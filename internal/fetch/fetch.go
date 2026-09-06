// Package fetch retrieves normalized EOD market quotes from configured providers.
package fetch

import (
	"context"
	"net/http"
	"time"
)

// DefaultTimeout is the HTTP request timeout for market data calls.
const DefaultTimeout = 15 * time.Second

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"

// Quote is a normalized view of a single market data point.
type Quote struct {
	Source    string
	Symbol    string
	Name      string
	Price     float64
	PrevClose float64
	ChangePct float64
	Currency  string
	Bid       float64
	Ask       float64
	QuotedAt  time.Time
	FetchedAt time.Time
}

// FetchResult groups the raw response and normalized quotes for one provider run.
type FetchResult struct {
	Source    string
	URL       string
	Status    int
	Payload   []byte
	Quotes    []Quote
	FetchedAt time.Time
}

// Provider is the abstraction each data source implements.
type Provider interface {
	Name() string
	Fetch(ctx context.Context) (FetchResult, error)
}

// NewClient returns an HTTP client suitable for market data APIs.
func NewClient() *http.Client {
	return &http.Client{Timeout: DefaultTimeout}
}

// cloneRequest adds the required User-Agent header to a request.
func cloneRequest(req *http.Request) *http.Request {
	out := req.Clone(req.Context())
	out.Header.Set("User-Agent", userAgent)
	return out
}
