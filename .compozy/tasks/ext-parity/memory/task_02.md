# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Build the generic typed resource boundary in `internal/resources` on top of the task 01 raw kernel.
- Scope includes typed codecs, registry/store adapters, projector adapter seams, and required unit/integration tests.
- Scope excludes family migration beyond representative codec/adapter coverage and excludes boot wiring or reconcile-driver implementation.

## Important Decisions

- Treat the approved PRD + TechSpec + ADR set as the design artifact for this run; no separate design approval loop is needed.
- Keep decode, validate, encode, and max-bytes enforcement at codec boundaries; typed store/projector surfaces must not accept raw JSON payloads.
- Preserve task 01 raw-kernel semantics and layer typed façades over the existing `Kernel` instead of reworking authority/CAS behavior.
- Model `bundle.activation` as the only mixed-kind projector adapter outlier in this task; avoid a generic dependency-bag abstraction.
- Use an opaque `ProjectorRegistration` seam so raw reconcile input and dependency bags stay inside `internal/resources` even though later boot wiring can register projectors from other packages.
- Implement validation on typed writes by encoding and then round-tripping through `DecodeAndValidate`, keeping one canonical validator boundary despite the `KindCodec` interface exposing only `DecodeAndValidate` and `Encode`.

## Learnings

- `internal/resources` currently exposes only raw-kernel contracts and tests; there is no typed store/projector surface yet.
- Task 01 already established the precedent that the PRD/TechSpec counts as the approved design artifact for execution tasks in this workflow.
- Representative domain types already expose validation entry points in `internal/automation/model`, `internal/bridges`, and `internal/bundles/model`, which can anchor codec behavior without migrating full families yet.
- The package-level contract tests now cover typed store authority, codec round-trips, explicit mixed-kind dependency decoding for `bundle.activation`, and AST checks that validator/projector contracts do not expose `json.RawMessage`.
- `internal/resources` package coverage is now above the task threshold with the new typed-boundary tests.
- `make verify` passed after the typed-boundary changes; the only post-implementation correction needed was replacing a generic registry method with the package-level `RegisterCodec` helper because Go forbids generic methods on non-generic types.

## Files / Surfaces

- `internal/resources/`
- `internal/hooks/types.go`
- `internal/automation/model/types.go`
- `internal/bridges/types.go`
- `internal/bundles/model/model.go`
- `.codex/ledger/2026-04-15-MEMORY-typed-resource-boundary.md`
- `.compozy/tasks/extensibility-parity/memory/MEMORY.md`

## Errors / Corrections

- Go does not allow generic methods on a non-generic type, so `CodecRegistry.Register[T]` was corrected into the package-level helper `RegisterCodec[T](registry, codec)`.

## Ready for Next Run

- Task 02 is implemented, verified, and tracked as completed. Only the local code-only completion commit remains for this run.
