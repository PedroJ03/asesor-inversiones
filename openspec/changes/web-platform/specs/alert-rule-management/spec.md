# Alert Rule Management Specification

## Purpose

Define advisor-only web rule management as a contract-first consumer of alert-engine.

## Requirements

### Requirement: Manage rules through the frozen alert-engine contract

After alert-engine is merged, the web platform MUST use the frozen `CreateRule`, `ListRules`, `RemoveRule`, `UpdateRuleState`, `LatestQuote`, and `History` methods rather than duplicating persistence or schema logic. It MUST support create, disable, remove, list, and recent-history views, with validation failures rendered as safe HTML errors.

#### Scenario: Create a valid rule
- GIVEN an authenticated advisor selects a valid watchlisted asset and threshold
- WHEN the advisor submits the rule form
- THEN the platform calls `CreateRule` and renders the persisted rule with its `armed` state
- AND a percentage rule displays its captured baseline

#### Scenario: Disable and remove a rule
- GIVEN an existing rule is listed
- WHEN the advisor disables or removes it
- THEN the platform calls the corresponding frozen contract method
- AND the refreshed list reflects the result without deleting alert history

#### Scenario: Reject invalid or duplicate rule
- GIVEN the engine rejects an invalid or duplicate request
- WHEN the advisor submits it
- THEN the platform renders a controlled validation error
- AND it does not claim that a rule was created

### Requirement: Keep alert management sequenced after the engine

Rule-management implementation MUST be applied only after the alert-engine merge and after its untracked OpenSpec artifacts are committed. During parallel development, platform specs and seed-backed tests MUST treat the alert-engine schema and methods as frozen; the platform MUST rebase after the engine merge.

#### Scenario: Engine contract unavailable
- GIVEN alert-engine has not merged into the platform worktree
- WHEN platform implementation is prepared
- THEN rule-management application is blocked rather than inventing substitute methods
