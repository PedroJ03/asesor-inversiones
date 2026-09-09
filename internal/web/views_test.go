package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

func testWatchlist() *config.Watchlist {
	return &config.Watchlist{
		USA: []config.Asset{
			{Symbol: "SPY", Label: "S and P 500"},
			{Symbol: "AAPL", Label: "Apple"},
		},
		Dolares: []string{"oficial", "blue"},
		Bonos:   []string{"AL30D"},
		Cripto: []config.CryptoAsset{
			{ID: "bitcoin", Label: "Bitcoin"},
		},
	}
}

func testStore(t *testing.T, quotes []fetch.Quote) *store.Store {
	t.Helper()
	s, err := store.Open("file:" + t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if len(quotes) > 0 {
		if err := s.SaveQuotes(quotes); err != nil {
			t.Fatalf("save quotes: %v", err)
		}
	}
	return s
}

func testViewServer(t *testing.T, s *store.Store) *viewServer {
	t.Helper()
	vs := newViewServer(s, testWatchlist())
	vs.clock = func() time.Time {
		return time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	}
	return vs
}

func TestDashboard_RendersFreshnessAndAlertBadge(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
		{Source: "dolarapi", Symbol: "blue", Price: 1200, ChangePct: 0.5, Currency: "ARS", FetchedAt: now.Add(-2 * time.Hour)},
	}
	s := testStore(t, quotes)

	// Seed two triggered alert snapshots.
	for i := 0; i < 2; i++ {
		_, err := s.InsertAlert(store.Alert{
			Source:      "yahoo",
			Symbol:      "SPY",
			Kind:        "pct",
			Threshold:   1.0,
			TriggeredAt: now.Add(-time.Duration(i) * time.Hour),
		})
		if err != nil {
			t.Fatalf("insert alert: %v", err)
		}
	}

	vs := testViewServer(t, s)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	vs.dashboardHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %q", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "última actualización") {
		t.Error("expected freshness label")
	}
	if !strings.Contains(body, "actualizada") {
		t.Error("expected current freshness state")
	}
	if !strings.Contains(body, "2") {
		t.Error("expected alert badge count of 2")
	}
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("expected full page response")
	}
}

func TestDashboard_Fragment(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	vs.dashboardHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if strings.Contains(body, "<nav") {
		t.Error("fragment must not contain navigation")
	}
	if !strings.Contains(body, "Inicio") {
		t.Error("expected dashboard content")
	}
}

