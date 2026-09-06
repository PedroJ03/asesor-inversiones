package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/config"
)

const coinGeckoBaseURL = "https://api.coingecko.com/api/v3/simple/price"

// CoinGecko fetches crypto prices from the CoinGecko public API.
type CoinGecko struct {
	client *http.Client
	ids    []string
	labels map[string]string
}

// NewCoinGecko creates a CoinGecko provider from the cripto watchlist section.
func NewCoinGecko(client *http.Client, watchlist *config.Watchlist) *CoinGecko {
	ids := make([]string, len(watchlist.Cripto))
	labels := make(map[string]string)
	for i, c := range watchlist.Cripto {
		ids[i] = c.ID
		labels[c.ID] = watchlist.CryptoLabel(c.ID)
	}
	return &CoinGecko{
		client: client,
		ids:    ids,
		labels: labels,
	}
}

// Name returns the provider name.
func (c *CoinGecko) Name() string { return "coingecko" }

// Fetch retrieves USD prices and 24h changes for the configured ids.
func (c *CoinGecko) Fetch(ctx context.Context) (FetchResult, error) {
	now := time.Now().UTC()
	result := FetchResult{
		Source:    c.Name(),
		FetchedAt: now,
	}

	if len(c.ids) == 0 {
		return result, nil
	}

	u, err := url.Parse(coinGeckoBaseURL)
	if err != nil {
		return result, fmt.Errorf("coingecko: parse base url: %w", err)
	}
	q := u.Query()
	q.Set("ids", strings.Join(c.ids, ","))
	q.Set("vs_currencies", "usd")
	q.Set("include_24hr_change", "true")
	u.RawQuery = q.Encode()
	result.URL = u.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return result, fmt.Errorf("coingecko: build request: %w", err)
	}
	req = cloneRequest(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("coingecko: request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return result, fmt.Errorf("coingecko: read body: %w", err)
	}

	result.Status = resp.StatusCode
	result.Payload = body
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("coingecko: HTTP %d: %s", resp.StatusCode, string(body))
	}

	quotes, err := parseCoinGecko(body, c.ids, c.labels, now)
	if err != nil {
		return result, err
	}
	result.Quotes = quotes

	return result, nil
}

// parseCoinGecko converts a CoinGecko simple/price response into quotes.
func parseCoinGecko(body []byte, ids []string, labels map[string]string, now time.Time) ([]Quote, error) {
	var payload map[string]struct {
		USD          float64 `json:"usd"`
		USD24hChange float64 `json:"usd_24h_change"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("coingecko: decode: %w", err)
	}

	var quotes []Quote
	for _, id := range ids {
		p, ok := payload[id]
		if !ok {
			continue
		}
		quotes = append(quotes, Quote{
			Source:    "coingecko",
			Symbol:    id,
			Name:      labels[id],
			Price:     p.USD,
			ChangePct: p.USD24hChange,
			Currency:  "USD",
			QuotedAt:  now,
			FetchedAt: now,
		})
	}

	return quotes, nil
}
