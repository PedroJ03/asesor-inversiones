package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPWAManifestValid(t *testing.T) {
	req := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	w := httptest.NewRecorder()
	Assets.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Fatalf("expected application/manifest+json, got %q", ct)
	}

	body := w.Body.Bytes()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}

	required := []string{"name", "short_name", "start_url", "display", "theme_color", "background_color", "icons"}
	for _, key := range required {
		if _, ok := m[key]; !ok {
			t.Errorf("manifest missing required field %q", key)
		}
	}

	icons, ok := m["icons"].([]any)
	if !ok || len(icons) != 2 {
		t.Fatalf("expected 2 icons, got %d", len(icons))
	}

	wantSizes := map[string]bool{"192x192": false, "512x512": false}
	for _, icon := range icons {
		im, ok := icon.(map[string]any)
		if !ok {
			t.Fatalf("icon entry is not an object: %v", icon)
		}
		sizes, _ := im["sizes"].(string)
		if _, exists := wantSizes[sizes]; !exists {
			t.Errorf("unexpected icon size %q", sizes)
			continue
		}
		wantSizes[sizes] = true

		if purpose, _ := im["purpose"].(string); !strings.Contains(purpose, "maskable") || !strings.Contains(purpose, "any") {
			t.Errorf("icon %q purpose must include maskable and any, got %q", sizes, purpose)
		}

		src, _ := im["src"].(string)
		if src == "" {
			t.Errorf("icon %q missing src", sizes)
			continue
		}

		// The declared icon file must be served.
		iconReq := httptest.NewRequest("GET", src, nil)
		iconW := httptest.NewRecorder()
		Assets.ServeHTTP(iconW, iconReq)
		if iconW.Code != http.StatusOK {
			t.Errorf("icon %q not served: status %d", src, iconW.Code)
		}
		if ct := iconW.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("icon %q content type = %q, want image/png", src, ct)
		}
	}

	for size, found := range wantSizes {
		if !found {
			t.Errorf("missing icon size %q", size)
		}
	}
}

func TestPWAServiceWorkerServed(t *testing.T) {
	req := httptest.NewRequest("GET", "/sw.js", nil)
	w := httptest.NewRecorder()
	Assets.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("expected javascript content type, got %q", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "SHELL_ASSETS") {
		t.Error("service worker missing SHELL_ASSETS declaration")
	}
	if !strings.Contains(body, "cache-first") || !strings.Contains(body, "stale-while-revalidate") {
		t.Error("service worker missing caching strategy comments")
	}
}

func TestPWAServiceWorkerAssetListMatchesEmbeddedFiles(t *testing.T) {
	// The shell assets declared in sw.js must exist in the embedded assets.
	req := httptest.NewRequest("GET", "/sw.js", nil)
	w := httptest.NewRecorder()
	Assets.ServeHTTP(w, req)

	body := w.Body.String()
	declared := []string{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "'/assets/") || strings.HasPrefix(line, "'/offline.html'") {
			path := strings.Trim(line, "',")
			declared = append(declared, path)
		}
	}

	if len(declared) == 0 {
		t.Fatal("no shell assets declared in sw.js")
	}

	for _, assetPath := range declared {
		req := httptest.NewRequest("GET", assetPath, nil)
		w := httptest.NewRecorder()
		Assets.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("declared shell asset %q not served: status %d", assetPath, w.Code)
		}
	}
}

func TestPWAOfflineMarkerPage(t *testing.T) {
	req := httptest.NewRequest("GET", "/offline.html", nil)
	w := httptest.NewRecorder()
	Assets.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "offline cached page") {
		t.Error("offline page missing explicit marker")
	}
	if strings.Contains(body, "última actualización") {
		t.Error("offline page must not fabricate a freshness timestamp")
	}
	if !strings.Contains(body, "asesor helper") {
		t.Error("offline page missing brand")
	}
}

func TestPWAFirstLoadBudget(t *testing.T) {
	// Render a realistic full page through the shell to include nav, meta tags, etc.
	w := httptest.NewRecorder()
	if err := Shell("Inicio", "inicio", PlaceholderPage("Inicio")).Render(context.Background(), w); err != nil {
		t.Fatalf("render shell: %v", err)
	}
	htmlBytes := w.Body.Len()

	staticFiles := map[string]string{
		"htmx.min.js":       "assets/htmx.min.js",
		"main.css":          "assets/main.css",
		"manifest":          "assets/manifest.webmanifest",
		"icon-192.png":      "assets/icon-192.png",
		"icon-512.png":      "assets/icon-512.png",
	}

	total := htmlBytes
	breakdown := []string{"HTML shell"}
	for name, file := range staticFiles {
		f, err := assetsFS.Open(file)
		if err != nil {
			t.Fatalf("open %s: %v", file, err)
		}
		size, err := f.(io.Seeker).Seek(0, io.SeekEnd)
		if err != nil {
			t.Fatalf("seek %s: %v", file, err)
		}
		_ = f.Close()
		total += int(size)
		breakdown = append(breakdown, name)
	}

	const budget = 200 * 1024 // 200 KB
	if total >= budget {
		t.Fatalf("first-load budget exceeded: %d bytes (budget %d bytes)", total, budget)
	}

	t.Logf("first-load budget total: %d bytes (HTML=%d, breakdown=%v)", total, htmlBytes, breakdown)
}

func TestPWAHTMXScriptTagHasSRIAndDefer(t *testing.T) {
	w := httptest.NewRecorder()
	if err := Shell("Test", "inicio", PlaceholderPage("Test")).Render(context.Background(), w); err != nil {
		t.Fatalf("render shell: %v", err)
	}

	html := w.Body.String()
	scriptStart := strings.Index(html, "<script")
	scriptEnd := strings.Index(html, "</script>")
	if scriptStart == -1 || scriptEnd == -1 {
		t.Fatal("shell missing htmx script tag")
	}
	script := html[scriptStart : scriptEnd+len("</script>")]

	if !strings.Contains(script, "src=\"/assets/htmx.min.js\"") {
		t.Error("htmx script src missing or incorrect")
	}
	if !strings.Contains(script, "defer") {
		t.Error("htmx script missing defer attribute")
	}
	wantIntegrity := "sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V"
	if !strings.Contains(script, wantIntegrity) {
		t.Errorf("htmx script missing expected integrity hash %q", wantIntegrity)
	}
}

func TestPWAManifestLinkAndThemeColor(t *testing.T) {
	w := httptest.NewRecorder()
	if err := Shell("Test", "inicio", PlaceholderPage("Test")).Render(context.Background(), w); err != nil {
		t.Fatalf("render shell: %v", err)
	}

	html := w.Body.String()
	if !strings.Contains(html, `<link rel="manifest" href="/manifest.webmanifest">`) {
		t.Error("shell missing manifest link")
	}
	if !strings.Contains(html, `<meta name="theme-color" content="#0f172a">`) {
		t.Error("shell missing theme-color meta")
	}
}

func TestPWAServiceWorkerRegistrationScript(t *testing.T) {
	w := httptest.NewRecorder()
	if err := Shell("Test", "inicio", PlaceholderPage("Test")).Render(context.Background(), w); err != nil {
		t.Fatalf("render shell: %v", err)
	}

	html := w.Body.String()
	if !strings.Contains(html, "navigator.serviceWorker.register('/sw.js')") {
		t.Error("shell missing service worker registration")
	}
}
