---
status: resolved
file: internal/drivers/claude/claude.go
line: 639
severity: medium
author: claude-code
provider_ref:
---

# Issue 003: buildEnv and helpers duplicated across all four drivers

## Review Comment

The `buildEnv`, `dropLegacyRuntimeEnvKeys`, and `resolveAGHBinaryPath`/`resolveHookCommandPath` functions are nearly identical across all four driver packages (claude, codex, opencode, pi). Each implementation:

1. Inherits `os.Environ()`
2. Drops legacy `AGI_*`/`COLLAB_*` prefixes
3. Merges caller-provided env vars
4. Sets `AGH_AGENT_NAME` and `AGH_BIN`
5. Sorts and formats as `key=value` slices

The only variation is OpenCode adding `OPENCODE_CONFIG_CONTENT` and Pi/OpenCode accepting an extra `aghBinaryPath` parameter.

This duplication increases the risk of inconsistent bug fixes (e.g., if the `backupFile` scoping pattern or env var logic needs a fix, it must be applied in four places). Extract the shared env-building logic into a shared internal package (e.g., `internal/driverutil` or `internal/drivers/shared`), keeping only driver-specific additions in each driver.

## Triage

- Decision: `invalid`
- Notes:
  - The review comment identifies duplication across the driver packages, but it does not identify a concrete behavioral bug in `internal/drivers/claude/claude.go` or a failing scenario unique to this batch.
  - The duplicated helpers are not byte-identical: OpenCode adds `OPENCODE_CONFIG_CONTENT`, Pi/OpenCode accept an explicit hook binary path, and Claude/Codex intentionally resolve their own executable path. Extracting them safely would require a broader shared-driver refactor across packages outside this review batch.
  - Given the scoped remediation workflow, this is an architectural improvement request rather than an actionable defect. Closing as `invalid` for this batch keeps the review focused on reproducible correctness issues.
