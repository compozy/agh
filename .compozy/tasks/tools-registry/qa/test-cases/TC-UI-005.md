# TC-UI-005 — Storybook/route stories cover native/extension/MCP/conflicted/unavailable/auth-required

- **Priority:** P2
- **Type:** UI / visual states
- **Trace:** Task 13

## Test Steps

1. Open Storybook (or route equivalent) for `web/src/systems/tools/**`.
   - **Expected:** At least one story per state: native callable, extension callable, extension unhealthy, MCP authenticated, MCP needs_login, MCP expired, MCP invalid, conflicted, denied by policy, approval-required, unavailable.
2. Visual diff per viewport (375/768/1280) against approved baseline.
3. No story uses real tokens or sentinel values.

## Automation

- **Target:** E2E
- **Status:** Existing
- **Command/Spec:** `make bun-test web/src/systems/tools/**.stories`
