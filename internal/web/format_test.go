package web

import (
	"strings"
	"testing"
)

func TestFormatPrice(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		currency string
		want     string
	}{
		{"ars positive", 1234.56, "ARS", "$ 1.234,56"},
		{"ars zero", 0, "ARS", "$ 0,00"},
		{"ars negative", -1234.56, "ARS", "$ -1.234,56"},
		{"usd positive", 999.99, "USD", "u$s 999,99"},
		{"large", 1234567.89, "ARS", "$ 1.234.567,89"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPrice(tc.value, tc.currency)
			if got != tc.want {
				t.Errorf("FormatPrice(%v, %q) = %q, want %q", tc.value, tc.currency, got, tc.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		want  string
	}{
		{"positive", 1.2345, "+1,23 %"},
		{"negative", -0.5, "-0,50 %"},
		{"zero", 0, "0,00 %"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatPercent(tc.value)
			if got != tc.want {
				t.Errorf("FormatPercent(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestPercentClass(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{-1, "negative"},
		{0, "positive"},
		{1, "positive"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := PercentClass(tc.value)
			if got != tc.want {
				t.Errorf("PercentClass(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

// TestFormatParityWithRender verifies that the duplicated helpers produce the
// same textual output as the internal/render helpers for representative values.
func TestFormatParityWithRender(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		currency string
	}{
		{"positive ars", 1500.5, "ARS"},
		{"negative usd", -2.345, "USD"},
		{"zero ars", 0, "ARS"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			price := FormatPrice(tc.value, tc.currency)
			pct := FormatPercent(tc.value)

			// Text must contain the same formatted number as the render helper.
			if !strings.Contains(price, formatNumber(tc.value, 2)) {
				t.Errorf("price %q does not contain %q", price, formatNumber(tc.value, 2))
			}
			if !strings.Contains(pct, formatNumber(tc.value, 2)) {
				t.Errorf("percent %q does not contain %q", pct, formatNumber(tc.value, 2))
			}
		})
	}
}