func TestReportCurrent_RendersSections(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
		{Source: "dolarapi", Symbol: "blue", Price: 1200, ChangePct: 0.5, Currency: "ARS", FetchedAt: now.Add(-2 * time.Hour)},
		{Source: "data912", Symbol: "AL30D", Price: 45, ChangePct: -0.3, Currency: "USD", FetchedAt: now.Add(-30 * time.Minute)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/reporte", nil)
	w := httptest.NewRecorder()
	vs.reportCurrentHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Acciones USA", "Dólares", "Bonos", "última actualización"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
}

// TestReportCurrent_DifferentiatedBrief checks the report reads as a daily
// brief: explicit day header, prominent freshness banner, signed change
// emphasis, and no per-asset navigation links.
func TestReportCurrent_DifferentiatedBrief(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/reporte", nil)
	w := httptest.NewRecorder()
	vs.reportCurrentHandler(w, req)

	body := w.Body.String()
	for _, want := range []string{"Reporte del día", "15/06/2024", "report__banner", "report-list", "+1,20 %"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
	if strings.Contains(body, `href="/activos/`) {
		t.Error("report rows must not link to asset detail")
	}
}

// TestWatchlist_RendersListRowsWithDetailLinks checks the redesigned
// single-column asset list: full-row links, symbol/name split, price and
// change chip on the right.
func TestWatchlist_RendersListRowsWithDetailLinks(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos", nil)
	w := httptest.NewRecorder()
	vs.watchlistHandler(w, req)

	body := w.Body.String()
	for _, want := range []string{
		`class="asset-row"`,
		`href="/activos/yahoo/SPY"`,
		"asset-row__symbol", "SPY",
		"asset-row__name", "S and P 500",
		"u$s 500,00",
		"asset-row__change", "+1,20 %",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
	if strings.Contains(body, "quote-card") {
		t.Error("watchlist must not render the old card grid")
	}
}

// TestWatchlist_MissingQuoteRendersDash checks that a missing quote shows a
// dash placeholder instead of an empty price cell.
func TestWatchlist_MissingQuoteRendersDash(t *testing.T) {
	s := testStore(t, nil)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos", nil)
	w := httptest.NewRecorder()
	vs.watchlistHandler(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "asset-row__price") {
		t.Fatal("expected asset row markup")
	}
	if !strings.Contains(body, ">—</span>") {
		t.Errorf("expected dash placeholder for missing quotes, got %q", body)
	}
}

func TestReportDated_InvalidDateReturns400(t *testing.T) {
	s := testStore(t, nil)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/reporte/mal", nil)
	w := httptest.NewRecorder()
	vs.reportDatedHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportDated_NoDataReturns404(t *testing.T) {
	s := testStore(t, nil)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/reporte/2024-01-01", nil)
	req.SetPathValue("date", "2024-01-01")
	w := httptest.NewRecorder()
	vs.reportDatedHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "No hay datos") {
		t.Errorf("expected controlled 404 message, got %q", body)
	}
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("expected full shell page for 404")
	}
}

func TestReportDated_AvailableDateRendersData(t *testing.T) {
	reportDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	// Seed a quote whose fetched_at falls inside the requested date in the
	// advisor's local timezone (America/Argentina/Buenos_Aires).
	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 1234.5, ChangePct: 1.5, Currency: "USD", FetchedAt: reportDate.Add(10 * time.Hour)},
	}
	s := testStore(t, quotes)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/reporte/2024-06-15", nil)
	req.SetPathValue("date", "2024-06-15")
	w := httptest.NewRecorder()
	vs.reportDatedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %q", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Reporte del 2024-06-15") {
		t.Errorf("expected dated report title, got %q", body)
	}
	if !strings.Contains(body, "15/06/2024") {
		t.Errorf("expected human-readable day under the title, got %q", body)
	}
	if !strings.Contains(body, "u$s 1.234,50") {
		t.Errorf("expected seeded price to be rendered, got %q", body)
	}
	if !strings.Contains(body, "última actualización") {
		t.Error("expected freshness label")
	}
}

func TestWatchlist_GroupsBySection(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
		{Source: "dolarapi", Symbol: "blue", Price: 1200, ChangePct: 0.5, Currency: "ARS", FetchedAt: now.Add(-2 * time.Hour)},
		{Source: "data912", Symbol: "AL30D", Price: 45, ChangePct: -0.3, Currency: "USD", FetchedAt: now.Add(-30 * time.Minute)},
		{Source: "coingecko", Symbol: "bitcoin", Price: 70000, ChangePct: 2.0, Currency: "USD", FetchedAt: now.Add(-10 * time.Minute)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos", nil)
	w := httptest.NewRecorder()
	vs.watchlistHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Acciones USA", "S and P 500", "Dólares", "blue", "Bonos", "AL30D", "Cripto", "Bitcoin"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
	if !strings.Contains(body, "última actualización") {
		t.Error("expected freshness label")
	}
}

func TestWatchlist_StaleQuoteRendersStaleWarning(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-25 * time.Hour)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos", nil)
	w := httptest.NewRecorder()
	vs.watchlistHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %q", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "freshness--stale") {
		t.Error("expected stale freshness class")
	}
	if !strings.Contains(body, "desactualizada") {
		t.Error("expected stale warning label")
	}
	if strings.Contains(body, "freshness--current") {
		t.Error("stale quote must not render current freshness class")
	}
}

