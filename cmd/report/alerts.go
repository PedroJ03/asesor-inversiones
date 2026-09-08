package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/alert"
	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

// alertQuoteMaxAge matches the platform-wide freshness policy: quotes older
// than a day are considered stale and never trigger alerts.
const alertQuoteMaxAge = 24 * time.Hour

// evaluateAlerts evaluates every enabled rule against the freshest stored
// quotes and persists triggered alerts and rule state transitions. The caller
// is responsible for treating a returned error as non-fatal.
func evaluateAlerts(st *store.Store) error {
	rules, err := st.ListRules(true)
	if err != nil {
		return fmt.Errorf("list rules: %w", err)
	}

	result, err := alert.Run(toAlertRules(rules), resolveStoreQuote(st), storeAlertPersister{st: st}, time.Now().UTC(), alertQuoteMaxAge)
	if err != nil {
		return err
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

// storeAlertPersister adapts the report pipeline's store to the alert engine's
// persistence interface.
type storeAlertPersister struct {
	st *store.Store
}

func (p storeAlertPersister) InsertTriggered(a alert.Alert) error {
	_, err := p.st.InsertAlert(toStoreAlert(a))
	return err
}

func (p storeAlertPersister) TransitionState(ruleID int64, to string) error {
	return p.st.UpdateRuleState(ruleID, to)
}

// resolveStoreQuote resolves a rule's asset to the newest stored quote.
func resolveStoreQuote(st *store.Store) func(string, string) (alert.Quote, bool, error) {
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
