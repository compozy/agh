---
status: completed
domain: Config
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_00
---

# Task 01: Config Package

## Overview

Implement the `internal/config` package that handles TOML configuration loading with 2-level merge (global + workspace), home directory layout, agent definition parsing from `AGENT.md` frontmatter, and the built-in provider registry. This is the foundation package with zero internal dependencies — everything else builds on it.

Also implement `internal/logger` (slog setup) and `internal/version` (build metadata) as trivial companion packages.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST load and validate TOML configuration from `~/.agh/config.toml`
- MUST support 2-level merge: global config + workspace `.agh/config.toml` overlay
- MUST support `AGH_HOME` env var to override default `~/.agh` location
- MUST define and validate all config sections: `[daemon]`, `[http]`, `[defaults]`, `[limits]`, `[permissions]`, `[providers.*]`, `[observability]`, `[log]`
- MUST parse `AGENT.md` files from `~/.agh/agents/<name>/AGENT.md` (YAML frontmatter + Markdown body)
- MUST implement built-in provider registry with hardcoded ACP commands for known agents (claude, codex, gemini, opencode, copilot, cursor, kiro, pi)
- MUST support provider config override via `[providers.*]` in config.toml
- MUST resolve fields via chain: AGENT.md override → Provider config.toml → Built-in defaults
- MUST support `mcp_servers` in both ProviderConfig and AgentDef, merged (not replaced)
- MUST ensure home directory layout (`~/.agh/agents/`, `~/.agh/sessions/`, `~/.agh/logs/`)
- MUST use functional options pattern for any complex constructors
</requirements>

## Subtasks

- [x] 1.1 Define all config structs (Config, DaemonConfig, HTTPConfig, LimitsConfig, PermissionsConfig, ProviderConfig, ObservabilityConfig, LogConfig)
- [x] 1.2 Implement TOML loading with 2-level merge (global + workspace deep-merge)
- [x] 1.3 Implement config validation (required fields, value ranges, port bounds)
- [x] 1.4 Implement HomePaths struct and EnsureHomeLayout for directory creation
- [x] 1.5 Implement AgentDef parsing from AGENT.md (YAML frontmatter + Markdown body extraction)
- [x] 1.6 Implement built-in provider registry with resolution chain
- [x] 1.7 Implement MCPServer merging logic (agent-level + provider-level)
- [x] 1.8 Implement `internal/logger` package (slog setup with configurable level + file output to `~/.agh/logs/agh.log`)
- [x] 1.9 Implement `internal/version` package (build metadata injection via ldflags)
- [x] 1.10 Implement .env loading via godotenv in config loading path

## Implementation Details

Create the following files:

- `internal/config/config.go` — Config struct, Load(), Validate(), defaults
- `internal/config/home.go` — HomePaths, EnsureHomeLayout
- `internal/config/merge.go` — 2-level TOML merge logic
- `internal/config/agent.go` — AgentDef parsing from AGENT.md
- `internal/config/provider.go` — ProviderConfig, built-in registry, resolution chain
- `internal/logger/logger.go` — slog setup
- `internal/version/version.go` — build metadata

### Relevant Files

- `CLAUDE.md` — Architecture principles and package layout
- `.compozy/tasks/agh-v2/_techspec.md` — Config section, data models, filesystem layout

### Old Project Reference

- `.old_project/internal/config/config.go` — TOML loading, top-level config struct
- `.old_project/internal/config/home.go` — Home path handling and directory resolution
- `.old_project/internal/config/discovery.go` — Agent definition discovery/parsing
- `.old_project/internal/config/roles.go` — Provider registry and role definitions
- `.old_project/internal/config/limits.go` — Duration parsing and validation patterns for config limits
- `.old_project/internal/frontmatter/frontmatter.go` — YAML frontmatter parsing

### Related ADRs

- [ADR-004: Self-Contained Agent Directory With AGENT.md](../adrs/adr-004.md) — Agent definition format
- [ADR-005: Built-In Provider Registry With ACP Commands](../adrs/adr-005.md) — Provider registry design

## Deliverables

- `internal/config/` package with full TOML loading, merge, validation, agent parsing, provider registry
- `internal/logger/` package with slog setup
- `internal/version/` package with build metadata
- Unit tests with 80%+ coverage **(REQUIRED)**
- Example `config.toml` at project root

## Tests

- Unit tests:
  - [x] Load valid TOML config with all sections
  - [x] 2-level merge: workspace overrides global values correctly
  - [x] 2-level merge: workspace adds new values without clobbering global
  - [x] Validation: reject invalid port (0, 65536+)
  - [x] Validation: reject unknown permission mode
  - [x] AGH_HOME env var overrides default home location
  - [x] EnsureHomeLayout creates all required directories
  - [x] Parse AGENT.md: valid frontmatter + Markdown body
  - [x] Parse AGENT.md: missing required fields (name, provider) returns error
  - [x] Built-in provider registry returns correct commands for known agents
  - [x] Provider config.toml override merges with built-in defaults
  - [x] Resolution chain: AGENT.md model overrides provider default_model
  - [x] MCP servers merge: agent-level + provider-level combined
  - [x] Load config from non-existent file returns defaults (not error)
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt + lint + test + build)
- Config loads correctly with default values when no config file exists
- Agent definitions parseable from `~/.agh/agents/<name>/AGENT.md`
