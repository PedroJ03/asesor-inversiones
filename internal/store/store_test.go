package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func sampleQuotes() []fetch.Quote {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	return []fetch.Quote{
		{
			Source:    "yahoo",
			Symbol:    "SPY",
			Name:      "S&P 500 (SPY)",
			Price:     600.5,
			PrevClose: 595.0,
			ChangePct: 0.92,
			Currency:  "USD",
			QuotedAt:  now,
			FetchedAt: now,
		},
		{
			Source:    "dolarapi",
			Symbol:    "blue",
			Name:      "Blue",
			Price:     1545,
			Bid:       1525,
			Ask:       1545,
			Currency:  "ARS",
			QuotedAt:  now,
			FetchedAt: now,
		},
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nested", "data", "asesor.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}

func TestSaveRawAndQuotes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	raw := []byte(`{"test": "payload"}`)
	if err := s.SaveRaw("dolarapi", "https://dolarapi.com/v1/dolares", 200, raw, now); err != nil {
		t.Fatalf("save raw: %v", err)
	}

	quotes := sampleQuotes()
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	got, err := s.QuotesBySource("yahoo")
	if err != nil {
		t.Fatalf("quotes by source: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 yahoo quote, got %d", len(got))
	}
	if got[0].Symbol != "SPY" {
		t.Errorf("symbol: want SPY, got %s", got[0].Symbol)
	}
	if got[0].Name != "S&P 500 (SPY)" {
		t.Errorf("name: want %q, got %q", "S&P 500 (SPY)", got[0].Name)
	}
	if got[0].Price != 600.5 {
		t.Errorf("price: want 600.5, got %v", got[0].Price)
	}

	gotDolar, err := s.QuotesBySource("dolarapi")
	if err != nil {
		t.Fatalf("quotes by source dolarapi: %v", err)
	}
	if len(gotDolar) != 1 {
		t.Fatalf("want 1 dolarapi quote, got %d", len(gotDolar))
	}
	if gotDolar[0].Bid != 1525 {
		t.Errorf("bid: want 1525, got %v", gotDolar[0].Bid)
	}
}

func TestSaveQuotesEmpty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	if err := s.SaveQuotes(nil); err != nil {
		t.Fatalf("save empty quotes: %v", err)
	}
	got, err := s.QuotesBySource("any")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want 0 quotes, got %d", len(got))
	}
}

func TestCreateAndListRules(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now)
	if err != nil {
		t.Fatalf("create valid rule: %v", err)
	}
	if id <= 0 {
		t.Fatalf("want positive id, got %d", id)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	r := rules[0]
	if r.Source != "yahoo" || r.Symbol != "SPY" || r.Kind != "value" || r.Direction != "above" {
		t.Errorf("unexpected rule identity: %+v", r)
	}
	if r.Threshold != 600 || r.BaselinePrice != 595 {
		t.Errorf("unexpected threshold/baseline: %+v", r)
	}
	if !r.Enabled || r.State != "armed" {
		t.Errorf("want enabled and armed, got enabled=%v state=%s", r.Enabled, r.State)
	}
}

func TestCreateRuleValidation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		source    string
		symbol    string
		kind      string
		direction string
		threshold float64
		baseline  float64
	}{
		{"invalid source", "unknown", "SPY", "value", "above", 600, 595},
		{"invalid kind", "yahoo", "SPY", "delta", "above", 600, 595},
		{"invalid direction", "yahoo", "SPY", "value", "up", 600, 595},
		{"zero threshold", "yahoo", "SPY", "value", "above", 0, 595},
		{"negative threshold", "yahoo", "SPY", "value", "above", -5, 595},
		{"zero baseline", "yahoo", "SPY", "value", "above", 600, 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.CreateRule(tt.source, tt.symbol, tt.kind, tt.direction, tt.threshold, tt.baseline, now)
			if !errors.Is(err, ErrInvalidRule) {
				t.Fatalf("want ErrInvalidRule, got %v", err)
			}
		})
	}
}

