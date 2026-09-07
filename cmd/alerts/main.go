// Command alerts manages and evaluates alert rules from stored quotes.
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/alert"
	"github.com/PedroJ03/asesor-inversiones/internal/config"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

var (
	validSources = map[string]struct{}{
		"yahoo":     {},
		"dolarapi":  {},
		"data912":   {},
		"coingecko": {},
	}

	validKinds = map[string]struct{}{
		"value": {},
		"pct":   {},
	}

	validDirections = map[string]struct{}{
		"above": {},
		"below": {},
	}
)

const (
	watchlistPath = "watchlist.yaml"
	dbDefault     = "data/asesor.db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: alerts <run|add|list|remove|history> [flags]")
		os.Exit(1)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	var err error
	switch subcommand {
	case "run":
		err = runAlerts(args)
	case "add":
		err = runAdd(args)
	case "list":
		err = runList(args)
	case "remove":
		err = runRemove(args)
	case "history":
		err = runHistory(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", subcommand)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runAlerts(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dbFlag := fs.String("db", dbDefault, "path to SQLite database")
	maxAgeFlag := fs.Duration("max-age", 24*time.Hour, "maximum quote age before stale")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := store.Open(*dbFlag)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	rules, err := st.ListRules(true)
	if err != nil {
		return fmt.Errorf("list rules: %w", err)
	}

	now := time.Now().UTC()
	result := alert.Evaluate(toAlertRules(rules), resolveQuote(st), now, *maxAgeFlag)

	for _, t := range result.Transitions {
		if err := st.UpdateRuleState(t.RuleID, t.To); err != nil {
			return fmt.Errorf("update rule %d state: %w", t.RuleID, err)
		}
	}

	for _, a := range result.Triggered {
		if _, err := st.InsertAlert(toStoreAlert(a)); err != nil {
			return fmt.Errorf("insert alert %s/%s: %w", a.Source, a.Symbol, err)
		}
	}

	for _, a := range result.Triggered {
		fmt.Printf("ALERT %s/%s %s %.4f (price=%.4f baseline=%.4f at=%s)\n",
			a.Source, a.Symbol, a.Kind, a.Threshold,
			a.ObservedPrice, a.BaselinePrice, a.TriggeredAt.Format(time.RFC3339))
	}

	for _, w := range result.Warnings {
		fmt.Fprintln(os.Stderr, "WARNING:", w)
	}

	return nil
}

func runAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	dbFlag := fs.String("db", dbDefault, "path to SQLite database")
	watchlistFlag := fs.String("watchlist", watchlistPath, "path to watchlist YAML file")
	sourceFlag := fs.String("source", "", "quote source (yahoo, dolarapi, data912, coingecko)")
	symbolFlag := fs.String("symbol", "", "asset symbol")
	kindFlag := fs.String("kind", "", "rule kind (value, pct)")
	directionFlag := fs.String("direction", "", "rule direction (above, below)")
	thresholdFlag := fs.Float64("threshold", 0, "threshold value")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sourceFlag == "" || *symbolFlag == "" || *kindFlag == "" || *directionFlag == "" || *thresholdFlag == 0 {
		return errors.New("missing required flag: -source, -symbol, -kind, -direction, -threshold")
	}

	wl, err := config.Load(*watchlistFlag)
	if err != nil {
		return fmt.Errorf("load watchlist: %w", err)
	}

	if err := validateAdd(*sourceFlag, *symbolFlag, *kindFlag, *directionFlag, *thresholdFlag, wl); err != nil {
		return err
	}

	st, err := store.Open(*dbFlag)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	quote, ok, err := st.LatestQuote(*sourceFlag, *symbolFlag)
	if err != nil {
		return fmt.Errorf("latest quote: %w", err)
	}
	if !ok {
		return fmt.Errorf("no stored quote for %s/%s; fetch quotes before adding a rule", *sourceFlag, *symbolFlag)
	}

	id, err := st.CreateRule(*sourceFlag, *symbolFlag, *kindFlag, *directionFlag, *thresholdFlag, quote.Price, quote.FetchedAt)
	if err != nil {
		return fmt.Errorf("create rule: %w", err)
	}

	fmt.Printf("Rule %d created for %s/%s (%s %s %.4f)\n", id, *sourceFlag, *symbolFlag, *kindFlag, *directionFlag, *thresholdFlag)
	fmt.Printf("Captured baseline price=%.4f from quote fetched at %s\n", quote.Price, quote.FetchedAt.Format(time.RFC3339))
	return nil
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	dbFlag := fs.String("db", dbDefault, "path to SQLite database")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := store.Open(*dbFlag)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	rules, err := st.ListRules(false)
	if err != nil {
		return fmt.Errorf("list rules: %w", err)
	}

	if len(rules) == 0 {
		fmt.Println("No rules.")
		return nil
	}

	fmt.Printf("%-4s %-10s %-10s %-6s %-10s %-10s %-10s %-10s %-8s\n", "ID", "SOURCE", "SYMBOL", "KIND", "DIRECTION", "THRESHOLD", "BASELINE", "STATE", "ENABLED")
	for _, r := range rules {
		enabled := "no"
		if r.Enabled {
			enabled = "yes"
		}
		fmt.Printf("%-4d %-10s %-10s %-6s %-10s %-10.4f %-10.4f %-10s %-8s\n",
			r.ID, r.Source, r.Symbol, r.Kind, r.Direction, r.Threshold, r.BaselinePrice, r.State, enabled)
	}
	return nil
}

