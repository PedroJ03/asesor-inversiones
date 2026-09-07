package alert

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func fixedResolver(quotes map[string]Quote) func(string, string) (Quote, bool, error) {
	return func(source, symbol string) (Quote, bool, error) {
		key := source + "/" + symbol
		q, ok := quotes[key]
		if !ok {
			return Quote{}, false, nil
		}
		return q, true, nil
	}
}

func errorResolver(err error) func(string, string) (Quote, bool, error) {
	return func(string, string) (Quote, bool, error) {
		return Quote{}, false, err
	}
}

func TestEvaluateValueBoundaries(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)

	cases := []struct {
		name      string
		direction string
		threshold float64
		price     float64
		want      bool
	}{
		{"value above exact", "above", 600, 600, true},
		{"value above higher", "above", 600, 601, true},
		{"value above lower", "above", 600, 599, false},
		{"value below exact", "below", 600, 600, true},
		{"value below lower", "below", 600, 599, true},
		{"value below higher", "below", 600, 601, false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rules := []Rule{{
				ID:        1,
				Source:    "yahoo",
				Symbol:    "SPY",
				Kind:      "value",
				Direction: tt.direction,
				Threshold: tt.threshold,
				State:     "armed",
			}}
			resolve := fixedResolver(map[string]Quote{"yahoo/SPY": {Price: tt.price, FetchedAt: fresh}})
			res := Evaluate(rules, resolve, now, 24*time.Hour)

			if tt.want {
				if len(res.Triggered) != 1 {
					t.Fatalf("want 1 trigger, got %d", len(res.Triggered))
				}
				if len(res.Transitions) != 1 || res.Transitions[0].To != "triggered" {
					t.Fatalf("want armed->triggered transition, got %+v", res.Transitions)
				}
			} else {
				if len(res.Triggered) != 0 || len(res.Transitions) != 0 {
					t.Fatalf("want no trigger/transition, got triggered=%d transitions=%d", len(res.Triggered), len(res.Transitions))
				}
			}
		})
	}
}

func TestEvaluatePercentageBoundaries(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)

	cases := []struct {
		name      string
		direction string
		threshold float64
		price     float64
		want      bool
		wantPct   float64
	}{
		{"pct above exact", "above", 5, 105, true, 5},
		{"pct above higher", "above", 5, 106, true, 6},
		{"pct above lower", "above", 5, 104.99, false, 0},
		{"pct below exact", "below", 5, 95, true, -5},
		{"pct below lower", "below", 5, 94, true, -6},
		{"pct below higher", "below", 5, 96, false, 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rules := []Rule{{
				ID:            1,
				Source:        "yahoo",
				Symbol:        "SPY",
				Kind:          "pct",
				Direction:     tt.direction,
				Threshold:     tt.threshold,
				BaselinePrice: 100,
				State:         "armed",
			}}
			resolve := fixedResolver(map[string]Quote{"yahoo/SPY": {Price: tt.price, FetchedAt: fresh}})
			res := Evaluate(rules, resolve, now, 24*time.Hour)

			if tt.want {
				if len(res.Triggered) != 1 {
					t.Fatalf("want 1 trigger, got %d", len(res.Triggered))
				}
				if res.Triggered[0].ObservedChangePct != tt.wantPct {
					t.Errorf("want observed pct %v, got %v", tt.wantPct, res.Triggered[0].ObservedChangePct)
				}
			} else {
				if len(res.Triggered) != 0 {
					t.Fatalf("want no trigger, got %d", len(res.Triggered))
				}
			}
		})
	}
}

func TestEvaluateDedupeAndRearmCycle(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)

	rule := Rule{
		ID:        1,
		Source:    "yahoo",
		Symbol:    "SPY",
		Kind:      "value",
		Direction: "above",
		Threshold: 600,
		State:     "armed",
	}
	resolveUp := fixedResolver(map[string]Quote{"yahoo/SPY": {Price: 605, FetchedAt: fresh}})
	resolveDown := fixedResolver(map[string]Quote{"yahoo/SPY": {Price: 595, FetchedAt: fresh}})

	// First evaluation: armed + true -> trigger and transition.
	res1 := Evaluate([]Rule{rule}, resolveUp, now, 24*time.Hour)
	if len(res1.Triggered) != 1 || len(res1.Transitions) != 1 {
		t.Fatalf("first eval want trigger+transition, got %+v", res1)
	}

	// Second evaluation: triggered + true -> no-op.
	rule.State = "triggered"
	res2 := Evaluate([]Rule{rule}, resolveUp, now, 24*time.Hour)
	if len(res2.Triggered) != 0 || len(res2.Transitions) != 0 {
		t.Fatalf("dedupe want no trigger/transition, got %+v", res2)
	}

	// Third evaluation: triggered + false -> re-arm silently.
	res3 := Evaluate([]Rule{rule}, resolveDown, now, 24*time.Hour)
	if len(res3.Triggered) != 0 {
		t.Fatalf("re-arm want no trigger, got %d", len(res3.Triggered))
	}
	if len(res3.Transitions) != 1 || res3.Transitions[0].To != "armed" {
		t.Fatalf("re-arm want triggered->armed, got %+v", res3.Transitions)
	}
}

func TestEvaluateGuards(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		resolve  func(string, string) (Quote, bool, error)
		wantWarn string
	}{
		{
			name: "missing quote",
			resolve: func(string, string) (Quote, bool, error) {
				return Quote{}, false, nil
			},
			wantWarn: "no quote available",
		},
		{
			name:     "stale quote",
			resolve:  fixedResolver(map[string]Quote{"yahoo/SPY": {Price: 605, FetchedAt: now.Add(-25 * time.Hour)}}),
			wantWarn: "quote stale",
		},
		{
			name:     "resolve error",
			resolve:  errorResolver(errors.New("boom")),
			wantWarn: "resolve error",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rules := []Rule{{
				ID:        1,
				Source:    "yahoo",
				Symbol:    "SPY",
				Kind:      "value",
				Direction: "above",
				Threshold: 600,
				State:     "armed",
			}}
			res := Evaluate(rules, tt.resolve, now, 24*time.Hour)
			if len(res.Triggered) != 0 || len(res.Transitions) != 0 {
				t.Fatalf("want no trigger/transition, got triggered=%d transitions=%d", len(res.Triggered), len(res.Transitions))
			}
			if len(res.Warnings) != 1 {
				t.Fatalf("want 1 warning, got %d", len(res.Warnings))
			}
			if !strings.Contains(res.Warnings[0], tt.wantWarn) {
				t.Errorf("want warning containing %q, got %q", tt.wantWarn, res.Warnings[0])
			}
		})
	}
}

func TestEvaluateOrphanDoesNotAbort(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Hour)

	rules := []Rule{
		{ID: 1, Source: "yahoo", Symbol: "SPY", Kind: "value", Direction: "above", Threshold: 600, State: "armed"},
		{ID: 2, Source: "yahoo", Symbol: "ORPHAN", Kind: "value", Direction: "above", Threshold: 1, State: "armed"},
	}
	// Only SPY is resolvable; ORPHAN is missing.
	resolve := fixedResolver(map[string]Quote{
		"yahoo/SPY": {Price: 605, FetchedAt: fresh},
	})

	res := Evaluate(rules, resolve, now, 24*time.Hour)
	if len(res.Triggered) != 1 || res.Triggered[0].Symbol != "SPY" {
		t.Fatalf("want SPY trigger only, got %+v", res.Triggered)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "ORPHAN") {
		t.Fatalf("want orphan warning, got %+v", res.Warnings)
	}
}
