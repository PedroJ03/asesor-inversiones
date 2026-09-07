// Package store persists raw API payloads and normalized quotes in SQLite.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/PedroJ03/asesor-inversiones/internal/fetch"
	_ "modernc.org/sqlite"
)

// Store wraps a SQLite connection for the report pipeline.
type Store struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at dbPath and runs migrations.
func Open(dbPath string) (*Store, error) {
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS raw_fetch (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			url TEXT,
			fetched_at DATETIME NOT NULL,
			http_status INTEGER,
			payload TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			symbol TEXT NOT NULL,
			name TEXT,
			price REAL,
			prev_close REAL,
			change_pct REAL,
			currency TEXT,
			bid REAL,
			ask REAL,
			quoted_at DATETIME,
			fetched_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_quotes_source_fetched ON quotes(source, fetched_at)`,
		`CREATE INDEX IF NOT EXISTS idx_quotes_symbol ON quotes(symbol)`,
		`CREATE TABLE IF NOT EXISTS alert_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			symbol TEXT NOT NULL,
			kind TEXT CHECK(kind IN ('value','pct')) NOT NULL,
			direction TEXT CHECK(direction IN ('above','below')) NOT NULL,
			threshold REAL NOT NULL,
			baseline_price REAL NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			state TEXT CHECK(state IN ('armed','triggered')) NOT NULL DEFAULT 'armed',
			created_at DATETIME NOT NULL,
			UNIQUE(source,symbol,kind,direction,threshold)
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rule_id INTEGER REFERENCES alert_rules(id) ON DELETE SET NULL,
			source TEXT NOT NULL,
			symbol TEXT NOT NULL,
			kind TEXT NOT NULL,
			threshold REAL NOT NULL,
			observed_price REAL,
			observed_change_pct REAL,
			baseline_price REAL,
			quote_fetched_at DATETIME,
			triggered_at DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_triggered_at ON alerts(triggered_at DESC)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// SaveRaw records a provider's raw API response.
func (s *Store) SaveRaw(source, url string, status int, payload []byte, fetchedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO raw_fetch (source, url, fetched_at, http_status, payload) VALUES (?, ?, ?, ?, ?)`,
		source, url, fetchedAt.UTC(), status, string(payload),
	)
	if err != nil {
		return fmt.Errorf("save raw: %w", err)
	}
	return nil
}

// SaveQuotes records a batch of normalized quotes.
func (s *Store) SaveQuotes(quotes []fetch.Quote) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO quotes
		(source, symbol, name, price, prev_close, change_pct, currency, bid, ask, quoted_at, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, q := range quotes {
		quotedAt := sql.NullTime{Time: q.QuotedAt.UTC(), Valid: !q.QuotedAt.IsZero()}
		_, err := stmt.Exec(
			q.Source, q.Symbol, q.Name, q.Price, q.PrevClose, q.ChangePct,
			q.Currency, q.Bid, q.Ask, quotedAt, q.FetchedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert quote %s/%s: %w", q.Source, q.Symbol, err)
		}
	}

	return tx.Commit()
}

// QuoteRecord mirrors fetch.Quote for readback.
type QuoteRecord struct {
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

// QuotesBySource returns the most recent quotes for a provider, newest first.
func (s *Store) QuotesBySource(source string) ([]QuoteRecord, error) {
	rows, err := s.db.Query(`
		SELECT source, symbol, name, price, prev_close, change_pct, currency, bid, ask, quoted_at, fetched_at
		FROM quotes
		WHERE source = ?
		ORDER BY fetched_at DESC, symbol ASC
	`, source)
	if err != nil {
		return nil, fmt.Errorf("query quotes: %w", err)
	}
	defer rows.Close()

	var out []QuoteRecord
	for rows.Next() {
		var r QuoteRecord
		var quotedAt sql.NullTime
		err := rows.Scan(
			&r.Source, &r.Symbol, &r.Name, &r.Price, &r.PrevClose, &r.ChangePct,
			&r.Currency, &r.Bid, &r.Ask, &quotedAt, &r.FetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan quote: %w", err)
		}
		r.QuotedAt = quotedAt.Time
		out = append(out, r)
	}

	return out, rows.Err()
}

// Rule mirrors an alert_rules row.
type Rule struct {
	ID            int64
	Source        string
	Symbol        string
	Kind          string
	Direction     string
	Threshold     float64
	BaselinePrice float64
	Enabled       bool
	State         string
	CreatedAt     time.Time
}

// Alert mirrors an alerts row.
type Alert struct {
	ID               int64
	RuleID           sql.NullInt64
	Source           string
	Symbol           string
	Kind             string
	Threshold        float64
	ObservedPrice    sql.NullFloat64
	ObservedChangePct sql.NullFloat64
	BaselinePrice    sql.NullFloat64
	QuoteFetchedAt   sql.NullTime
	TriggeredAt      time.Time
}

var (
	// ErrInvalidRule indicates a rule failed validation.
	ErrInvalidRule = errors.New("invalid rule")
	// ErrDuplicateRule indicates a rule with the same identity already exists.
	ErrDuplicateRule = errors.New("duplicate rule")
)

var validSources = map[string]struct{}{
	"yahoo":    {},
	"dolarapi": {},
	"data912":  {},
	"coingecko": {},
}

// CreateRule inserts a validated alert rule and returns its generated id.
func (s *Store) CreateRule(source, symbol, kind, direction string, threshold, baselinePrice float64, createdAt time.Time) (int64, error) {
	if _, ok := validSources[source]; !ok {
		return 0, fmt.Errorf("%w: invalid source %q", ErrInvalidRule, source)
	}
	if kind != "value" && kind != "pct" {
		return 0, fmt.Errorf("%w: invalid kind %q", ErrInvalidRule, kind)
	}
	if direction != "above" && direction != "below" {
		return 0, fmt.Errorf("%w: invalid direction %q", ErrInvalidRule, direction)
	}
	if threshold <= 0 {
		return 0, fmt.Errorf("%w: threshold must be positive", ErrInvalidRule)
	}
	if baselinePrice <= 0 {
		return 0, fmt.Errorf("%w: baseline price must be positive", ErrInvalidRule)
	}

	var exists int
	err := s.db.QueryRow(`
		SELECT 1 FROM alert_rules
		WHERE source = ? AND symbol = ? AND kind = ? AND direction = ? AND threshold = ?
	`, source, symbol, kind, direction, threshold).Scan(&exists)
	if err == nil {
		return 0, ErrDuplicateRule
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("check duplicate: %w", err)
	}

	res, err := s.db.Exec(`
		INSERT INTO alert_rules
		(source, symbol, kind, direction, threshold, baseline_price, enabled, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, source, symbol, kind, direction, threshold, baselinePrice, 1, "armed", createdAt.UTC())
	if err != nil {
		return 0, fmt.Errorf("insert rule: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

// RemoveRule deletes a rule by id, preserving its alert history via ON DELETE SET NULL.
func (s *Store) RemoveRule(id int64) error {
	res, err := s.db.Exec("DELETE FROM alert_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("remove rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("rule %d not found", id)
	}
	return nil
}

// LatestQuote returns the newest stored quote for a source and symbol.
func (s *Store) LatestQuote(source, symbol string) (QuoteRecord, bool, error) {
	var r QuoteRecord
	var quotedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT source, symbol, name, price, prev_close, change_pct, currency, bid, ask, quoted_at, fetched_at
		FROM quotes
		WHERE source = ? AND symbol = ?
		ORDER BY fetched_at DESC, id DESC
		LIMIT 1
	`, source, symbol).Scan(
		&r.Source, &r.Symbol, &r.Name, &r.Price, &r.PrevClose, &r.ChangePct,
		&r.Currency, &r.Bid, &r.Ask, &quotedAt, &r.FetchedAt,
	)
	if err == sql.ErrNoRows {
		return r, false, nil
	}
	if err != nil {
		return r, false, fmt.Errorf("query latest quote: %w", err)
	}
	r.QuotedAt = quotedAt.Time
	return r, true, nil
}

// UpdateRuleState sets the state of a rule.
func (s *Store) UpdateRuleState(id int64, state string) error {
	if state != "armed" && state != "triggered" {
		return fmt.Errorf("%w: invalid state %q", ErrInvalidRule, state)
	}
	_, err := s.db.Exec("UPDATE alert_rules SET state = ? WHERE id = ?", state, id)
	if err != nil {
		return fmt.Errorf("update rule state: %w", err)
	}
	return nil
}

// InsertAlert records a triggered alert snapshot.
func (s *Store) InsertAlert(a Alert) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO alerts
		(rule_id, source, symbol, kind, threshold, observed_price, observed_change_pct, baseline_price, quote_fetched_at, triggered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.RuleID, a.Source, a.Symbol, a.Kind, a.Threshold, a.ObservedPrice, a.ObservedChangePct, a.BaselinePrice, a.QuoteFetchedAt, a.TriggeredAt.UTC())
	if err != nil {
		return 0, fmt.Errorf("insert alert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

// History returns the most recent alert snapshots, bounded by limit.
func (s *Store) History(limit int) ([]Alert, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit must be positive", ErrInvalidRule)
	}
	rows, err := s.db.Query(`
		SELECT id, rule_id, source, symbol, kind, threshold, observed_price, observed_change_pct, baseline_price, quote_fetched_at, triggered_at
		FROM alerts
		ORDER BY triggered_at DESC, id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var out []Alert
	for rows.Next() {
		var a Alert
		err := rows.Scan(
			&a.ID, &a.RuleID, &a.Source, &a.Symbol, &a.Kind, &a.Threshold,
			&a.ObservedPrice, &a.ObservedChangePct, &a.BaselinePrice, &a.QuoteFetchedAt, &a.TriggeredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListRules returns persisted rules, optionally filtering to enabled rules only.
func (s *Store) ListRules(enabledOnly bool) ([]Rule, error) {
	query := `
		SELECT id, source, symbol, kind, direction, threshold, baseline_price, enabled, state, created_at
		FROM alert_rules
	`
	if enabledOnly {
		query += "WHERE enabled = 1 "
	}
	query += "ORDER BY created_at DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	var out []Rule
	for rows.Next() {
		var r Rule
		var enabled int
		err := rows.Scan(
			&r.ID, &r.Source, &r.Symbol, &r.Kind, &r.Direction, &r.Threshold,
			&r.BaselinePrice, &enabled, &r.State, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		r.Enabled = enabled == 1
		out = append(out, r)
	}

	return out, rows.Err()
}
