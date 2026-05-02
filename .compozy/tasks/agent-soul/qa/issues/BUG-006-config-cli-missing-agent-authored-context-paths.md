# BUG-006: Config CLI Could Not Set Agent Soul And Heartbeat Overlay Paths

**Severity:** Medium
**Priority:** P1
**Type:** Functional
**Status:** Fixed

## Environment

- **Build:** current branch during `agent-soul-qa` continuation on 2026-05-02.
- **OS:** macOS local isolated QA lab.
- **Browser:** not in scope.
- **URL:** `agh config set` CLI.
- **Live provider/LLM:** not required; reproduced through the real CLI against the isolated runtime home.

## Summary

During TC-REG-005, valid config overlay writes for `[agents.soul]` and `[agents.heartbeat]` failed through `agh config set` even though those paths were listed in the runtime config surface. This prevented operators and agents from managing authored-context limits through the documented config lifecycle.

## Behavioral Impact

- **Operator/User Goal:** operators could not tune `agents.soul.context_projection_bytes` or Heartbeat cadence through the CLI.
- **Agent Behavior:** agent-manageable config surfaces were incomplete for Agent Soul and Heartbeat.
- **Business Outcome:** the feature was not fully configurable through the expected operator surface.
- **Cross-Surface State:** `internal/config/tool_surface.go` exposed the keys, but the CLI mutation allowlist rejected them.

## Reproduction

```bash
AGH_HOME="$LAB/.agh/runtime" ./bin/agh config set agents.soul.context_projection_bytes 1536 --scope workspace --workspace agent-soul-lab -o json
```

Observed before the fix:

- CLI exited non-zero with `config path "agents.soul.context_projection_bytes" is not supported by config set`.

## Expected

The CLI must accept valid Agent Soul and Heartbeat config paths, persist valid overlays, and reject invalid values with field-specific validation errors without mutating the current config.

## Root Cause

`internal/config/tool_surface.go` already listed Agent Soul and Heartbeat keys, but `internal/cli/config.go` omitted those paths from `configScalarMutationKinds`.

## Fix

- Added Agent Soul and Heartbeat scalar mutation paths to `internal/cli/config.go`.
- Added `TestConfigSetSupportsAgentAuthoredContextPaths` in `internal/cli/config_test.go`.

## Verification

- Focused regression: `.compozy/tasks/agent-soul/qa/evidence/BUG-006-config-cli-focused-go-test.log`.
- Live lab valid overlay evidence:
  - `.compozy/tasks/agent-soul/qa/evidence/TC-REG-005-config-soul-context-set-1536.json`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-REG-005-config-heartbeat-default-set-25m.json`
- Live lab invalid rejection evidence:
  - `.compozy/tasks/agent-soul/qa/evidence/TC-REG-005-config-invalid-rejection-summary.json`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-REG-005-config-soul-invalid-stderr.log`
  - `.compozy/tasks/agent-soul/qa/evidence/TC-REG-005-config-heartbeat-invalid-stderr.log`

## Impact

- **Users Affected:** operators and agents managing runtime config through CLI.
- **Frequency:** always for the omitted Agent Soul and Heartbeat paths before the fix.
- **Workaround:** none inside `agh config set`.

## Related

- Test Case: TC-REG-005
