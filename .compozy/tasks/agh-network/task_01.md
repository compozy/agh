---
status: completed
title: Network protocol core
type: backend
complexity: high
dependencies: []
---

# Task 01: Network protocol core

## Overview

Create the foundational `internal/network` protocol package that models AGH Network envelopes, kind-specific bodies, validation, interaction lifecycle state, and route token derivation. This task establishes the protocol boundary that every later transport, router, and delivery feature will depend on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create a new `internal/network` package that owns RFC v0 envelope types and kind-specific bodies without importing `daemon`, `api`, or `cli`
- MUST support all seven message kinds defined by the tech spec, including per-kind required field validation and lifecycle-related body parsing
- MUST enforce protocol validation rules for `space`, `peer_id`, `interaction_id`, targeted versus broadcast `to`, expiration, and reason-code usage
- MUST provide deterministic route token derivation and reusable lifecycle helpers that later router code can call directly
- MUST preserve opaque `ext` payloads while leaving AGH workflow and handoff metadata optional and non-normative
</requirements>

## Subtasks
- [x] 1.1 Create protocol types for the shared envelope and all kind-specific bodies under `internal/network`
- [x] 1.2 Implement validation and normalization helpers for field rules, expiration handling, and targeted versus broadcast semantics
- [x] 1.3 Model interaction lifecycle ownership and terminal-state decisions in package-local helpers
- [x] 1.4 Add unit tests for parsing, validation, lifecycle transitions, and route token derivation

## Implementation Details

This task should stay entirely inside the protocol layer. Follow the implementation sections of the tech spec for field rules, ordering, and reason-code semantics instead of re-explaining the protocol here.
It should model and round-trip `ext` cleanly, but it must not embed coordinator-mode workflow policy, circuit-breaker state, or saga/compensation logic.

### Relevant Files
- `.compozy/tasks/agh-network/_techspec.md` - Normative implementation design, data model, and receiver rules
- `docs/rfcs/003_agh-network-v0.md` - RFC v0 protocol requirements for envelope and processing semantics
- `docs/rfcs/004_agh-network-v1.md` - RFC v1 notes that the v0 implementation must remain compatible with where specified
- `internal/network/envelope.go` - New protocol envelope and kind-body definitions
- `internal/network/validate.go` - New validation and normalization surface
- `internal/network/lifecycle.go` - New interaction lifecycle helpers and ownership rules

### Dependent Files
- `internal/network/transport.go` - Transport layer will publish and consume these protocol types
- `internal/network/router.go` - Router logic will depend on validation and lifecycle helpers
- `internal/network/delivery.go` - Delivery formatting will consume normalized envelopes
- `internal/network/manager.go` - Orchestrator will expose protocol operations through runtime interfaces

### Related ADRs
- [ADR-002: Session-as-Peer Identity Model](adrs/adr-002.md) - Establishes the peer identity shape the protocol must validate
- [ADR-005: Runtime-Created Spaces with Explicit Session Opt-In](adrs/adr-005.md) - Defines how spaces behave in v0 and what the protocol can assume

## Deliverables
- New `internal/network` protocol files for envelopes, kinds, validation, and lifecycle helpers
- Stable exported APIs for later transport and router tasks
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for protocol parsing and normalization boundaries **(REQUIRED)**

## Tests
- Unit tests:
- [ ] Valid envelopes for each supported kind normalize correctly
- [ ] Invalid `space`, `peer_id`, `to`, or `interaction_id` fields are rejected with descriptive errors
- [ ] Lifecycle helpers reject invalid ownership and invalid terminal-state regressions
- [ ] Route token derivation matches known vectors and remains deterministic
- [ ] `ext` metadata round-trips without dropping unknown keys or making AGH workflow/handoff keys mandatory
- Integration tests:
- [ ] End-to-end protocol fixtures from the tech spec can be parsed and re-serialized without semantic drift
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/network` exists as the protocol owner with no forbidden imports
- Later transport and router tasks can depend on protocol helpers without redefining envelope rules
