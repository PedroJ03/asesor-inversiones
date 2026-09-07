# Alert Rules Specification

## Purpose

Define persisted threshold rules and their lifecycle independently of report rendering.

## ADDED Requirements

### Requirement: Persist validated alert rules

The store MUST persist each rule in `alert_rules` with `id`, `source`, `symbol`, `kind`, `direction`, `threshold`, `baseline_price`, `enabled`, `state`, and `created_at`. `kind` MUST be `value` or `pct`; `direction` MUST be `above` or `below`; `state` MUST be `armed` or `triggered`; and the identity `(source, symbol, kind, direction, threshold)` MUST be unique.

#### Scenario: Create a valid value rule
- GIVEN a watchlisted asset and a latest stored quote with a positive price
- WHEN `CreateRule` receives a positive value threshold and captured baseline
- THEN the rule is stored enabled and armed with its creation timestamp

#### Scenario: Reject invalid rule input
- GIVEN an unknown source, non-watchlisted symbol, invalid kind/direction, or non-positive threshold
- WHEN the advisor creates a rule
- THEN creation fails and no rule is persisted

#### Scenario: Reject duplicate identity
- GIVEN a rule already exists with the same source, symbol, kind, direction, and threshold
- WHEN the advisor creates the same rule
- THEN creation fails with a uniqueness error

### Requirement: Capture an immutable percentage baseline

The system MUST capture `baseline_price` from the latest stored quote at rule creation. A percentage rule MUST be rejected when no quote exists, and its baseline MUST NOT be edited by evaluation or rule listing; changing it requires removal and recreation.

#### Scenario: Create percentage rule from latest quote
- GIVEN a valid watchlisted asset with a latest stored quote
- WHEN a percentage rule is created
- THEN its baseline and quote timestamp are reported and the baseline remains fixed

#### Scenario: Reject percentage rule without quote
- GIVEN no stored quote exists for the requested asset
- WHEN a percentage rule is created
- THEN creation fails and no baseline or rule is stored

### Requirement: List and remove rules without destroying history

`ListRules(enabledOnly)` MUST return persisted rules, including current state. `RemoveRule` MUST remove the rule while preserving its alert history through nullable rule references with `ON DELETE SET NULL`.

#### Scenario: List rules by enabled status
- GIVEN enabled and disabled rules exist
- WHEN rules are listed with either filter
- THEN the result contains the matching rules and their `armed` or `triggered` state

#### Scenario: Remove rule preserves snapshots
- GIVEN a rule has generated alert snapshots
- WHEN the rule is removed
- THEN the rule is absent, snapshots remain queryable, and their `rule_id` is null
