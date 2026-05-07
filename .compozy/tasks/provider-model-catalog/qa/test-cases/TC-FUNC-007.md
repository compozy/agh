# TC-FUNC-007: Partial Source Success vs All-Source Failure

**Priority:** P0
**Type:** Functional
**Module:** `internal/modelcatalog.Service.ListModels`
**Requirement:** TechSpec Safety Invariants SI-13.
**Status:** Not Run

## Objective

Verify that:
- A list call succeeds when at least one source delivers usable rows or a stale cache exists.
- A list call fails (deterministic error) only when every usable source fails AND no stale cache exists.

## Preconditions

- [ ] Catalog with multiple sources registered for one provider.
- [ ] Stub control over each source's success/failure.

## Test Steps

1. **Partial success.**
   - Force `models.dev` 5xx; let `builtin` return rows.
   - Command: `agh provider models list codex -o json`.
   - **Expected:** Exit 0; rows from `builtin` returned; status reports `models_dev` as failed; `last_error` redacted.
2. **All-source failure with stale cache.**
   - Run a successful refresh first; then force every source to fail.
   - **Expected:** List returns stale rows with `stale=true`; no error to operator.
3. **All-source failure with no stale cache.**
   - Wipe SQLite catalog tables (test harness only); force every source to fail.
   - **Expected:** List returns deterministic error referencing the failed sources; CLI exit non-zero with structured JSON error in `-o json` mode.
4. **Refresh during all-source failure remains coalesced.**
   - Issue two concurrent refreshes for the same provider.
   - **Expected:** One subprocess/network attempt per source; status batch returned identically to both callers.

## Audit Coverage

- C6 task tree (Task 03, Task 05).
- SI-4, SI-13.

## Pass Criteria

- Steps 1-2 succeed without error.
- Step 3 fails with structured error and non-zero exit.
- Step 4 shows single underlying call.

## Failure Criteria

- Partial failure reported as global failure.
- All-source failure with stale cache returns error.
- Coalescing breaks under concurrent refresh.
