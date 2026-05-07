# TC-INT-005: Extension Source - Success and Denial Through Host API

**Priority:** P0
**Type:** Integration
**Systems:** `internal/extension`, `internal/modelcatalog`, Host API.
**Requirement:** ADR-003, TechSpec Extensibility Integration Plan, Task 08.
**Status:** Not Run

## Objective

Verify the extension `model.source` end-to-end path: AGH calls extension `models/list`, validates and persists rows, and surfaces the daemon-owned projection through Host API; capability denial is deterministic.

## Preconditions

- [ ] Extension fixture with `model.source` capability for provider `codex`.
- [ ] Capability grants toggleable.

## Test Steps

1. **Extension grant present, valid rows.**
   - Trigger Host API `models/refresh` for `codex`.
   - **Expected:** Extension subprocess invoked; AGH validates rows; SQLite catalog updated; status `succeeded`; `models/list` returns rows including extension priority 100.
2. **Extension returns invalid row.**
   - **Expected:** Row dropped; source status records redacted error referencing the offending field; valid rows persist.
3. **Capability missing for `models/list`.**
   - **Expected:** Deterministic capability error returned; no rows leaked.
4. **Capability missing for `models/refresh`.**
   - **Expected:** No subprocess invoked; source status unchanged.
5. **Capability missing for `models/status`.**
   - **Expected:** Deterministic capability error.
6. **Extension declares provider it has no grant for.**
   - **Expected:** Refresh fails closed with capability error; valid grants for other providers unaffected.

## Audit Coverage

- C5, C6 (Task 08), C11 disruption probe.
- SI-8, SI-9.

## Pass Criteria

- Steps 1-2 produce correct catalog state.
- Steps 3-6 deterministically denied.

## Failure Criteria

- Denied call returns rows.
- Invalid extension row breaks persistence.
- Subprocess invoked without grant.