func TestAssetDetail_HappyPath(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	quotes := []fetch.Quote{
		{Source: "yahoo", Symbol: "AAPL", Name: "Apple Inc.", Price: 180, Bid: 179.5, Ask: 180.5, ChangePct: 1.5, Currency: "USD", FetchedAt: now.Add(-5 * time.Minute)},
		{Source: "yahoo", Symbol: "AAPL", Name: "Apple Inc.", Price: 179, Bid: 178.5, Ask: 179.5, ChangePct: 1.0, Currency: "USD", FetchedAt: now.Add(-26 * time.Hour)},
	}
	s := testStore(t, quotes)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos/yahoo/AAPL", nil)
	req.SetPathValue("source", "yahoo")
	req.SetPathValue("symbol", "AAPL")
	w := httptest.NewRecorder()
	vs.assetDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %q", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Apple", "u$s 180,50", "+1,50 %", "Compra:", "Venta:", "última actualización"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
}

func TestAssetDetail_MissingPairReturns404(t *testing.T) {
	s := testStore(t, nil)
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos/yahoo/NOEXISTE", nil)
	req.SetPathValue("source", "yahoo")
	req.SetPathValue("symbol", "NOEXISTE")
	w := httptest.NewRecorder()
	vs.assetDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("expected full shell page for 404")
	}
}

func TestAssetDetail_Fragment(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "AAPL", Price: 180, ChangePct: 1.5, Currency: "USD", FetchedAt: now.Add(-5 * time.Minute)},
	})
	vs := testViewServer(t, s)

	req := httptest.NewRequest("GET", "/activos/yahoo/AAPL", nil)
	req.SetPathValue("source", "yahoo")
	req.SetPathValue("symbol", "AAPL")
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	vs.assetDetailHandler(w, req)

	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if !strings.Contains(body, "Apple") {
		t.Error("expected asset content")
	}
}

func TestViews_RenderHelper(t *testing.T) {
	// Direct render coverage: full page and fragment selection.
	vs := testViewServer(t, testStore(t, nil))

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		vs.render(w, req, "inicio", "Test", PlaceholderPage("Test"))
		if !strings.Contains(w.Body.String(), "<!doctype html>") {
			t.Error("expected full page")
		}
	})

	t.Run("fragment", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		vs.render(w, req, "inicio", "Test", PlaceholderPage("Test"))
		if strings.Contains(w.Body.String(), "<!doctype html>") {
			t.Error("expected fragment only")
		}
	})
}

func TestAlertBadgeCount(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)

	for i := 0; i < 3; i++ {
		_, err := s.InsertAlert(store.Alert{
			Source:      "yahoo",
			Symbol:      "SPY",
			Kind:        "pct",
			Threshold:   1.0,
			TriggeredAt: now.Add(-time.Duration(i) * time.Hour),
		})
		if err != nil {
			t.Fatalf("insert alert: %v", err)
		}
	}

	vs := testViewServer(t, s)
	count, err := vs.triggeredAlertCount()
	if err {
		t.Fatal("expected no alert count error")
	}
	if count != 3 {
		t.Fatalf("expected alert count 3, got %d", count)
	}
}

func TestFreshnessComponent_Render(t *testing.T) {
	at := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	info := FreshnessInfo{State: FreshnessCurrent, At: at}
	w := httptest.NewRecorder()
	if err := Freshness(info).Render(context.Background(), w); err != nil {
		t.Fatalf("render freshness: %v", err)
	}
	body := w.Body.String()
	if !strings.Contains(body, "última actualización") {
		t.Error("expected freshness label")
	}
	if !strings.Contains(body, "actualizada") {
		t.Error("expected state label")
	}
}

func TestNullAlertHistory(t *testing.T) {
	// Ensure a null rule_id does not break history reads.
	s := testStore(t, nil)
	_, err := s.InsertAlert(store.Alert{
		RuleID:      sql.NullInt64{},
		Source:      "dolarapi",
		Symbol:      "blue",
		Kind:        "value",
		Threshold:   1000,
		TriggeredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert alert with null rule_id: %v", err)
	}
	alerts, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
}
