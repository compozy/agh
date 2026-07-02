---
schema_version: 1
review_kind: implementation
round: 3
verdict: SHIP
reviewer_runtime: claude
reviewer_model: opus
generated_at: 2026-07-02T03:52:00Z
---

# Summary

Round 3 applies exactly the two round-2 nits (N-201 digest single-envelope flush-skip and N-202 task-detail
`designation_rollups` read surface) across 8 files, and both are implemented correctly — the flush wait is
gated on a real same-peer head candidate so single digests deliver immediately without rendering as a batch,
and the rollup reader is task-scoped, error-wrapped, codegen co-shipped (openapi + web typings + sdk contracts),
and behaviorally tested in the canonical suites. No new blocker was introduced and no prior behavior regressed;
the residual items are low-severity risks (a new unindexed `task_id` scan on the hot task-detail path, secondary-
read failure coupling) plus the three non-blocking risks already carried forward under the round-2 SHIP verdict.

# Blockers

None.

# Risks

## R-301 — N-202 read enrichment runs an unindexed `task_id` scan on the hot task-detail path

- File: internal/store/globaldb/global_db.go
- Line: 202
- Issue: `task_designation_rollups` is defined with `designation_group_id` as PRIMARY KEY and `task_id` as an
  FK column with no covering index (global_db.go:202-207; the only new index in v47 is on
  `task_runs(task_id, designation_group_id)`, not on the rollups table). Round 3 wires
  `taskDetailPayload` (internal/api/core/tasks.go:262-276) to call `ListTaskDesignationRollups` filtered by
  `task_id` on every `GetTask`, `AddTaskDependency`, and `RemoveTaskDependency` — the read path issued by
  `global_db_network_preferences.go:359-366` is `... WHERE task_id = ? ORDER BY created_at DESC ...`, a full
  table scan since SQLite does not auto-index FK columns. Before round 3 the rollups table was only read by
  its PK (`previousDesignationRollupComplete` dedup), so this is a newly-introduced unindexed scan on a core
  read. The table is tiny today (one row per fan-out group), so the cost is negligible at alpha scale, but it
  is a latent perf smell on a hot endpoint.
- Suggested fix: Add `CREATE INDEX IF NOT EXISTS idx_task_designation_rollups_task ON task_designation_rollups(task_id, created_at DESC);`
  in both the fresh-schema statement set and an appended migration, matching the existing rollup ordering.

## R-302 — Task-detail GET now 500s when the secondary rollup read fails

- File: internal/api/core/tasks.go
- Line: 271
- Issue: `taskDetailPayload` returns a zeroed payload and a wrapped error whenever `ListTaskDesignationRollups`
  errors (tasks.go:271-273), and every caller (`GetTask` 254-258, `AddTaskDependency` 731-735,
  `RemoveTaskDependency` 774-778) turns that into `h.respondError(c, StatusForTaskError(err), err)` — a 500,
  since a generic store error carries no task-domain status mapping. The primary task view was already
  retrieved successfully at that point, so a transient failure in a secondary, almost-always-empty enrichment
  now fails the core task-detail read (and both dependency-mutation responses) rather than degrading to a
  detail payload without rollups. Low probability (a rollup read failing generally implies the global DB is
  already unhealthy) but it couples a primary read's availability to an enrichment.
- Suggested fix: Treat the rollup enrichment as best-effort — on error, log with `slog` (task_id) and return
  the base `TaskDetailPayloadFromView` payload without `DesignationRollups`, rather than failing the whole
  request.

## R-303 — Partial fan-out failure still leaves orphaned sibling runs (carried from round-2 R-201, unchanged)

- File: internal/api/core/tasks.go
- Line: 1245
- Issue: `enqueueFanOutTaskRuns` (tasks.go:1245-1272) still loops `manager.EnqueueRun` once per designation and
  returns on the first error, leaving already-enqueued siblings persisted as `queued` runs while the handler
  returns 500 and `PutTaskDesignationRollup` is skipped (tasks.go:1210-1228). Round 3 did not touch this path,
  so the round-2 risk persists verbatim: a failure on designation ≥2 strands a live run under an abandoned
  group id, and a client retry mints a fresh `groupID`. Non-blocking (accepted under the round-2 SHIP verdict),
  but relevant before autonomous fan-out runs at scale.
