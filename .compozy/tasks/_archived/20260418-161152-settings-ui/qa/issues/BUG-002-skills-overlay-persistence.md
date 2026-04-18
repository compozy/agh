# BUG-002: Nested Skills Overlay Writes Produced Invalid TOML

## Summary

- Status: `Fixed`
- Severity: `Critical`
- Origin case: `TC-FUNC-005`
- Surface: `/settings/skills`

## Symptom

Saving the skills page could fail with a TOML parse error when the mutation wrote nested configuration such as `skills.marketplace`, leaving the operator with a rejected save even though the edited values were valid.

## Root Cause

The overlay editing path in `internal/config/persistence.go` did not correctly normalize nested TOML tree values or compute the replacement block boundaries for nested tables. The resulting fragment rendered malformed TOML for nested sections.

## Fix

- Normalize nested values before building TOML trees.
- Use line-end offsets that include the full replacement span.
- Render nested overlay fragments without duplicating parent table headers.
- Add a regression test for `[skills.marketplace]` overlay creation.

## Regression Coverage

- `internal/config/persistence_test.go`
- `web/e2e/settings.spec.ts`
- `go test ./internal/config ./internal/settings ./internal/daemon -count=1`
- `make verify`

## Evidence

- `qa/screenshots/TC-FUNC-005-skills-applied-now-vs-restart.png`
