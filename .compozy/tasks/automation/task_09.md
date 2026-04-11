---
status: completed
title: Add extension Host API automation methods and automation hook events
type: backend
complexity: high
dependencies:
  - task_06
---

# Task 09: Add extension Host API automation methods and automation hook events

## Overview

Expose automation to extensions through the Host API and hook runtime described in the TechSpec. This task makes automation observable, manageable, and extensible for extensions without moving the subsystem out of the daemon or duplicating runtime behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add the `automation/*` Host API methods and typed method registry entries needed for extension read, write, and external trigger-fire workflows.
2. MUST extend the capability map so each new automation Host API method requires the correct `automation.read` or `automation.write` capability.
3. MUST emit the automation lifecycle hook events described in the hardened TechSpec, including pre-fire, post-fire, run-completed, and run-failed events.
4. MUST support extension-originated `ext.*` trigger events through `automation/triggers/fire` while reusing the same trigger-engine and dispatcher path as built-in events.
</requirements>

## Subtasks
- [x] 9.1 Add automation Host API request and response types to the extension contract and protocol registries
- [x] 9.2 Extend host API handler dispatch and capability enforcement for automation methods
- [x] 9.3 Add automation lifecycle hook event names and payload emission points
- [x] 9.4 Add extension-fired trigger ingress for `ext.*` events through the existing trigger engine
- [x] 9.5 Add tests for capability checks, Host API behavior, and hook-driven prompt mutation or cancellation

## Implementation Details

Follow the TechSpec sections "Extension Integration", "Host API Methods", "Hook Events", and "Custom Trigger Sources". The implementation should layer on top of the built-in automation manager from task 06 instead of giving extensions a second automation runtime.

### Relevant Files
- `internal/extension/host_api.go` — Host API dispatch and method handlers belong here
- `internal/extension/contract/host_api.go` — Canonical typed Host API spec registry needs additive automation entries
- `internal/extension/protocol/host_api.go` — Wire-level method constants must include the automation methods
- `internal/extension/capability.go` — Capability mapping must enforce `automation.read` and `automation.write`
- `internal/hooks/events.go` — Automation lifecycle event names should be defined with the rest of the hook taxonomy
- `internal/hooks/payloads.go` — Hook payload structures may need additive automation event payload types

### Dependent Files
- `internal/automation/trigger.go` — Extension-fired `ext.*` events must route into the existing trigger engine
- `internal/automation/dispatch.go` — Pre-fire hook mutation or cancellation affects dispatch input

### Related ADRs
- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Directly governs Host API exposure and extension-observable hook points
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires extension-fired events to share the same execution path

## Deliverables
- Additive automation Host API methods, capability wiring, and wire contracts
- Automation lifecycle hook events and extension-fired trigger ingress
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for extension automation management and hook behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Each `automation/*` Host API method maps to the expected `automation.read` or `automation.write` capability requirement
  - [x] Host API method registries in contract and protocol packages include the new automation methods in stable wire order
  - [x] Automation hook event payloads include the required identifiers, prompt fields, and retry metadata for pre-fire and run-failed events
  - [x] `automation/triggers/fire` rejects non-`ext.*` custom event names when the method is intended for extension-provided events
- Integration tests:
  - [x] An extension with `automation.write` can call `automation/jobs/create` and receive the created job payload
  - [x] An extension calling `automation/triggers/fire` with `event = "ext.github.push"` produces matched runs through the normal trigger engine
  - [x] A synchronous `automation.job.pre_fire` hook can mutate the prompt or cancel the fire before dispatch
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Extensions can observe and manage automation through supported Host API and hook surfaces
- Extension-originated trigger events reuse the built-in automation execution path instead of bypassing it
