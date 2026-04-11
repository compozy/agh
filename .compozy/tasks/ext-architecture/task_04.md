---
status: completed
title: Capability checker and source-trust tiers
type: backend
complexity: medium
dependencies:
  - task_03
---

# Task 04: Capability checker and source-trust tiers

## Overview

Implement the capability-scoped security model that enforces per-extension grants at both the hook dispatch boundary and the Host API boundary. The `CapabilityChecker` maps extensions to their declared capabilities and applies source-trust tier policy (bundled, user, workspace, marketplace). This is the core of ADR-003 and prevents a compromised extension from exceeding its declared privileges.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/extension/capability.go` with `CapabilityChecker` struct
- MUST define `ExtensionSource` enum with values: `SourceBundled`, `SourceUser`, `SourceWorkspace`, `SourceMarketplace` (matching existing `skills.SkillSource` pattern)
- MUST implement source-trust tier policy per ADR-003:
  - bundled, user, workspace: all capabilities granted by default (`*`)
  - marketplace: restricted (no `permission.*`, no `session.write`, no `memory.write` without explicit allowlist)
- MUST implement `Check(extName, capability string) error` returning typed `ErrCapabilityDenied` with the method, required grants, and granted grants in the error data
- MUST implement `CheckHostAPI(extName, method string) error` enforcing `granted_actions` (method-level) AND `granted_security` (family-level) per protocol spec section 5.2
- MUST implement `Register(extName string, source ExtensionSource, manifest *Manifest)` that computes effective grants from manifest + source-tier policy
- MUST apply source-tier policy BEFORE consulting manifest requests (tier acts as a ceiling)
- MUST return all security grant denials via `-32001 capability_denied` equivalent Go error
</requirements>

## Subtasks
- [x] 4.1 Define `ExtensionSource` enum with documented trust tier semantics
- [x] 4.2 Implement `CapabilityChecker` struct with register, check, and check Host API methods
- [x] 4.3 Implement source-trust tier policy with default grants per tier
- [x] 4.4 Implement typed `ErrCapabilityDenied` error with structured data
- [x] 4.5 Write table-driven tests for all source-tier combinations

## Implementation Details

New file `internal/extension/capability.go` and `internal/extension/capability_test.go`. Extends the package created in task 03.

See TechSpec "Capability Checker" section and ADR-003 for the security model. See `_protocol.md` sections 4.5 and 5.2 for the dual-layer enforcement model (granted_actions AND granted_security).

The `ExtensionSource` enum should mirror the existing `skills.SkillSource` pattern for consistency.

### Relevant Files
- `internal/extension/manifest.go` — Provides `Manifest.Security.Capabilities` and `Manifest.Actions.Requires` (task 03)
- `internal/skills/types.go` — Existing `SkillSource` enum to mirror for `ExtensionSource`
- `internal/hooks/permission.go` — Existing permission enforcement guard pattern to follow

### Dependent Files
- `internal/extension/manager.go` — Will invoke `Register()` for each extension at load time (task 06)
- `internal/extension/host_api.go` — Will invoke `CheckHostAPI()` for every method call (task 07)

### Related ADRs
- [ADR-003: Capability-Scoped Security Model](adrs/adr-003.md) — This task implements the normative decision

## Deliverables
- New `internal/extension/capability.go` with `CapabilityChecker`, `ExtensionSource`, `ErrCapabilityDenied`
- Source-trust tier default policy map
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `Check()` succeeds when extension has granted capability
  - [x] `Check()` returns `ErrCapabilityDenied` when extension lacks capability
  - [x] `CheckHostAPI()` succeeds when both granted_actions and granted_security are satisfied
  - [x] `CheckHostAPI()` fails when granted_actions missing even if granted_security satisfies
  - [x] `CheckHostAPI()` fails when granted_security missing even if granted_actions satisfies
  - [x] Bundled source grants all capabilities by default
  - [x] User source grants all capabilities by default
  - [x] Workspace source grants all capabilities by default
  - [x] Marketplace source denies `permission.*` without explicit allowlist
  - [x] Marketplace source denies `session.write` without explicit allowlist
  - [x] Marketplace source denies `memory.write` without explicit allowlist
  - [x] Marketplace source allows `session.read`, `memory.read`, `observe.read` by default
  - [x] `Register()` applies source-tier ceiling to manifest requests
  - [x] `ErrCapabilityDenied` includes method, required, and granted fields in data
  - [x] Wildcard grant `["*"]` authorizes all capabilities in that family
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All source-tier combinations covered in tests
