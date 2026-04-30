# TC-FUNC-004: Config Mutable Tool Family With Trust-Root And Secret Denials

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__config_show/list/get/set/unset/diff/path` reuse the existing validated config writer and validation pipeline, that allowed paths mutate identically across tool/CLI/HTTP/UDS, and that forbidden paths return deterministic denials. Confirm `set`/`unset` require mutating approval.

## Traceability

- Task: task_05 (Config Mutable Tool Family).
- TechSpec: "Mutable Surface Policy → agh__config", "Config Lifecycle", "Old vs New Effective Behavior", "Post-Implementation Residual Checks".
- ADRs: ADR-002, ADR-006.
- Surfaces: `internal/tools/builtin/config.go`, `internal/daemon/native_config_hook_tools.go`, `internal/config/{config.go,merge.go,persistence.go,tools.go}`, `internal/cli/config.go`.

## Preconditions

- Isolated `AGH_HOME` from `agh-qa-bootstrap`.
- A workspace overlay file the daemon reads.
- An ACP approval channel that auto-approves mutating calls in this lab (manifest variable).
- Sequential write discipline: never run `agh config set` and the equivalent tool invoke in parallel against the same overlay (`CLAUDE.md` rule).

## Test Steps

1. **Allowed path round-trip parity (CLI ↔ tool ↔ HTTP/UDS):**
   ```bash
   # CLI baseline
   agh config show -o json | tee qa/logs/TC-FUNC-004/config-show-cli.json
   agh tool invoke agh__config_show -o json | tee qa/logs/TC-FUNC-004/config-show-tool.json
   diff <(jq -S . qa/logs/TC-FUNC-004/config-show-cli.json) \
        <(jq -S . qa/logs/TC-FUNC-004/config-show-tool.json)

   # Allowed mutation: e.g., session.history.max_records or memory.local_dir
   agh tool invoke agh__config_set --input '{"path":"defaults.history_limit","value":42,"scope":"workspace"}' -o json \
     | tee qa/logs/TC-FUNC-004/config-set-tool.json
   agh config get defaults.history_limit -o json | tee qa/logs/TC-FUNC-004/config-get-after.json
   ```
   - **Expected:** CLI ↔ tool show output is identical. After tool `set`, CLI `get` reflects the value. Persistence file diff reflects only the targeted field.

2. **Trust-root denial:**
   ```bash
   agh tool invoke agh__config_set --input '{"path":"daemon.bind","value":"127.0.0.1:9999","scope":"workspace"}' -o json \
     | tee qa/logs/TC-FUNC-004/config-set-trust-root.json
   ```
   - **Expected:** Returns `error.code=CONFIG_TRUST_ROOT_FORBIDDEN`. CLI parity: `agh config set daemon.bind 127.0.0.1:9999 --scope workspace` returns the same error code.

3. **Secret denial:**
   ```bash
   agh tool invoke agh__config_set --input '{"path":"providers.codex.api_key_env","value":"X","scope":"workspace"}' -o json \
     | tee qa/logs/TC-FUNC-004/config-set-secret.json
   ```
   - **Expected:** Returns `error.code=CONFIG_SECRET_PATH_FORBIDDEN`. CLI parity returns identical reason code.

4. **Scope denial:**
   ```bash
   agh tool invoke agh__config_set --input '{"path":"defaults.history_limit","value":42,"scope":"runtime"}' -o json \
     | tee qa/logs/TC-FUNC-004/config-set-scope.json
   ```
   - **Expected:** `CONFIG_SCOPE_NOT_ALLOWED` (write scope must be `global` or `workspace`).

5. **Validation failure:**
   ```bash
   agh tool invoke agh__config_set --input '{"path":"defaults.history_limit","value":-1,"scope":"workspace"}' -o json \
     | tee qa/logs/TC-FUNC-004/config-set-validation.json
   ```
   - **Expected:** `CONFIG_VALIDATION_FAILED` from the existing validator; nothing persisted.

6. **Approval requirement:**
   - With approval channel disabled (toggle via test hook), retry Step 1's allowed mutation.
   - **Expected:** Tool returns `error.code=approval_unreachable` (or `approval_required` with `approval_unreachable` reason) and no write reaches the persistence layer.

7. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     -d '{"path":"defaults.history_limit","value":7,"scope":"workspace"}' \
     "http://localhost/api/tools/agh__config_set/invoke" | tee qa/logs/TC-FUNC-004/config-set-uds.json
   ```
   - **Expected:** UDS invoke matches the tool/CLI behavior.

8. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestConfig" -count=1 | tee qa/logs/TC-FUNC-004/builtin-tests.log
   go test ./internal/config -count=1 | tee qa/logs/TC-FUNC-004/config-pkg-tests.log
   go test ./internal/daemon -run "TestNativeConfig" -count=1 | tee qa/logs/TC-FUNC-004/daemon-tests.log
   ```

## Evidence To Capture

- All allow/deny `qa/logs/TC-FUNC-004/config-*.json` payloads with their tool/CLI/UDS variants.
- `qa/logs/TC-FUNC-004/builtin-tests.log`, `config-pkg-tests.log`, `daemon-tests.log`.
- The persisted overlay file before/after the allowed mutation.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| `agh__config_unset` of allowed path | `{"path":"defaults.history_limit","scope":"workspace"}` | Removes value; identical between CLI and tool |
| `[mcp_servers]` `auth.*` secret field | Set `mcp_servers.foo.auth.access_token` | `CONFIG_SECRET_PATH_FORBIDDEN` |
| `memory.global_dir` | Set to `/tmp/mem` | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| Path that doesn't exist in schema | `defaults.unknown_field` | `CONFIG_PATH_FORBIDDEN` (or validator-equivalent code) |
| Concurrent tool + CLI set on same overlay | run both in parallel | Per CLAUDE.md, this is a workflow violation; expected behavior is one writer's success and the other's deterministic mismatch — the regression run must NOT trigger this in parallel |

## Channels Exercised

- Tool invoke (daemon native provider).
- CLI (`agh config *`).
- HTTP/UDS `/api/tools/*/invoke` and `/api/config/*` if available.

## Related Test Cases

- TC-SEC-004 (config trust-root/secret/scope denial parity).
- TC-INT-002 (transport parity).
- TC-FUNC-005 (hook mutation rides on the same approval/validation gate).
