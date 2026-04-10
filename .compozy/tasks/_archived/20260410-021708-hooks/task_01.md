---
status: completed
title: Core types and hook taxonomy
type: backend
complexity: medium
dependencies: []
---

# Task 1: Core types and hook taxonomy

## Overview

Define the foundational types for the hooks platform in a new `internal/hooks` package: the `HookEvent` enum with sync eligibility classification, `HookSource`, `HookMode`, `RegisteredHook`, `ResolvedHook`, and all event-specific payload/patch type pairs. This is the dependency-free base that every subsequent task builds on.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/hooks/` package with no imports from other `internal/` packages except stdlib
- MUST define `HookEvent` as a typed string enum with compile-time constants for all 27 events in the taxonomy
- MUST define a `SyncEligible` property per event — `message.delta`, `event.pre_record`, `event.post_record`, `permission.resolved`, `permission.denied` are async-only
- MUST define `HookSource` enum: Native, Config, AgentDefinition, Skill
- MUST define `HookMode` enum: Sync, Async
- MUST define `RegisteredHook` struct per TechSpec "Core Interfaces" section
- MUST define all event-specific payload and patch type pairs per TechSpec "Data Models" section
- MUST define `HookRunRecord` with `PatchApplied json.RawMessage` field
- SHOULD define `HookDecl` as the declarative source record for config/agent/skill declarations
</requirements>

## Subtasks
- [x] 1.1 Create `internal/hooks/` package directory and `doc.go`
- [x] 1.2 Define `HookEvent` enum with all 27 events and sync eligibility lookup
- [x] 1.3 Define `HookSource`, `HookMode`, `HookExecutorKind` enums
- [x] 1.4 Define `RegisteredHook`, `ResolvedHook`, `HookDecl` structs
- [x] 1.5 Define event-specific payload/patch type pairs for each family
- [x] 1.6 Define `HookRunRecord` observability struct with patch audit field
- [x] 1.7 Write unit tests for type validation and sync eligibility classification

## Implementation Details

Create new files in `internal/hooks/`:
- `events.go` — HookEvent enum, sync eligibility map, event family constants
- `types.go` — RegisteredHook, ResolvedHook, HookDecl, HookRunRecord, HookSource, HookMode
- `payloads.go` — All event-specific payload and patch types

Reference TechSpec "Hook Taxonomy", "Core Interfaces", and "Data Models" sections for type definitions.

### Relevant Files
- `internal/skills/types.go` — Current HookDecl/HookEvent definitions (lines 55-70) to understand what's being replaced
- `internal/session/interfaces.go` — Session type used in payload types (line 22)
- `internal/acp/types.go` — AgentEvent type referenced in event payloads

### Dependent Files
- `internal/hooks/` — All subsequent hooks package files will import these types

### Related ADRs
- [ADR-002: Use a Dotted Hook Taxonomy with Rich Families](../adrs/adr-002.md) — Defines the event taxonomy
- [ADR-005: Use Typed Per-Event Dispatch Functions](../adrs/adr-005.md) — Requires concrete payload/patch types per event
- [ADR-012: Classify Events into Sync-Eligible and Async-Only](../adrs/adr-012.md) — Defines which events are async-only

## Deliverables
- `internal/hooks/events.go` with complete HookEvent enum and sync eligibility
- `internal/hooks/types.go` with all core structs
- `internal/hooks/payloads.go` with all event-specific payload/patch types
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] All 27 HookEvent constants are defined and have non-empty string values
  - [x] `SyncEligible` returns true for `session.pre_create` and false for `message.delta`
  - [x] `SyncEligible` returns false for all 5 async-only events
  - [x] `SyncEligible` returns true for all sync-eligible events
  - [x] `HookSource` ordering: Native < Config < AgentDefinition < Skill
  - [x] `RegisteredHook` with `Required=true` and `Mode=Async` fails validation
  - [x] `RegisteredHook` with `Mode=Sync` on async-only event fails validation
  - [x] All payload/patch types serialize to JSON and back without data loss
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- No imports from other `internal/` packages (except stdlib)
- All 27 events defined with correct sync eligibility
