# Alert CLI Specification

## Purpose

Define the separate `cmd/alerts` developer CLI without changing report or rendering behavior.

## ADDED Requirements

### Requirement: Run alert evaluation from stored quotes

`cmd/alerts run` MUST open the selected database, resolve latest stored quotes, evaluate enabled rules, print triggered alerts and warnings, and use a 24-hour default staleness limit. `-db` MUST select the database and `-max-age` MUST override the limit. Rule-local warnings MUST yield exit code 0; store or persistence failures MUST yield a non-zero exit.

#### Scenario: Run prints triggers and warnings
- GIVEN a database containing a triggered rule and another rule with a stale quote
- WHEN `alerts run -db path` executes
- THEN it prints the alert and warning and exits successfully

#### Scenario: Run reports store failure
- GIVEN an unavailable or invalid database path
- WHEN `alerts run -db path` executes
- THEN it reports the store error and exits non-zero

### Requirement: Add validated rules through flags

`cmd/alerts add` MUST accept `-db`, `-source`, `-symbol`, `-kind`, `-direction`, and `-threshold`. It MUST validate source whitelist and watchlist membership, capture the latest quote as baseline, reject invalid or missing input, and report baseline price and timestamp on success.

#### Scenario: Add reports captured baseline
- GIVEN a valid watchlisted asset with a latest quote
- WHEN `alerts add` receives valid flags
- THEN it persists the rule and prints the captured baseline and quote timestamp

#### Scenario: Add rejects missing quote
- GIVEN valid asset flags but no latest stored quote
- WHEN `alerts add` executes
- THEN it reports a validation error and does not create a rule

### Requirement: Inspect and remove rules and history

`cmd/alerts list -db` MUST print rules with their state. `remove -db -id` MUST remove the selected rule without deleting snapshots. `history -db -limit` MUST print bounded denormalized history; invalid identifiers and limits MUST be command errors.

#### Scenario: List displays state
- GIVEN armed and triggered persisted rules
- WHEN `alerts list` executes
- THEN output identifies each rule and its current state

#### Scenario: History survives removal
- GIVEN a rule with a persisted trigger that has been removed
- WHEN `alerts history -limit 10` executes
- THEN the snapshot remains visible without requiring the deleted rule

### Requirement: Preserve report boundaries and offline evaluation

The CLI MUST be a separate `cmd/alerts` binary. The change MUST NOT modify report or render behavior, and evaluation MUST use stored quotes without network calls.

#### Scenario: Report remains unchanged
- GIVEN the alert functionality is built
- WHEN the existing report command runs
- THEN its source and rendered behavior remain unaffected
