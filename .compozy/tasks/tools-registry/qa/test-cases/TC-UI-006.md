# TC-UI-006 — Session projection display matches task_11 endpoint semantics

- **Priority:** P2
- **Type:** UI / session view
- **Trace:** Task 13, ADR-006

## Test Steps

1. Open session detail view rendering session-callable tools.
   - **Expected:** Tools shown match `GET /api/sessions/{id}/tools` payload exactly.
2. Tools hidden by session lineage / approval / availability are absent (no diagnostic rows in session view).
3. Operator users with elevated scope see a "switch to operator view" affordance backed by `GET /api/tools`; session view never silently includes denied tools.

## Automation

- **Target:** E2E
- **Status:** Existing
- **Command/Spec:** `make test-e2e-web` (`tools-session` spec)
