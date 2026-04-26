---
status: pending
title: Situation Surface Providers
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 04: Situation Surface Providers

## Overview
Build the bounded runtime context that agents need to act without shell snippets or hidden assumptions. This task renders self identity, workspace/session facts, active task context, inbox summary, peer roster, capabilities, limits, and provenance through existing prompt provider/augmenter seams and the `/agent/context` contract.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-002, and ADR-009 before implementing situation providers
- REFERENCE TECHSPEC for section order and bounding rules
- FOCUS ON "WHAT" - stable context facts and prompt injection, not task claiming or spawning
- MINIMIZE CODE - reuse `session.PromptProvider` and `session.PromptInputAugmenter`
- TESTS REQUIRED - render ordering, truncation, absent data, and provenance must be covered
- NO WORKAROUNDS - do not hard-code CLI snippets as a substitute for first-class context APIs
</critical>

<requirements>
- MUST render `/agent/context` sections in stable order: `self`, `workspace`, `session`, `task`, `inbox_summary`, `peer_roster`, `capabilities`, `limits`, `provenance`.
- MUST bound every list section and include truncation metadata.
- MUST use existing session prompt assembly/augmenter seams and daemon composition root wiring.
- MUST omit unavailable sections cleanly without inventing placeholder facts.
- MUST include `agent_name` and `session_id` provenance in rendered context, hooks/logs where relevant, and task-run payloads if available.
- MUST not implement broad post-MVP memory extraction, eval/replay, or cross-daemon network evolution.
</requirements>

## Subtasks
- [ ] 4.1 Define situation provider interfaces and section renderers in the daemon/session boundary.
- [ ] 4.2 Render self/workspace/session/capability/limit sections from existing runtime state.
- [ ] 4.3 Render task/inbox/peer sections from task and network services when available.
- [ ] 4.4 Wire prompt startup and bounded dynamic update injection through existing prompt seams.
- [ ] 4.5 Add tests for stable ordering, truncation metadata, missing services, provenance, and prompt assembly.
- [ ] 4.6 Update contracts or web generated artifacts if `/agent/context` DTO shape changes from task_02.

## Implementation Details
Keep the first implementation local-daemon focused. Reference external projects only for context shaping ideas; do not port their abstractions or create a new memory/plugin stack.

### Relevant Files
- `internal/session/interfaces.go` - `PromptInputAugmenter` and prompt assembly interfaces.
- `internal/session/manager_helpers.go` - startup prompt assembly.
- `internal/daemon/harness_context*.go` - existing prompt/context augmentation precedent.
- `internal/network/*` - peer/channel facts for bounded roster and inbox summaries.
- `internal/task/live.go` and `internal/task/live_types.go` - task/run view facts.
- `internal/skills/*` - capability and skill catalog facts.
- `.resources/hermes/agent/context_engine.py` - reference for context section selection.
- `.resources/hermes/agent/context_references.py` - reference for provenance-bearing context references.
- `.resources/hermes/agent/prompt_builder.py` - reference for assembling bounded prompt context.
- `.resources/claude-code/context.ts` - reference for local context assembly.

### Dependent Files
- `internal/api/udsapi/*` and `internal/api/httpapi/*` - task_06 exposes context endpoints.
- `internal/cli/*` - task_06 exposes `agh me context`.
- `packages/site/content/runtime/core/autonomy/*` - task_16 documents section semantics.

### Related ADRs
- [ADR-001: Phased Autonomy Kernel Scope](adrs/adr-001.md) - local MVP context boundaries.
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - context backs CLI-first agent controls.
- [ADR-008: Memory Provenance Before Rich Memory Scopes](adrs/adr-008.md) - provenance before broad memory scope.
- [ADR-009: Autonomy Hooks and Extension Points Are First-Class Contracts](adrs/adr-009.md) - provider extensibility.

## Deliverables
- Situation provider and renderer stack wired through daemon/session seams.
- Stable `/agent/context` payload assembly behind a service boundary.
- Prompt startup/dynamic context tests.
- Unit tests with 80%+ coverage for renderers **(REQUIRED)**.
- Integration tests proving prompt assembly receives situation context without provider-specific hacks **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] Rendering preserves the required section order and omits unavailable sections.
  - [ ] Peer roster, inbox, capabilities, and task lists truncate deterministically and report truncation metadata.
  - [ ] `agent_name`, `session_id`, workspace, provider/model, and limits appear when present.
  - [ ] Missing network/task/skill services do not panic and do not fabricate data.
  - [ ] Provider output is stable for snapshot-style assertions without current-time nondeterminism.
- Integration tests:
  - [ ] A created session receives startup situation context through `PromptProvider` or prompt assembly.
  - [ ] A prompt submitted after task assignment includes the active task envelope without duplicating previous context.
  - [ ] Existing harness context tests continue to pass with the new provider stack.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agents can inspect a compact runtime situation without reading daemon internals.
- Context remains bounded, deterministic, and extensible.
