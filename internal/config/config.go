// Package config loads and validates the daily report watchlist.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Asset represents a tracked symbol with an optional display label.
type Asset struct {
	Symbol string `yaml:"symbol"`
	Label  string `yaml:"label"`
}

// CryptoAsset represents a CoinGecko id with a display label.
type CryptoAsset struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

// Watchlist holds all assets the report must cover.
type Watchlist struct {
	USA     []Asset       `yaml:"usa"`
	Dolares []string      `yaml:"dolares"`
	Bonos   []string      `yaml:"bonos"`
	Cripto  []CryptoAsset `yaml:"cripto"`
}

// Load reads a YAML watchlist from path.
func Load(path string) (*Watchlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read watchlist: %w", err)
	}

	var wl Watchlist
	if err := yaml.Unmarshal(data, &wl); err != nil {
		return nil, fmt.Errorf("parse watchlist: %w", err)
	}

	if err := wl.Validate(); err != nil {
		return nil, err
	}

	return &wl, nil
}

// Validate returns an error if the watchlist is empty or inconsistent.
func (w *Watchlist) Validate() error {
	var errs []string

	if len(w.USA) == 0 {
		errs = append(errs, "usa section is empty")
	}
	if len(w.Dolares) == 0 {
		errs = append(errs, "dolares section is empty")
	}
	if len(w.Bonos) == 0 {
		errs = append(errs, "bonos section is empty")
	}
	if len(w.Cripto) == 0 {
		errs = append(errs, "cripto section is empty")
	}

	for i, a := range w.USA {
		if strings.TrimSpace(a.Symbol) == "" {
			errs = append(errs, fmt.Sprintf("usa[%d] missing symbol", i))
		}
	}

	bondRe := regexp.MustCompile(`^[A-Z0-9]+$`)
	seenBonds := make(map[string]struct{})
	for _, b := range w.Bonos {
		b = strings.TrimSpace(b)
		if !bondRe.MatchString(b) {
			errs = append(errs, fmt.Sprintf("invalid bond symbol %q", b))
		}
		if _, ok := seenBonds[b]; ok {
			errs = append(errs, fmt.Sprintf("duplicate bond symbol %q", b))
		}
		seenBonds[b] = struct{}{}
	}

	seenDolares := make(map[string]struct{})
	dolarRe := regexp.MustCompile(`^[a-z0-9]+$`)
	for _, d := range w.Dolares {
		d = strings.TrimSpace(d)
		if !dolarRe.MatchString(d) {
			errs = append(errs, fmt.Sprintf("invalid dolar casa %q", d))
		}
		if _, ok := seenDolares[d]; ok {
			errs = append(errs, fmt.Sprintf("duplicate dolar casa %q", d))
		}
		seenDolares[d] = struct{}{}
	}

	seenCrypto := make(map[string]struct{})
	for i, c := range w.Cripto {
		if strings.TrimSpace(c.ID) == "" {
			errs = append(errs, fmt.Sprintf("cripto[%d] missing id", i))
		}
		if _, ok := seenCrypto[c.ID]; ok {
			errs = append(errs, fmt.Sprintf("duplicate crypto id %q", c.ID))
		}
		seenCrypto[c.ID] = struct{}{}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid watchlist: %s", strings.Join(errs, "; "))
	}

	return nil
}

// USASymbols returns the raw symbol list for USA providers.
func (w *Watchlist) USASymbols() []string {
	out := make([]string, len(w.USA))
	for i, a := range w.USA {
		out[i] = a.Symbol
	}
	return out
}

// LabelForSymbol returns the configured label for a USA symbol, or the symbol itself.
func (w *Watchlist) LabelForSymbol(symbol string) string {
	for _, a := range w.USA {
		if a.Symbol == symbol {
			if a.Label != "" {
				return a.Label
			}
			return a.Symbol
		}
	}
	return symbol
}

// CryptoLabel returns the configured label for a CoinGecko id, or the id itself.
func (w *Watchlist) CryptoLabel(id string) string {
	for _, c := range w.Cripto {
		if c.ID == id {
			if c.Label != "" {
				return c.Label
			}
			return c.ID
		}
	}
	return id
}
