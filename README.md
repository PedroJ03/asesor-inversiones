# Daily Market Report

Go prototype that fetches end-of-day market data from free APIs, persists it to SQLite, and renders a mobile-first HTML report for an Argentine investment advisor.

## Quick start

```bash
# Generate today's report
go run ./cmd/report

# Open the result
xdg-open reporte/index.html
```

Default flags:

- `-watchlist watchlist.yaml` — asset configuration
- `-db data/asesor.db` — SQLite database
- `-out reporte` — output directory for HTML files

## Project layout

```
cmd/report/main.go          # CLI entry point
internal/config/            # watchlist.yaml loading and validation
internal/fetch/             # Provider interface + Yahoo, dolarapi, data912, CoinGecko
internal/store/             # SQLite persistence (raw payloads + normalized quotes)
internal/render/            # mobile-first HTML report (Spanish UI)
watchlist.yaml              # editable assets
```

## Edit the watchlist

Update `watchlist.yaml` to change the symbols tracked by each provider:

- `usa` — Yahoo Finance symbols (stocks/ETFs). Index proxies such as SPY/QQQ/DIA are used because index values require a separate license.
- `dolares` — dolarapi.com `casa` values, e.g. `oficial`, `blue`, `bolsa`, `contadoconliqui`.
- `bonos` — data912.com bond symbols. Suffix `D` = MEP in ARS, `C` = cable in USD.
- `cripto` — CoinGecko ids, e.g. `bitcoin`, `ethereum`.

## Swapping Yahoo for Tiingo

The USA provider implements the small `fetch.Provider` interface. Replacing Yahoo with Tiingo (or any paid source) only requires adding a new provider under `internal/fetch/` and swapping it in `cmd/report/main.go`; `store` and `render` stay untouched.

## Tests

```bash
go test ./...
```

All provider tests run offline against JSON fixtures in `internal/fetch/testdata/`.

## Data sources and attribution

- USA: Yahoo Finance (unofficial endpoint; for prototyping only)
- Argentine FX: [dolarapi.com](https://dolarapi.com/)
- Argentine sovereign bonds: [data912.com](https://data912.com/)
- Crypto: [CoinGecko](https://www.coingecko.com/) (public API; attribution shown in report footer)

## Notes

- The report is generated manually in this prototype. WhatsApp delivery and hosting are out of scope for now.
- CoinGecko's free public tier has rate limits; heavy use may trigger temporary blocks.
- Yahoo Finance is not a licensed commercial data feed; treat it as swappable scaffolding.
