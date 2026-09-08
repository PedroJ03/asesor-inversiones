package web

import (
	"errors"
	"testing"
	"time"
)

func TestAssessFreshness(t *testing.T) {
	now := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		at        time.Time
		err       error
		wantState FreshnessState
	}{
		{
			name:      "current at exactly 24h",
			at:        now.Add(-24 * time.Hour),
			wantState: FreshnessCurrent,
		},
		{
			name:      "current within threshold",
			at:        now.Add(-23 * time.Hour),
			wantState: FreshnessCurrent,
		},
		{
			name:      "stale beyond threshold",
			at:        now.Add(-24*time.Hour - time.Second),
			wantState: FreshnessStale,
		},
		{
			name:      "missing zero time",
			at:        time.Time{},
			wantState: FreshnessMissing,
		},
		{
			name:      "unavailable on error",
			at:        now,
			err:       errors.New("db down"),
			wantState: FreshnessUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessFreshness(tc.at, now, tc.err)
			if got.State != tc.wantState {
				t.Errorf("AssessFreshness state = %q, want %q", got.State, tc.wantState)
			}
		})
	}
}

func TestNewestFreshness(t *testing.T) {
	now := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		records   []FreshnessInfo
		wantState FreshnessState
		wantAt    time.Time
	}{
		{
			name:      "empty returns missing",
			records:   nil,
			wantState: FreshnessMissing,
		},
		{
			name: "unavailable wins over current",
			records: []FreshnessInfo{
				{State: FreshnessCurrent, At: now},
				{State: FreshnessUnavailable},
			},
			wantState: FreshnessUnavailable,
		},
		{
			name: "picks newest timestamp",
			records: []FreshnessInfo{
				{State: FreshnessCurrent, At: now.Add(-2 * time.Hour)},
				{State: FreshnessCurrent, At: now.Add(-1 * time.Hour)},
			},
			wantState: FreshnessCurrent,
			wantAt:    now.Add(-1 * time.Hour),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewestFreshness(tc.records)
			if got.State != tc.wantState {
				t.Errorf("state = %q, want %q", got.State, tc.wantState)
			}
			if !tc.wantAt.IsZero() && !got.At.Equal(tc.wantAt) {
				t.Errorf("at = %v, want %v", got.At, tc.wantAt)
			}
		})
	}
}

func TestFreshnessInfo_Label(t *testing.T) {
	cases := []struct {
		state FreshnessState
		want  string
	}{
		{FreshnessCurrent, "actualizada"},
		{FreshnessStale, "desactualizada"},
		{FreshnessMissing, "sin datos"},
		{FreshnessUnavailable, "no disponible"},
	}
	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			f := FreshnessInfo{State: tc.state}
			if got := f.Label(); got != tc.want {
				t.Errorf("Label() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFreshnessInfo_TimestampFormatted(t *testing.T) {
	at := time.Date(2024, 1, 15, 14, 30, 0, 0, arLocation)
	f := FreshnessInfo{State: FreshnessCurrent, At: at}
	got := f.TimestampFormatted()
	want := "15 de enero de 2024, 14:30"
	if got != want {
		t.Errorf("TimestampFormatted() = %q, want %q", got, want)
	}

	missing := FreshnessInfo{State: FreshnessMissing}
	if missing.TimestampFormatted() != "" {
		t.Error("missing freshness should return empty timestamp")
	}
}
