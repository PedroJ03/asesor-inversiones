# Web Pages Specification

## Purpose

Define the authenticated, server-rendered advisor views for the web platform.

## Requirements

### Requirement: Serve advisor views through an authentication seam

The web server MUST expose `GET /`, `/reporte`, `/reporte/{date}`, `/activos`, `/activos/{source}/{symbol}`, `/alertas`, and `/healthz`. All data-bearing views MUST pass through an identity-agnostic advisor authentication middleware seam; the seam MUST be replaceable without changing handlers. `/healthz` MUST return liveness without exposing portfolio data.

#### Scenario: Authenticated dashboard request
- GIVEN the auth seam identifies an advisor
- WHEN the advisor requests `GET /`
- THEN the server returns a successful HTML dashboard response
- AND the response contains current snapshot and alert information

#### Scenario: Unauthenticated data request
- GIVEN the auth seam rejects the request
- WHEN the client requests a data-bearing route
- THEN the server returns the seam's unauthenticated response
- AND no quote, rule, or alert data is rendered

#### Scenario: Archived report route
- GIVEN an authenticated advisor requests a valid `YYYY-MM-DD` date
- WHEN `GET /reporte/{date}` is handled
- THEN the response renders that date's available report data
- AND an invalid or unavailable date produces a controlled client-visible error

### Requirement: Render mobile-first pages with progressive enhancement

Pages MUST render usable HTML without JavaScript, use the `asesor helper` brand, and provide dashboard, daily report, watchlist, asset detail, and alerts views. Interactive rule and symbol flows MAY use htmx HTML fragments but MUST preserve a server-rendered fallback.

#### Scenario: JavaScript-disabled watchlist
- GIVEN an authenticated client with JavaScript disabled
- WHEN the client requests `/activos`
- THEN grouped watchlist sections and latest values render as HTML
- AND navigation remains usable in a single column at 720px or less

#### Scenario: Freshness is visible
- GIVEN stored quotes have a fetch timestamp
- WHEN any quote-bearing page renders
- THEN it prominently shows `última actualización` and a stale/current presentation
