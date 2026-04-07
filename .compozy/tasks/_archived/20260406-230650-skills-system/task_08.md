---
status: completed
title: Config changes (SkillsConfig + merge overlay)
type: backend
complexity: medium
dependencies: []
---

# Task 08: Config changes (SkillsConfig + merge overlay)

## Overview

Add `SkillsConfig` to the AGH configuration system with TOML loading, validation, merge overlay support, and `SkillsDir` in `HomePaths`. This follows the exact same patterns used by `MemoryConfig` and `DreamConfig` in the existing config package.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `SkillsConfig` struct to `config/config.go` with `Enabled bool`, `DisabledSkills []string`, `PollInterval time.Duration`
- MUST add `Skills SkillsConfig` field to the `Config` struct
- MUST set defaults: `Enabled: true`, `PollInterval: 3 * time.Second`
- MUST add `skillsOverlay` struct to `config/merge.go` with pointer fields following existing overlay pattern
- MUST add `Skills skillsOverlay` field to `configOverlay` struct
- MUST wire `Skills.Apply()` in `configOverlay.Apply()` method
- MUST add `SkillsDir string` to `HomePaths` struct in `config/home.go`
- MUST set `SkillsDir` to `filepath.Join(root, "skills")` in `ResolveHomePathsFrom()`
- MUST add `SkillsDir` to `EnsureHomeLayout()` directory creation list
- MUST use `time.Duration` for `PollInterval` (not string) — matches `DreamConfig.CheckInterval` pattern
</requirements>

## Subtasks
- [x] 8.1 Add `SkillsConfig` struct and field to `Config` in `config.go`
- [x] 8.2 Set defaults in `Default()` and `DefaultWithHome()` functions
- [x] 8.3 Add `skillsOverlay` and wire in `merge.go`
- [x] 8.4 Add `SkillsDir` to `HomePaths` in `home.go`
- [x] 8.5 Update `EnsureHomeLayout()` to create skills directory
- [x] 8.6 Write unit tests for config loading, merging, and home paths

## Implementation Details

Follow the `MemoryConfig`/`memoryOverlay` pattern exactly. See TechSpec "Config (internal/config)" section.

### Relevant Files
- `internal/config/config.go` — Config struct (line 97), MemoryConfig (line 81) as pattern
- `internal/config/merge.go` — memoryOverlay (line 73), dreamOverlay (line 79) as patterns
- `internal/config/home.go` — HomePaths (line 34), EnsureHomeLayout (line 95)

### Dependent Files
- `daemon/daemon.go` — Will read cfg.Skills at boot (task_10)
- `cli/skill.go` — Will read config for directory resolution (task_11)

## Deliverables
- Modified `internal/config/config.go` with SkillsConfig
- Modified `internal/config/merge.go` with skillsOverlay
- Modified `internal/config/home.go` with SkillsDir
- Updated tests in `internal/config/config_test.go`, `merge_test.go`, `home_test.go`
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Default config has Skills.Enabled = true
  - [x] Default config has Skills.PollInterval = 3s
  - [x] TOML config with `[skills]` section parses correctly
  - [x] TOML config with `skills.enabled = false` overrides default
  - [x] TOML config with `skills.poll_interval` as duration parses correctly
  - [x] TOML config with `skills.disabled_skills` list parses correctly
  - [x] Merge overlay applies skills config from workspace overlay
  - [x] HomePaths includes SkillsDir at expected path
  - [x] EnsureHomeLayout creates skills directory
  - [x] Unknown keys under `[skills]` cause error (strict TOML decoding)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- `make verify` passes (existing tests not broken)
- Config pattern matches existing MemoryConfig/DreamConfig exactly
