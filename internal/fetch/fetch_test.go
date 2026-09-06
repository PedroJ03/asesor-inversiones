package fetch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseYahooOne(t *testing.T) {
	t.Parallel()

	body := loadFixture(t, "yahoo.json")
	labels := map[string]string{"AAPL": "Apple"}
	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	q, raw, err := parseYahooOne(body, labels, now)
	if err != nil {
		t.Fatalf("parseYahooOne: %v", err)
	}

	if q.Symbol != "AAPL" {
		t.Errorf("symbol: want AAPL, got %s", q.Symbol)
	}
	if q.Name != "Apple Inc." {
		t.Errorf("name: want %q, got %q", "Apple Inc.", q.Name)
	}
	if q.Price != 328.21 {
		t.Errorf("price: want 328.21, got %v", q.Price)
	}
	if q.PrevClose != 325.0 {
		t.Errorf("prevClose: want 325.0, got %v", q.PrevClose)
	}
	if q.ChangePct != 1.0 {
		t.Errorf("changePct: want 1.0, got %v", q.ChangePct)
	}
	if q.Currency != "USD" {
		t.Errorf("currency: want USD, got %s", q.Currency)
	}
	if q.Source != "yahoo" {
		t.Errorf("source: want yahoo, got %s", q.Source)
	}
	if raw == nil {
		t.Error("expected raw map, got nil")
	}
}

func TestParseDolarAPI(t *testing.T) {
	t.Parallel()

	body := loadFixture(t, "dolarapi.json")
	casas := []string{"oficial", "blue"}
	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	quotes, err := parseDolarAPI(body, casas, now)
	if err != nil {
		t.Fatalf("parseDolarAPI: %v", err)
	}

	if len(quotes) != 2 {
		t.Fatalf("want 2 quotes, got %d", len(quotes))
	}

	oficial := quotes[0]
	if oficial.Symbol != "oficial" {
		t.Errorf("symbol: want oficial, got %s", oficial.Symbol)
	}
	if oficial.Price != 1530 {
		t.Errorf("price (venta): want 1530, got %v", oficial.Price)
	}
	if oficial.Bid != 1480 {
		t.Errorf("bid (compra): want 1480, got %v", oficial.Bid)
	}
	if oficial.Ask != 1530 {
		t.Errorf("ask (venta): want 1530, got %v", oficial.Ask)
	}
	if oficial.Currency != "ARS" {
		t.Errorf("currency: want ARS, got %s", oficial.Currency)
	}
	if !oficial.QuotedAt.Equal(time.Date(2026, 9, 3, 18, 55, 0, 0, time.UTC)) {
		t.Errorf("quotedAt: got %v", oficial.QuotedAt)
	}
}

func TestParseData912(t *testing.T) {
	t.Parallel()

	body := loadFixture(t, "data912.json")
	symbols := []string{"AL30D", "AE38C"}
	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	quotes, err := parseData912(body, symbols, now)
	if err != nil {
		t.Fatalf("parseData912: %v", err)
	}

	if len(quotes) != 2 {
		t.Fatalf("want 2 quotes, got %d", len(quotes))
	}

	// Map by symbol to avoid ordering assumptions.
	bySymbol := make(map[string]Quote)
	for _, q := range quotes {
		bySymbol[q.Symbol] = q
	}

	al30d, ok := bySymbol["AL30D"]
	if !ok {
		t.Fatal("missing AL30D quote")
	}
	if al30d.Currency != "ARS" {
		t.Errorf("AL30D currency: want ARS, got %s", al30d.Currency)
	}
	if al30d.ChangePct != -0.5 {
		t.Errorf("AL30D changePct: want -0.5, got %v", al30d.ChangePct)
	}
	wantPrev := 955.0 / (1 + -0.5/100)
	if al30d.PrevClose != wantPrev {
		t.Errorf("AL30D prevClose: want %v, got %v", wantPrev, al30d.PrevClose)
	}

	ae38c, ok := bySymbol["AE38C"]
	if !ok {
		t.Fatal("missing AE38C quote")
	}
	if ae38c.Currency != "USD" {
		t.Errorf("AE38C currency: want USD, got %s", ae38c.Currency)
	}
}

func TestParseCoinGecko(t *testing.T) {
	t.Parallel()

	body := loadFixture(t, "coingecko.json")
	ids := []string{"bitcoin", "ethereum"}
	labels := map[string]string{"bitcoin": "Bitcoin", "ethereum": "Ethereum"}
	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	quotes, err := parseCoinGecko(body, ids, labels, now)
	if err != nil {
		t.Fatalf("parseCoinGecko: %v", err)
	}

	if len(quotes) != 2 {
		t.Fatalf("want 2 quotes, got %d", len(quotes))
	}

	bySymbol := make(map[string]Quote)
	for _, q := range quotes {
		bySymbol[q.Symbol] = q
	}

	btc, ok := bySymbol["bitcoin"]
	if !ok {
		t.Fatal("missing bitcoin quote")
	}
	if btc.Price != 81155 {
		t.Errorf("bitcoin price: want 81155, got %v", btc.Price)
	}
	if btc.Currency != "USD" {
		t.Errorf("bitcoin currency: want USD, got %s", btc.Currency)
	}
	if btc.ChangePct != 4.142482682606519 {
		t.Errorf("bitcoin changePct: got %v", btc.ChangePct)
	}
}

func TestNewClientHasTimeout(t *testing.T) {
	t.Parallel()
	c := NewClient()
	if c.Timeout != DefaultTimeout {
		t.Errorf("timeout: want %v, got %v", DefaultTimeout, c.Timeout)
	}
}
