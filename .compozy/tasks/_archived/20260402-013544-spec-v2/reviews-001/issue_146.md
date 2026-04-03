---
status: resolved
file: internal/drivers/codex/codex.go
line: 603
severity: medium
author: claude-reviewer
---

# Issue 146: Substantial code duplication across all three driver packages



## Review Comment

The following functions are duplicated verbatim (or near-verbatim) across all three driver packages (`codex`, `opencode`, `pi`):

1. `buildEnv(agentName string, envVars map[string]string) []string` - identical in all three (codex:603, opencode:1119, pi:633)
2. `cloneAnyMap(values map[string]any) map[string]any` - identical in all three (codex:630, opencode:1274, pi:612)
3. `waitWithTimeout(ctx, proc, timeout)` - near-identical in all three (codex:662, opencode:1209, pi:676) — codex variant uses `WaitContext` while opencode/pi use raw goroutine+channel
4. `allocatedPID(allocated, cmd)` - identical in all three (codex:684, opencode:1237, pi:704)
5. `closeAllocated(allocated)` - identical in all three (codex:694, opencode:1247, pi:714)
6. `isExpectedStopExit(err)` - identical in all three (codex:712, opencode:1265, pi:732)
7. `isProcessGone(err)` - identical in all three (codex:717, opencode:1270, pi:737)
8. `resolveConfigPath(workDir, configPath)` - identical in all three (codex:651, opencode:1146, pi:660)
9. `translateModel(model)` - identical in all three (codex:491, opencode:1099, pi:513)
10. `isResearcherRole(role)` - identical in codex:512 and opencode:1304
11. `coalesce`/`firstNonEmpty` - identical in opencode:1291/1300 and pi:624

The `Stop` method body is also nearly identical across all three drivers, differing only in the error message prefix string.

This violates the single responsibility principle and creates a maintenance burden. Bug fixes need to be applied in all three packages.

**Suggested fix**: Extract the shared utilities into a common `internal/drivers/driverutil` (or similar) package. The shared `Stop` logic could be extracted into a helper that takes a driver name prefix parameter.

## Triage

- Decision: `invalid`
- Notes:
  - The review identifies real duplication, but this batch is for correctness and regression fixes inside the listed files, not cross-package architectural cleanup.
  - The duplicated helpers are stable, tiny, and already covered by package-local tests. Consolidating them into a shared utility package would widen scope across multiple drivers without fixing a demonstrated runtime defect.
  - The recommended `driverutil` extraction would also force API churn in all driver packages and associated tests. That is refactor work, not a production bug fix for this review batch.
  - Resolution: closed as a non-blocking refactor recommendation; no production change was required for this batch.