- Suggested fix: Enqueue the designation group atomically (single tx spanning all `ReserveQueuedRun` calls) or
  roll back already-enqueued siblings on partial failure; add a test asserting no orphan `queued` rows remain
  after a mid-group enqueue failure.

## R-304 — Rollup-completion status-back double-send race (carried from round-2 R-202, unchanged)

- File: internal/daemon/network_task_status_observer.go
- Line: 245
- Issue: `recomputeAndSendDesignationRollup` still gates the completion message on a non-atomic
  read-compute-persist-check-send around `previousDesignationRollupComplete` (observer.go:245-266); two sibling
  runs terminating concurrently in different sessions can both read `wasComplete=false` and both send the
  `task_fanout_rollup` message. Round 3 did not touch the observer. Cosmetic (duplicate thread post), not data
  corruption, and low-probability given the single-worker happy path; carried non-blocking.
- Suggested fix: Detect the incomplete→complete transition inside a single `BEGIN IMMEDIATE` tx that returns
  whether this call flipped `Complete`, and send only on that signal; or serialize rollup handling per group id.

## R-305 — Positive capability_match and positive keyword-filter deliveries remain untested (carried from round-2 R-203, unchanged)

- File: internal/network/router_test.go
- Line: 436
- Issue: The delivery suites still cover only the negative paths — `router_test.go:436` asserts digest fallback
  when capability match scores zero, and `manager_test.go:1914` asserts digest fallback when a keyword filter
  misses — with no subtest asserting (a) a capability-matched peer is activated as `full` when its catalog
  overlaps the intent terms, or (b) a `full`-mode subscription with a matching keyword filter delivers `full`.
  Both are the happy paths of features whose failure modes (over-/under-delivery) directly affect the round's
  token-cost goal. Round 3 added no delivery tests beyond the N-201 single-digest case.
- Suggested fix: Add the two positive-path subtests to the canonical `router_test`/`manager_test` suites.

# Nits

## N-301 — New rollup read surface is undocumented for agents

- File: skills/agh/references/tasks-and-orchestration.md
- Line: 73
- Issue: N-202 makes the aggregated designation rollup observable via task detail (`designation_rollups`), but
  the skill/docs only describe writing fan-out and status-back — nothing tells agents they can now read the
  rollup back (e.g. from `agh task get <id> -o json`), leaving the round-2 N-202 "make it observable" intent
  half-documented.
- Suggested fix: Add one line noting the aggregated result is readable from task detail via
  `designation_rollups` (mirroring the existing `auto_enqueue_on_ready` read-back note at line 77).

# Evidence

Read in full (first-hand): round-1 and round-2 findings; the megaround packet
(`.compozy/tasks/network-token-optimization/codex-megaround-packet.txt`); the plan
(`.codex/plans/20260701T200023-0300*…megaround.md`); root `CLAUDE.md` and `internal/CLAUDE.md`.

