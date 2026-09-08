package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// WebQuoteKey identifies a quote by its provider source and symbol.
type WebQuoteKey struct{ Source, Symbol string }

// WebHistoryDefaultLimit is the default number of history rows returned by
// QuoteHistory when the caller explicitly chooses to use it.
const WebHistoryDefaultLimit = 100

// WebHistoryMaxLimit is the hard ceiling for QuoteHistory to prevent unbounded
// reads.
const WebHistoryMaxLimit = 500

// LatestSnapshots returns the newest quote row for each requested (source,
// symbol) pair. Missing pairs are omitted from the map, distinguishing them
// from zero-valued data. An empty pairs slice returns an empty map without
// issuing a query.
func (s *Store) LatestSnapshots(pairs []WebQuoteKey) (map[WebQuoteKey]QuoteRecord, error) {
	out := make(map[WebQuoteKey]QuoteRecord, len(pairs))
	if len(pairs) == 0 {
		return out, nil
	}

	pairSet := make(map[WebQuoteKey]struct{}, len(pairs))
	for _, p := range pairs {
		pairSet[p] = struct{}{}
	}

	args := make([]any, 0, len(pairSet)*2)
	placeholders := make([]string, 0, len(pairSet))
	for p := range pairSet {
		args = append(args, p.Source, p.Symbol)
		placeholders = append(placeholders, "(?,?)")
	}

	query := fmt.Sprintf(`
		WITH ranked AS (
			SELECT
				source, symbol, name, price, prev_close, change_pct,
				currency, bid, ask, quoted_at, fetched_at,
				ROW_NUMBER() OVER (PARTITION BY source, symbol ORDER BY fetched_at DESC, id DESC) AS rn
			FROM quotes
			WHERE (source, symbol) IN (%s)
		)
		SELECT source, symbol, name, price, prev_close, change_pct,
			currency, bid, ask, quoted_at, fetched_at
		FROM ranked
		WHERE rn = 1
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query latest snapshots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r QuoteRecord
		var quotedAt sql.NullTime
		if err := rows.Scan(
			&r.Source, &r.Symbol, &r.Name, &r.Price, &r.PrevClose, &r.ChangePct,
			&r.Currency, &r.Bid, &r.Ask, &quotedAt, &r.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		r.QuotedAt = quotedAt.Time
		out[WebQuoteKey{Source: r.Source, Symbol: r.Symbol}] = r
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return out, nil
}

// QuoteHistory returns history rows for a single (source, symbol) pair inside a
// closed [from, to] time range. Results are ordered by fetched_at ascending,
// then id ascending, so ties are deterministic. Invalid ranges or limits return
// an error instead of widening the query.
func (s *Store) QuoteHistory(source, symbol string, from, to time.Time, limit int) ([]QuoteRecord, error) {
	if from.IsZero() || to.IsZero() || from.After(to) {
		return nil, fmt.Errorf("invalid history range: from %v to %v", from, to)
	}
	if limit <= 0 || limit > WebHistoryMaxLimit {
		return nil, fmt.Errorf("invalid history limit: %d (max %d)", limit, WebHistoryMaxLimit)
	}

	rows, err := s.db.Query(`
		SELECT source, symbol, name, price, prev_close, change_pct,
			currency, bid, ask, quoted_at, fetched_at
		FROM quotes
		WHERE source = ? AND symbol = ? AND fetched_at >= ? AND fetched_at <= ?
		ORDER BY fetched_at ASC, id ASC
		LIMIT ?
	`, source, symbol, from.UTC(), to.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var out []QuoteRecord
	for rows.Next() {
		var r QuoteRecord
		var quotedAt sql.NullTime
		if err := rows.Scan(
			&r.Source, &r.Symbol, &r.Name, &r.Price, &r.PrevClose, &r.ChangePct,
			&r.Currency, &r.Bid, &r.Ask, &quotedAt, &r.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		r.QuotedAt = quotedAt.Time
		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history: %w", err)
	}
	return out, nil
}
