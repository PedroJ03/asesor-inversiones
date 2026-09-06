package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultWatchlist(t *testing.T) {
	t.Parallel()

	// Tests run from inside internal/config.
	wl, err := Load(filepath.Join("..", "..", "watchlist.yaml"))
	if err != nil {
		t.Fatalf("load default watchlist: %v", err)
	}

	if len(wl.USA) != 6 {
		t.Errorf("want 6 USA assets, got %d", len(wl.USA))
	}
	if len(wl.Dolares) != 4 {
		t.Errorf("want 4 dolares, got %d", len(wl.Dolares))
	}
	if len(wl.Bonos) != 2 {
		t.Errorf("want 2 bonos, got %d", len(wl.Bonos))
	}
	if len(wl.Cripto) != 2 {
		t.Errorf("want 2 cripto, got %d", len(wl.Cripto))
	}

	if got := wl.LabelForSymbol("SPY"); got != "S&P 500 (SPY)" {
		t.Errorf("SPY label: want %q, got %q", "S&P 500 (SPY)", got)
	}
	if got := wl.CryptoLabel("bitcoin"); got != "Bitcoin" {
		t.Errorf("bitcoin label: want %q, got %q", "Bitcoin", got)
	}
}

func TestValidateErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "empty sections",
			content: "usa: []\ndolares: []\nbonos: []\ncripto: []\n",
			wantErr: "usa section is empty",
		},
		{
			name:    "invalid bond symbol",
			content: "usa:\n  - symbol: AAPL\ndolares:\n  - blue\nbonos:\n  - AL-30\ncripto:\n  - id: bitcoin\n",
			wantErr: "invalid bond symbol",
		},
		{
			name:    "invalid dolar casa",
			content: "usa:\n  - symbol: AAPL\ndolares:\n  - Blue\nbonos:\n  - AL30\ncripto:\n  - id: bitcoin\n",
			wantErr: "invalid dolar casa",
		},
		{
			name:    "duplicate bond symbol",
			content: "usa:\n  - symbol: AAPL\ndolares:\n  - blue\nbonos:\n  - AL30\n  - AL30\ncripto:\n  - id: bitcoin\n",
			wantErr: "duplicate bond symbol",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "watchlist.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}

			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
