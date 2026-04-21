---
status: completed
title: Harness context resolver and turn-origin foundations
type: backend
complexity: high
dependencies: []
---

# Task 01: Harness context resolver and turn-origin foundations

## Overview

Introduce the daemon-owned foundation that resolves harness behavior from durable session context plus per-turn origin and runtime signals. This task creates the authoritative runtime model that all later prompt, reentry, and detached-work tasks will consume, replacing the earlier idea of a foundational `HarnessProfile` enum.

<critical>
- ALWAYS READ `_techspec.md` and ADRs before starting (`_prd.md` is absent; requirements come from the TechSpec)
- REFERENCE TECHSPEC sections "Core Model", "Workstream 1: Harness Context Resolution", and "Key Decisions"
- FOCUS ON "WHAT" - centralize context resolution and policy derivation; do not start layering prompt sections or reentry behavior here
- MINIMIZE CODE - one resolver path owned by `internal/daemon`, not scattered policy checks across session/runtime packages
- TESTS REQUIRED - matrix coverage for session type, turn origin, and derived policy is mandatory
- GREENFIELD: nao reintroduzir um enum de profile como fonte da verdade; qualquer label de profile deve ser derivado
</critical>

<requirements>
- MUST add a daemon-owned resolver that derives harness policy from session type, channel presence, turn origin, and runtime metadata
- MUST keep durable session context and per-turn origin as separate axes in the model
- MUST define and validate support for a synthetic turn origin without yet implementing full synthetic event persistence
- MUST encode the resolution matrix in tests so policy decisions are deterministic and reviewable
- SHOULD keep the first implementation internal-only with no new user-facing config knobs
</requirements>

## Subtasks
- [x] 1.1 Introduce the core runtime types for resolved harness context and policy
- [x] 1.2 Add the daemon-owned resolver entrypoint and keep call ownership out of `internal/session`
- [x] 1.3 Thread the resolved policy into the startup and prompt pipeline seams without changing user-facing config
- [x] 1.4 Define the valid matrix for session type, channel state, and turn origin combinations
- [x] 1.5 Add unit and integration coverage for deterministic resolution behavior

## Implementation Details

See TechSpec "Workstream 1: Harness Context Resolution" and ADR-001. The main outcome is one place in the daemon that can answer "what harness behavior applies to this session and this turn?" without relying on prompt-specific heuristics or duplicated policy checks later in the stack.

### Relevant Files
- `internal/session/session.go` - durable session type semantics already live here and should remain the source for the session-side axis
- `internal/session/interfaces.go` - current `TurnSource` definition is the natural seam for origin expansion
- `internal/session/manager_prompt.go` - prompt dispatch already normalizes turn source and is the first consumer of the new policy
- `internal/daemon/boot.go` - boot composition is where daemon-owned policy wiring begins
- `internal/daemon/harness_context.go` - new home for the resolver and the resolved-policy model introduced by this task

### Dependent Files
- `internal/session/manager_start.go` - startup policy consumers will depend on the resolved context introduced here
- `internal/daemon/composed_assembler.go` - later section selection will consume the resolved policy
- `internal/daemon/daemon.go` - later augmenter composition and runtime wiring will consume the resolver output
- `internal/daemon/task_runtime.go` - detached harness work will depend on the same session/turn policy vocabulary
- `internal/session/manager_test.go` - prompt-path tests will need to assert the new policy semantics

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - Defines the core architectural direction for this task

### External References
- `.resources/claude-code/utils/systemPrompt.ts` - shows a real layered policy order where runtime mode is derived before prompt composition
- `.resources/openclaw/src/agents/bootstrap-mode.ts` - demonstrates explicit mode resolution from turn origin plus workspace/bootstrap eligibility
- `.resources/hermes/gateway/session.py` - good reference for session-context prompt derivation and reset reasoning
- `.resources/openfang/crates/openfang-types/src/agent.rs` - useful precedent for separating schedule/runtime identity metadata from later policy application

## Deliverables
- Daemon-owned harness context resolver and resolved-policy types
- Synthetic turn origin added to the internal model vocabulary without exposing new public config **(REQUIRED)**
- Deterministic resolution matrix coverage for session and turn combinations **(REQUIRED)**
- No remaining foundational dependency on a top-level `HarnessProfile` enum **(REQUIRED)**
- Table-driven resolver tests covering session type, channel presence, and turn origin permutations **(REQUIRED)**
- Integration coverage proving one shared resolver result is consumed consistently by startup and prompt paths **(REQUIRED)**
- Unit and integration tests with >=80% coverage for the new resolver paths **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `SessionTypeUser` plus `TurnOriginUser` resolves the baseline policy with no network or synthetic overlays enabled
  - [x] `SessionTypeUser` plus bound channel plus `TurnOriginNetwork` resolves network-aware sections and augmenters without mutating the durable session axis
  - [x] `SessionTypeSystem` plus `TurnOriginSynthetic` resolves a distinct synthetic-capable policy only when internal runtime metadata is present
  - [x] Empty, unknown, or mismatched turn-origin inputs fail validation with descriptive errors instead of silently defaulting
  - [x] Derived diagnostic labels or profile-like tags remain stable for identical input tuples across repeated resolver calls
- Integration tests:
  - [x] Session startup and `PromptWithOpts` consume the same resolved policy for one session without diverging section or augmenter decisions
  - [x] `PromptNetwork` drives the resolver through the network path and produces the expected network-aware runtime behavior end to end
  - [x] Resolver output remains stable across session resume or boot-time metadata reconstruction for the same session/turn tuple
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Harness behavior is resolved from session context plus turn origin rather than a foundational profile enum
- Later prompt and detached-runtime tasks can consume one authoritative daemon-owned policy model
