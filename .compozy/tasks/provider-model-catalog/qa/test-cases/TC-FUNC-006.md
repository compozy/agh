# TC-FUNC-006: Stale Fallback Preserves Prior Successful Rows

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog`
**Requirement:** TechSpec Safety Invariants SI-4.
**Status:** Not Run

## Objective

Verify that when a source refresh fails after at least one prior successful refresh, AGH preserves the previously stored rows, marks them stale, and surfaces the redacted `last_error` through projection and status.

## Preconditions

- [ ] Catalog seeded via successful refresh of a stub source.
- [ ] Stub source can be flipped to fail on demand.

## Test Steps

1. **Successful refresh.**
   - Trigger refresh; assert `model_catalog_rows` has rows for the source with `stale=0`.
   - **Expected:** Source status `last_success_at` populated; rows readable via projection.
2. **Force failure on next refresh.**
   - Stub returns 5xx; trigger `agh provider models refresh codex --source models_dev`.
   - **Expected:** Source status records `refresh_state="failed"`, `last_error` redacted; previous rows now flagged `stale=1`.
3. **List after failure returns stale rows with markers.**
   - Command: `agh provider models list codex --include-stale -o json`.
   - **Expected:** Rows present with `stale=true`; `availability_state` either `available_stale` or `unavailable_stale` if previous live row existed; `unknown` otherwise.
4. **Without `--include-stale`, projection still includes stale rows but flags them.**
   - **Expected:** Default behavior surfaces stale rows tagged `stale=true` (TechSpec keeps stale rows usable as fallback).
5. **Daemon restart preserves stale rows.**
   - Restart daemon; reissue list.
   - **Expected:** Same rows present, still flagged stale; no row loss.

## Audit Coverage

- C6 task tree (Task 03, Task 05).
- SI-4, SI-13.

## Pass Criteria

- Stale rows persist across refresh failure and daemon restart.
- `last_error` redacted (no API key / OAuth / env secret string).

## Failure Criteria

- Failure clears prior rows.
- Status loses `last_success_at`.
- Stale rows missing the `stale` flag.
