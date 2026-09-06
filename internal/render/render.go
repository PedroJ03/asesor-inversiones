// Package render generates the mobile-first HTML daily market report.
package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	_ "time/tzdata"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
)

//go:embed template.html
var templates embed.FS

var arLocation *time.Location

func init() {
	var err error
	arLocation, err = time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		arLocation = time.UTC
	}
}

// ReportData carries everything the template needs to render the report.
type ReportData struct {
	DateString string
	USA        []fetch.Quote
	Dolares    []fetch.Quote
	Bonos      []fetch.Quote
	Cripto     []fetch.Quote
	Warnings   []string
}

// FromQuotes builds report sections from a flat quote list.
func FromQuotes(quotes []fetch.Quote, warnings []string, reportTime time.Time) ReportData {
	data := ReportData{
		DateString: dateInSpanish(reportTime.In(arLocation)),
		Warnings:   warnings,
	}
	for _, q := range quotes {
		switch q.Source {
		case "yahoo":
			data.USA = append(data.USA, q)
		case "dolarapi":
			data.Dolares = append(data.Dolares, q)
		case "data912":
			data.Bonos = append(data.Bonos, q)
		case "coingecko":
			data.Cripto = append(data.Cripto, q)
		}
	}
	return data
}

// Render executes the embedded template and returns the HTML bytes.
func Render(data ReportData) ([]byte, error) {
	tmpl, err := template.New("template.html").Funcs(template.FuncMap{
		"price": formatPrice,
		"pct":   formatPercent,
	}).ParseFS(templates, "template.html")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

var monthsES = []string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

func dateInSpanish(t time.Time) string {
	return fmt.Sprintf("%d de %s de %d", t.Day(), monthsES[t.Month()-1], t.Year())
}

func formatPrice(value float64, currency string) template.HTML {
	symbol := "$"
	if currency == "USD" {
		symbol = "u$s"
	}
	return template.HTML(fmt.Sprintf("%s %s", symbol, formatNumber(value, 2)))
}

func formatPercent(value float64) template.HTML {
	sign := ""
	cls := "positive"
	if value < 0 {
		cls = "negative"
	} else if value > 0 {
		sign = "+"
	}
	return template.HTML(fmt.Sprintf(`<span class="%s">%s%s %%</span>`, cls, sign, formatNumber(value, 2)))
}

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
