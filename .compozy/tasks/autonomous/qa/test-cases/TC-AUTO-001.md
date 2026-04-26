## TC-AUTO-001: Coordinator Config Defaults And Resolver Precedence

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

### Objective

Verify typed `[autonomy.coordinator]` config defaults, strict validation, workspace override
precedence, and daemon resolver behavior without starting coordinator runtime behavior from config
loading alone.

### Traceability

- Task: task_01, Autonomy Config Foundation.
- TechSpec: Coordinator config, Coordinator Trigger, Manual Control Contract.
- ADR: ADR-001 and ADR-005.
- Resource lesson: Paperclip config/schema references favor explicit typed validation over loose maps.
- Surfaces: `internal/config`, `internal/daemon`, global/workspace `config.toml`.

### Preconditions

- Isolated global `AGH_HOME` and temp workspace with `.agh/config.toml`.
- Built-in coordinator agent definition is available or represented by the test fixture.
- No daemon coordinator runtime is already running in the isolated home.

### Test Steps

1. Load default config with no autonomy section.
   - **Expected:** Coordinator config resolves to conservative defaults: disabled, `agent_name=coordinator`, TTL `2h`, max children `5`, and one active coordinator per workspace.

2. Load global config with valid `[autonomy.coordinator]` provider/model/TTL/limits.
   - **Expected:** Config validates and the daemon-facing resolver returns the global values without spawning, stopping, or prompting a session.

3. Add workspace override values for provider, model, TTL, max children, and enabled state.
   - **Expected:** Workspace override wins over global config, and omitted workspace values fall through to global/bundled defaults.

4. Try invalid values: empty agent name, invalid TTL, negative max children, unknown key, and unsupported provider/model.
   - **Expected:** Loading fails with wrapped field-path validation errors and does not mutate ambient environment or workspace files.

5. Start or inspect the daemon after config load without enqueuing executable work.
   - **Expected:** No coordinator session exists and no coordinator bootstrap log/event appears.

### Evidence To Capture

- `qa/logs/TC-AUTO-001/config-defaults.log`
- `qa/logs/TC-AUTO-001/config-workspace-override.log`
- `qa/logs/TC-AUTO-001/config-invalid-values.log`
- `qa/logs/TC-AUTO-001/daemon-no-coordinator.json`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Workspace disables global | global enabled, workspace disabled | Resolver returns disabled for workspace |
| Empty provider/model | `provider=""`, `model=""` | Falls through without validation failure |
| Unknown TOML key | `autonomy.coordinator.foo` | Strict loader rejects config |
| Boundary TTL | `1m`, `24h` | Accepted when within configured bounds |

### Related Test Cases

- TC-AUTO-013: Coordinator bootstrap uses resolved config.
- TC-AUTO-016: Docs describe coordinator config accurately.
