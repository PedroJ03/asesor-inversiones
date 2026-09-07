package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func writeWatchlist(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "watchlist.yaml")
	data := `usa:
  - symbol: SPY
  - symbol: QQQ
dolares:
  - oficial
  - blue
bonos:
  - AL30D
  - GD30D
cripto:
  - id: bitcoin
  - id: ethereum
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write watchlist: %v", err)
	}
	return path
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

func TestAddValidCapturesBaseline(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	wl := writeWatchlist(t, t.TempDir())
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 600.5, FetchedAt: now},
	})

	if err := runAdd([]string{
		"-db", dbPath,
		"-watchlist", wl,
		"-source", "yahoo",
		"-symbol", "SPY",
		"-kind", "value",
		"-direction", "above",
		"-threshold", "590",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	if rules[0].BaselinePrice != 600.5 {
		t.Errorf("baseline: want 600.5, got %v", rules[0].BaselinePrice)
	}
}

func TestAddRejectsMissingQuote(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	wl := writeWatchlist(t, t.TempDir())

	err := runAdd([]string{
		"-db", dbPath,
		"-watchlist", wl,
		"-source", "yahoo",
		"-symbol", "SPY",
		"-kind", "value",
		"-direction", "above",
		"-threshold", "590",
	})
	if err == nil {
		t.Fatal("expected error for missing quote")
	}
	if !strings.Contains(err.Error(), "no stored quote") {
		t.Errorf("want missing quote error, got %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}
}

func TestAddRejectsWatchlist(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	wl := writeWatchlist(t, t.TempDir())
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "TSLA", Price: 300, FetchedAt: now},
	})

	err := runAdd([]string{
		"-db", dbPath,
		"-watchlist", wl,
		"-source", "yahoo",
		"-symbol", "TSLA",
		"-kind", "value",
		"-direction", "above",
		"-threshold", "290",
	})
	if err == nil {
		t.Fatal("expected error for non-watchlisted symbol")
	}
	if !strings.Contains(err.Error(), "not in watchlist") {
		t.Errorf("want watchlist error, got %v", err)
	}
}

func TestAddInvalidFlags(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	wl := writeWatchlist(t, t.TempDir())
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 600, FetchedAt: now},
	})

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "invalid source",
			args: []string{"-db", dbPath, "-watchlist", wl, "-source", "binance", "-symbol", "SPY", "-kind", "value", "-direction", "above", "-threshold", "590"},
		},
		{
			name: "invalid kind",
			args: []string{"-db", dbPath, "-watchlist", wl, "-source", "yahoo", "-symbol", "SPY", "-kind", "delta", "-direction", "above", "-threshold", "590"},
		},
		{
			name: "invalid direction",
			args: []string{"-db", dbPath, "-watchlist", wl, "-source", "yahoo", "-symbol", "SPY", "-kind", "value", "-direction", "up", "-threshold", "590"},
		},
		{
			name: "zero threshold",
			args: []string{"-db", dbPath, "-watchlist", wl, "-source", "yahoo", "-symbol", "SPY", "-kind", "value", "-direction", "above", "-threshold", "0"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := runAdd(tt.args); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf strings.Builder
		b := make([]byte, 1024)
		for {
			n, err := r.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if err != nil {
				break
			}
		}
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	out := <-done
	_ = r.Close()
	os.Stdout = stdout
	return out
}

func TestListDisplaysState(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	if _, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runList([]string{"-db", dbPath}); err != nil {
			t.Fatalf("list: %v", err)
		}
	})

	if !strings.Contains(out, "SPY") {
		t.Errorf("output should contain SPY, got %q", out)
	}
	if !strings.Contains(out, "armed") {
		t.Errorf("output should contain state, got %q", out)
	}
}

func TestRemovePreservesHistory(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	ruleID, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 605, FetchedAt: now},
	})

	if err := runAlerts([]string{"-db", dbPath, "-max-age", "24h"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if err := runRemove([]string{"-db", dbPath, "-id", "0"}); err == nil {
		t.Fatal("expected error for invalid id")
	}

	if err := runRemove([]string{"-db", dbPath, "-id", "999"}); err == nil {
		t.Fatal("expected error for missing rule")
	}

	if err := runRemove([]string{"-db", dbPath, "-id", strconv.FormatInt(ruleID, 10)}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("want 1 history entry, got %d", len(history))
	}
	if history[0].RuleID.Valid {
		t.Errorf("rule_id should be null after removal")
	}
}

func TestHistoryOutput(t *testing.T) {
	t.Parallel()
	dbPath, s := newTestStore(t)
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	ruleID, err := s.CreateRule("yahoo", "SPY", "value", "above", 600, 595, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	seedQuotes(t, s, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 605, FetchedAt: now},
	})

	if err := runAlerts([]string{"-db", dbPath, "-max-age", "24h"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if err := runHistory([]string{"-db", dbPath, "-limit", "0"}); err == nil {
		t.Fatal("expected error for invalid limit")
	}

	out := captureStdout(t, func() {
		if err := runHistory([]string{"-db", dbPath, "-limit", "10"}); err != nil {
			t.Fatalf("history: %v", err)
		}
	})

	if !strings.Contains(out, "SPY") {
		t.Errorf("output should contain SPY, got %q", out)
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 || history[0].RuleID.Int64 != ruleID {
		t.Errorf("history should reference rule %d", ruleID)
	}
}
