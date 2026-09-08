package alert

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// fakePersister records persisted outcomes and can be made to fail.
type fakePersister struct {
	alerts      []Alert
	transitions []StateChange
	failInsert  bool
	failState   bool
}

func (f *fakePersister) InsertTriggered(a Alert) error {
	if f.failInsert {
		return errors.New("insert failed")
	}
	f.alerts = append(f.alerts, a)
	return nil
}

func (f *fakePersister) TransitionState(ruleID int64, to string) error {
	if f.failState {
		return errors.New("state update failed")
	}
	f.transitions = append(f.transitions, StateChange{RuleID: ruleID, To: to})
	return nil
}

func runnerTestRule(state string) Rule {
	return Rule{
		ID:        7,
		Source:    "yahoo",
		Symbol:    "NVDA",
		Kind:      "value",
		Direction: "above",
		Threshold: 500,
		State:     state,
	}
}

func TestRun_PersistsTriggerAndTransition(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	resolve := fixedResolver(map[string]Quote{"yahoo/NVDA": {Price: 505, FetchedAt: now.Add(-time.Hour)}})
	persist := &fakePersister{}

	result, err := Run([]Rule{runnerTestRule("armed")}, resolve, persist, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(persist.alerts) != 1 {
		t.Fatalf("want 1 inserted alert, got %d", len(persist.alerts))
	}
	if persist.alerts[0].Symbol != "NVDA" || persist.alerts[0].ObservedPrice != 505 {
		t.Errorf("unexpected alert snapshot: %+v", persist.alerts[0])
	}
	if len(persist.transitions) != 1 || persist.transitions[0].RuleID != 7 || persist.transitions[0].To != "triggered" {
		t.Fatalf("want armed->triggered transition, got %+v", persist.transitions)
	}
	if len(result.Triggered) != 1 {
		t.Fatalf("want 1 triggered in result, got %d", len(result.Triggered))
	}
}

func TestRun_IdempotentWhileConditionHolds(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	resolve := fixedResolver(map[string]Quote{"yahoo/NVDA": {Price: 505, FetchedAt: now.Add(-time.Hour)}})
	persist := &fakePersister{}

	result, err := Run([]Rule{runnerTestRule("triggered")}, resolve, persist, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(persist.alerts) != 0 || len(persist.transitions) != 0 {
		t.Fatalf("still-triggered condition must not duplicate alerts, got alerts=%d transitions=%d",
			len(persist.alerts), len(persist.transitions))
	}
	if len(result.Triggered) != 0 {
		t.Fatalf("want no triggers on re-evaluation, got %d", len(result.Triggered))
	}
}

func TestRun_RearmsWhenConditionStopsHolding(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	resolve := fixedResolver(map[string]Quote{"yahoo/NVDA": {Price: 480, FetchedAt: now.Add(-time.Hour)}})
	persist := &fakePersister{}

	result, err := Run([]Rule{runnerTestRule("triggered")}, resolve, persist, now, 24*time.Hour)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(persist.alerts) != 0 {
		t.Fatalf("re-arm must not insert alerts, got %d", len(persist.alerts))
	}
	if len(persist.transitions) != 1 || persist.transitions[0].To != "armed" {
		t.Fatalf("want triggered->armed transition, got %+v", persist.transitions)
	}
	if len(result.Transitions) != 1 || result.Transitions[0].From != "triggered" {
		t.Fatalf("want re-arm in result, got %+v", result.Transitions)
	}
}

func TestRun_PersistFailureReturnsError(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	resolve := fixedResolver(map[string]Quote{"yahoo/NVDA": {Price: 505, FetchedAt: now.Add(-time.Hour)}})

	t.Run("insert failure", func(t *testing.T) {
		persist := &fakePersister{failInsert: true}
		_, err := Run([]Rule{runnerTestRule("armed")}, resolve, persist, now, 24*time.Hour)
		if err == nil || !strings.Contains(err.Error(), "persist alert") {
			t.Fatalf("want insert persistence error, got %v", err)
		}
	})

	t.Run("state failure", func(t *testing.T) {
		persist := &fakePersister{failState: true}
		_, err := Run([]Rule{runnerTestRule("armed")}, resolve, persist, now, 24*time.Hour)
		if err == nil || !strings.Contains(err.Error(), "persist rule") {
			t.Fatalf("want state persistence error, got %v", err)
		}
	})
}
