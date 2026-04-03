---
status: completed
domain: Core/CLI
type: Feature Implementation
scope: Full
complexity: high
dependencies:
    - task_01
    - task_03
---

# Task 4: Install/Uninstall Plugin Lifecycle

## Overview

Extend `agh install` to install global driver plugins alongside roles, and add `agh uninstall` to remove them. The install command detects which CLIs are available on the machine, extracts embedded plugin assets, and installs them to each CLI's global config directory using the appropriate mechanism (Claude=formal plugin, Codex=file merge, OpenCode/Pi=file copy).

<critical>
- ALWAYS READ the TechSpec before starting
- CHECK each CLI's global config directory paths for the current platform
- Codex merge logic MUST handle edge cases (missing file, existing AGH hooks, malformed JSON)
- TESTS REQUIRED — every task MUST include tests in deliverables
- Use t.TempDir() for all file system tests — never write to real global config dirs
</critical>

<requirements>
- MUST implement `plugins.DetectDrivers()` using `exec.LookPath` for claude, codex, opencode, pi binaries
- MUST implement `plugins.Install(homeDir)` that installs plugins for all detected drivers
- MUST implement `plugins.Uninstall(homeDir)` that removes AGH plugins from all driver configs
- Claude install: MUST create local marketplace structure and run `claude plugin install agh@agh-local --scope user`
- Codex install: MUST merge AGH hooks into `~/.codex/hooks.json` with `"_agh": true` marker for identification
- Codex install: MUST copy `agh-forwarder.sh` to `~/.codex/hooks/` with executable permissions (0o755)
- Codex install: MUST ensure `codex_hooks = true` in `~/.codex/config.toml`
- OpenCode install: MUST copy `agh-hook.ts` to `~/.config/opencode/plugins/`
- Pi install: MUST copy `agh-hook.ts` to `~/.pi/agent/extensions/`
- Claude uninstall: MUST run `claude plugin uninstall agh --scope user`
- Codex uninstall: MUST remove only AGH-tagged entries from `~/.codex/hooks.json`, remove forwarder script
- OpenCode uninstall: MUST remove `~/.config/opencode/plugins/agh-hook.ts`
- Pi uninstall: MUST remove `~/.pi/agent/extensions/agh-hook.ts`
- MUST return InstallReport/UninstallReport with installed/skipped/failed drivers
- MUST update `internal/cli/install.go` to call `plugins.Install()` after roles install
- MUST add `agh uninstall` command in `internal/cli/root.go`
- MUST handle missing directories gracefully (create if needed on install, skip if missing on uninstall)
</requirements>

## Subtasks
- [x] 4.1 Create `internal/plugins/install.go` with `DetectDrivers()`, `Install()`, `Uninstall()` and report types
- [x] 4.2 Implement Claude install/uninstall (local marketplace + plugin CLI)
- [x] 4.3 Implement Codex install/uninstall (JSON merge with markers + script copy + config.toml feature flag)
- [x] 4.4 Implement OpenCode install/uninstall (file copy/remove)
- [x] 4.5 Implement Pi install/uninstall (file copy/remove)
- [x] 4.6 Update `internal/cli/install.go` to include plugin installation
- [x] 4.7 Create `agh uninstall` command and register in root.go

## Implementation Details

### Relevant Files
- `internal/plugins/install.go` — New file: Install(), Uninstall(), DetectDrivers()
- `internal/plugins/embed.go` — Read from (created in task_01)
- `internal/cli/install.go` — Modify to add plugin installation
- `internal/cli/root.go` — Add uninstall command registration (around line 54)

### Dependent Files
- `internal/plugins/embed.go` (task_01) — provides Assets embed.FS
- All driver files (task_03) — must be done so install is coherent with driver behavior

### Related ADRs
- [ADR-001: Global Plugins Over Per-Session File Generation](adrs/adr-001.md) — Install-once approach
- [ADR-002: Hybrid Plugin Strategy](adrs/adr-002.md) — Claude=formal plugin, Codex=merge, OpenCode/Pi=copy

## Deliverables
- `internal/plugins/install.go` with full install/uninstall logic for all 4 drivers
- `internal/plugins/install_test.go` with comprehensive tests
- Updated `internal/cli/install.go` with plugin installation
- New `agh uninstall` command
- Updated tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `DetectDrivers()` returns correct list when CLIs are present/absent (mock exec.LookPath)
  - [x] Install creates correct files in Claude plugin directory structure
  - [x] Install merges Codex hooks.json correctly when file is empty/missing
  - [x] Install merges Codex hooks.json correctly when file has existing hooks (preserves them)
  - [x] Install merges Codex hooks.json correctly when AGH hooks already exist (idempotent update)
  - [x] Install handles malformed Codex hooks.json gracefully (backup + overwrite)
  - [x] Install copies agh-forwarder.sh with executable permissions
  - [x] Install ensures codex_hooks feature flag in config.toml
  - [x] Install copies OpenCode plugin to correct directory
  - [x] Install copies Pi extension to correct directory
  - [x] Install creates target directories if they don't exist
  - [x] Install returns report with correct installed/skipped counts
  - [x] Uninstall removes Claude plugin
  - [x] Uninstall removes only AGH entries from Codex hooks.json (preserves user hooks)
  - [x] Uninstall removes OpenCode plugin file
  - [x] Uninstall removes Pi extension file
  - [x] Uninstall handles missing files/dirs gracefully (no error)
  - [x] Uninstall returns report with correct removed/skipped counts
- Integration tests:
  - [x] Full install → verify files exist → uninstall → verify files removed
- Test coverage target: >=80%

## Success Criteria
- `agh install` installs roles AND driver plugins for all detected CLIs
- `agh uninstall` cleanly removes all AGH plugins without affecting user's config
- Codex install is idempotent (running twice produces same result)
- Install/uninstall reports clearly show what was done
- `make verify` passes (fmt + lint + test + build)
