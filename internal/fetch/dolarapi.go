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

const dolarAPIURL = "https://dolarapi.com/v1/dolares"

// DolarAPI fetches Argentine FX rates from dolarapi.com.
type DolarAPI struct {
	client *http.Client
	casas  []string
}

// NewDolarAPI creates a DolarAPI provider from the dolares watchlist section.
func NewDolarAPI(client *http.Client, watchlist *config.Watchlist) *DolarAPI {
	return &DolarAPI{
		client: client,
		casas:  append([]string{}, watchlist.Dolares...),
	}
}

// Name returns the provider name.
func (d *DolarAPI) Name() string { return "dolarapi" }

// Fetch retrieves all FX rates and filters them to the configured casas.
func (d *DolarAPI) Fetch(ctx context.Context) (FetchResult, error) {
	now := time.Now().UTC()
	result := FetchResult{
		Source:    d.Name(),
		URL:       dolarAPIURL,
		FetchedAt: now,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dolarAPIURL, nil)
	if err != nil {
		return result, fmt.Errorf("dolarapi: build request: %w", err)
	}
	req = cloneRequest(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("dolarapi: request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return result, fmt.Errorf("dolarapi: read body: %w", err)
	}

	result.Status = resp.StatusCode
	result.Payload = body
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("dolarapi: HTTP %d: %s", resp.StatusCode, string(body))
	}

	quotes, err := parseDolarAPI(body, d.casas, now)
	if err != nil {
		return result, err
	}
	result.Quotes = quotes

	return result, nil
}

// parseDolarAPI converts a dolarapi.com response into filtered quotes.
func parseDolarAPI(body []byte, casas []string, now time.Time) ([]Quote, error) {
	var rows []dolarRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("dolarapi: decode: %w", err)
	}

	wanted := make(map[string]struct{})
	for _, c := range casas {
		wanted[c] = struct{}{}
	}

	var quotes []Quote
	for _, r := range rows {
		casa := strings.TrimSpace(r.Casa)
		if _, ok := wanted[casa]; !ok {
			continue
		}

		quotedAt, _ := time.Parse(time.RFC3339Nano, r.FechaActualizacion)
		name := strings.TrimSpace(r.Nombre)
		if name == "" {
			name = casa
		}

		quotes = append(quotes, Quote{
			Source:    "dolarapi",
			Symbol:    casa,
			Name:      name,
			Price:     r.Venta,
			Bid:       r.Compra,
			Ask:       r.Venta,
			Currency:  "ARS",
			QuotedAt:  quotedAt,
			FetchedAt: now,
		})
	}

	return quotes, nil
}

type dolarRow struct {
	Casa               string  `json:"casa"`
	Nombre             string  `json:"nombre"`
	Compra             float64 `json:"compra"`
	Venta              float64 `json:"venta"`
	FechaActualizacion string  `json:"fechaActualizacion"`
}
