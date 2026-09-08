package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestSetRuleEnabled_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, time.Now())
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	if err := s.SetRuleEnabled(id, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Enabled {
		t.Fatalf("expected rule disabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state unchanged armed, got %q", rules[0].State)
	}

	if err := s.SetRuleEnabled(id, true); err != nil {
		t.Fatalf("enable rule: %v", err)
	}
	rules, err = s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if !rules[0].Enabled {
		t.Fatalf("expected rule enabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state still armed, got %q", rules[0].State)
	}
}

func TestSetRuleEnabled_MissingID(t *testing.T) {
	s := newTestStore(t)

	err := s.SetRuleEnabled(999, false)
	if err == nil {
		t.Fatalf("expected error for missing rule")
	}
	if !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("expected ErrInvalidRule, got %v", err)
	}
}

func TestSetRuleEnabled_ListRulesExcludesDisabled(t *testing.T) {
	s := newTestStore(t)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, time.Now())
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	all, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list all rules: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 rule in unfiltered list, got %d", len(all))
	}

	enabledOnly, err := s.ListRules(true)
	if err != nil {
		t.Fatalf("list enabled rules: %v", err)
	}
	if len(enabledOnly) != 1 {
		t.Fatalf("expected 1 enabled rule, got %d", len(enabledOnly))
	}

	if err := s.SetRuleEnabled(id, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}

	all, err = s.ListRules(false)
	if err != nil {
		t.Fatalf("list all rules after disable: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 rule in unfiltered list after disable, got %d", len(all))
	}

	enabledOnly, err = s.ListRules(true)
	if err != nil {
		t.Fatalf("list enabled rules after disable: %v", err)
	}
	if len(enabledOnly) != 0 {
		t.Fatalf("expected 0 enabled rules after disable, got %d", len(enabledOnly))
	}
}

func TestSetRuleEnabled_HistoryUntouched(t *testing.T) {
	s := newTestStore(t)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, time.Now())
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	_, err = s.InsertAlert(Alert{
		RuleID:      sql.NullInt64{Int64: id, Valid: true},
		Source:      "yahoo",
		Symbol:      "SPY",
		Kind:        "value",
		Threshold:   550,
		TriggeredAt: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	if err := s.SetRuleEnabled(id, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected history untouched, got %d rows", len(history))
	}
}
