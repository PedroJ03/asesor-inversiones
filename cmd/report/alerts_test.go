package main

import (
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

func testReportStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir() + "/report-test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// TestEvaluateAlerts_PersistsTriggerAndStateFlips seeds a temp store with an
// NVDA value rule above threshold plus a matching fresh quote, then asserts
// the wiring end to end: alert inserted, rule state flipped to triggered.
func TestEvaluateAlerts_PersistsTriggerAndStateFlips(t *testing.T) {
	st := testReportStore(t)
	if err := st.SaveQuotes([]fetch.Quote{{
		Source:    "yahoo",
		Symbol:    "NVDA",
		Price:     505,
		ChangePct: 1.0,
		Currency:  "USD",
		FetchedAt: time.Now().UTC().Add(-time.Hour),
	}}); err != nil {
		t.Fatalf("save quotes: %v", err)
	}
	if _, err := st.CreateRule("yahoo", "NVDA", "value", "above", 500, 480, time.Now().UTC()); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if err := evaluateAlerts(st); err != nil {
		t.Fatalf("evaluateAlerts: %v", err)
	}

	history, err := st.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want 1 inserted alert, got %d", len(history))
	}
	if history[0].Symbol != "NVDA" {
		t.Errorf("want alert for NVDA, got %s/%s", history[0].Source, history[0].Symbol)
	}
	if !history[0].ObservedPrice.Valid || history[0].ObservedPrice.Float64 != 505 {
		t.Errorf("want observed price 505, got %+v", history[0].ObservedPrice)
	}

	rules, err := st.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 || rules[0].State != "triggered" {
		t.Fatalf("want rule state triggered, got %+v", rules)
	}
}

// TestEvaluateAlerts_IdempotentAndRearms runs the orchestrator a second time
// while the condition still holds (no duplicate alert), then refreshes the
// quote below the threshold and asserts the rule returns to armed.
func TestEvaluateAlerts_IdempotentAndRearms(t *testing.T) {
	st := testReportStore(t)
	now := time.Now().UTC()
	if err := st.SaveQuotes([]fetch.Quote{{
		Source:    "yahoo",
		Symbol:    "NVDA",
		Price:     505,
		ChangePct: 1.0,
		Currency:  "USD",
		FetchedAt: now.Add(-time.Hour),
	}}); err != nil {
		t.Fatalf("save quotes: %v", err)
	}
	if _, err := st.CreateRule("yahoo", "NVDA", "value", "above", 500, 480, now); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if err := evaluateAlerts(st); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// Second run with the still-triggered condition must not duplicate alerts.
	if err := evaluateAlerts(st); err != nil {
		t.Fatalf("second run: %v", err)
	}
	history, err := st.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want exactly 1 alert after repeated runs, got %d", len(history))
	}

	// A newer quote below the threshold stops the condition: rule re-arms.
	if err := st.SaveQuotes([]fetch.Quote{{
		Source:    "yahoo",
		Symbol:    "NVDA",
		Price:     470,
		ChangePct: -1.0,
		Currency:  "USD",
		FetchedAt: now,
	}}); err != nil {
		t.Fatalf("save updated quote: %v", err)
	}
	if err := evaluateAlerts(st); err != nil {
		t.Fatalf("re-arm run: %v", err)
	}

	rules, err := st.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 || rules[0].State != "armed" {
		t.Fatalf("want rule re-armed, got %+v", rules)
	}
	history, err = st.History(10)
	if err != nil {
		t.Fatalf("history after re-arm: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("re-arm must not insert alerts, got %d", len(history))
	}
}

// TestEvaluateAlerts_DisabledRulesSkipped checks the enabled-only filter.
func TestEvaluateAlerts_DisabledRulesSkipped(t *testing.T) {
	st := testReportStore(t)
	now := time.Now().UTC()
	if err := st.SaveQuotes([]fetch.Quote{{
		Source:    "yahoo",
		Symbol:    "NVDA",
		Price:     505,
		ChangePct: 1.0,
		Currency:  "USD",
		FetchedAt: now.Add(-time.Hour),
	}}); err != nil {
		t.Fatalf("save quotes: %v", err)
	}
	id, err := st.CreateRule("yahoo", "NVDA", "value", "above", 500, 480, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := st.SetRuleEnabled(id, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}

	if err := evaluateAlerts(st); err != nil {
		t.Fatalf("evaluateAlerts: %v", err)
	}

	history, err := st.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("disabled rule must not trigger, got %d alerts", len(history))
	}
	rules, err := st.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if rules[0].State != "armed" {
		t.Fatalf("disabled rule state must stay armed, got %q", rules[0].State)
	}
}
