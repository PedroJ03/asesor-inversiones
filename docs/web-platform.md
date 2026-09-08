# Web platform

Server-rendered web UI for the daily market report. It reads the same SQLite database and watchlist as `cmd/report`; run the report CLI to fetch fresh data, then browse the platform.

## Run locally

```bash
go build -o web ./cmd/web
WEB_AUTH_PASSWORD=change-me \
  WEB_SESSION_SECRET=$(openssl rand -hex 32) \
  WEB_DB_PATH=./data/asesor.db \
  WATCHLIST_PATH=./watchlist.yaml \
  ./web
```

The server listens on `:8080` by default. Change the address with `WEB_ADDR` (e.g. `127.0.0.1:3000`).

## Environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `WEB_AUTH_PASSWORD` | yes | — | Shared login password |
| `WEB_SESSION_SECRET` | yes | — | HMAC key for signed session cookies |
| `WEB_DB_PATH` | no | `./asesor.db` | SQLite database written by `cmd/report` |
| `WATCHLIST_PATH` | no | `./watchlist.yaml` | Asset configuration |
| `WEB_ADDR` | no | `:8080` | Listen address |

Generate `WEB_SESSION_SECRET` with `openssl rand -hex 32` or similar.

## Compose with `cmd/report`

`cmd/report` fetches market data and writes it to the database:

```bash
go run ./cmd/report -db ./data/asesor.db -watchlist ./watchlist.yaml
```

`cmd/web` only reads that database; it never fetches data. Refresh the report on a schedule (e.g. cron) and the platform will show the latest quotes, report pages, and rule history on the next request.

## Deploy with Caddy

The PWA requires HTTPS and a valid origin to install. Place Caddy in front of the Go binary:

```caddy
{
    email you@example.com
}

asesor.example.com {
    reverse_proxy 127.0.0.1:8080
    encode gzip zstd
}
```

```bash
caddy run --config Caddyfile
```

Caddy obtains Let's Encrypt certificates automatically. The service worker and manifest will then be served over HTTPS, which is required for installability and push readiness.

## Tunnel fallback

If you do not want to expose a VPS port, run the binary locally and expose it through a tunnel:

- **Tailscale Funnel**: `tailscale funnel --bg 8080`
- **Cloudflare Tunnel**: `cloudflared tunnel --url http://localhost:8080`

Both terminate TLS and give the platform a public HTTPS origin.

## Rollback

1. Stop the binary (`SIGINT` or `SIGTERM`).
2. Revert the web-specific files: `cmd/web/`, `internal/web/`, `internal/store/web_*.go`, and the `templ` dependency in `go.mod`/`go.sum`.
3. `internal/store/store.go`, `internal/render/`, and `cmd/report/` are untouched by this change and do not need to be reverted.

## Health check

`GET /healthz` is unauthenticated and returns `200 OK` when the server is running.
