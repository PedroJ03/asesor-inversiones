// Command alerts manages and evaluates alert rules from stored quotes.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/alert"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

const dbDefault = "data/asesor.db"

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
