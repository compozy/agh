---
status: completed
title: Ordered prompt augmentation composite over the current manager seam
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 03: Ordered prompt augmentation composite over the current manager seam

## Overview

Introduce ordered prompt augmentation on top of the current single `PromptInputAugmenter` seam without forcing an immediate `session.Manager` API break. This task turns the existing one-slot augmenter hook into a daemon-composed pipeline with explicit ordering, aggregate budget handling, and clear critical-vs-noncritical failure behavior.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, `task_01.md`, and the current memory augmenter before starting
- REFERENCE TECHSPEC sections "Workstream 3: Turn Augmentation Pipeline" and "Important Constraint"
- FOCUS ON "WHAT" - add deterministic augmenter composition over the existing seam; do not redesign the entire prompt dispatch API here
- MINIMIZE CODE - compose in the daemon first and preserve the current manager injection point until a later phase proves otherwise
- TESTS REQUIRED - ordering, aggregate budgets, failure policy, and unchanged stored-input semantics must all be covered
- GREENFIELD: nao esconder falhas do pipeline; o comportamento de `critical` vs `warn-and-continue` precisa ficar explicito
</critical>

<requirements>
- MUST keep the current manager-facing augmenter seam working while introducing ordered composition behind it
- MUST support deterministic augmenter ordering and aggregate budget behavior
- MUST preserve "store original input, dispatch augmented input" semantics for user and network turns
- MUST distinguish fail-fast augmenters from warning-only augmenters
- SHOULD make the memory augmenter one participant in the composite rather than a special one-off path
</requirements>

## Subtasks
- [x] 3.1 Introduce a daemon-side composite augmenter that adapts to the current manager seam
- [x] 3.2 Define augmenter ordering, aggregate budget handling, and criticality semantics
- [x] 3.3 Move the current memory recall augmenter behind the composite path
- [x] 3.4 Preserve stored-input versus dispatched-input behavior for real user and network turns
- [x] 3.5 Add focused coverage for ordering, budget, and failure behavior

## Implementation Details

See TechSpec "Workstream 3: Turn Augmentation Pipeline" and ADR-002. This task should leave AGH with a usable pipeline architecture while keeping the current injection point stable enough for the rest of the runtime to evolve incrementally.

### Relevant Files
- `internal/session/interfaces.go` - defines the current `PromptInputAugmenter` seam that must remain compatible in phase one
- `internal/session/manager.go` - owns the augmenter injection point from the daemon
- `internal/session/manager_prompt.go` - prompt dispatch path that currently stores one message and dispatches one possibly augmented message
- `internal/daemon/daemon.go` - daemon wiring is the correct place to inject a composite instead of a single augmenter
- `internal/memory/recall.go` - current memory augmenter implementation that should become one composite participant
- `internal/daemon/prompt_input_composite.go` - new daemon-side composite introduced by this task

### Dependent Files
- `internal/session/manager_test.go` - prompt-path tests need to assert ordered augmentation while stored input stays canonical
- `internal/memory/recall_test.go` - memory-specific coverage may need to validate composite participation
- `internal/daemon/daemon_test.go` - wiring tests should assert the daemon installs a composite augmenter path
- `internal/session/manager_hooks_test.go` - hook-observable prompt behavior may need updates if augmentation errors are now surfaced differently

### Related ADRs
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - This task is the concrete phase-one implementation of the staged augmenter strategy

### External References
- `.resources/openclaw/src/agents/system-prompt-contribution.ts` - useful precedent for stable/dynamic prompt contributions without collapsing everything into one builder
- `.resources/openclaw/test/helpers/agents/prompt-composition-scenarios.ts` - strong reference for scenario-based prompt composition regression coverage
- `.resources/claude-code/utils/systemPrompt.ts` - shows explicit layered ordering and addenda precedence in a production harness
- `.resources/hermes/gateway/run.py` - useful runtime analogy for composing ephemeral context before dispatch while keeping one session-facing seam
- `.resources/openfang/crates/openfang-runtime/src/tool_policy.rs` - useful model for explicit policy layering and deny-wins behavior around runtime composition

## Deliverables
- Daemon-side composite augmenter that still satisfies the current `PromptInputAugmenter` injection contract
- Explicit augmenter order, budget, and criticality semantics **(REQUIRED)**
- Memory recall participating through the composite path **(REQUIRED)**
- Stored-input versus dispatched-input invariants preserved for user and network turns **(REQUIRED)**
- Regression coverage for blank augmenter output, noncritical failure continuation, and fail-fast critical behavior **(REQUIRED)**
- Unit and integration tests with >=80% coverage for the new composite and affected prompt path files **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Composite augmenters run in deterministic order even when registration order differs from execution priority
  - [x] Aggregate budget logic trims or rejects later augmenter output once the declared budget ceiling is reached
  - [x] A critical augmenter failure aborts dispatch before driver submission and returns a visible error path
  - [x] A noncritical augmenter failure is logged and the next augmenter still executes with the last valid message state
  - [x] Memory recall participates through the composite path and does not bypass ordering or budget rules
  - [x] Blank or whitespace-only augmenter output never clobbers an already valid dispatch message
- Integration tests:
  - [x] Stored user input remains canonical in the session DB while the driver receives the fully augmented dispatch text
  - [x] `PromptNetwork` continues to preserve stored-input invariants while still dispatching an augmented network-originated message
  - [x] Daemon wiring installs the composite path so a session with memory recall plus one additional augmenter produces stable end-to-end dispatch behavior
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- AGH has ordered turn augmentation without a breaking manager API redesign
- Later runtime augmenters can be added through one composite path instead of ad hoc hooks
