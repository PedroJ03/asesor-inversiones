package web

import (
	"fmt"
	"time"

	_ "time/tzdata"
)

var arLocation *time.Location

func init() {
	var err error
	arLocation, err = time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		arLocation = time.UTC
	}
}

// FreshnessState describes how recent a quote is.
type FreshnessState string

const (
	// FreshnessCurrent means the quote is no older than 24 hours.
	FreshnessCurrent FreshnessState = "current"
	// FreshnessStale means the quote is older than 24 hours.
	FreshnessStale FreshnessState = "stale"
	// FreshnessMissing means no quote row exists for the requested pair.
	FreshnessMissing FreshnessState = "missing"
	// FreshnessUnavailable means reading the quote failed.
	FreshnessUnavailable FreshnessState = "unavailable"
)

// FreshnessThreshold is the 24-hour boundary used across the platform.
const FreshnessThreshold = 24 * time.Hour

// FreshnessInfo carries the assessed freshness of a quote.
type FreshnessInfo struct {
	State FreshnessState
	At    time.Time
}

// AssessFreshness returns the freshness state for a timestamp. Errors map to
// Unavailable; a zero timestamp maps to Missing; exactly 24 hours is Current.
func AssessFreshness(at time.Time, now time.Time, err error) FreshnessInfo {
	if err != nil {
		return FreshnessInfo{State: FreshnessUnavailable}
	}
	if at.IsZero() {
		return FreshnessInfo{State: FreshnessMissing}
	}
	if now.Sub(at) <= FreshnessThreshold {
		return FreshnessInfo{State: FreshnessCurrent, At: at}
	}
	return FreshnessInfo{State: FreshnessStale, At: at}
}

// NewestFreshness reduces a collection of records to the single freshness that
// should be shown at the top of a quote-bearing page. It picks the newest
// non-zero timestamp and preserves its state; if all are zero it returns Missing.
func NewestFreshness(records []FreshnessInfo) FreshnessInfo {
	var newest FreshnessInfo
	found := false
	for _, r := range records {
		if r.State == FreshnessUnavailable {
			return FreshnessInfo{State: FreshnessUnavailable}
		}
		if !r.At.IsZero() && (!found || r.At.After(newest.At)) {
			newest = r
			found = true
		}
	}
	if !found {
		return FreshnessInfo{State: FreshnessMissing}
	}
	return newest
}

// Class returns the CSS modifier class for the state.
func (f FreshnessInfo) Class() string {
	return fmt.Sprintf("freshness freshness--%s", f.State)
}

// Label returns a short human label for the state.
func (f FreshnessInfo) Label() string {
	switch f.State {
	case FreshnessCurrent:
		return "actualizada"
	case FreshnessStale:
		return "desactualizada"
	case FreshnessMissing:
		return "sin datos"
	case FreshnessUnavailable:
		return "no disponible"
	}
	return "no disponible"
}

// TimestampFormatted returns the freshness timestamp in the advisor's local
// timezone, or an empty string when there is no timestamp.
func (f FreshnessInfo) TimestampFormatted() string {
	if f.At.IsZero() {
		return ""
	}
	t := f.At.In(arLocation)
	return fmt.Sprintf("%d de %s de %d, %02d:%02d", t.Day(), monthName(t.Month()), t.Year(), t.Hour(), t.Minute())
}

var monthsES = []string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

func monthName(m time.Month) string {
	return monthsES[m-1]
}
