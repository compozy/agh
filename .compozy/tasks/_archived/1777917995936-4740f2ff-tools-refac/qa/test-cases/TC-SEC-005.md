# TC-SEC-005: Hook Secret-Input Rejection And Source-Immutable Enforcement

**Priority:** P0 (Critical)
**Type:** Security / Redaction
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-30
**Last Updated:** 2026-04-30

## Objective

Prove that hook mutation tools reject secret-bearing inputs (env, command args, headers) with `HOOK_SECRET_INPUT_FORBIDDEN` and that source-owned hook declarations remain structurally immutable through the tool surface (`HOOK_SOURCE_IMMUTABLE`). Confirm parity across tool, CLI, and HTTP/UDS.

## Traceability

- Task: task_06.
- TechSpec: "Mutable Surface Policy → agh__hooks", "Hooks".
- ADR: ADR-006.
- Surfaces: `internal/tools/builtin/hooks.go`, `internal/hooks/{normalize.go,permission.go}`, `internal/config/hooks.go`.

## Preconditions

- Isolated `AGH_HOME`.
- Two hook fixtures:
  - `hook-config` — workspace overlay (mutable source).
  - `hook-skill` — bundled skill (`HookSourceSkill`, immutable).
- Approval channel auto-approves.

## Test Steps

1. **Secret-bearing env rejection:**
   ```bash
   agh tool invoke agh__hooks_create --input '{
     "id":"hook-secret",
     "matcher":"PreToolUse",
     "command":"/usr/bin/echo",
     "env":{"OPENAI_API_KEY":"sk-..."}
   }' -o json | tee qa/logs/TC-SEC-005/tool-create-secret-env.json
   ```
   - **Expected:** `HOOK_SECRET_INPUT_FORBIDDEN`.

2. **Secret-bearing arg rejection:**
   ```bash
   agh tool invoke agh__hooks_create --input '{
     "id":"hook-secret-arg",
     "matcher":"PreToolUse",
     "command":"/usr/bin/echo",
     "args":["--password","hunter2"]
   }' -o json | tee qa/logs/TC-SEC-005/tool-create-secret-arg.json
   ```
   - **Expected:** Either `HOOK_SECRET_INPUT_FORBIDDEN` (if the validator detects sensitive-key conventions) OR `HOOK_VALIDATION_FAILED` per existing rules. Whichever the validator returns must match CLI behavior. Document which one in the evidence file and confirm parity.

3. **Source-immutable update:**
   ```bash
   agh tool invoke agh__hooks_update --input '{"id":"hook-skill","command":"/usr/bin/echo bypass"}' -o json \
     | tee qa/logs/TC-SEC-005/tool-update-skill.json
   ```
   - **Expected:** `HOOK_SOURCE_IMMUTABLE`.

4. **Source-immutable delete:**
   ```bash
   agh tool invoke agh__hooks_delete --input '{"id":"hook-skill"}' -o json \
     | tee qa/logs/TC-SEC-005/tool-delete-skill.json
   ```
   - **Expected:** `HOOK_SOURCE_IMMUTABLE`.

5. **Source-immutable disable (enabled flip):**
   ```bash
   agh tool invoke agh__hooks_disable --input '{"id":"hook-skill"}' -o json \
     | tee qa/logs/TC-SEC-005/tool-disable-skill.json
   ```
   - **Expected:** `HOOK_SOURCE_IMMUTABLE` (per shared workflow memory: `HookSourceSkill` immutability is enforced on enable/disable).

6. **CLI parity:**
   ```bash
   agh hooks update hook-skill --command "/usr/bin/echo bypass" -o json | tee qa/logs/TC-SEC-005/cli-update-skill.json
   ```
   - **Expected:** Same `HOOK_SOURCE_IMMUTABLE` code.

7. **HTTP/UDS parity:**
   ```bash
   curl -s -X POST --unix-socket "$AGH_HOME/run/sock/uds.sock" \
     -d '{"id":"hook-skill","command":"/usr/bin/echo bypass"}' \
     "http://localhost/api/tools/agh__hooks_update/invoke" | tee qa/logs/TC-SEC-005/uds-update-skill.json
   ```

8. **Allowed mutation on `hook-config`:**
   ```bash
   agh tool invoke agh__hooks_update --input '{"id":"hook-config","command":"/usr/bin/echo allowed"}' -o json \
     | tee qa/logs/TC-SEC-005/tool-update-config.json
   ```
   - **Expected:** Success.

9. Run focused Go tests:
   ```bash
   go test ./internal/tools/builtin -run "TestHook" -count=1 | tee qa/logs/TC-SEC-005/builtin-tests.log
   go test ./internal/hooks ./internal/config -count=1 | tee qa/logs/TC-SEC-005/hooks-config-tests.log
   ```

## Evidence To Capture

- All allow / deny payloads per surface.
- Test logs.

## Edge Cases And Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Hook stored under workspace overlay but with sensitive key in env | `env={"FOO_SECRET":"x"}` (key matches sensitivity rule) | `HOOK_SECRET_INPUT_FORBIDDEN` |
| Update of source-owned hook that only changes priority | priority-only diff | Still `HOOK_SOURCE_IMMUTABLE` — source-owned means structurally immutable |
| Source-owned hook deletion via overlay shadowing | adding overlay row to mask | Out of scope for tool surface; must remain operator-only |

## Channels Exercised

- Tool / CLI / HTTP/UDS for hook mutation.

## Related Test Cases

- TC-FUNC-005 (hook tool family parity).
- TC-SEC-004 (config secret/scope denial parity).
