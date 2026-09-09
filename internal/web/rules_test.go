package web

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

func testRuleServer(t *testing.T, s *store.Store) *ruleServer {
	t.Helper()
	rs := newRuleServer(s, testWatchlist())
	rs.clock = func() time.Time {
		return time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	}
	return rs
}

func TestRuleServer_CreateRuleValid(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "value")
	form.Set("direction", "above")
	form.Set("threshold", "550")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "/alertas" {
		t.Fatalf("expected redirect to /alertas, got %q", loc)
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Source != "yahoo" || rules[0].Symbol != "SPY" {
		t.Fatalf("unexpected rule pair: %s/%s", rules[0].Source, rules[0].Symbol)
	}
	if rules[0].BaselinePrice != 500 {
		t.Fatalf("expected baseline 500, got %f", rules[0].BaselinePrice)
	}
}

func TestRuleServer_CreateRuleNoBaseline(t *testing.T) {
	s := testStore(t, nil)
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "value")
	form.Set("direction", "above")
	form.Set("threshold", "550")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "no hay cotizaci") {
		t.Fatalf("expected baseline error message, got %q", body)
	}
	if strings.Contains(body, "creada") {
		t.Error("error page must not claim success")
	}
}

func TestRuleServer_CreateRuleDuplicate(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	for i := 0; i < 2; i++ {
		form := url.Values{}
		form.Set("source", "yahoo|SPY")
		form.Set("kind", "value")
		form.Set("direction", "above")
		form.Set("threshold", "550")

		req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		if i == 0 {
			rs.ruleCreateHandler(w, req)
			if w.Code != http.StatusSeeOther {
				t.Fatalf("first create failed: %d", w.Code)
			}
			continue
		}
		rs.ruleCreateHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request for duplicate, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Ya existe") {
			t.Fatalf("expected duplicate error, got %q", body)
		}
		if strings.Contains(body, "creada") {
			t.Error("error page must not claim success")
		}
	}
}

func TestRuleServer_CreateRuleInvalidThreshold(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "value")
	form.Set("direction", "above")
	form.Set("threshold", "-10")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "umbral") {
		t.Fatalf("expected threshold error, got %q", body)
	}
}

func TestRuleServer_CreateRuleHtmxFragment(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "pct")
	form.Set("direction", "above")
	form.Set("threshold", "5")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected fragment OK, got %d: %q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if !strings.Contains(body, "Reglas") {
		t.Error("expected rules section")
	}
	if !strings.Contains(body, "desde") {
		t.Error("expected percent baseline to be shown as '>= pct desde baseline'")
	}
}

