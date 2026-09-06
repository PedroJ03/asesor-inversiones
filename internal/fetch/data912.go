package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
)

const data912URL = "https://data912.com/live/arg_bonds"

// Data912 fetches Argentine sovereign bond quotes from data912.com.
type Data912 struct {
	client  *http.Client
	symbols []string
}

// NewData912 creates a Data912 provider from the bonos watchlist section.
func NewData912(client *http.Client, watchlist *config.Watchlist) *Data912 {
	return &Data912{
		client:  client,
		symbols: append([]string{}, watchlist.Bonos...),
	}
}

// Name returns the provider name.
func (d *Data912) Name() string { return "data912" }

// Fetch retrieves all bond rows and filters them to the configured symbols.
func (d *Data912) Fetch(ctx context.Context) (FetchResult, error) {
	now := time.Now().UTC()
	result := FetchResult{
		Source:    d.Name(),
		URL:       data912URL,
		FetchedAt: now,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, data912URL, nil)
	if err != nil {
		return result, fmt.Errorf("data912: build request: %w", err)
	}
	req = cloneRequest(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("data912: request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return result, fmt.Errorf("data912: read body: %w", err)
	}

	result.Status = resp.StatusCode
	result.Payload = body
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("data912: HTTP %d: %s", resp.StatusCode, string(body))
	}

	quotes, err := parseData912(body, d.symbols, now)
	if err != nil {
		return result, err
	}
	result.Quotes = quotes

	return result, nil
}

// parseData912 converts a data912.com response into filtered bond quotes.
func parseData912(body []byte, symbols []string, now time.Time) ([]Quote, error) {
	var rows []bondRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("data912: decode: %w", err)
	}

	wanted := make(map[string]struct{})
	for _, s := range symbols {
		wanted[s] = struct{}{}
	}

	var quotes []Quote
	for _, r := range rows {
		symbol := strings.TrimSpace(r.Symbol)
		if _, ok := wanted[symbol]; !ok {
			continue
		}

		currency := "ARS"
		if strings.HasSuffix(symbol, "C") {
			currency = "USD"
		}

		prevClose := 0.0
		if r.PctChange != 0 {
			prevClose = r.C / (1 + r.PctChange/100)
		}

		quotes = append(quotes, Quote{
			Source:    "data912",
			Symbol:    symbol,
			Name:      symbol,
			Price:     r.C,
			PrevClose: prevClose,
			ChangePct: r.PctChange,
			Currency:  currency,
			Bid:       r.PxBid,
			Ask:       r.PxAsk,
			QuotedAt:  now,
			FetchedAt: now,
		})
	}

	return quotes, nil
}

type bondRow struct {
	Symbol    string  `json:"symbol"`
	PxBid     float64 `json:"px_bid"`
	PxAsk     float64 `json:"px_ask"`
	C         float64 `json:"c"`
	PctChange float64 `json:"pct_change"`
}
