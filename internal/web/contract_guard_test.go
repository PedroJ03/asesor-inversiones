package web

import (
	"testing"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/store"
)

// platformEngineContract lists every frozen store method consumed by the web
// platform's rule-management and read layers. This compile-time assertion makes
// the build fail loudly if the engine contract disappears, rather than letting
// the platform invent substitute methods.
type platformEngineContract interface {
	CreateRule(source, symbol, kind, direction string, threshold, baselinePrice float64, createdAt time.Time) (int64, error)
	ListRules(enabledOnly bool) ([]store.Rule, error)
	RemoveRule(id int64) error
	SetRuleEnabled(id int64, enabled bool) error
	UpdateRuleState(id int64, state string) error
	LatestQuote(source, symbol string) (store.QuoteRecord, bool, error)
	History(limit int) ([]store.Alert, error)
}

var _ platformEngineContract = (*store.Store)(nil)

func TestPlatformEngineContractGuard(t *testing.T) {
	// The assertion above is compile-time; this test exists so the guard is
	// exercised by `go test` and reported in coverage tooling.
}