func TestRuleServer_AlertsPageRendersRulesAndHistory(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})

	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	_, err = s.CreateRule("dolarapi", "blue", "pct", "above", 5, 1200, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	_, err = s.InsertAlert(store.Alert{
		RuleID:      sql.NullInt64{Int64: 1, Valid: true},
		Source:      "yahoo",
		Symbol:      "SPY",
		Kind:        "value",
		Threshold:   550,
		TriggeredAt: now.Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	rs := testRuleServer(t, s)
	req := httptest.NewRequest("GET", "/alertas", nil)
	w := httptest.NewRecorder()
	rs.alertsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %q", http.StatusOK, w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"S and P 500", "blue", "Reglas", "Historial reciente", "<!doctype html>"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
}

func TestRuleServer_AlertsFragment(t *testing.T) {
	s := testStore(t, nil)
	rs := testRuleServer(t, s)

	req := httptest.NewRequest("GET", "/alertas", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	rs.alertsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if !strings.Contains(body, "Alertas") {
		t.Error("expected alerts content")
	}
}

func TestRuleServer_DisableViaPut(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("enabled", "false")

	req := httptest.NewRequest("PUT", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleUpdateHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != id {
		t.Fatalf("expected rule %d, got %+v", id, rules)
	}
	if rules[0].Enabled {
		t.Fatalf("expected rule disabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state unchanged armed, got %q", rules[0].State)
	}
}

func TestRuleServer_EnableViaPut(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := s.SetRuleEnabled(id, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("enabled", "true")

	req := httptest.NewRequest("PUT", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleUpdateHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if !rules[0].Enabled {
		t.Fatalf("expected rule enabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state unchanged armed, got %q", rules[0].State)
	}
}

func TestRuleServer_DisableHtmxFragment(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("enabled", "false")

	req := httptest.NewRequest("PUT", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleUpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected fragment OK, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if !strings.Contains(body, "deshabilitada") {
		t.Error("expected disabled chip in fragment")
	}
	if !strings.Contains(body, "Reactivar") {
		t.Error("expected re-enable affordance in fragment")
	}
}

func TestRuleServer_EnableHtmxFragment(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := s.SetRuleEnabled(1, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("enabled", "true")

	req := httptest.NewRequest("PUT", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleUpdateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected fragment OK, got %d: %q", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "<!doctype html>") {
		t.Error("fragment must not contain full html document")
	}
	if !strings.Contains(body, "Deshabilitar") {
		t.Error("expected disable affordance in fragment")
	}
}

func TestRuleServer_DeleteRulePreservesHistory(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	id, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	_, err = s.InsertAlert(store.Alert{
		RuleID:      sql.NullInt64{Int64: id, Valid: true},
		Source:      "yahoo",
		Symbol:      "SPY",
		Kind:        "value",
		Threshold:   550,
		TriggeredAt: now.Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("insert alert: %v", err)
	}
	rs := testRuleServer(t, s)

	req := httptest.NewRequest("DELETE", "/alertas/reglas/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleDeleteHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected rule removed, got %d", len(rules))
	}

	history, err := s.History(10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected history preserved, got %d", len(history))
	}
}

func TestRuleServer_FallbackPostDisable(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("action", "deshabilitar")

	req := httptest.NewRequest("POST", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleActionFallbackHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if rules[0].Enabled {
		t.Fatalf("expected rule disabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state unchanged armed, got %q", rules[0].State)
	}
}

func TestRuleServer_FallbackPostEnable(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if err := s.SetRuleEnabled(1, false); err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("action", "habilitar")

	req := httptest.NewRequest("POST", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleActionFallbackHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if !rules[0].Enabled {
		t.Fatalf("expected rule enabled")
	}
	if rules[0].State != "armed" {
		t.Fatalf("expected state unchanged armed, got %q", rules[0].State)
	}
}

func TestRuleServer_FallbackPostDelete(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("action", "eliminar")

	req := httptest.NewRequest("POST", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleActionFallbackHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}

	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected rule removed, got %d", len(rules))
	}
}

func TestRuleServer_FallbackPostUnknownAction(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	_, err := s.CreateRule("yahoo", "SPY", "value", "above", 550, 500, now)
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("action", "desconocida")

	req := httptest.NewRequest("POST", "/alertas/reglas/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	rs.ruleActionFallbackHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", w.Code)
	}
}

// TestNormalizeThreshold covers the comma-permissive threshold input: decimal
// commas, surrounding spaces, and trivially attached currency markers.
func TestNormalizeThreshold(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    float64
		wantErr bool
	}{
		{"decimal comma", "1,5", 1.5, false},
		{"comma with decimals", "1500,50", 1500.5, false},
		{"surrounding spaces", " 2 ", 2, false},
		{"currency prefix", "$500", 500, false},
		{"adviser price format", "u$s 1.200,50", 0, true}, // thousands dot is not accepted
		{"non numeric", "abc", 0, true},
		{"empty", "", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := strconv.ParseFloat(normalizeThreshold(tc.raw), 64)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected parse error for %q, got %v", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFloat(normalizeThreshold(%q)): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("normalizeThreshold(%q) parsed %v; want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// TestRuleServer_CreateRuleCommaThreshold proves the create handler accepts a
// comma decimal threshold and stores the parsed float.
func TestRuleServer_CreateRuleCommaThreshold(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "value")
	form.Set("direction", "above")
	form.Set("threshold", "1500,50")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %q", w.Code, w.Body.String())
	}
	rules, err := s.ListRules(false)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 || rules[0].Threshold != 1500.5 {
		t.Fatalf("want threshold 1500.5, got %+v", rules)
	}
}

// TestRuleServer_CreateRuleNonNumericThreshold rejects input that still fails
// parsing with the controlled error style.
func TestRuleServer_CreateRuleNonNumericThreshold(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, []fetch.Quote{
		{Source: "yahoo", Symbol: "SPY", Price: 500, ChangePct: 1.2, Currency: "USD", FetchedAt: now.Add(-1 * time.Hour)},
	})
	rs := testRuleServer(t, s)

	form := url.Values{}
	form.Set("source", "yahoo|SPY")
	form.Set("kind", "value")
	form.Set("direction", "above")
	form.Set("threshold", "abc")

	req := httptest.NewRequest("POST", "/alertas/reglas/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	rs.ruleCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "umbral") {
		t.Fatalf("expected controlled threshold error, got %q", body)
	}
	rules, _ := s.ListRules(false)
	if len(rules) != 0 {
		t.Fatalf("invalid threshold must not create a rule, got %d", len(rules))
	}
}

// TestAlertsPage_ShowsEvaluationHint guards the clarity copy explaining when
// alerts are evaluated.
func TestAlertsPage_ShowsEvaluationHint(t *testing.T) {
	s := testStore(t, nil)
	rs := testRuleServer(t, s)

	req := httptest.NewRequest("GET", "/alertas", nil)
	w := httptest.NewRecorder()
	rs.alertsHandler(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "Las alertas se evalúan cada vez que se actualiza el reporte") {
		t.Errorf("expected evaluation hint, got %q", body)
	}
}

// TestAlertsPage_DirectionLabelsFollowKind checks the server-side rendering
// switch: pct labels with the baseline helper, value labels by default.
func TestAlertsPage_DirectionLabelsFollowKind(t *testing.T) {
	s := testStore(t, nil)
	rs := testRuleServer(t, s)

	t.Run("kind pct via query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/alertas?kind=pct", nil)
		w := httptest.NewRecorder()
		rs.alertsHandler(w, req)

		body := w.Body.String()
		for _, want := range []string{"Sube al menos", "Baja al menos", "respecto al precio base capturado al crear la regla"} {
			if !strings.Contains(body, want) {
				t.Errorf("expected body to contain %q", want)
			}
		}
		if strings.Contains(body, "Mayor o igual que") {
			t.Error("pct form must not render value-style labels")
		}
	})

	t.Run("default kind value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/alertas", nil)
		w := httptest.NewRecorder()
		rs.alertsHandler(w, req)

		body := w.Body.String()
		for _, want := range []string{"Mayor o igual que", "Menor o igual que"} {
			if !strings.Contains(body, want) {
				t.Errorf("expected body to contain %q", want)
			}
		}
		if strings.Contains(body, "Sube al menos") {
			t.Error("value form must not render pct-style labels")
		}
	})
}

// TestRulesList_PctConditionShowsBaselineFrom renders pct rules as a single
// condition: ">= 2,00 % desde u$s 118,50".
func TestRulesList_PctConditionShowsBaselineFrom(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	s := testStore(t, nil)
	if _, err := s.CreateRule("dolarapi", "blue", "pct", "above", 2, 118.5, now); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	rs := testRuleServer(t, s)

	req := httptest.NewRequest("GET", "/alertas", nil)
	w := httptest.NewRecorder()
	rs.alertsHandler(w, req)

	body := w.Body.String()
	for _, want := range []string{"≥ 2,00 %", "desde", "u$s 118,50"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}
}

func TestRuleServer_ParseAssetValue(t *testing.T) {
	cases := []struct {
		in         string
		wantSource string
		wantSymbol string
	}{
		{"yahoo|SPY", "yahoo", "SPY"},
		{"dolarapi|blue", "dolarapi", "blue"},
		{"", "", ""},
		{"invalid", "", ""},
	}
	for _, tc := range cases {
		source, symbol := parseAssetValue(tc.in)
		if source != tc.wantSource || symbol != tc.wantSymbol {
			t.Errorf("parseAssetValue(%q) = %q,%q; want %q,%q", tc.in, source, symbol, tc.wantSource, tc.wantSymbol)
		}
	}
}

func TestRuleServer_ClassifyRuleError(t *testing.T) {
	cases := []struct {
		in   error
		want string
	}{
		{store.ErrInvalidRule, "no es válida"},
		{store.ErrDuplicateRule, "Ya existe"},
		{errors.New("db down"), "No se pudo guardar"},
	}
	for _, tc := range cases {
		got := classifyRuleError(tc.in)
		if !strings.Contains(got, tc.want) {
			t.Errorf("classifyRuleError(%v) = %q; want containing %q", tc.in, got, tc.want)
		}
	}
}
