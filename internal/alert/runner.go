package alert

import (
	"fmt"
	"time"
)

// Persister receives the outcomes of an evaluation run. It decouples the pure
// engine from any concrete storage implementation.
type Persister interface {
	// InsertTriggered records a triggered alert snapshot.
	InsertTriggered(a Alert) error
	// TransitionState persists a rule state change.
	TransitionState(ruleID int64, to string) error
}

// Run evaluates the rules against the resolver and persists every trigger and
// state transition. Transitions are persisted before alerts so that a partially
// failed run still leaves rules in a state the next pass can recover from.
// It returns the evaluation result alongside any persistence error; warnings
// are surfaced to the caller through the result.
func Run(rules []Rule, resolve func(string, string) (Quote, bool, error), persist Persister, now time.Time, maxAge time.Duration) (Result, error) {
	result := Evaluate(rules, resolve, now, maxAge)

	for _, t := range result.Transitions {
		if err := persist.TransitionState(t.RuleID, t.To); err != nil {
			return result, fmt.Errorf("persist rule %d state: %w", t.RuleID, err)
		}
	}
	for _, a := range result.Triggered {
		if err := persist.InsertTriggered(a); err != nil {
			return result, fmt.Errorf("persist alert %s/%s: %w", a.Source, a.Symbol, err)
		}
	}
	return result, nil
}
