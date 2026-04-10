---
status: completed
title: Config and agent-definition hook declarations
type: backend
complexity: medium
dependencies:
  - task_01
---

# Task 8: Config and agent-definition hook declarations

## Overview

Extend `internal/config` to parse hook declarations from TOML config layers (policy, user, workspace) and extend agent-definition loading to emit hook declarations. Both feed into the hooks registry as declaration sources alongside Go-native and skill hooks.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST extend config schema to support hook declarations under a `[hooks]` or `[[hooks.declarations]]` section
- MUST parse hook fields: name, event, mode, required, priority, timeout, matcher, executor (command, args, env)
- MUST validate declarations at config load time using hooks normalization from task_02
- MUST respect existing config precedence (policy → user → workspace)
- MUST extend agent-definition parsing to support hooks in agent def YAML/TOML
- MUST scope agent-definition hooks to execute only for matching agent type sessions
- MUST export a function that returns `[]hooks.HookDecl` for the registry to consume
- SHOULD apply default priority 500 for config hooks and 100 for agent-definition hooks
</requirements>

## Subtasks
- [x] 8.1 Extend config schema with hook declaration fields
- [x] 8.2 Implement TOML parsing for hook declarations in config loader
- [x] 8.3 Implement hook declaration parsing in agent-definition loader
- [x] 8.4 Implement validation using hooks normalization functions
- [x] 8.5 Export `HookDeclarations()` function from config for registry consumption
- [x] 8.6 Write unit tests for config and agent-definition hook parsing

## Implementation Details

Modify existing files:
- `internal/config/config.go` — Add hooks section to config struct
- `internal/config/loader.go` or relevant loading file — Parse hook declarations from TOML
- Agent definition loading code — Parse hooks from agent definitions

Reference TechSpec "Declaration Model" section. Reference existing config loading patterns in `internal/config/`.

### Relevant Files
- `internal/config/config.go:104-112` — Current SkillsConfig with AllowedMarketplaceHooks
- `internal/config/` — Config loading and validation patterns
- `internal/hooks/types.go` (task_01) — HookDecl type to use
- `internal/hooks/normalize.go` (task_02) — Normalization functions for validation

### Dependent Files
- `internal/hooks/hooks.go` (task_06) — Registry Rebuild consumes these declarations
- `internal/daemon/boot.go` — Wiring (task_09) connects config declarations to registry

### Related ADRs
- [ADR-004: Support Four Declaration Sources](../adrs/adr-004.md) — Config and agent-definition as sources
- [ADR-011: Simplify Ordering](../adrs/adr-011.md) — Default priority 500 for config, 100 for agent-def

## Deliverables
- Extended config schema with hook declarations
- Config hook parsing and validation
- Agent-definition hook parsing
- Export function for registry consumption
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Parse valid TOML hook declaration with all fields
  - [x] Parse TOML hook with minimal fields (name + event + command) — defaults applied
  - [x] Invalid event in config hook fails validation with descriptive error
  - [x] Config hook `required: true` with `mode: async` fails validation
  - [x] Config hooks from multiple precedence levels merge correctly
  - [x] Agent-definition hook parsed from YAML with agent name scope
  - [x] Agent-definition hook scoped to agent type — matcher includes agent name
  - [x] `HookDeclarations()` returns combined config + agent-def declarations
  - [x] Empty config hooks section returns empty list (not nil)
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Config hooks integrate with existing config precedence
- Agent-definition hooks are properly scoped to agent type
