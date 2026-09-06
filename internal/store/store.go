// Package store persists raw API payloads and normalized quotes in SQLite.
package store

import (
	"database/sql"
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
