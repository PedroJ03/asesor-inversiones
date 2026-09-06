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

const yahooChartURLTemplate = "https://query1.finance.yahoo.com/v8/finance/chart/%s?range=5d&interval=1d"

// Yahoo fetches EOD stock/ETF quotes from Yahoo Finance's chart endpoint.
type Yahoo struct {
	client  *http.Client
	symbols []string
	labels  map[string]string
}

// NewYahoo creates a Yahoo provider from the USA watchlist section.
func NewYahoo(client *http.Client, watchlist *config.Watchlist) *Yahoo {
	labels := make(map[string]string)
	for _, a := range watchlist.USA {
		labels[a.Symbol] = watchlist.LabelForSymbol(a.Symbol)
	}
	return &Yahoo{
		client:  client,
		symbols: watchlist.USASymbols(),
		labels:  labels,
	}
}

// Name returns the provider name.
func (y *Yahoo) Name() string { return "yahoo" }

// Fetch retrieves each configured symbol and returns a combined result.
func (y *Yahoo) Fetch(ctx context.Context) (FetchResult, error) {
	now := time.Now().UTC()
	result := FetchResult{
		Source:    y.Name(),
		URL:       yahooChartURLTemplate,
		Status:    http.StatusOK,
		FetchedAt: now,
	}

	rawResponses := make([]map[string]interface{}, 0, len(y.symbols))
	for _, sym := range y.symbols {
		url := fmt.Sprintf(yahooChartURLTemplate, sym)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return result, fmt.Errorf("yahoo %s: build request: %w", sym, err)
		}
		req = cloneRequest(req)

		resp, err := y.client.Do(req)
		if err != nil {
			return result, fmt.Errorf("yahoo %s: request: %w", sym, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return result, fmt.Errorf("yahoo %s: read body: %w", sym, err)
		}

		if resp.StatusCode != http.StatusOK {
			return result, fmt.Errorf("yahoo %s: HTTP %d: %s", sym, resp.StatusCode, string(body))
		}

		quote, raw, err := parseYahooOne(body, y.labels, now)
		if err != nil {
			return result, fmt.Errorf("yahoo %s: %w", sym, err)
		}
		result.Quotes = append(result.Quotes, quote)
		rawResponses = append(rawResponses, raw)
	}

	combined, err := json.Marshal(rawResponses)
	if err != nil {
		return result, fmt.Errorf("yahoo: combine raw payloads: %w", err)
	}
	result.Payload = combined

	return result, nil
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta chartMeta `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type chartMeta struct {
	Currency               string  `json:"currency"`
	Symbol                 string  `json:"symbol"`
	RegularMarketTime      int64   `json:"regularMarketTime"`
	RegularMarketPrice     float64 `json:"regularMarketPrice"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
	ChartPreviousClose     float64 `json:"chartPreviousClose"`
	LongName               string  `json:"longName"`
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// parseYahooOne converts a single Yahoo chart response body into a Quote and the raw map.
func parseYahooOne(body []byte, labels map[string]string, now time.Time) (Quote, map[string]interface{}, error) {
	var wrapper chartResponse
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return Quote{}, nil, fmt.Errorf("decode: %w", err)
	}
	if wrapper.Chart.Error != nil {
		return Quote{}, nil, fmt.Errorf("API error %s: %s", wrapper.Chart.Error.Code, wrapper.Chart.Error.Description)
	}
	if len(wrapper.Chart.Result) == 0 {
		return Quote{}, nil, fmt.Errorf("empty result")
	}

	q := wrapper.Chart.Result[0].Meta
	quote := Quote{
		Source:    "yahoo",
		Symbol:    q.Symbol,
		Name:      firstNonEmpty(q.LongName, labels[q.Symbol], q.Symbol),
		Price:     q.RegularMarketPrice,
		PrevClose: q.ChartPreviousClose,
		ChangePct: q.RegularMarketChangePercent,
		Currency:  q.Currency,
		QuotedAt:  time.Unix(q.RegularMarketTime, 0).UTC(),
		FetchedAt: now,
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)
	return quote, raw, nil
}
