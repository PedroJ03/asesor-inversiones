// Command web runs the server-rendered web platform.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
	"github.com/PedroJ03/asesor-inversiones/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer s.Close()

	wl, err := config.Load(cfg.WatchlistPath)
	if err != nil {
		return fmt.Errorf("load watchlist: %w", err)
	}

	deps := web.Dependencies{
		Authorizer:  web.NewCookieAuthorizer(cfg.Password, cfg.Secret),
		Assets:      web.Assets,
		RenderLogin: renderLogin,
		Store:       s,
		Watchlist:   wl,
	}

	mux := web.NewMux(deps)

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintln(os.Stdout, "web server listening on", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		fmt.Fprintln(os.Stdout, "received signal", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

type serverConfig struct {
	Addr          string
	Password      string
	Secret        string
	DBPath        string
	WatchlistPath string
}

func loadConfig() (serverConfig, error) {
	var cfg serverConfig
	cfg.Addr = os.Getenv("WEB_ADDR")
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	cfg.Password = os.Getenv("WEB_AUTH_PASSWORD")
	cfg.Secret = os.Getenv("WEB_SESSION_SECRET")
	if cfg.Password == "" || cfg.Secret == "" {
		return cfg, errors.New("WEB_AUTH_PASSWORD and WEB_SESSION_SECRET must be set")
	}
	cfg.DBPath = os.Getenv("WEB_DB_PATH")
	if cfg.DBPath == "" {
		cfg.DBPath = "./asesor.db"
	}
	cfg.WatchlistPath = os.Getenv("WATCHLIST_PATH")
	if cfg.WatchlistPath == "" {
		cfg.WatchlistPath = "./watchlist.yaml"
	}
	return cfg, nil
}

func renderLogin(w http.ResponseWriter, r *http.Request, loginError string) {
	if RequestIsHX(r) {
		_ = web.Fragment(web.LoginPage(loginError)).Render(r.Context(), w)
		return
	}
	_ = web.Shell("Ingresar", "", web.LoginPage(loginError)).Render(r.Context(), w)
}

// RequestIsHX reports whether the request was issued by htmx.
func RequestIsHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
