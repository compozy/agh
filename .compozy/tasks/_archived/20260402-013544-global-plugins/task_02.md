---
status: completed
domain: Core/Kernel
type: Refactor
scope: Full
complexity: medium
---

# Task 2: AGH_* Environment Variables and Remove BuildHookConfig from Interface

## Overview

Standardize all environment variables from the dual `COLLAB_*`/`AGI_*` prefixes to a unified `AGH_*` prefix across the entire codebase, and remove `BuildHookConfig()` from the `AgentDriver` interface along with the `HookConfig` struct. These are foundational changes that all driver tasks depend on.

<critical>
- ALWAYS READ the TechSpec before starting
- This task touches many files — verify ALL references are updated with Grep before finishing
- TESTS REQUIRED — every task MUST include tests in deliverables
- The project is alpha — no backward compatibility shims needed
</critical>

<requirements>
- MUST rename all env vars from COLLAB_*/AGI_* to AGH_* (see mapping in ADR-003)
- MUST update `agentRuntimeEnv()` in `internal/kernel/api.go` to emit AGH_* vars only
- MUST update `agh hook-event` CLI in `internal/cli/hooks.go` to read AGH_* vars only
- MUST update `buildEnv()` in all 4 drivers to set AGH_* vars only
- MUST add AGH_SESSION_ID, AGH_SOCKET, AGH_BIN to driver buildEnv() (currently only AGI_AGENT_NAME is set)
- MUST remove `BuildHookConfig(agentName string, hookEndpoint string) (*HookConfig, error)` from AgentDriver interface
- MUST remove `HookConfig` struct from types.go
- MUST remove `UnimplementedDriver.BuildHookConfig` stub
- MUST remove `BuildHookConfig()` method implementations from all 4 drivers
- MUST remove all callers of BuildHookConfig in Start() methods
- MUST update all test files to use AGH_* env var names
</requirements>

## Subtasks
- [x] 2.1 Update `internal/kernel/api.go` `agentRuntimeEnv()` — replace all COLLAB_*/AGI_* with AGH_* equivalents
- [x] 2.2 Update `internal/cli/hooks.go` — read AGH_SESSION_ID, AGH_SOCKET, AGH_AGENT instead of dual prefixes
- [x] 2.3 Update all 4 driver `buildEnv()` functions — set AGH_AGENT_NAME, AGH_SESSION_ID, AGH_SOCKET, AGH_BIN
- [x] 2.4 Remove `BuildHookConfig` from AgentDriver interface, HookConfig struct, and UnimplementedDriver stub in `internal/kernel/types.go`
- [x] 2.5 Remove `BuildHookConfig()` method implementations from all 4 drivers
- [x] 2.6 Update all test files to match new env var names and removed interface method

## Implementation Details

### Relevant Files
- `internal/kernel/types.go` — Remove BuildHookConfig from interface (line 64), HookConfig struct (lines 152-158), UnimplementedDriver stub (lines 551-554)
- `internal/kernel/api.go` — Update agentRuntimeEnv() (lines 843-869)
- `internal/cli/hooks.go` — Update env var reads (lines 65-93)
- `internal/drivers/claude/claude.go` — Update buildEnv() (line 702), remove BuildHookConfig() (lines 363-396)
- `internal/drivers/codex/codex.go` — Update buildEnv() (line 612), remove BuildHookConfig() (lines 300-333)
- `internal/drivers/opencode/opencode.go` — Update buildEnv() (line 1128), remove BuildHookConfig() (line 463)
- `internal/drivers/pi/pi.go` — Update buildEnv() (line 642), remove BuildHookConfig() (lines 313-343)

### Dependent Files
- `internal/kernel/api_test.go` — Update env var assertions (lines 1044-1081)
- `internal/cli/hooks_test.go` — Update env var setup in tests
- `internal/drivers/claude/claude_test.go` — Update env var assertions (lines 72-74)
- `internal/drivers/codex/codex_test.go` — Update env var assertions (lines 66-68)
- `internal/drivers/opencode/opencode_test.go` — Update env var assertions (lines 179-181)
- `internal/drivers/pi/pi_test.go` — Update env var assertions (lines 64-66)

### Related ADRs
- [ADR-003: AGH_* Environment Variable Prefix Standardization](adrs/adr-003.md) — Defines the variable mapping
- [ADR-004: Remove BuildHookConfig from AgentDriver Interface](adrs/adr-004.md) — Defines the interface change

## Deliverables
- Updated env vars across all files (zero references to COLLAB_* or AGI_* remaining)
- Cleaned AgentDriver interface (6 methods, no BuildHookConfig)
- No HookConfig struct in types.go
- Updated tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `agentRuntimeEnv()` returns only AGH_* prefixed env vars
  - [x] `agentRuntimeEnv()` includes AGH_AGENT_NAME, AGH_SESSION_ID, AGH_SOCKET
  - [x] `agh hook-event` reads AGH_SESSION_ID and AGH_SOCKET correctly
  - [x] `agh hook-event` fails gracefully when AGH_SESSION_ID is missing
  - [x] All 4 driver `buildEnv()` functions set AGH_AGENT_NAME, AGH_SESSION_ID, AGH_SOCKET, AGH_BIN
  - [x] AgentDriver interface compiles without BuildHookConfig
  - [x] UnimplementedDriver compiles without BuildHookConfig stub
  - [x] No references to COLLAB_* or AGI_* exist in codebase (grep verification)
- Test coverage target: >=80%

## Success Criteria
- `grep -r "COLLAB_\|AGI_" internal/` returns zero results (excluding comments explaining migration)
- AgentDriver interface has exactly 6 methods (Name, Start, SendMessage, Stop, ParseHookEvent, HealthCheck, DetectReady — 7 actually, but no BuildHookConfig)
- All existing tests pass with updated env var names
- `make verify` passes (fmt + lint + test + build)
