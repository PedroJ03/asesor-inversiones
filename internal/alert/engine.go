// Package alert implements a pure alert evaluation engine over stored quotes.
package alert

import (
	"fmt"
	"time"
)

// Rule is the input view of an alert rule for evaluation.
type Rule struct {
	ID            int64
	Source        string
	Symbol        string
	Kind          string // "value" or "pct"
	Direction     string // "above" or "below"
	Threshold     float64
	BaselinePrice float64
	State         string // "armed" or "triggered"
}

// Quote is the input view of the latest stored quote for a rule's asset.
type Quote struct {
	Price     float64
	FetchedAt time.Time
}

// Alert represents a triggered alert snapshot.
type Alert struct {
	RuleID          int64
	Source          string
	Symbol          string
	Kind            string
	Threshold       float64
	ObservedPrice   float64
	ObservedChangePct float64
	BaselinePrice   float64
	QuoteFetchedAt  time.Time
	TriggeredAt     time.Time
}

// StateChange records a rule transition.
type StateChange struct {
	RuleID int64
	From   string
	To     string
}

// Result is the output of Evaluate.
type Result struct {
	Triggered   []Alert
	Transitions []StateChange
	Warnings    []string
}

// Evaluate checks each rule against the quote resolver and returns triggers,
// state transitions, and warnings. It performs no I/O.
func Evaluate(rules []Rule, resolve func(string, string) (Quote, bool, error), now time.Time, maxAge time.Duration) Result {
	var res Result

	for _, rule := range rules {
		quote, ok, err := resolve(rule.Source, rule.Symbol)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s/%s: resolve error: %v", rule.Source, rule.Symbol, err))
			continue
		}
		if !ok {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s/%s: no quote available", rule.Source, rule.Symbol))
			continue
		}
		if now.Sub(quote.FetchedAt) > maxAge {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s/%s: quote stale (fetched %s)", rule.Source, rule.Symbol, quote.FetchedAt.Format(time.RFC3339)))
			continue
		}

		condition := evaluateCondition(rule, quote.Price)

		switch rule.State {
		case "armed":
			if condition {
				res.Triggered = append(res.Triggered, buildAlert(rule, quote, now))
				res.Transitions = append(res.Transitions, StateChange{RuleID: rule.ID, From: "armed", To: "triggered"})
			}
		case "triggered":
			if !condition {
				res.Transitions = append(res.Transitions, StateChange{RuleID: rule.ID, From: "triggered", To: "armed"})
			}
		}
	}

	return res
}

func evaluateCondition(rule Rule, price float64) bool {
	switch rule.Kind {
	case "value":
		if rule.Direction == "above" {
			return price >= rule.Threshold
		}
		return price <= rule.Threshold
	case "pct":
		drift := (price - rule.BaselinePrice) / rule.BaselinePrice * 100
		if rule.Direction == "above" {
			return drift >= rule.Threshold
		}
		return drift <= -rule.Threshold
	}
	return false
}

func buildAlert(rule Rule, quote Quote, now time.Time) Alert {
	alert := Alert{
		RuleID:         rule.ID,
		Source:         rule.Source,
		Symbol:         rule.Symbol,
		Kind:           rule.Kind,
		Threshold:      rule.Threshold,
		ObservedPrice:  quote.Price,
		BaselinePrice:  rule.BaselinePrice,
		QuoteFetchedAt: quote.FetchedAt,
		TriggeredAt:    now,
	}
	if rule.Kind == "pct" {
		alert.ObservedChangePct = (quote.Price - rule.BaselinePrice) / rule.BaselinePrice * 100
	}
	return alert
}
