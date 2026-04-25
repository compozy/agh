## TC-FUNC-002: CLI Config And Setup Lifecycle

**Priority:** P1 (High)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 45 minutes
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

### Objective

Verify that CLI setup and configuration commands inspect, mutate, validate, and report AGH configuration safely, with redaction rules aligned to MCP auth and managed install/update/uninstall behavior.

### Traceability

- Task: task_08, CLI Config and Setup Lifecycle.
- TechSpec: issues 36, 37, 39, 40, 41, 42, and 43.
- ADR: ADR-001 CLI/setup/release hardening track and ADR-003 redaction baseline.
- Surfaces: `internal/cli`, `internal/config/persistence.go`, `internal/config/bootstrap.go`, `internal/config/config.go`, settings contracts where compatible, site config/install/update/uninstall/completion docs.

### Preconditions

- Isolated temp `AGH_HOME` and workspace.
- Config contains ordinary fields and secret-bearing MCP/env values that must be redacted in output.
- Managed and unmanaged install states can be simulated with `AGH_MANAGED`.

### Test Steps

1. Run `agh config path`, `show`, `list`, and `get` in human and JSON modes.
   - **Expected:** Commands read from the resolved config path and redact all sensitive MCP/auth/env values.

2. Run `agh config set` for representative scalar and nested fields.
   - **Expected:** Mutations go through config persistence APIs, preserve TOML structure as supported, and fail validation without partially writing invalid state.

3. Run `agh config validate` and `agh config check`.
   - **Expected:** Both report the same validation result, and `check` behaves as the documented alias.

4. Run shell completion commands for supported shells.
   - **Expected:** Commands emit valid completion scripts without touching shell files unless explicitly redirected by the operator.

5. Simulate managed and unmanaged `agh update` and `agh uninstall`.
   - **Expected:** Managed state reports package-manager-owned behavior; unmanaged state is idempotent and clear about files it will or will not remove.

6. Review docs and web compatibility.
   - **Expected:** CLI reference and installation docs match actual command names/flags; web settings contracts remain compatible with redaction semantics.

### Evidence To Capture

- `qa/logs/TC-FUNC-002/go-test-cli-config.log`
- `qa/logs/TC-FUNC-002/config-show-redacted.json`
- `qa/logs/TC-FUNC-002/config-validation.log`
- `qa/logs/TC-FUNC-002/managed-lifecycle.log`

### Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Invalid set | Bad field value | Validation failure, no partial write |
| Secret env | MCP env/token-like value | Redacted in show/list/get |
| Managed install | `AGH_MANAGED=homebrew` | Update/uninstall report manager guidance |
| Completion | bash/zsh/fish/powershell | Valid script output |

### Related Test Cases

- TC-FUNC-003: `.env` repair and extension environment diagnostics.
- TC-REG-002: Site config/setup docs.
