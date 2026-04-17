---
status: completed
title: "Add extension surface registry and resource grant config"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 04: Add extension surface registry and resource grant config

## Overview

Create the static policy table and operator-config surface that make resource publication server-authoritative for extensions. This task centralizes kind legality, scope ceilings, manifest requests, and operator allowlists so later handshake and snapshot flows all compute grants from one place.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a static `internal/extension/surfaces` registry that declares which resource kinds are publishable by extensions, which scopes are legal for each kind, and which kinds remain daemon-only.
2. MUST add `[extensions.resources]` operator configuration under `internal/config.ExtensionsConfig` for resource allowlists, scope ceilings, and publication rate-limit policy described in the TechSpec "Authority and Validation Rules" section.
3. MUST make `CapabilityChecker` compute effective grants as the intersection of surface legality, source-tier ceilings, operator config, manifest request, and session-mode narrowing.
4. MUST keep grant computation daemon-derived rather than manifest-self-asserted so later handshakes and reads use the same authoritative source of truth.
</requirements>

## Subtasks

- [x] 4.1 Create the static extension surface registry for first-wave resource kinds
- [x] 4.2 Add `[extensions.resources]` config parsing, merge behavior, and validation
- [x] 4.3 Update capability computation to derive granted kinds and scopes from the shared policy chain
- [x] 4.4 Add coverage for manifest requests, source-tier ceilings, and operator policy precedence

## Implementation Details

Follow the TechSpec sections "Data Models", "Authority and Validation Rules", "Integration Points", and "Technical Considerations". This task should stop at static policy and configuration; it should not yet modify the subprocess handshake or Host API payloads.

### Relevant Files

- `internal/extension/surfaces/` — New static table for resource kind legality, scope rules, and future consumers
- `internal/extension/manifest.go` — Manifest requests need first-class resource-family coverage aligned to the surface table
- `internal/extension/capability.go` — Grant computation must intersect policy, source tier, manifest request, and session mode
- `internal/config/config.go` — Add `[extensions.resources]` to the daemon config model
- `internal/config/merge.go` — Config overlay semantics must preserve explicit precedence for resource policy

### Dependent Files

- `internal/subprocess/handshake.go` — Later handshake payloads depend on the computed grant set from this task
- `internal/extension/manager.go` — Extension startup later depends on one authoritative grant computation path
- `sdk/typescript/src/types.ts` — SDK initialize metadata later depends on the new grant fields

### Related ADRs

- [ADR-001: Adopt a Shared Resource Runtime as the Authoritative Extensibility Control Plane](adrs/adr-001.md) — Requires one consistent authority model across manifest, runtime, and config
- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Defines server-computed grants, scope ceilings, and read restrictions
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Keeps policy and grants explicit rather than hidden in untyped payloads

## Deliverables

- A new `internal/extension/surfaces` policy table for first-wave resource kinds
- `[extensions.resources]` config parsing, validation, and merge behavior
- Updated capability computation that derives resource grants from daemon-side policy instead of self-declared payloads
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for manifest, config, and grant-derivation behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] a manifest request for an illegal kind or illegal scope is rejected by the surface registry before handshake time
  - [x] operator config narrowing removes kinds or scopes that the manifest requested but policy does not allow
  - [x] source-tier ceilings prevent lower-trust extensions from receiving grants above their maximum allowed scope
  - [x] config merge keeps explicit precedence for surface legality, source-tier ceilings, operator config, manifest request, and session mode
- Integration tests:
  - [x] a workspace-scoped extension cannot obtain global publication scope even if its manifest requests it
  - [x] the effective grant set exposed to extension startup is derived from daemon policy rather than manifest self-assertion
  - [x] rate-limit policy under `[extensions.resources]` round-trips through config load and merge paths
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The daemon computes one authoritative grant set for extension resource publication and reads
- Manifest, config, and capability policy stop drifting across separate extension call sites

## Verification Evidence

- `go test ./internal/extension/surfaces ./internal/extension ./internal/config`
- `go test -cover ./internal/extension/surfaces ./internal/extension ./internal/config`
  - `internal/extension/surfaces`: `83.6%`
  - `internal/extension`: `80.0%`
  - `internal/config`: `83.0%`
- `go test -tags integration ./internal/extension -run 'TestManagerIntegration(WorkspaceExtensionCannotReceiveGlobalResourceScope|ResourceGrantsComeFromDaemonPolicy)$'`
- `make verify`
