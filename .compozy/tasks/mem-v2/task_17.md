---
status: pending
title: CLI Memory Hard Cut
type: backend
complexity: high
dependencies:
  - task_14
  - task_16
---

# Task 17: CLI Memory Hard Cut

## Overview

Replace the current CLI memory surface with the approved Slice 1 verbs, flags, scope/tier semantics, and output contracts. This task is the operator and agent-facing command-line hard cut that must stay in lockstep with the repaired TechSpec and the finalized daemon routes.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `API Endpoints`, `Agent Manageability Plan`, `Greenfield Delete Targets`, and `Development Sequencing` step 27.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside CLI command/client/output seams.
- TESTS REQUIRED: verb renames, structured output, scope/tier flags, deterministic errors, and command-path coverage must ship here.
- NO WORKAROUNDS: do not keep deprecated CLI verbs, aliases, or output shapes alive after the hard cut.
</critical>

<requirements>
- MUST replace legacy memory CLI verbs and flags with the approved Slice 1 surface.
- MUST keep CLI behavior aligned with the shared public contract and daemon transport semantics.
- MUST support structured output modes and deterministic error contracts for agent operation.
- MUST update client calls and command help/examples consistently with the hard cut.
- MUST add or update CLI tests for new verbs, removed verbs, flags, and output contracts.
</requirements>

## Subtasks
- [ ] 17.1 Rewrite the memory CLI command tree to the approved Slice 1 verb set and flag model.
- [ ] 17.2 Update CLI client calls and output shapers for the new memory payloads.
- [ ] 17.3 Remove deprecated/legacy memory command names and help text.
- [ ] 17.4 Add focused tests for command paths, structured outputs, and deterministic failure behavior.

## Implementation Details

See TechSpec `CLI verbs`, `Agent Manageability Plan`, `Greenfield Delete Targets`, and `Development Sequencing` step 27. This task should leave the CLI as the truthful agent/operator surface for memory operations; generated docs are updated later in `task_24`.

### Relevant Files
- `internal/cli/memory.go` — current memory CLI tree and output models to hard-cut.
- `internal/cli/client.go` — shared CLI client helpers used by memory commands.
- `internal/cli/memory_test.go` — focused command coverage for memory operations.
- `internal/cli/command_paths_test.go` — command-path guardrails for renamed or removed verbs.
- `internal/cli/api_state_verb_coverage_test.go` — coverage guardrails between public state verbs and CLI surfaces.
- `internal/cli/doc_test.go` — command documentation integrity checks affected by the hard cut.

### Dependent Files
- `packages/site/content/runtime/cli-reference/memory/**` — later generated CLI docs depend on the new command tree.
- `internal/tools/builtin/memory.go` — later native-tool surface should align to the same naming model where applicable.
- `.compozy/tasks/mem-v2/task_24.md` — CLI reference regeneration depends on the final command tree.
- `.compozy/tasks/mem-v2/task_23.md` — runtime docs task depends on final CLI semantics.

### Related ADRs
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — mutation CLI semantics.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — search/trace CLI semantics.
- [ADR-012: Slice 1 Fat Scope — Single TechSpec with Four Eixos](adrs/adr-012.md) — hard-cut rationale for shipping the full surface now.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — checked surfaces are extension manifests, host API, and provider SDKs; they are handled in separate tasks.
- Agent manageability: this task is the CLI owner for the final machine-readable memory operations surface.
- Config lifecycle: command help/examples and config-set guidance must reflect the backend memory config truth from `task_13`.

### Web/Docs Impact

- `web/`: none — checked surfaces are web memory/settings/session pages; they consume API/codegen, not CLI.
- `packages/site`: generated CLI memory docs and runtime guides must be updated later from this command tree.

## Deliverables

- Final Slice 1 memory CLI command tree and output contracts.
- Updated CLI client calls and help/examples.
- Focused tests for renamed/removed verbs, structured outputs, and deterministic errors.

## Tests

- Unit tests:
  - [ ] New memory verbs and flags parse correctly, including scope/tier selectors where applicable.
  - [ ] Removed or renamed legacy verbs fail or disappear according to the hard cut.
  - [ ] Structured CLI outputs match the finalized public payloads.
- Integration tests:
  - [ ] CLI commands succeed against the daemon-backed routes and surface deterministic errors for invalid inputs.
  - [ ] Command-path coverage tests prove no hidden legacy alias survives.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/hermes_cli/memory_setup.py`
- `.resources/hermes/tools/memory_tool.py`
- `.resources/codex/codex-cli`
- `.resources/claude-code/cli`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- The CLI exposes the final Slice 1 memory surface with no deprecated behavior left behind.
- Agent/operator automation can rely on structured CLI output and deterministic errors.

