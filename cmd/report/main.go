// Command report fetches market data and renders the daily HTML report.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "time/tzdata"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	"github.com/PedroJ03/asesor-inversiones/internal/render"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

var (
	watchlistPath = flag.String("watchlist", "watchlist.yaml", "path to watchlist YAML file")
	dbPath        = flag.String("db", "data/asesor.db", "path to SQLite database")
	outDir        = flag.String("out", "reporte", "output directory for the HTML report")
)

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	wl, err := config.Load(*watchlistPath)
	if err != nil {
		return fmt.Errorf("load watchlist: %w", err)
	}

	st, err := store.Open(*dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	client := fetch.NewClient()
	providers := []fetch.Provider{
		fetch.NewYahoo(client, wl),
		fetch.NewDolarAPI(client, wl),
		fetch.NewData912(client, wl),
		fetch.NewCoinGecko(client, wl),
	}

	var allQuotes []fetch.Quote
	var warnings []string

	for _, p := range providers {
		res, err := p.Fetch(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %s", p.Name(), err))
			continue
		}
		if err := st.SaveRaw(res.Source, res.URL, res.Status, res.Payload, res.FetchedAt); err != nil {
			return fmt.Errorf("save raw %s: %w", res.Source, err)
		}
		if err := st.SaveQuotes(res.Quotes); err != nil {
			return fmt.Errorf("save quotes %s: %w", res.Source, err)
		}
		allQuotes = append(allQuotes, res.Quotes...)
	}

	// Evaluate alert rules against the freshest stored quotes now that the
	// provider quotes are saved. A failure here must never abort report
	// generation: log it and keep going.
	if err := evaluateAlerts(st); err != nil {
		fmt.Fprintln(os.Stderr, "warning: alert evaluation:", err)
	}

	reportTime := time.Now().In(arLocation())
	data := render.FromQuotes(allQuotes, warnings, reportTime)
	html, err := render.Render(data)
	if err != nil {
		return fmt.Errorf("render report: %w", err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	dateFile := filepath.Join(*outDir, fmt.Sprintf("reporte-%s.html", reportTime.Format("2006-01-02")))
	indexFile := filepath.Join(*outDir, "index.html")

	if err := os.WriteFile(dateFile, html, 0o644); err != nil {
		return fmt.Errorf("write dated report: %w", err)
	}
	if err := os.WriteFile(indexFile, html, 0o644); err != nil {
		return fmt.Errorf("write index report: %w", err)
	}

	fmt.Println("Report generated:")
	fmt.Println(" ", dateFile)
	fmt.Println(" ", indexFile)
	if len(warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range warnings {
			fmt.Println(" -", w)
		}
	}
	fmt.Printf("\nQuotes fetched: %d\n", len(allQuotes))
	fmt.Println("Sources:", strings.Join(sourceNames(providers), ", "))

	return nil
}

func arLocation() *time.Location {
	loc, err := time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		return time.UTC
	}
	return loc
}

func sourceNames(providers []fetch.Provider) []string {
	out := make([]string, len(providers))
	for i, p := range providers {
		out[i] = p.Name()
	}
	return out
}
