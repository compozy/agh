---
status: pending
title: Extend extension protocol, capabilities, and instance-scoped launch negotiation
type: backend
complexity: high
dependencies:
  - task_02
  - task_03
---

# Task 04: Extend extension protocol, capabilities, and instance-scoped launch negotiation

## Overview

Update the extension runtime contract so channel-capable extensions can negotiate `channels/deliver`, advertise the right capabilities, and start with only the channel-instance metadata and bound secrets they are allowed to see. This task establishes the protocol and launch boundary that later Host API and delivery work will rely on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend the extension protocol and SDK contract so channel-capable extensions can explicitly negotiate the `channels/deliver` service during `initialize`.
2. MUST add channel-related capability and Host API authorization metadata so the daemon can distinguish channel-capable extensions from generic extensions.
3. MUST add instance-scoped launch negotiation that provides the extension only the channel instance metadata and bound secret material attached to that instance, without exposing arbitrary secret reads.
4. SHOULD keep the new protocol surface specific to delivery-oriented channel behavior rather than introducing a generic raw event-push mechanism.
</requirements>

## Subtasks
- [ ] 4.1 Extend the protocol and contract definitions for `channels/deliver` and channel-related Host API methods
- [ ] 4.2 Update capability-checking and manifest/runtime negotiation for channel-capable extensions
- [ ] 4.3 Add instance-scoped launch metadata and bound-secret injection behavior in the extension manager
- [ ] 4.4 Add unit and integration tests for negotiation, capability enforcement, and bound-secret scoping

## Implementation Details

Follow the TechSpec sections "Extension Protocol", "Secret Storage", and "Technical Considerations", along with the accepted ADRs for negotiated delivery and bound secret injection. This task should define the launch and protocol boundary; it should not yet implement inbound message handling or stream projection.

### Relevant Files
- `internal/extension/protocol/host_api.go` — Canonical Host API method registry where new channel methods and service names belong
- `internal/extension/contract/host_api.go` — Shared contract types for generated SDKs and protocol-aware tests
- `internal/extension/capability.go` — Host API method security and capability checks must expand for channel surfaces
- `internal/extension/manager.go` — Initialize negotiation and launch-time environment/runtime injection happen here
- `.compozy/tasks/ext-architecture/_protocol.md` — The normative protocol document should stay aligned with the new negotiated channel service

### Dependent Files
- `internal/extension/host_api.go` — Host API request handling later depends on the new method identifiers and capability checks
- `sdk/examples/secret-guard/extension.toml` — Existing example extension patterns are the closest reference for instance-scoped launch material and manifest wiring
- `internal/extension/manager_integration_test.go` — Existing runtime negotiation coverage should be extended for channel-capable extension startup

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Confirms platform adapters remain extensions even as the daemon owns the substrate
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — Requires explicit delivery-stream negotiation in the extension handshake
- [ADR-008: Bound Secret Injection per Channel Instance](adrs/adr-008.md) — Constrains launch-time secret exposure to bound instance-scoped material

## Deliverables
- Extended protocol and contract definitions for channel-capable extensions
- Capability and launch negotiation support for `channels/deliver` and bound secrets
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for extension negotiation and instance-scoped launch behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Capability checks reject channel Host API methods when the extension lacks the required action or security grant
  - [ ] Protocol negotiation rejects `channels/deliver` requests from extensions that do not declare the required capability surface
  - [ ] Instance-scoped launch material includes only the secrets bound to the selected channel instance
  - [ ] Launch negotiation does not expose arbitrary vault or secret lookup methods to the extension contract
- Integration tests:
  - [ ] A fake channel extension can negotiate `channels/deliver` during initialize and receives the expected runtime metadata
  - [ ] A non-channel extension still starts successfully without channel negotiation or channel-scoped launch bindings
  - [ ] Restarting a channel extension preserves the negotiated channel service surface across launches
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Channel-capable extensions have an explicit negotiated delivery surface in the protocol
- Launch-time secret exposure is limited to the channel instance being served
