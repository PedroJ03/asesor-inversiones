package web

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShell_FullPage(t *testing.T) {
	w := httptest.NewRecorder()
	body := PlaceholderPage("Test")
	err := Shell("Test", "inicio", body).Render(context.Background(), w)
	if err != nil {
		t.Fatalf("render shell: %v", err)
	}

	html := w.Body.String()
	if !strings.Contains(strings.ToLower(html), "<!doctype html>") {
		t.Error("expected full HTML document")
	}
	if !strings.Contains(html, "<nav") {
		t.Error("expected navigation")
	}
	if !strings.Contains(html, "Inicio") || !strings.Contains(html, "Reporte") || !strings.Contains(html, "Activos") || !strings.Contains(html, "Alertas") {
		t.Errorf("expected all four nav tabs, got body:\n%s", html)
	}
	if !strings.Contains(html, "Test") {
		t.Error("expected body content")
	}
}

func TestShell_Fragment(t *testing.T) {
	w := httptest.NewRecorder()
	body := PlaceholderPage("Fragment")
	err := Fragment(body).Render(context.Background(), w)
	if err != nil {
		t.Fatalf("render fragment: %v", err)
	}

	html := w.Body.String()
	if strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("fragment must not contain full HTML document")
	}
	if strings.Contains(html, "<nav") {
		t.Error("fragment must not contain navigation")
	}
	if !strings.Contains(html, "Fragment") {
		t.Error("expected fragment body content")
	}
}

func TestNav_RendersFourTabs(t *testing.T) {
	w := httptest.NewRecorder()
	err := Nav("inicio").Render(context.Background(), w)
	if err != nil {
		t.Fatalf("render nav: %v", err)
	}

	html := w.Body.String()
	for _, label := range []string{"Inicio", "Reporte", "Activos", "Alertas"} {
		if !strings.Contains(html, label) {
			t.Errorf("expected nav to contain %q", label)
		}
	}

	// Active tab should have aria-current="page".
	if !strings.Contains(html, `aria-current="page"`) {
		t.Error("expected active tab to have aria-current=page")
	}
}

func TestRequestIsHX(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if RequestIsHX(req) {
		t.Error("expected non-htmx request")
	}
	req.Header.Set("HX-Request", "true")
	if !RequestIsHX(req) {
		t.Error("expected htmx request")
	}
}
