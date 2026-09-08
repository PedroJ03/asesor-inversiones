package store

import (
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
)

func sampleHistoryQuotes() []fetch.Quote {
	base := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	return []fetch.Quote{
		{
			Source:    "yahoo",
			Symbol:    "SPY",
			Name:      "S&P 500 (SPY)",
			Price:     600.5,
			PrevClose: 595.0,
			ChangePct: 0.92,
			Currency:  "USD",
			QuotedAt:  base,
			FetchedAt: base.Add(-30 * time.Minute),
		},
		{
			Source:    "yahoo",
			Symbol:    "SPY",
			Name:      "S&P 500 (SPY)",
			Price:     601.0,
			PrevClose: 595.0,
			ChangePct: 1.01,
			Currency:  "USD",
			QuotedAt:  base,
			FetchedAt: base,
		},
		{
			Source:    "dolarapi",
			Symbol:    "blue",
			Name:      "Blue",
			Price:     1545,
			Bid:       1525,
			Ask:       1545,
			Currency:  "ARS",
			QuotedAt:  base,
			FetchedAt: base.Add(-15 * time.Minute),
		},
		{
			Source:    "dolarapi",
			Symbol:    "blue",
			Name:      "Blue",
			Price:     1550,
			Bid:       1530,
			Ask:       1550,
			Currency:  "ARS",
			QuotedAt:  base,
			FetchedAt: base,
		},
	}
}

func TestWebLatestSnapshots_NewestPerPair(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	if err := s.SaveQuotes(sampleHistoryQuotes()); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	pairs := []WebQuoteKey{
		{Source: "yahoo", Symbol: "SPY"},
		{Source: "dolarapi", Symbol: "blue"},
	}
	got, err := s.LatestSnapshots(pairs)
	if err != nil {
		t.Fatalf("latest snapshots: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("want 2 snapshots, got %d", len(got))
	}

	spy := got[WebQuoteKey{Source: "yahoo", Symbol: "SPY"}]
	if spy.Price != 601.0 {
		t.Errorf("SPY price: want 601.0, got %v", spy.Price)
	}

	blue := got[WebQuoteKey{Source: "dolarapi", Symbol: "blue"}]
	if blue.Price != 1550 {
		t.Errorf("blue price: want 1550, got %v", blue.Price)
	}
}

func TestWebLatestSnapshots_MissingPairIsolated(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	if err := s.SaveQuotes(sampleHistoryQuotes()); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	pairs := []WebQuoteKey{
		{Source: "yahoo", Symbol: "SPY"},
		{Source: "missing", Symbol: "MISSING"},
	}
	got, err := s.LatestSnapshots(pairs)
	if err != nil {
		t.Fatalf("latest snapshots: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("want 1 snapshot, got %d", len(got))
	}
	if _, ok := got[WebQuoteKey{Source: "yahoo", Symbol: "SPY"}]; !ok {
		t.Errorf("expected SPY snapshot to be present")
	}
	if _, ok := got[WebQuoteKey{Source: "missing", Symbol: "MISSING"}]; ok {
		t.Errorf("expected missing pair to be absent")
	}
}

func TestWebLatestSnapshots_EmptyBatchNoQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	// Seed data exists, but an empty batch must not return it.
	if err := s.SaveQuotes(sampleHistoryQuotes()); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	got, err := s.LatestSnapshots(nil)
	if err != nil {
		t.Fatalf("latest snapshots: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map, got %d entries", len(got))
	}

	got, err = s.LatestSnapshots([]WebQuoteKey{})
	if err != nil {
		t.Fatalf("latest snapshots: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map, got %d entries", len(got))
	}
}

func TestWebQuoteHistory_Range(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	quotes := sampleHistoryQuotes()
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	base := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	from := base.Add(-35 * time.Minute)
	to := base.Add(-15 * time.Minute)

	got, err := s.QuoteHistory("yahoo", "SPY", from, to, WebHistoryDefaultLimit)
	if err != nil {
		t.Fatalf("quote history: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("want 1 row in range, got %d", len(got))
	}
	if got[0].Price != 600.5 {
		t.Errorf("price: want 600.5, got %v", got[0].Price)
	}
}

func TestWebQuoteHistory_Bounds(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	base := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 598.0, FetchedAt: base.Add(-3 * time.Hour)},
		{Source: "yahoo", Symbol: "SPY", Price: 599.0, FetchedAt: base.Add(-2 * time.Hour)},
		{Source: "yahoo", Symbol: "SPY", Price: 600.0, FetchedAt: base.Add(-1 * time.Hour)},
		{Source: "yahoo", Symbol: "SPY", Price: 601.0, FetchedAt: base},
	}
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	got, err := s.QuoteHistory("yahoo", "SPY", base.Add(-4*time.Hour), base, 2)
	if err != nil {
		t.Fatalf("quote history: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows with limit 2, got %d", len(got))
	}
	if got[0].Price != 598.0 {
		t.Errorf("first price: want 598.0, got %v", got[0].Price)
	}
	if got[1].Price != 599.0 {
		t.Errorf("second price: want 599.0, got %v", got[1].Price)
	}
}

func TestWebQuoteHistory_DeterministicOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	base := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 100.0, FetchedAt: base},
		{Source: "yahoo", Symbol: "SPY", Price: 200.0, FetchedAt: base},
	}
	if err := s.SaveQuotes(quotes); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	got, err := s.QuoteHistory("yahoo", "SPY", base.Add(-time.Hour), base.Add(time.Hour), WebHistoryDefaultLimit)
	if err != nil {
		t.Fatalf("quote history: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 rows, got %d", len(got))
	}
	// Same fetched_at must order by id ascending, so the first inserted row is
	// returned first.
	if got[0].Price != 100.0 || got[1].Price != 200.0 {
		t.Errorf("order mismatch: got %v then %v", got[0].Price, got[1].Price)
	}
}

func TestWebQuoteHistory_InvalidLimitsAndRanges(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	base := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	if err := s.SaveQuotes(sampleHistoryQuotes()); err != nil {
		t.Fatalf("save quotes: %v", err)
	}

	cases := []struct {
		name   string
		from   time.Time
		to     time.Time
		limit  int
		wantOK bool
	}{
		{"zero limit", base.Add(-time.Hour), base.Add(time.Hour), 0, false},
		{"negative limit", base.Add(-time.Hour), base.Add(time.Hour), -1, false},
		{"over max limit", base.Add(-time.Hour), base.Add(time.Hour), WebHistoryMaxLimit + 1, false},
		{"reversed range", base.Add(time.Hour), base.Add(-time.Hour), WebHistoryDefaultLimit, false},
		{"zero from", time.Time{}, base, WebHistoryDefaultLimit, false},
		{"zero to", base, time.Time{}, WebHistoryDefaultLimit, false},
		{"valid", base.Add(-time.Hour), base.Add(time.Hour), WebHistoryDefaultLimit, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.QuoteHistory("yahoo", "SPY", tc.from, tc.to, tc.limit)
			if tc.wantOK && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestWebQuoteHistoryConstants(t *testing.T) {
	if WebHistoryDefaultLimit != 100 {
		t.Errorf("WebHistoryDefaultLimit: want 100, got %d", WebHistoryDefaultLimit)
	}
	if WebHistoryMaxLimit != 500 {
		t.Errorf("WebHistoryMaxLimit: want 500, got %d", WebHistoryMaxLimit)
	}
}
