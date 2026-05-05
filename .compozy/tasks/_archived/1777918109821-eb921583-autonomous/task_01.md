---
status: completed
title: Autonomy Config Foundation
type: backend
complexity: medium
dependencies: []
---

# Task 01: Autonomy Config Foundation

## Overview
Add the configuration foundation for the autonomy MVP without starting any coordinator behavior yet. This task makes coordinator policy explicit, validates provider/model settings, and preserves the workspace > global > bundled-default resolution contract required by later tasks.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001/ADR-005 before changing autonomy config
- REFERENCE TECHSPEC for implementation details - do not duplicate structs from the spec into production blindly
- FOCUS ON "WHAT" - implement config semantics and validation, not coordinator runtime behavior
- MINIMIZE CODE - keep daemon composition as the only wiring root
- TESTS REQUIRED - config parsing, validation, defaulting, and workspace overlay behavior are mandatory
- NO WORKAROUNDS - do not skip validation or add loose map-based config to avoid typed TOML work
</critical>

<requirements>
- MUST add typed global `[autonomy.coordinator]` configuration with conservative defaults.
- MUST validate enabled state, agent name, provider/model, TTL, max children, and coordinator uniqueness-related limits.
- MUST expose a coordinator config resolver contract that preserves workspace override > global config > bundled/default agent definition precedence.
- MUST support greenfield schema/config changes without compatibility branches for old alpha config.
- MUST not spawn, stop, or prompt coordinator sessions in this task.
- MUST document whether this task changes generated contracts, `web/`, or `packages/site`; if not applicable, say so in completion notes.
</requirements>

## Subtasks
- [x] 1.1 Add typed autonomy/coordinator config structs, defaults, and validation.
- [x] 1.2 Merge autonomy config through global and workspace `.agh/config.toml` overlays.
- [x] 1.3 Add a daemon-facing resolver interface and no-op implementation path for later coordinator bootstrap.
- [x] 1.4 Add config tests for defaults, workspace override precedence, invalid providers/models, TTL bounds, and unknown key rejection.
- [x] 1.5 Record contract/web/docs impact as not applicable or apply required generated updates if a public DTO is touched.

## Implementation Details
Keep config ownership in `internal/config` and wire only a resolver boundary in `internal/daemon`. Use existing TOML loading, validation, and workspace overlay patterns instead of adding a parallel config loader.

### Relevant Files
- `internal/config/config.go` - top-level config structs, defaults, and validation entry points.
- `internal/config/merge.go` - workspace overlay behavior and TOML merge semantics.
- `internal/config/config_test.go` - existing config default/overlay validation coverage.
- `internal/config/provider.go` - provider/model resolution precedent for coordinator validation.
- `internal/daemon/` - future composition root for the coordinator resolver.
- `.resources/multica/CLI_AND_DAEMON.md` - reference for daemon/CLI separation and config boundaries.
- `.resources/paperclip/cli/src/config/schema.ts` - reference for explicit config schema validation.
- `.resources/hermes/cli-config.yaml.example` - reference for provider/model policy expressed as config.

### Dependent Files
- `internal/daemon/*` - later tasks consume the resolver without importing config everywhere.
- `internal/api/contract/*` - task_02 may expose config read DTOs.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` - task_16 documents the final config.

### Related ADRs
- [ADR-001: Phased Autonomy Kernel Scope](adrs/adr-001.md) - defines MVP scope.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - defines coordinator configurability and precedence.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - config must not remove manual flows.

## Deliverables
- Typed autonomy coordinator config with defaults and validation.
- Workspace overlay merge behavior covered by tests.
- Resolver contract ready for coordinator bootstrap.
- Unit tests with 80%+ coverage for touched config code **(REQUIRED)**.
- Integration-style config load tests for global + workspace overlay **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Default config returns coordinator auto-start disabled/enabled according to TechSpec default and non-zero safe limits.
  - [x] Workspace coordinator override wins over global config for provider, model, TTL, max children, and enabled flag.
  - [x] Invalid TTL, negative max children, empty agent name, and unknown provider/model return wrapped validation errors with field paths.
  - [x] Unknown autonomy TOML keys are rejected by the existing strict config loader.
  - [x] Resolver returns bundled/default coordinator identity when no global or workspace override exists.
- Integration tests:
  - [x] `Load(WithWorkspaceRoot(...))` merges autonomy config without clobbering existing providers, hooks, network, memory, or skills sections.
  - [x] Config edits do not mutate process environment or ambient workspace config.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Later tasks can consume coordinator policy through one resolver boundary.
- No coordinator runtime behavior starts from task creation or config loading alone.

## Completion Notes
- Implemented typed `[autonomy.coordinator]` config in `internal/config` with strict TOML overlay support, conservative defaults, provider/model and safety-limit validation, and resolver fallback semantics.
- Added `RuntimeDeps.CoordinatorConfig` in `internal/daemon` as the no-runtime-behavior resolver boundary for later coordinator bootstrap.
- Verification: `go test ./internal/config -run 'Autonomy|Coordinator'`, `go test ./internal/daemon -run 'CoordinatorConfig'`, `go test ./internal/config`, `go test ./internal/daemon`, `go test -cover ./internal/config` (`81.2%`), and `make verify` all passed.
- Generated contracts/public DTOs: not changed; `make codegen` was not required.
- `web/`: not changed.
- `packages/site`: not changed.
