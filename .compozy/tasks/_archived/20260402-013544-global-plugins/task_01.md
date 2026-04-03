---
status: completed
domain: Core/Plugins
type: Feature Implementation
scope: Full
complexity: medium
---

# Task 1: Plugin Assets and Embed Package

## Overview

Create the `internal/plugins/` package with static plugin/extension files for all four drivers (Claude, Codex, OpenCode, Pi) and a `go:embed` declaration to bundle them into the AGH binary. These assets are the hook forwarder plugins that get installed globally once and forward tool events to `agh hook-event` using `AGH_*` environment variables for dynamic configuration.

<critical>
- ALWAYS READ the TechSpec before starting
- REFERENCE TECHSPEC for exact plugin content — do not invent
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create Claude plugin with `.claude-plugin/plugin.json` manifest and `hooks/hooks.json` with PostToolUse hook
- MUST create Codex `hooks.json` and `agh-forwarder.sh` shell script that reads stdin JSON and calls `agh hook-event`
- MUST create OpenCode TypeScript plugin with `tool.execute.before` and `tool.execute.after` hooks
- MUST create Pi TypeScript extension with `pi.on("tool_execution_start/end")` handlers
- MUST use `AGH_AGENT_NAME` as gating variable in all plugins (absent = no-op)
- MUST reference `AGH_BIN` env var for agh binary path with fallback to "agh"
- MUST embed all assets via `go:embed` in `embed.go`
- MUST verify all embedded assets are accessible at runtime
</requirements>

## Subtasks
- [x] 1.1 Create `internal/plugins/` directory structure with subdirectories for each driver
- [x] 1.2 Create Claude plugin assets (`plugin.json`, `hooks/hooks.json`)
- [x] 1.3 Create Codex plugin assets (`hooks.json`, `agh-forwarder.sh`)
- [x] 1.4 Create OpenCode plugin asset (`agh-hook.ts`)
- [x] 1.5 Create Pi plugin asset (`agh-hook.ts`)
- [x] 1.6 Create `internal/plugins/embed.go` with `//go:embed` declarations exposing `Assets embed.FS`

## Implementation Details

### Relevant Files
- `internal/plugins/embed.go` — New file: `go:embed` declarations
- `internal/plugins/claude/.claude-plugin/plugin.json` — New file: Claude plugin manifest
- `internal/plugins/claude/hooks/hooks.json` — New file: Claude hook config
- `internal/plugins/codex/hooks.json` — New file: Codex hook config
- `internal/plugins/codex/agh-forwarder.sh` — New file: Codex forwarder script
- `internal/plugins/opencode/agh-hook.ts` — New file: OpenCode plugin
- `internal/plugins/pi/agh-hook.ts` — New file: Pi extension

### Dependent Files
- `internal/plugins/install.go` (task_04) — will read from `Assets embed.FS`

### Related ADRs
- [ADR-001: Global Plugins Over Per-Session File Generation](adrs/adr-001.md) — Defines the plugin-per-driver approach
- [ADR-002: Hybrid Plugin Strategy](adrs/adr-002.md) — Defines Claude=formal plugin, Codex=file merge, OpenCode/Pi=standalone file

## Deliverables
- `internal/plugins/embed.go` with embedded filesystem
- All 7 plugin asset files for 4 drivers
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Embedded FS contains all expected files (claude, codex, opencode, pi)
  - [x] Claude `plugin.json` is valid JSON with required fields (name, description, version)
  - [x] Claude `hooks.json` is valid JSON with PostToolUse hook referencing `$AGH_AGENT_NAME`
  - [x] Codex `hooks.json` is valid JSON with PostToolUse hook pointing to forwarder script
  - [x] Codex `agh-forwarder.sh` contains `AGH_AGENT_NAME` gating and `AGH_BIN` reference
  - [x] OpenCode `agh-hook.ts` contains `AGH_AGENT_NAME` check and `tool.execute.before/after` handlers
  - [x] Pi `agh-hook.ts` contains `AGH_AGENT_NAME` check and `tool_execution_start/end` handlers
  - [x] All embedded files are non-empty and readable
- Test coverage target: >=80%

## Success Criteria
- `internal/plugins.Assets` exposes all plugin files via `embed.FS`
- All plugin files contain correct `AGH_*` env var references
- All plugins have `AGH_AGENT_NAME` gating (no-op when absent)
- `make verify` passes (fmt + lint + test + build)
