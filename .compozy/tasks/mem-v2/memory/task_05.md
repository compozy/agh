# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Delivered the Memory v2 Slice 1 write controller and durable `memory_decisions` WAL seam.
- Runtime HTTP core memory writes/deletes and extension Host API memory store/forget now propose controller decisions instead of mutating curated files directly.
- Low-level `Store.Write` and `Store.Delete` remain storage-local primitives for tests, replay, and internal raw mutation; public runtime paths use `ProposeWrite` / `ProposeDelete`.

## Important Decisions

- `internal/memory/controller` owns deterministic ADD/UPDATE/DELETE/NOOP/REJECT decisions using lexical scan results, exact content, entity/attribute slot matching, filename collision, and conservative ambiguity NOOP fallback.
- Scanner rejections are persisted as WAL decisions and emit `memory.write.rejected`; callers adapt `OpReject` into transport-specific validation errors without bypassing the audit row.
- `Store.ApplyDecision` inserts the WAL row before any file/catalog mutation. Failed mutations keep `applied_at` empty so replay can reconcile pending decisions.
- Decision idempotency keys include candidate hash, op, targets, target filename, post hash, frontmatter hash, and prompt version.
- Revert uses persisted `prior_content` for UPDATE/DELETE and a current-content hash guard for ADD so rollback cannot silently delete changed files.
- Scanner rejection telemetry stores redaction-safe rule traces plus sample byte counts in `RuleHit.Details`; raw rejected material is not copied into audit reasons.

## Learnings

- `make verify` first failed on `funlen` for `Controller.Decide`; splitting preparation/rejection/write helpers fixed the shape without changing rule order.
- Focused coverage briefly exposed `internal/memory` at 79.8%; adding destructive-revert and invalid delete validation coverage raised it to 80.1%.
- A full `make verify` run hit one non-reproducing macOS `TempDir RemoveAll` cleanup failure in `TestHostAPIHandlerSessionsEventsSupportsSinceFilter`; isolated `go test -race ./internal/extension -count=1` passed, and the next full `make verify` passed.

## Files / Surfaces

- `internal/memory/controller/`: deterministic decision controller and controller tests.
- `internal/memory/decision.go`: proposal, apply, revert, target listing, decision events, and WAL persistence.
- `internal/memory/store.go`: raw mutation extraction and storage seam preservation.
- `internal/memory/replay.go`: pending decision replay continues through raw storage to avoid future controller recursion.
- `internal/api/core/memory.go`: HTTP core write/delete reroute through controller proposals.
- `internal/extension/host_api.go`: extension Host API memory store/forget reroute through controller proposals.
- `internal/memory/store_memv2_test.go` and `internal/extension/host_api_test.go`: WAL/controller/runtime coverage updates.

## Errors / Corrections

- Corrected `Controller.Decide` lint shape instead of suppressing `funlen`.
- Adjusted an extension memory fixture away from repo-path-like content because the task 04 scanner intentionally rejects direct repository path material.

## Ready for Next Run

- Task 06 can treat `memory_decisions.id`, `TargetFilename`, scope, workspace ID, agent identity, and decision events as stable provenance inputs for deterministic recall and shadow rules.
- Task 07 should revisit scanner live-gate thresholds using the persisted rejection telemetry before promoting provider-facing writes broadly.
