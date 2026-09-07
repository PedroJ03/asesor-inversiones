package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

func newTestStore(t *testing.T) (string, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return path, s
}

func seedQuotes(t *testing.T, s *store.Store, quotes []fetch.Quote) {
	t.Helper()
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("seed quotes: %v", err)
	}
}

func TestRunTriggersAndWarnings(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 605, FetchedAt: now},
		{Source: "yahoo", Symbol: "QQQ", Price: 500, FetchedAt: now.Add(-25 * time.Hour)},
	})

	if _, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now); err != nil {
		t.Fatalf("create spy rule: %v", err)
	}
	if _, err := s.CreateRule("yahoo", "QQQ", "value", "above", 600, 595, now); err != nil {
		t.Fatalf("create qqq rule: %v", err)
	}

	if err := runAlerts([]string{"-db", dbPath, "-max-age", "24h"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(rules))
	}
	for _, r := range rules {
		if r.Symbol == "SPY" && r.State != "triggered" {
			t.Errorf("SPY should be triggered, got %s", r.State)
		}
		if r.Symbol == "QQQ" && r.State != "armed" {
			t.Errorf("QQQ should remain armed, got %s", r.State)
		}
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want 1 triggered alert, got %d", len(history))
	}
	if history[0].Symbol != "SPY" {
		t.Errorf("want SPY alert, got %s", history[0].Symbol)
	}
}

func TestRunStoreFailureExitsNonZero(t *testing.T) {
	t.Parallel()
	if err := runAlerts([]string{"-db", "/nonexistent/path/to/db/asesor.db", "-max-age", "24h"}); err == nil {
		t.Fatal("expected error for invalid db path")
	}
}
