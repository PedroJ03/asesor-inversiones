# Web Read Layer Specification

## Purpose

Define API-ready store reads used by pages and future JSON twins without changing protected store files.

## Requirements

### Requirement: Read batched latest snapshots

The store MUST provide a read operation that returns at most one newest quote for every requested `(source, symbol)` pair in one logical batch. Results MUST retain source, symbol, price, change data, and fetch timestamp, and MUST distinguish missing pairs from zero-valued data.

#### Scenario: Batch watchlist snapshot
- GIVEN a watchlist contains pairs across multiple sources and stored history has several rows per pair
- WHEN the web read is requested
- THEN it returns only the newest row for each pair
- AND one missing pair does not suppress the other results

#### Scenario: Empty batch
- GIVEN no pairs are requested
- WHEN the batch read is called
- THEN it returns an empty result without issuing an unbounded history scan

### Requirement: Read bounded pair history

The store MUST provide range and limit reads for one `(source, symbol)` pair, ordered deterministically by fetch time, with a bounded default and explicit maximum limit. Invalid ranges or limits MUST return an error rather than silently widening the query.

#### Scenario: Asset detail history
- GIVEN a pair has quotes inside and outside a requested time range
- WHEN history is requested with a valid limit
- THEN only in-range rows up to the limit are returned in deterministic order

#### Scenario: Reject unsafe limit
- GIVEN a caller supplies a negative, zero, or over-maximum limit
- WHEN the history read is called
- THEN it returns a validation error and does not perform an unbounded read

### Requirement: Preserve parallel-development isolation

These reads MUST be added only in `internal/store/web_reads.go`; they MUST consume existing store types and MUST NOT edit `internal/store/store.go`, providers, report rendering, or CLI report code. Returned read DTOs MUST remain reusable by future JSON handlers.

#### Scenario: Contract-safe platform change
- GIVEN alert-engine concurrently owns `store.go`
- WHEN web reads are implemented
- THEN the platform change is confined to the new read file and composes with the frozen store contract
