---
status: completed
title: "Add typed codecs, stores, and projector adapters"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Add typed codecs, stores, and projector adapters

## Overview

Build the typed boundary that keeps raw bytes inside the persistence kernel while exposing concrete Go types to domain code. This task is where the runtime stops leaking untyped payloads and gains stable contracts for validators, stores, and single-kind projectors.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add `KindCodec[T]`, typed `Draft[T]`, typed `Record[T]`, and typed `Store[T]` façades as described in the TechSpec "Core Interfaces" section.
2. MUST keep `MutationActor` on typed `Get` and `List` so read filtering remains enforced at the typed boundary rather than only in raw transport code.
3. MUST add thin adapter seams that let single-kind domain projectors consume typed records while keeping `ProjectionInput` and raw dependency bags internal to `internal/resources`, with `bundle.activation` treated as the only planned mixed-kind outlier in this pack.
4. MUST ensure decode and validate happen at the codec boundary only, without reintroducing `json.RawMessage` into validators or projector call sites.
</requirements>

## Subtasks

- [x] 2.1 Add typed draft and record façades over the raw resource kernel
- [x] 2.2 Add a codec registry pattern for kind-specific decode, encode, and max-bytes enforcement
- [x] 2.3 Add typed projector adapter seams for single-kind projectors and the explicit `bundle.activation` mixed-kind adapter escape hatch
- [x] 2.4 Add contract tests proving typed store reads and projections do not leak raw JSON

## Implementation Details

Follow the TechSpec sections "Core Interfaces", "Authority and Validation Rules", and "Technical Considerations". This task should build the generic typed boundary only; it should not migrate concrete families to resources yet beyond representative adapter and codec coverage. Treat `bundle.activation` as the only concrete mixed-kind adapter target for now and avoid inventing a broader generic dependency-bag abstraction until a second real outlier exists.

### Relevant Files

- `internal/resources/` — Add the typed store façade, codec contracts, and projector adapter scaffolding
- `internal/hooks/types.go` — Representative hook-binding spec types that later adopt codec-backed records
- `internal/automation/model/types.go` — Representative automation types that later need typed resource records
- `internal/bridges/types.go` — Representative bridge desired-state types that later need typed projector adapters

### Dependent Files

- `internal/daemon/boot.go` — Later boot wiring depends on registering codecs and projector adapters explicitly
- `internal/extension/manager.go` — Extension publication later depends on typed encode and decode boundaries for migrated families
- `internal/bundles/service.go` — Mixed-kind bundle work later depends on the explicit projector adapter escape hatch

### Related ADRs

- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Keeps read filtering enforced at the typed store edge
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Defines the typed façade pattern this task must implement

## Deliverables

- Typed draft, record, store, and codec contracts in `internal/resources`
- Adapter scaffolding for single-kind typed projectors and the explicit `bundle.activation` mixed-kind projector escape hatch
- Contract tests proving codecs own decode and encode responsibility without raw JSON leakage
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for typed store behavior over the raw SQLite-backed kernel **(REQUIRED)**

## Tests

- Unit tests:
  - [x] a typed store `Get` and `List` path still rejects records outside the actor source, granted kinds, or scope boundary
  - [x] a codec decode failure rejects invalid raw payloads before typed records are returned to domain code
  - [x] encode and decode round-trip through the same codec without losing version, scope, owner, or source metadata
  - [x] a typed projector adapter decodes its primary kind once per reconcile input and does not expose raw bytes to the domain projector
- Integration tests:
  - [x] a fake codec plus SQLite-backed typed store can persist, load, and list typed records using the raw kernel from task 01
  - [x] a mixed-kind adapter can decode dependency bags explicitly without forcing `TypedProjector[T]` to accept heterogeneous raw payloads
  - [x] contract tests fail if a validator or projector tries to depend on `json.RawMessage` directly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Domain-facing store and projector contracts are typed end to end
- Raw bytes remain confined to the persistence and transport boundary rather than leaking into domain packages
