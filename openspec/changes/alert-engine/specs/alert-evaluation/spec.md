# Alert Evaluation Specification

## Purpose

Define offline, deterministic evaluation of stored quotes, state transitions, and auditable alert history.

## ADDED Requirements

### Requirement: Evaluate inclusive value and baseline-drift thresholds

`Evaluate(rules, resolve, now, maxAge)` MUST perform no I/O or network calls. Value rules MUST trigger at `price >= threshold` for `above` and `price <= threshold` for `below`. Percentage rules MUST compute `(price - baseline_price) / baseline_price * 100` and trigger at drift `>= +threshold` for `above` or `<= -threshold` for `below`, for every provider.

#### Scenario: Value equality triggers
- GIVEN an armed value rule and a quote exactly equal to its threshold
- WHEN evaluation resolves a fresh quote
- THEN the result contains a trigger and a transition to `triggered`

#### Scenario: Percentage drift uses immutable baseline
- GIVEN an armed percentage rule with baseline 100 and an above threshold of 5
- WHEN the fresh quote is 105
- THEN evaluation triggers using 5 percent drift, regardless of provider metadata

### Requirement: Enforce the two-state transition machine

For a fresh quote, armed plus true MUST insert one alert snapshot and transition to triggered; armed plus false MUST remain armed; triggered plus true MUST be a no-op; and triggered plus false MUST transition silently to armed. Each rule MUST be evaluated independently.

#### Scenario: Deduplicate while condition remains true
- GIVEN a triggered rule whose condition remains true
- WHEN evaluation runs again
- THEN no alert is inserted and no duplicate trigger is returned

#### Scenario: Automatically re-arm after reset
- GIVEN a triggered rule whose condition becomes false
- WHEN evaluation runs
- THEN state becomes armed without inserting an alert

### Requirement: Guard stale, missing, and orphaned quotes

A missing quote, quote error, or `fetched_at` older than `maxAge` (24 hours by default) MUST produce a warning, no trigger, and no state change. Rules whose asset is no longer resolvable MUST be skipped with a warning. Warnings for one rule MUST NOT prevent independent rules from being evaluated.

#### Scenario: Stale quote is fail-safe
- GIVEN an armed rule and a quote older than the allowed age
- WHEN evaluation runs
- THEN it returns a warning and leaves the rule armed

#### Scenario: Orphan does not abort run
- GIVEN one orphaned rule and one rule with a fresh resolvable quote
- WHEN evaluation runs
- THEN it warns for the orphan and evaluates the other rule normally

### Requirement: Persist denormalized alert history

Each trigger MUST persist an `alerts` snapshot containing source, symbol, kind, threshold, observed price, observed percentage when applicable, baseline, quote fetched time, and `triggered_at`. Snapshots MUST survive rule edits or deletion, and `History(limit)` MUST return the requested bounded history.

#### Scenario: Trigger records auditable timestamps
- GIVEN a fresh quote satisfies an armed rule
- WHEN the trigger is persisted
- THEN history contains the observed values plus quote and trigger timestamps
