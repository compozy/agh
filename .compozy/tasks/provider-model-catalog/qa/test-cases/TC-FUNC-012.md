# TC-FUNC-012: Extension Capability Missing / Revoked = Denial

**Priority:** P1
**Type:** Functional
**Module:** `internal/extension/host_api_models.go`
**Requirement:** ADR-003.
**Status:** Not Run

## Objective

Verify extension Host API methods (`models/list`, `models/refresh`, `models/status`) honor capability grants and surface deterministic denial errors when grants are missing or revoked, without leaking daemon internals.

## Preconditions

- [ ] Extension fixture with grants toggleable per method.

## Test Steps

1. **All three grants present.**
   - **Expected:** Host API succeeds; payload matches daemon-owned projection (not raw extension payload).
2. **`models/list` grant missing.**
   - **Expected:** Host API returns deterministic capability error; no rows leaked.
3. **`models/refresh` grant missing.**
   - **Expected:** Refresh denied; no source status changes; no subprocess executed.
4. **`models/status` grant missing.**
   - **Expected:** Status request denied; no source status read.
5. **Grant revoked mid-run.**
   - Trigger list, then revoke grant, then trigger again.
   - **Expected:** Second call denied; no cached payload returned.

## Audit Coverage

- C6 task tree (Task 08).
- SI-8, SI-9.

## Pass Criteria

- Capability gate enforced on every call.
- Errors deterministic.

## Failure Criteria

- Missing grant still returns rows or status.
- Error surface leaks daemon internals.
