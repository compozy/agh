# TC-SEC-004: Config Trust-Root, Secret, And Scope Denial Parity

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove the deterministic config denial taxonomy returns the same code across tool, CLI, and HTTP/UDS for forbidden trust-root paths, secret-bearing paths, scope-not-allowed mutations, and validation failures.

## Traceability

- Task: task_05.
- TechSpec: "Mutable Surface Policy → agh__config", "Config Lifecycle".
- ADR: ADR-006.
- Surfaces: `internal/tools/builtin/config.go`, `internal/config/{persistence.go,merge.go,tools.go}`, `internal/cli/config.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Approval channel auto-approves mutating calls.

## Test Steps

For each forbidden path below, exercise tool, CLI, and UDS surfaces and confirm matching deterministic error codes.

| Path | Forbidden because | Expected code |
|------|-------------------|---------------|
| `daemon.bind` | trust-root daemon transport | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `http.bind` | trust-root | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `permissions.allow_unknown_tools` | trust-root | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `memory.global_dir` | trust-root | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `providers.codex.command` | trust-root provider transport | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `providers.codex.api_key_env` | secret env binding | `CONFIG_SECRET_PATH_FORBIDDEN` |
| `mcp_servers.foo.transport` | trust-root | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `mcp_servers.foo.auth.access_token` | secret | `CONFIG_SECRET_PATH_FORBIDDEN` |
| `sandboxes.local.backend` | trust-root | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `log.retention` | trust-root for audit | `CONFIG_TRUST_ROOT_FORBIDDEN` |
| `defaults.history_limit` (scope=runtime) | scope not allowed | `CONFIG_SCOPE_NOT_ALLOWED` |
| `defaults.history_limit=-1` (scope=workspace) | validator fails | `CONFIG_VALIDATION_FAILED` |
| `defaults.unknown_field` | unknown path | `CONFIG_PATH_FORBIDDEN` |

For each row:

```bash
# Tool
agh tool invoke agh__config_set --input '{"path":"<path>","value":"<v>","scope":"workspace"}' -o json \
  | tee qa/logs/TC-SEC-004/tool-<path>.json
# CLI
agh config set <path> <v> --scope workspace -o json | tee qa/logs/TC-SEC-004/cli-<path>.json
# UDS
curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
  -d '{"path":"<path>","value":"<v>","scope":"workspace"}' \
  "http://localhost/api/tools/agh__config_set/invoke" | tee qa/logs/TC-SEC-004/uds-<path>.json
```

Compare:

```bash
jq -r '.error.code' qa/logs/TC-SEC-004/tool-<path>.json
jq -r '.error.code' qa/logs/TC-SEC-004/cli-<path>.json
jq -r '.error.code' qa/logs/TC-SEC-004/uds-<path>.json
```

- **Expected:** All three surfaces return the same `error.code`. Persisted overlay file is unchanged.

After every row, confirm overlay file is byte-identical to its pre-test snapshot:

```bash
diff $AGH_HOME/config.toml qa/logs/TC-SEC-004/config.toml.before
```

Run focused Go tests:

```bash
go test ./internal/config -run "TestPolicy|TestForbidden|TestValidate" -count=1 \
  | tee qa/logs/TC-SEC-004/config-tests.log
go test ./internal/tools/builtin -run "TestConfig" -count=1 | tee qa/logs/TC-SEC-004/builtin-tests.log
```

## Evidence To Capture

- All allow / deny payloads per surface, per row.
- Pre/post overlay file diff.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Path that the writer would normally accept but with secret-marker added at runtime | dynamic config schema flag | `CONFIG_SECRET_PATH_FORBIDDEN` consistent across surfaces |
| Two consecutive forbidden writes from the same surface | rapid retries | Each call returns the same deterministic code; no overlay drift |
| Tool surface bypassed via direct package import in a test | NOT applicable in QA — production code only | n/a |

## Channels Exercised

- Tool, CLI, HTTP/UDS.
- Persisted overlay file.

## Related Test Cases

- TC-FUNC-004 (config tool family parity for allowed paths).
- TC-SEC-005 (hook secret-input rejection).
