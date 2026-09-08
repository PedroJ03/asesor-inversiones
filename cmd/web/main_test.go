package main

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("WEB_AUTH_PASSWORD", "pw")
	t.Setenv("WEB_SESSION_SECRET", "secret")
	t.Setenv("WEB_ADDR", ":9000")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Password != "pw" {
		t.Errorf("expected password pw, got %q", cfg.Password)
	}
	if cfg.Secret != "secret" {
		t.Errorf("expected secret secret, got %q", cfg.Secret)
	}
	if cfg.Addr != ":9000" {
		t.Errorf("expected addr :9000, got %q", cfg.Addr)
	}
}

func TestLoadConfig_MissingCredentials(t *testing.T) {
	os.Unsetenv("WEB_AUTH_PASSWORD")
	os.Unsetenv("WEB_SESSION_SECRET")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error when credentials are missing")
	}
	if !strings.Contains(err.Error(), "WEB_AUTH_PASSWORD") || !strings.Contains(err.Error(), "WEB_SESSION_SECRET") {
		t.Fatalf("expected error to mention env vars, got %v", err)
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
