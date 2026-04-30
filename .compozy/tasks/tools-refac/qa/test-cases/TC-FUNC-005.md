# TC-FUNC-005: Hook Management Tool Family With Source-Immutable Protection

**Priority:** P0 (Critical)
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 30 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove `agh__hooks_*` read and mutation tools reuse the existing hook normalization, validation, and permission rules. Confirm:

- Read tools (`list`, `info`, `events`, `runs`) match CLI/HTTP/UDS output.
- Mutation tools (`create`, `update`, `delete`, `enable`, `disable`) work for config-backed/overlay-backed declarations.
- Source-owned (extension- or skill-owned) hook declarations remain structurally immutable through tools.
- Secret-bearing inputs are rejected.
- Mutating operations require approval.

## Traceability

- Task: task_06 (Hook Management Tool Family).
- TechSpec: "Mutable Surface Policy → agh__hooks", "Hooks", "Implementation Steps".
- ADRs: ADR-002, ADR-006.
- Surfaces: `internal/tools/builtin/hooks.go`, `internal/daemon/native_config_hook_tools.go`, `internal/hooks/{normalize.go,permission.go,introspection.go}`, `internal/config/hooks.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two hook fixtures pre-loaded:
  - `hook-config` — declared via workspace config overlay (mutable source).
  - `hook-skill` — supplied by a bundled skill (`HookSourceSkill`) with mutable=false (source-owned).
- ACP approval channel auto-approves mutating calls in this lab.

## Test Steps

1. **Read parity:**
   ```bash
   agh hooks list -o json | tee qa/logs/TC-FUNC-005/cli-hooks-list.json
   agh tool invoke agh__hooks_list -o json | tee qa/logs/TC-FUNC-005/tool-hooks-list.json
   diff <(jq -S . qa/logs/TC-FUNC-005/cli-hooks-list.json) \
        <(jq -S . qa/logs/TC-FUNC-005/tool-hooks-list.json)
   ```
   Repeat for `hooks_info`, `hooks_events`, `hooks_runs`.
   - **Expected:** Identical output.

2. **Allowed mutation on config-backed hook:**
   ```bash
   agh tool invoke agh__hooks_update --input '{
     "id":"hook-config",
     "matcher":"PostToolUse",
     "command":"/usr/bin/echo updated"
   }' -o json | tee qa/logs/TC-FUNC-005/tool-hooks-update.json
   ```
   - **Expected:** Update applies through `internal/hooks/normalize.go` + `internal/hooks/permission.go`. Output reflects the mutation. The persisted overlay file diff is minimal and consistent with the `agh hooks update` CLI behavior.

3. **Source-immutable denial:**
   ```bash
   agh tool invoke agh__hooks_update --input '{"id":"hook-skill","command":"/usr/bin/echo bypass"}' -o json \
     | tee qa/logs/TC-FUNC-005/tool-hooks-update-skill.json
   ```
   - **Expected:** `error.code=HOOK_SOURCE_IMMUTABLE`.
   ```bash
   agh tool invoke agh__hooks_delete --input '{"id":"hook-skill"}' -o json \
     | tee qa/logs/TC-FUNC-005/tool-hooks-delete-skill.json
   ```
   - **Expected:** Same `HOOK_SOURCE_IMMUTABLE` code.

4. **Secret-bearing input rejection:**
   ```bash
   agh tool invoke agh__hooks_create --input '{
     "id":"hook-secret",
     "matcher":"PreToolUse",
     "command":"/usr/bin/echo",
     "env":{"AWS_ACCESS_KEY_ID":"AKIA..."}
   }' -o json | tee qa/logs/TC-FUNC-005/tool-hooks-create-secret.json
   ```
   - **Expected:** `error.code=HOOK_SECRET_INPUT_FORBIDDEN`. Hook is not persisted.

5. **Validation failure:**
   ```bash
   agh tool invoke agh__hooks_create --input '{"id":"hook-invalid","matcher":"NotAValidMatcher"}' -o json \
     | tee qa/logs/TC-FUNC-005/tool-hooks-create-invalid.json
   ```
   - **Expected:** `HOOK_VALIDATION_FAILED`.

6. **Approval gating:**
   - With approval channel disabled, retry Step 2.
   - **Expected:** `HOOK_APPROVAL_REQUIRED` (with `approval_unreachable` reason or equivalent). No write applied.

7. **Enable/disable:**
   ```bash
   agh tool invoke agh__hooks_disable --input '{"id":"hook-config"}' -o json \
     | tee qa/logs/TC-FUNC-005/tool-hooks-disable.json
   agh tool invoke agh__hooks_enable --input '{"id":"hook-config"}' -o json \
     | tee qa/logs/TC-FUNC-005/tool-hooks-enable.json
   ```
   - **Expected:** Hook `enabled` field flips. CLI parity: `agh hooks disable hook-config` matches.

8. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     -d '{"id":"hook-config","matcher":"PostToolUse","command":"/usr/bin/echo via-uds"}' \
     "http://localhost/api/tools/agh__hooks_update/invoke" | tee qa/logs/TC-FUNC-005/uds-hooks-update.json
   ```

9. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestHook" -count=1 | tee qa/logs/TC-FUNC-005/builtin-tests.log
   go test ./internal/hooks ./internal/config -count=1 | tee qa/logs/TC-FUNC-005/hooks-config-tests.log
   ```

## Evidence To Capture

- CLI ↔ tool diffs for read parity.
- Allow / deny payloads.
- Persisted overlay diff for allowed mutation.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Update only `enabled` field on a source-owned hook | `agh__hooks_disable --input '{"id":"hook-skill"}'` | `HOOK_SOURCE_IMMUTABLE` (enabled flipping is treated as mutation) — confirms `HookSourceSkill` immutability is enforced |
| Create with executor command requiring trust-root path | command pointing to `/etc/passwd` | `HOOK_VALIDATION_FAILED` |
| Update preserves priority/timeout fields | tool input with new priority | Persisted overlay reflects priority; normalization enforces legal range |
| Concurrent tool + CLI mutation on same hook | sequential per CLAUDE.md | Each write is independently consistent; no test runs them in parallel |

## Channels Exercised

- Tool invoke / CLI / HTTP/UDS for hooks.
- Persisted overlay file diff.

## Related Test Cases

- TC-SEC-005 (hook secret-input + source-immutable denial sweep).
- TC-INT-002 (transport parity).
- TC-FUNC-004 (config tool family rides on the same approval gate).
