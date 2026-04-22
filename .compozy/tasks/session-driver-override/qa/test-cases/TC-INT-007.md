## TC-INT-007: Legacy Blank-Provider Metadata Repairs Once and Then Stays Deterministic

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Legacy repair and reconcile
**Traceability:** Task 03 repair requirements; ADR-003 and ADR-005; TechSpec "Legacy metadata repair" and "Testing Approach"

---

### Objective

Verify that an inactive legacy session with blank `provider` metadata is repaired exactly once from the stored agent and workspace config, persisted immediately, and then behaves like an ordinary persisted-provider session.

---

### Preconditions

- [ ] Legacy stopped-session metadata fixture `LEGACY-META-BLANK` exists with `provider == ""`.
- [ ] The stored agent still resolves in the target workspace.
- [ ] Reconcile or resume can be triggered and backend logs are captured.
- [ ] `session.json` and the global `sessions` row can be inspected before and after repair.

---

### Test Steps

1. Confirm the legacy fixture starts with blank `provider` metadata.
   **Expected:** `session.json` and any indexed state show the pre-feature blank provider state before the first read/repair.

2. Trigger the first read path that should repair the session, using resume or reconcile.
   **Expected:** AGH resolves the effective provider, persists it immediately, and completes the read path or resume preparation using that repaired value.

3. Inspect `session.json`, SQLite, and logs after repair.
   **Expected:** Both persistence layers now contain the repaired provider, and logs identify the repair with `phase=legacy_repair`.

4. Trigger the same read path again without changing config.
   **Expected:** The session behaves like an ordinary persisted-provider session; AGH does not keep leaving blanks behind or perform repeated repair writes.

---

### Evidence to Capture

- Before/after `session.json` state.
- Before/after SQLite state for the same session id.
- Repair log lines containing the provider and repair phase.
- Evidence from the second read proving deterministic post-repair behavior.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Repair via resume | Blank provider metadata | Provider is repaired before resume continues. |
| Repair via reconcile | Stopped session scan | Global index receives the repaired provider. |
| Second read after repair | Same workspace config | No repeated blank-provider fallback behavior occurs. |

---

### Related Test Cases

- `TC-INT-006` for the schema migration layer
- `TC-INT-005` for explicit failure when a persisted provider cannot resolve

---

### Notes

This case addresses the highest remaining cross-task storage risk noted in shared workflow memory. Treat any repeated blank state after repair as a blocking defect.
