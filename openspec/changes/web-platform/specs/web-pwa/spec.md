# Web PWA Specification

## Purpose

Define the installable, mobile-first shell and honest freshness presentation.

## Requirements

### Requirement: Provide an installable HTTPS shell

The platform MUST provide a valid manifest, app icons, and a minimal service worker suitable for installation over HTTPS. The shell MUST use a single-column layout at 720px or less, a bottom tab bar, and near-zero blocking JavaScript. The first load MUST remain below 200 KB total, including the pinned `htmx.org@2.0.10` asset with SRI.

#### Scenario: Installable production visit
- GIVEN an advisor visits the site over HTTPS with a compatible browser
- WHEN the browser evaluates the page and manifest
- THEN the app is installable with its icons and `asesor helper` name
- AND the service worker can cache the declared shell assets

#### Scenario: Mobile performance budget
- GIVEN a first visit on a mobile connection
- WHEN the initial page finishes loading
- THEN total transferred first-load assets are under 200 KB
- AND no JavaScript dependency blocks the primary HTML content

#### Scenario: Offline or stale shell
- GIVEN previously cached shell assets exist and the network is unavailable
- WHEN the installed app opens
- THEN the shell remains usable where cached
- AND cached data is not presented as newly fetched data

### Requirement: Present data freshness honestly

Every quote-bearing view MUST show `última actualización` using the stored fetch timestamp and MUST distinguish current, stale, missing, and unavailable data. Service-worker caching MUST NOT remove or falsify this state.

#### Scenario: Stale quote warning
- GIVEN the latest stored quote exceeds the freshness threshold
- WHEN a page renders
- THEN it shows the timestamp and a visible stale warning
- AND it does not imply a live quote
