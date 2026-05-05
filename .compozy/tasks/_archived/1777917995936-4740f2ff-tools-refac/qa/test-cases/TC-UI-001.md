# TC-UI-001: Web Settings And Automation Spot-Check

**Priority:** P1 (High)
**Type:** UI Spot-Check
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Spot-check that `web/src/systems/automation/*` and `web/src/systems/settings/*` render against the post-cut DTOs without console errors. Confirm settings → MCP servers shows redacted `auth_status` (`token_present: true|false`), and tasks views render the `run_id`-keyed contract.

## Traceability

- Tasks: task_07 (automation DTO), task_09 (autonomy hard cut), task_10 (MCP auth status), task_11 (docs/codegen alignment).
- TechSpec: "Web/Docs Impact".
- ADRs: ADR-005, ADR-006.
- Surfaces: `web/src/systems/{tasks,automation,settings}/*`.

## Preconditions

- TC-REG-004 passed.
- Isolated `AGH_HOME` daemon running with the bootstrap manifest.
- `AGH_WEB_API_PROXY_TARGET` exported per the bootstrap manifest (do not hardcode `localhost:2123`).

## Test Steps

1. Start the dev server:
   ```bash
   make web-dev | tee qa/logs/TC-UI-001/web-dev.log
   ```

2. Open the dev URL (typically `http://localhost:3000`) in a browser at desktop 1280px.

3. Navigate to **Automation** → confirm:
   - Job list renders fixture data without console errors.
   - Trigger list renders.
   - Run history panel renders the new DTO fields.
   - Capture screenshot under `qa/screenshots/TC-UI-001/automation.png`.

4. Navigate to **Settings → MCP servers** → confirm:
   - Each server entry shows `auth_status` with redacted fields.
   - `token_present` boolean visible; no token / code / secret string.
   - Capture screenshot under `qa/screenshots/TC-UI-001/settings-mcp.png`.

5. Navigate to **Tasks** views (any panel that consumes `web/src/systems/tasks/*`) → confirm:
   - Run rows render `run_id`, `claim_token_hash` (if displayed at all), and no raw `claim_token` text.
   - Capture screenshot under `qa/screenshots/TC-UI-001/tasks.png`.

6. Capture browser console output (DevTools) for the entire visit:
   - **Expected:** No DTO-related runtime errors. If a TypeScript build emitted them at runtime, fail TC-UI-001 and link to a `BUG-*.md`.

## Evidence To Capture

- `qa/screenshots/TC-UI-001/{automation,settings-mcp,tasks}.png`.
- Browser console log saved as `qa/logs/TC-UI-001/console.log`.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Settings page with FAILED MCP server | mock OAuth expired | UI shows `expired` status without secret material |
| Automation panel with empty job list | fresh daemon | Empty state renders without errors |
| Tasks view during in-flight claim | run is `claimed` | UI displays current state and lease metadata; no raw token |

## Channels Exercised

- Browser ↔ daemon (via dev proxy).
- Web fixtures and live state.

## Related Test Cases

- TC-FUNC-006, TC-FUNC-008 (mutable family parity for automation/MCP auth).
- TC-REG-004 (web Vitest lanes).
- TC-SEC-001, TC-SEC-002 (redaction sweeps).
