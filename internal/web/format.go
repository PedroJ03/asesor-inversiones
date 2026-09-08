package web

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatPrice mirrors internal/render's formatPrice helper. It returns a
// Spanish-locale price string with thousands separators and the currency
// symbol: "$" for ARS and "u$s" for USD.
func FormatPrice(value float64, currency string) string {
	symbol := "$"
	if currency == "USD" {
		symbol = "u$s"
	}
	return fmt.Sprintf("%s %s", symbol, formatNumber(value, 2))
}

// formatNumber mirrors internal/render's formatNumber helper. Negative values
// keep a leading minus sign; the integer part uses '.' as thousands separator
// and the decimal part uses ','.
func formatNumber(n float64, decimals int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	format := "%." + strconv.Itoa(decimals) + "f"
	s := fmt.Sprintf(format, n)
	parts := strings.Split(s, ".")
	intPart := parts[0]

	var b strings.Builder
	b.WriteString(sign)
	for i := 0; i < len(intPart); i++ {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteByte(intPart[i])
	}
	if decimals > 0 {
		b.WriteByte(',')
		b.WriteString(parts[1])
	}
	return b.String()
}

// PercentClass returns the CSS class for a percentage change. It mirrors the
// class selection in internal/render's formatPercent helper.
func PercentClass(value float64) string {
	if value < 0 {
		return "negative"
	}
	return "positive"
}

// FormatPercent mirrors internal/render's formatPercent helper text, without
// the surrounding HTML. It returns the signed percentage string (e.g. "+1,23 %"
// or "-0,50 %"). The caller is responsible for wrapping it with PercentClass.
func FormatPercent(value float64) string {
	sign := ""
	if value > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%s %%", sign, formatNumber(value, 2))
}