Round-3 delta isolation (first-hand): diffed `impl-review-diff-round2.patch` vs `impl-review-diff-round3.patch`
by blob hash — round 3 changed exactly 8 files: `internal/network/delivery.go`,
`internal/network/delivery_test.go`, `internal/api/contract/tasks.go`, `internal/api/core/tasks.go`,
`internal/api/core/tasks_test.go`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`,
`sdk/typescript/src/generated/contracts.ts`. This matches the stated N-201 + N-202 scope with no stray edits.

N-201 verification (first-hand): `collectDigestBatch` (delivery.go:688-706) now guards the flush wait with
`c.digestFlushInterval > 0 && c.hasDigestBatchCandidate(sessionID, first)`; `hasDigestBatchCandidate`
(delivery.go:926-934 → inboundQueue.hasDigestBatchCandidate 1199-1209) peeks the post-dequeue head and returns
true only for a same-peer digest item, using the same head-consecutive predicate as `dequeueDigestBatch`
(1177-1197), so `false` implies `dequeueBatch → nil → batch=[first]` with no lost coalescing. Lock discipline
matches the existing `dequeue`/`dequeueDigestBatch` pattern (acquire `c.mu`, release, then `q.mu`); single
per-session worker means no concurrent dequeue during the wait; token mismatch on `dropSession` short-circuits
safely. Test `delivery_test.go` "Should skip flush wait when one digest envelope is queued" uses a 1-hour flush
interval and asserts <500ms latency plus no `<network-digest-batch` wrapper — a real behavioral gate.

N-202 verification (first-hand): `TaskDesignationRollupPayload` contract (contract/tasks.go:206-212) and
`DesignationRollups` field (222); `taskDetailPayload` (tasks.go:262-276) reads task-scoped rollups (limit 20)
with `%w` wrapping and no `_` discards; `TaskDesignationRollupPayloadsFromStore` (tasks.go:2540+) clones summary
via `cloneRawMessage` (network.go:861-866). All three `TaskDetailResponse` producers route through
`taskDetailPayload` (consistent). Store read `ListTaskDesignationRollups` (global_db_network_preferences.go:344-395)
is parameterized, `ORDER BY created_at DESC, designation_group_id ASC` (not `ORDER BY 0`), positive-limit
validated, rows closed with `errors.Join`. Interface wired via `store.NetworkPreferenceStore` (store.go:155-156)
→ `api/core.NetworkStore`; stub `ListTaskDesignationRollupsFn` added (network_stub.go:405-411). Codegen co-ship
confirmed: `designation_rollups` present in `openapi/agh.json` (3), `web/src/generated/agh-openapi.d.ts` (3), and
`sdk/typescript/src/generated/contracts.ts` (`TaskDesignationRollupPayload` + field). Migration integrity:
`task_designation_rollups` table created both in fresh-schema (global_db.go:716) and the appended v47 migration
`migrateTaskRunDesignations` (global_db.go:1248-1268) — no reorder/rename. `TaskDetailPayloadFromView(nil)` is
nil-safe (tasks.go:2520). Happy-path test (`tasks_test.go`) adds a post-fan-out GET asserting one rollup with the
stored group id/task id/summary and bumps `getTaskCalls` 3→4; task ids are uniformly `task-1`, so the stub
assertion does not false-fatal.

Workspace isolation (N-202): rollups are task-scoped (no `workspace_id` column); the read is only reachable after
`manager.GetTask(ctx, taskID, actor)` authorizes the task view, and `task_id` is globally unique, so no
cross-workspace leak is introduced — consistent with the round-2 assessment.

Carried-forward risk state re-verified first-hand: R-201 (tasks.go:1245-1272) unchanged; R-202
(observer.go:245-266) unchanged; R-203 (router_test.go / manager_test.go) still negative-path only.

Scoped-write-contract conflict (reported per contract §0, not written): repo `CLAUDE.md` mandates a per-session
Memory Ledger under `.claude/ledger/` and, for plan-mode acceptance, a plan file under `.claude/plans/`. This
read-only review's scoped-write contract permits writing only this findings file, so no ledger/plan artifact was
created; this is the required conflict report.

Limitations: I did not run `make verify`, `make codegen`, `make codegen-check`, or the Go/Bun test suites
(read-only review of the working tree at commit `669f6b503 savepoint`). Findings are static analysis plus
first-hand source reads; every N-201/N-202 claim affecting the verdict was verified by direct source read.

# Deferred Or Follow-Up

- R-301: add a `task_designation_rollups(task_id, created_at DESC)` index (fresh schema + appended migration)
  before fan-out volume grows.
- R-302: make the task-detail rollup enrichment best-effort (log + omit) instead of failing the request.
- R-303 / R-304 / R-305: the three round-2 risks remain open by design under the SHIP verdict — close R-303
  (atomic fan-out + orphan test), R-304 (transactional completion-transition detector), and R-305 (two
  positive-path delivery tests) before autonomous fan-out and activation run at scale.
- N-301: document the `designation_rollups` read-back path in the AGH skill reference.
- Pre-existing (carried from rounds 1–2, out of this diff's fault): `networkChannelEntry` lookup at
  `global_db_task_claim.go` filters by `channel` only (not `workspace_id`) — close when next touched.
