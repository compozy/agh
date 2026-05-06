---
status: completed
title: Config and Settings Backend
type: backend
complexity: high
dependencies:
  - task_07
  - task_11
---

# Task 13: Config and Settings Backend

## Overview

Extend the configuration and settings backend so Memory v2 can be operated through stable config/default/validation surfaces. This task owns the server-side lifecycle of new memory keys and settings payloads, including merge semantics, tool-surface mutability, and the operator-facing settings model consumed later by web and docs tasks.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Config Lifecycle`, `Assumptions / Defaults`, and `Development Sequencing` step 23.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside config/default/validation/settings seams.
- TESTS REQUIRED: defaults, merge/overlay, validation, tool-surface mutability, and settings payload round-trips must ship here.
- NO WORKAROUNDS: new memory config cannot exist only in docs or UI; structs, defaults, validation, settings payloads, and tests move together.
</critical>

<requirements>
- MUST add every approved Slice 1 memory config key, default, and validation rule to backend config structs.
- MUST expose the same settings state through backend settings payloads and mutation parsing.
- MUST update agent-facing config tool-surface rules for new mutable memory paths.
- MUST keep provider, recall, dreaming, and memory-scope settings aligned with the runtime semantics established by earlier tasks.
- MUST add config-focused tests for defaults, overlay behavior, invalid values, and restart-safe serialization.
</requirements>

## Subtasks
- [x] 13.1 Extend backend config structs, defaults, and validation with all approved Slice 1 memory keys.
- [x] 13.2 Update backend settings payloads and mutation parsing for the expanded memory model.
- [x] 13.3 Extend config tool-surface mutability rules for new memory paths.
- [x] 13.4 Add focused tests for defaults, merges, validation, and settings payload parity.

## Implementation Details

See TechSpec `Config Lifecycle`, `Assumptions / Defaults`, and `Development Sequencing` step 23. This task is backend-only: it owns the truth that later CLI/web/docs tasks consume, not the final public UI presentation.

### Relevant Files
- `internal/config/config.go` — backend config structs and defaults for memory settings.
- `internal/config/tool_surface.go` — agent-facing mutable path policy for config writes.
- `internal/config/config_test.go` — defaults, validation, and merge coverage.
- `internal/api/core/settings.go` — settings mutation parsing and response shaping.
- `internal/api/contract/settings.go` — backend settings payloads for memory configuration.
- `internal/cli/config.go` — config CLI path coverage that later CLI/docs tasks depend on.

### Dependent Files
- `web/src/systems/settings/adapters/settings-api.ts` — later web settings task depends on final backend payloads.
- `web/src/systems/settings/types.ts` — generated or mirrored UI types depend on this shape.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` — later docs task depends on final keys/defaults.
- `.compozy/tasks/mem-v2/task_14.md` — public contract task depends on stable settings payloads.
- `.compozy/tasks/mem-v2/task_21.md` — web settings task depends on final backend configuration semantics.

### Related ADRs
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — provider selection/config surface implications.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — controller-mode and write-policy settings.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — recall/dream defaults and knobs.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: expose provider-related config and settings state in a way later extension/provider surfaces can reflect without private wiring.
- Agent manageability: this task prepares the settings/config truth that later CLI/HTTP/UDS/web surfaces must expose consistently.
- Config lifecycle: this task is the lifecycle owner for new memory keys, defaults, validation, merge/overlay behavior, and backend settings payloads.

### Web/Docs Impact

- `web/`: backend settings payload changes will later affect `web/src/systems/settings/**`, but no frontend code is modified here.
- `packages/site`: memory/configuration docs and references must be updated later from these finalized keys/defaults; no docs change happens in this task.

## Deliverables

- Expanded backend memory config structs, defaults, and validation.
- Updated backend settings payloads and mutation parsing for Memory v2.
- Updated config tool-surface policy for mutable memory paths.
- Focused config/settings tests for defaults, validation, and parity.

## Tests

- Unit tests:
  - [x] New memory config keys load with the approved defaults.
  - [x] Invalid memory config values fail with deterministic validation errors.
  - [x] Tool-surface path classification correctly allows or denies new memory config paths.
- Integration tests:
  - [x] Settings read/update payloads round-trip cleanly through backend parsing and serialization.
  - [x] Merge/overlay behavior preserves memory config semantics across root, workspace, and runtime layers.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/hermes_cli/memory_setup.py`
- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/codex/codex-rs/memories/write/src/runtime.rs`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 config is fully represented in backend structs, defaults, validation, and settings payloads.
- Later CLI/web/docs tasks can consume this backend truth without inventing parallel config models.
