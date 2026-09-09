package web

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
)

// isEmojiRune reports whether r falls in the emoji/symbol codepoint ranges the
// UI is required to keep free of: pictographs, dingbats, arrows, and misc
// symbol blocks. It mirrors the repository's verification grep for *.templ.
func isEmojiRune(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF:
		return true
	case r >= 0x2600 && r <= 0x27BF:
		return true
	case r >= 0x2190 && r <= 0x21FF:
		return true
	case r >= 0x2B00 && r <= 0x2BFF:
		return true
	case r == 0xFE0F:
		return true
	}
	return false
}

func assertNoEmoji(t *testing.T, name, body string) {
	t.Helper()
	for _, r := range body {
		if isEmojiRune(r) {
			t.Errorf("%s contains forbidden emoji/symbol rune U+%04X", name, r)
			return
		}
	}
}

// TestRenderedPages_NoEmoji renders every primary page and asserts the UI
// stays free of emoji characters in labels, buttons, and links.
func TestRenderedPages_NoEmoji(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
		{Source: "dolarapi", Symbol: "blue", Price: 1200, ChangePct: 0.5, Currency: "ARS", FetchedAt: now.Add(-2 * time.Hour)},
	})

	vs := testViewServer(t, s)
	rs := testRuleServer(t, s)

	pages := []struct {
		name string
		body func() string
	}{
		{"dashboard", func() string {
			w := httptest.NewRecorder()
			vs.dashboardHandler(w, httptest.NewRequest("GET", "/", nil))
			return w.Body.String()
		}},
		{"report", func() string {
			w := httptest.NewRecorder()
			vs.reportCurrentHandler(w, httptest.NewRequest("GET", "/reporte", nil))
			return w.Body.String()
		}},
		{"watchlist", func() string {
			w := httptest.NewRecorder()
			vs.watchlistHandler(w, httptest.NewRequest("GET", "/activos", nil))
			return w.Body.String()
		}},
		{"alerts", func() string {
			w := httptest.NewRecorder()
			rs.alertsHandler(w, httptest.NewRequest("GET", "/alertas", nil))
			return w.Body.String()
		}},
		{"nav", func() string {
			w := httptest.NewRecorder()
			if err := Nav("inicio").Render(context.Background(), w); err != nil {
				t.Fatalf("render nav: %v", err)
			}
			return w.Body.String()
		}},
	}

	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			assertNoEmoji(t, p.name, p.body())
		})
	}
}
