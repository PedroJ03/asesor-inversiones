# Daily Market Report

[![CI](https://github.com/PedroJ03/asesor-inversiones/actions/workflows/ci.yml/badge.svg)](https://github.com/PedroJ03/asesor-inversiones/actions/workflows/ci.yml)

Self-hosted market briefing for an Argentine investment advisor. Go services fetch end-of-day data from free APIs, persist it to SQLite, and render a Spanish-language, mobile-first report — consumable as a static HTML file or through an authenticated web platform with an alert engine.

Built with Go · SQLite (pure-Go driver) · html/template + templ · htmx · PWA

## Highlights

| Capability | Detail |
|---|---|
| Daily report (CLI) | Fetches US equities, ARS exchange rates, sovereign bonds, and crypto; renders a dated, mobile-first HTML report with es-AR number formatting and a pinned Buenos Aires timezone. |
| Web platform | Server-rendered UI over the same database: session authentication, alert-rule management, freshness indicators, and an installable PWA that works offline. |
| Alert engine | Validated price rules with an armed/triggered lifecycle; triggered snapshots are persisted for history. |
| Four data providers | Yahoo Finance, DolarAPI, data912, and CoinGecko behind a single `Provider` interface. A failing provider degrades to a warning in the report instead of aborting the run. |
| SQLite persistence | CGo-free driver, WAL mode, indexed normalized quotes, plus raw API payloads archived for audit. |
| Offline test suite | 21 test files: providers parse recorded JSON fixtures and web handlers run against `httptest` — `go test ./...` needs no network. |

## Quick start

Generate today's report:

```bash
go run ./cmd/report
xdg-open reporte/index.html
```

Flags: `-watchlist watchlist.yaml` · `-db data/asesor.db` · `-out reporte`

Run the web platform:

```bash
go run ./cmd/report              # fetch fresh data first
go build -o web ./cmd/web
WEB_AUTH_PASSWORD=change-me \
  WEB_SESSION_SECRET=$(openssl rand -hex 32) \
  WEB_DB_PATH=./data/asesor.db \
  ./web
```

The server listens on `:8080` (health check at `/healthz`). Deployment with Caddy or a tunnel is covered in [docs/web-platform.md](docs/web-platform.md).

Run the tests:

```bash
go test ./...
```

## How it fits together

```
Yahoo · DolarAPI · data912 · CoinGecko
              │  one fetch.Provider interface
              ▼
        SQLite (WAL, CGo-free)
   raw payload archive + normalized quotes
              │
   ┌──────────┼───────────┐
   ▼          ▼           ▼
cmd/report  cmd/alerts  cmd/web
HTML report rule check  authenticated UI
```

`cmd/web` never fetches data: it reads the database `cmd/report` writes. Refresh on a cron and the platform serves the latest quotes on the next request.

## Project layout

| Path | Purpose |
|---|---|
| `cmd/report` | CLI: fetch → store → render (writes `reporte/index.html` plus a dated copy) |
| `cmd/alerts` | CLI: evaluate stored quotes against alert rules |
| `cmd/web` | Web platform: authentication, dashboard, rule CRUD, PWA |
| `internal/config` | `watchlist.yaml` loading and validation |
| `internal/fetch` | `Provider` interface + Yahoo, DolarAPI, data912, CoinGecko |
| `internal/store` | SQLite: migrations, raw archive, normalized quotes, alert rules and history |
| `internal/alert` | Rule evaluation engine |
| `internal/render` | Mobile-first HTML report (stdlib `html/template`, `go:embed`) |
| `internal/web` | Server-rendered web UI (`templ`, htmx, PWA assets) |
| `docs/web-platform.md` | Web run and deployment guide |

## Customizing the watchlist

`watchlist.yaml` drives what is tracked:

| Section | Provider | Examples |
|---|---|---|
| `usa` | Yahoo Finance | `SPY`, `AAPL` (index proxies — live index values require a separate license) |
| `dolares` | DolarAPI | `oficial`, `blue`, `bolsa`, `contadoconliqui` |
| `bonos` | data912 | `AL30D` — suffix `D` settles as MEP in ARS, `C` as cable in USD |
| `cripto` | CoinGecko | `bitcoin`, `ethereum` |

Validation rejects empty sections, malformed symbols, and duplicates.

## Swapping a data source

Every provider implements one small interface:

```go
type Provider interface {
    Name() string
    Fetch(ctx context.Context) (FetchResult, error)
}
```

Replacing Yahoo with a licensed feed (e.g. Tiingo) means adding one file under `internal/fetch/` and wiring it in `cmd/report/main.go` — storage and rendering stay untouched. Provider tests run offline against recorded JSON fixtures in `internal/fetch/testdata/`.

## Data sources

| Data | Source | Notes |
|---|---|---|
| US equities / ETFs | Yahoo Finance | Unofficial endpoint; prototyping only |
| ARS exchange rates | [dolarapi.com](https://dolarapi.com/) | Free public API |
| Sovereign bonds | [data912.com](https://data912.com/) | Free public API |
| Crypto | [CoinGecko](https://www.coingecko.com/) | Public tier, rate-limited; attribution shown in the report footer |

## Roadmap

- Configurable assets from the web UI (early exploration)
- Scheduled fetching on a hosted instance
- WhatsApp delivery of the daily report

## Disclaimer

Market data comes from free, unofficial endpoints and is intended for personal, non-commercial use. This project is not investment advice.

## License

[MIT](LICENSE).