func TestCreateRuleDuplicate(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	if _, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 590, now)
	if !errors.Is(err, ErrDuplicateRule) {
		t.Fatalf("want ErrDuplicateRule, got %v", err)
	}
}

func TestListRulesEnabledFilter(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	if _, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now); err != nil {
		t.Fatalf("create enabled rule: %v", err)
	}

	// Insert a disabled rule directly to test the filter without depending on future update methods.
	if _, err := s.db.Exec(`
		INSERT INTO alert_rules (source, symbol, kind, direction, threshold, baseline_price, enabled, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "dolarapi", "blue", "value", "below", 1500, 1600, 0, "armed", now.UTC()); err != nil {
		t.Fatalf("insert disabled rule: %v", err)
	}

	all, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 rules, got %d", len(all))
	}

	enabled, err := s.ListRules(true)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("want 1 enabled rule, got %d", len(enabled))
	}
	if enabled[0].Source != "yahoo" {
		t.Errorf("want yahoo rule, got %s", enabled[0].Source)
	}
}

func TestRemoveRulePreservesHistory(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	ruleID, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	alertID, err := s.InsertAlert(Alert{
		RuleID:        sql.NullInt64{Int64: ruleID, Valid: true},
		Source:        "yahoo",
		Symbol:        "SPY",
		Kind:          "value",
		Threshold:     600,
		ObservedPrice: sql.NullFloat64{Float64: 605, Valid: true},
		BaselinePrice: sql.NullFloat64{Float64: 595, Valid: true},
		TriggeredAt:   now,
	})
	if err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	if err := s.RemoveRule(ruleID); err != nil {
		t.Fatalf("remove rule: %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("want 0 rules, got %d", len(rules))
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want 1 history entry, got %d", len(history))
	}
	if history[0].ID != alertID {
		t.Errorf("want alert id %d, got %d", alertID, history[0].ID)
	}
	if history[0].RuleID.Valid {
		t.Errorf("want null rule_id after removal, got %d", history[0].RuleID.Int64)
	}
}

func TestLatestQuoteOrdering(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	older := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 590, FetchedAt: older},
		{Source: "yahoo", Symbol: "SPY", Price: 600, FetchedAt: newer},
	}
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	got, ok, err := s.LatestQuote("yahoo", "SPY")
	if err != nil {
		t.Fatalf("latest quote: %v", err)
	}
	if !ok {
		t.Fatalf("expected a latest quote")
	}
	if got.Price != 600 {
		t.Errorf("want newest price 600, got %v", got.Price)
	}
	if !got.FetchedAt.Equal(newer) {
		t.Errorf("want newest fetched_at, got %v", got.FetchedAt)
	}

	_, ok, err = s.LatestQuote("yahoo", "UNKNOWN")
	if err != nil {
		t.Fatalf("latest unknown: %v", err)
	}
	if ok {
		t.Errorf("want no quote for unknown symbol")
	}
}

func TestUpdateRuleState(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	ruleID, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if err := s.UpdateRuleState(ruleID, "triggered"); err != nil {
		t.Fatalf("update state: %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if rules[0].State != "triggered" {
		t.Errorf("want triggered, got %s", rules[0].State)
	}

	if err := s.UpdateRuleState(ruleID, "invalid"); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("want ErrInvalidRule, got %v", err)
	}
}

func TestHistoryLimit(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		if _, err := s.InsertAlert(Alert{
			Source:        "yahoo",
			Symbol:        "SPY",
			Kind:          "value",
			Threshold:     600,
			ObservedPrice: sql.NullFloat64{Float64: float64(600 + i), Valid: true},
			TriggeredAt:   now.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Fatalf("insert alert %d: %v", i, err)
		}
	}

	history, err := s.History(2)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("want 2 history entries, got %d", len(history))
	}
	if !history[0].TriggeredAt.Equal(now.Add(2 * time.Hour)) {
		t.Errorf("want newest entry first, got %v", history[0].TriggeredAt)
	}

	_, err = s.History(0)
	if !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("want ErrInvalidRule for zero limit, got %v", err)
	}
}
