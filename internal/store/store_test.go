package store

import (
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
