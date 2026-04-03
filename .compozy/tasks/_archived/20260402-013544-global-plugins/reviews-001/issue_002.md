---
status: resolved
file: internal/plugins/pi/agh-hook.ts
line: 11
severity: medium
author: claude-code
provider_ref:
---

# Issue 002: Pi hook uses unquoted shell interpolation in execSync

## Review Comment

The Pi plugin passes `aghBin` and `agent` via template literal string interpolation into `execSync`, which executes through the system shell:

```typescript
execSync(`${aghBin} hook-event --agent ${agent}`, {
    input: JSON.stringify(event),
    timeout: 5000,
    stdio: ["pipe", "pipe", "pipe"],
})
```

If `AGH_BIN` resolves to a path containing spaces (e.g., `/path/to my apps/agh`), the command will break. The Claude and Codex plugins correctly quote their variables (`"${AGH_BIN:-agh}"` and `"$AGENT"`), but the Pi plugin does not.

**Fix:** Use `execFileSync` instead of `execSync` to avoid shell interpretation entirely:

```typescript
import { execFileSync } from "node:child_process"

execFileSync(aghBin, ["hook-event", "--agent", agent], {
    input: JSON.stringify(event),
    timeout: 5000,
    stdio: ["pipe", "pipe", "pipe"],
})
```

## Triage

- Decision: `valid`
- Notes:
  - Confirmed in `internal/plugins/pi/agh-hook.ts`: the plugin shells out via `execSync(`${aghBin} hook-event --agent ${agent}`, ...)`, so both `AGH_BIN` and `AGH_AGENT_NAME` are parsed by the shell instead of being passed as argv.
  - This breaks valid executable paths containing spaces and needlessly broadens shell interpretation for data that is already available as discrete arguments.
  - The correct fix is to invoke the binary directly with `execFileSync` and a fixed argv slice, preserving the current best-effort hook semantics while removing shell quoting hazards.
  - Fixed by switching the Pi extension from `execSync` string interpolation to `execFileSync(aghBin, ["hook-event", "--agent", agent], ...)`.
  - Updated embedded asset coverage in `internal/plugins/embed_test.go` to assert the direct argv form is present and the old template-literal `execSync` shell invocation is absent.
  - Verification: `go test ./internal/plugins` and `make verify` both passed after the fix.
