---
status: completed
title: Transcript, hooks, and extension host support for synthetic turns
type: backend
complexity: high
dependencies:
  - task_04
---

# Task 05: Transcript, hooks, and extension host support for synthetic turns

## Overview

Propagate the new synthetic turn model through the transcript assembler, hook input classes, and extension host assumptions. This task closes the trust-boundary gap by ensuring synthetic runtime activity renders and propagates as daemon-originated work instead of being mistaken for human input in replay, hooks, or extension bridging.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_04.md` before starting
- REFERENCE TECHSPEC sections "Workstream 4: Synthetic Reentry Model", "Hooks and Transcript", and "Workstream 6"
- FOCUS ON "WHAT" - make downstream consumers understand synthetic turns correctly; do not add detached task-run wake-up logic here
- MINIMIZE CODE - extend the canonical transcript/hook/host surfaces instead of introducing a parallel replay or hook bus
- TESTS REQUIRED - transcript rendering, hook input classes, and extension-host turn identification all need coverage
- GREENFIELD: nenhuma superficie pode inferir que todo turn com prompt recente nasceu de `user_message`
</critical>

<requirements>
- MUST render synthetic events distinctly in transcript output rather than as ordinary user messages
- MUST add a dedicated hook input class for synthetic turns
- MUST update extension-host prompt submission logic so `turnID` and seed-event discovery do not depend solely on `user_message`
- MUST preserve existing user and network transcript behavior while adding synthetic support
- SHOULD keep transcript ordering and tool-call pairing behavior stable under mixed user, network, and synthetic turns
</requirements>

## Subtasks
- [x] 5.1 Extend transcript assembly to render synthetic runtime-originated input distinctly
- [x] 5.2 Add a synthetic hook input class and update hook dispatch classification
- [x] 5.3 Update extension-host prompt submission and seed-event discovery for non-user initiating events
- [x] 5.4 Verify mixed-turn ordering and tool/result continuity with synthetic events present
- [x] 5.5 Add targeted regression tests across transcript, hooks, and extension host surfaces

## Implementation Details

See TechSpec "Workstream 4: Synthetic Reentry Model", "Hooks and Transcript", and ADR-003. This task should leave downstream consumers able to answer "who initiated this turn?" correctly even when the daemon reenters a session internally.

### Relevant Files
- `internal/transcript/transcript.go` - canonical replay assembler that must render synthetic events as daemon-originated input
- `internal/session/manager_hooks.go` - current hook input-class mapping must grow a synthetic class
- `internal/session/transcript.go` - session transcript API should continue delegating to the canonical assembler with the new event semantics
- `internal/extension/host_api.go` - current prompt submission logic assumes the first stored `user_message` provides the turn id
- `internal/store/types.go` - event payload and validation changes from task_04 will be consumed here
- `internal/transcript/transcript_test.go` - transcript semantics must be extended and locked down with regression coverage

### Dependent Files
- `internal/session/manager_hooks_test.go` - hook input classes and lifecycle dispatch need synthetic-turn assertions
- `internal/extension/host_api_test.go` - prompt submission/seed-event logic needs non-user initiating coverage
- `internal/api/udsapi/transport_parity_integration_test.go` - transcript and event parity tests may need updates if synthetic turns surface in transport flows
- `internal/api/httpapi/transport_parity_integration_test.go` - transport parity should stay stable under the new event model

### Related ADRs
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - Prompt and hook surfaces must continue to honor the staged runtime seams
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - Synthetic turn support is the downstream contract needed by later reentry work

### External References
- `.resources/openclaw/src/agents/session-tool-result-guard.ts` - useful guard pattern for transcript correctness when synthetic/runtime events repair missing pairs
- `.resources/openclaw/src/agents/session-transcript-repair.ts` - concrete reference for transcript repair around synthetic or missing tool-result state
- `.resources/hermes/hermes_state.py` - useful model for replaying persisted conversation with reasoning/tool-call state intact
- `.resources/hermes/gateway/hooks.py` - good precedent for lifecycle classification without degrading into a generic event bus
- `.resources/openfang/crates/openfang-runtime/src/session_repair.rs` - strong reference for maintaining conversation consistency around inserted internal turns
- `.resources/claude-code/bridge/initReplBridge.ts` - useful example of filtering synthetic/internal messages at a bridge boundary

## Deliverables
- Transcript support for synthetic runtime-originated turns
- Dedicated synthetic hook input class **(REQUIRED)**
- Extension-host turn-id and seed-event logic that no longer relies solely on `user_message` **(REQUIRED)**
- Mixed-turn replay behavior validated under transcript and transport tests **(REQUIRED)**
- Regression coverage for tool-call pairing and ordering under mixed user, network, and synthetic turns **(REQUIRED)**
- Unit and integration tests with >=80% coverage for affected transcript/hook/host files **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Transcript output renders synthetic prompt input as daemon-originated content instead of a normal user role or message
  - [x] Hook dispatch maps synthetic turns to the dedicated synthetic input class without altering existing user or network classifications
  - [x] Extension-host turn-id discovery succeeds when the first persisted event for a turn is synthetic rather than `user_message`
  - [x] Mixed user, network, and synthetic turns preserve stable ordering in the transcript assembler
  - [x] Tool-call and tool-result pairing remains correct when a synthetic turn appears before or after tool activity in the same transcript window
- Integration tests:
  - [x] Session transcript APIs return consistent replay output when synthetic events exist alongside user and network events in the same session
  - [x] Extension prompt submission still produces valid seed events and a valid `turnID` after synthetic-path changes
  - [x] HTTP and UDS transcript consumers observe the same ordering and role semantics for synthetic events
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Transcript, hooks, and extension host can all distinguish daemon-originated turns from human-originated ones
- The runtime is ready for real synthetic reentry without corrupting replay or bridge semantics