func runRemove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	dbFlag := fs.String("db", dbDefault, "path to SQLite database")
	idFlag := fs.Int64("id", 0, "rule id to remove")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *idFlag <= 0 {
		return errors.New("invalid rule id")
	}

	st, err := store.Open(*dbFlag)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	if err := st.RemoveRule(*idFlag); err != nil {
		return fmt.Errorf("remove rule: %w", err)
	}
	fmt.Printf("Rule %d removed.\n", *idFlag)
	return nil
}

func runHistory(args []string) error {
	fs := flag.NewFlagSet("history", flag.ExitOnError)
	dbFlag := fs.String("db", dbDefault, "path to SQLite database")
	limitFlag := fs.Int("limit", 10, "maximum history entries")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *limitFlag <= 0 {
		return errors.New("invalid limit")
	}

	st, err := store.Open(*dbFlag)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	entries, err := st.History(*limitFlag)
	if err != nil {
		return fmt.Errorf("history: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No history.")
		return nil
	}

	fmt.Printf("%-4s %-10s %-10s %-6s %-10s %-12s %-12s %-12s %-20s\n",
		"ID", "SOURCE", "SYMBOL", "KIND", "THRESHOLD", "PRICE", "CHANGE_PCT", "BASELINE", "TRIGGERED_AT")
	for _, e := range entries {
		price := "-"
		if e.ObservedPrice.Valid {
			price = strconv.FormatFloat(e.ObservedPrice.Float64, 'f', 4, 64)
		}
		changePct := "-"
		if e.ObservedChangePct.Valid {
			changePct = strconv.FormatFloat(e.ObservedChangePct.Float64, 'f', 4, 64)
		}
		baseline := "-"
		if e.BaselinePrice.Valid {
			baseline = strconv.FormatFloat(e.BaselinePrice.Float64, 'f', 4, 64)
		}
		fmt.Printf("%-4d %-10s %-10s %-6s %-10.4f %-12s %-12s %-12s %-20s\n",
			e.ID, e.Source, e.Symbol, e.Kind, e.Threshold, price, changePct, baseline,
			e.TriggeredAt.Format(time.RFC3339))
	}
	return nil
}

func validateAdd(source, symbol, kind, direction string, threshold float64, wl *config.Watchlist) error {
	if _, ok := validSources[source]; !ok {
		return fmt.Errorf("invalid source %q: must be one of yahoo, dolarapi, data912, coingecko", source)
	}
	if _, ok := validKinds[kind]; !ok {
		return fmt.Errorf("invalid kind %q: must be value or pct", kind)
	}
	if _, ok := validDirections[direction]; !ok {
		return fmt.Errorf("invalid direction %q: must be above or below", direction)
	}
	if threshold <= 0 {
		return errors.New("threshold must be positive")
	}
	if !isWatchlisted(source, symbol, wl) {
		return fmt.Errorf("%s/%s is not in watchlist.yaml", source, symbol)
	}
	return nil
}

func isWatchlisted(source, symbol string, wl *config.Watchlist) bool {
	symbol = strings.TrimSpace(symbol)
	switch source {
	case "yahoo":
		for _, a := range wl.USA {
			if strings.EqualFold(strings.TrimSpace(a.Symbol), symbol) {
				return true
			}
		}
	case "dolarapi":
		for _, d := range wl.Dolares {
			if strings.EqualFold(strings.TrimSpace(d), symbol) {
				return true
			}
		}
	case "data912":
		for _, b := range wl.Bonos {
			if strings.EqualFold(strings.TrimSpace(b), symbol) {
				return true
			}
		}
	case "coingecko":
		for _, c := range wl.Cripto {
			if strings.EqualFold(strings.TrimSpace(c.ID), symbol) {
				return true
			}
		}
	}
	return false
}

func toAlertRules(rules []store.Rule) []alert.Rule {
	out := make([]alert.Rule, len(rules))
	for i, r := range rules {
		out[i] = alert.Rule{
			ID:            r.ID,
			Source:        r.Source,
			Symbol:        r.Symbol,
			Kind:          r.Kind,
			Direction:     r.Direction,
			Threshold:     r.Threshold,
			BaselinePrice: r.BaselinePrice,
			State:         r.State,
		}
	}
	return out
}

func resolveQuote(st *store.Store) func(string, string) (alert.Quote, bool, error) {
	return func(source, symbol string) (alert.Quote, bool, error) {
		q, ok, err := st.LatestQuote(source, symbol)
		if err != nil {
			return alert.Quote{}, false, err
		}
		if !ok {
			return alert.Quote{}, false, nil
		}
		return alert.Quote{Price: q.Price, FetchedAt: q.FetchedAt}, true, nil
	}
}

func toStoreAlert(a alert.Alert) store.Alert {
	return store.Alert{
		RuleID:            sql.NullInt64{Int64: a.RuleID, Valid: true},
		Source:            a.Source,
		Symbol:            a.Symbol,
		Kind:              a.Kind,
		Threshold:         a.Threshold,
		ObservedPrice:     sql.NullFloat64{Float64: a.ObservedPrice, Valid: true},
		ObservedChangePct: sql.NullFloat64{Float64: a.ObservedChangePct, Valid: a.Kind == "pct"},
		BaselinePrice:     sql.NullFloat64{Float64: a.BaselinePrice, Valid: true},
		QuoteFetchedAt:    sql.NullTime{Time: a.QuoteFetchedAt, Valid: !a.QuoteFetchedAt.IsZero()},
		TriggeredAt:       a.TriggeredAt,
	}
}
