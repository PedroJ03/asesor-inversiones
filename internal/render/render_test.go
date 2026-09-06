package render

import (
	"html"
	"strings"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
)

func textOf(rendered []byte) string {
	return html.UnescapeString(string(rendered))
}

func sampleReportData(t *testing.T) ReportData {
	t.Helper()
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	return FromQuotes([]fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Name: "S&P 500 (SPY)", Price: 600.5, PrevClose: 595.0, ChangePct: 0.92, Currency: "USD", QuotedAt: now, FetchedAt: now},
		{Source: "dolarapi", Symbol: "blue", Name: "Blue", Price: 1545, Bid: 1525, Ask: 1545, Currency: "ARS", QuotedAt: now, FetchedAt: now},
		{Source: "data912", Symbol: "AL30D", Price: 955.0, PrevClose: 959.8, ChangePct: -0.5, Currency: "ARS", QuotedAt: now, FetchedAt: now},
		{Source: "data912", Symbol: "AE38C", Price: 75.8, ChangePct: 0.26, Currency: "USD", QuotedAt: now, FetchedAt: now},
		{Source: "coingecko", Symbol: "bitcoin", Name: "Bitcoin", Price: 81155, ChangePct: 4.14, Currency: "USD", QuotedAt: now, FetchedAt: now},
	}, []string{"Yahoo rate-limited temporarily"}, now)
}

func TestRenderContainsSections(t *testing.T) {
	t.Parallel()

	data := sampleReportData(t)
	html, err := Render(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	required := []string{
		"Reporte diario de mercado",
		"4 de septiembre de 2026",
		"Estados Unidos",
		"Dólares",
		"Bonos soberanos",
		"Criptomonedas",
		"S&P 500 (SPY)",
		"Blue",
		"AL30D",
		"AE38C",
		"Bitcoin",
		"Fuentes",
		"coingecko.com",
		"Yahoo rate-limited temporarily",
	}
	text := textOf(html)
	for _, s := range required {
		if !strings.Contains(text, s) {
			t.Errorf("rendered HTML missing %q", s)
		}
	}
}

func TestRenderFormatsPrices(t *testing.T) {
	t.Parallel()

	data := sampleReportData(t)
	html, err := Render(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Spanish locale: thousands separator = '.', decimal = ','
	checks := []string{
		"u$s 600,50",
		"$ 1.545,00",
		"u$s 75,80",
		"u$s 81.155,00",
	}
	for _, s := range checks {
		if !strings.Contains(string(html), s) {
			t.Errorf("rendered HTML missing price %q", s)
		}
	}
}

func TestRenderFormatsPercents(t *testing.T) {
	t.Parallel()

	data := sampleReportData(t)
	html, err := Render(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(string(html), "+0,92 %") {
		t.Error("missing positive percent")
	}
	if !strings.Contains(string(html), "-0,50 %") {
		t.Error("missing negative percent")
	}
}

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   float64
		want string
	}{
		{1234.56, "1.234,56"},
		{-1234.56, "-1.234,56"},
		{0.0, "0,00"},
		{1000000.0, "1.000.000,00"},
	}

	for _, tc := range cases {
		got := formatNumber(tc.in, 2)
		if got != tc.want {
			t.Errorf("formatNumber(%v): want %q, got %q", tc.in, tc.want, got)
		}
	}
}
