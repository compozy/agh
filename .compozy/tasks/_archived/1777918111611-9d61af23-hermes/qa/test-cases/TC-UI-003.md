## TC-UI-003: Web Generated Contract Compatibility

**Priority:** P1 (High)
**Type:** UI
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that backend contract changes from Hermes tasks remain type-safe and coherent in the web generated OpenAPI types, adapters, hooks, fixtures, and route/component tests.

### Traceability

- Tasks: task_02, task_03, task_04, task_07, task_09.
- TechSpec: API payloads for observe health, session failure, automation delivery errors, settings extension env diagnostics, and memory health/history.
- ADRs: ADR-001, ADR-002, ADR-003, ADR-005.
- Surfaces: `web/src/generated/agh-openapi.d.ts`, daemon/session/automation/settings adapters, fixtures, hooks, route tests, `make codegen-check`.

### Preconditions

- Generated OpenAPI files are current.
- Fixture payloads include observe `retention`, `failures`, `agent_probes`, session `failure`, automation `scheduler` and `delivery_error`, memory endpoints, and extension env diagnostics.
- Web dependencies are installed.

### Test Steps

1. Run `make codegen-check`.
   - **Expected:** Generated OpenAPI artifacts are current and no backend contract drift is detected.

2. Run web typecheck.
   - **Expected:** Web compiles against generated memory, health, session, automation, and settings DTOs.

3. Run daemon adapter and fixture tests.
   - **Expected:** Tests assert observe health retention/failures/agent probes map correctly.

4. Run session adapter/component tests.
   - **Expected:** Session failure payloads remain accepted and rendered without breaking resume/session flows.

5. Run automation and settings focused tests.
   - **Expected:** Scheduler/delivery fields and extension env diagnostics are preserved.

6. Verify memory generated endpoints.
   - **Expected:** `getMemoryHealth` and `listMemoryHistory` exist in generated types and can be imported by future consumers.

### Evidence To Capture

- `qa/logs/TC-UI-003/codegen-check.log`
- `qa/logs/TC-UI-003/web-typecheck.log`
- `qa/logs/TC-UI-003/web-contract-vitest.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Missing generated endpoint | No `getMemoryHealth` | Codegen/typecheck failure |
| Optional health field absent | No `agent_probes` | Adapters tolerate optional field |
| New required backend field | Fixture missing field | Typecheck/test reveals drift |
| Redacted fields | Crash/auth/env sentinel | Fixtures do not expose raw values |

### Related Test Cases

- TC-INT-002: Observe health backend.
- TC-FUNC-001: Memory backend/API.
