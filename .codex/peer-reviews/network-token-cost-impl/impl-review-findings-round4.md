---
schema_version: 1
review_kind: implementation
round: 4
verdict: SHIP
reviewer_runtime: claude
reviewer_model: opus
generated_at: 2026-07-02T04:35:00Z
---

# Summary

Round 4 is a doc-only change that applies round-3 nit N-301 by adding a designated fan-out block to
`skills/agh/references/tasks-and-orchestration.md`, and every claim in it — the `agh task fan-out`
command, `designation_group_id`, `designation.brief`, `agh.runtime` status-back, and the
`designation_rollups` read surface — was verified truthful against the shipped implementation.
The change is broader than N-301's single-line suggestion but correctly fills the megaround packet C5
requirement to document fan-out (which `git diff HEAD` confirms was entirely absent from this file
before this change), so it is a completeness gain, not scope creep; no blockers or risks were found.

# Blockers

None.

# Risks

None.

# Nits

## N-401 — Rollup read-back line does not name the concrete command

- File: skills/agh/references/tasks-and-orchestration.md
- Line: 75
- Issue: The new line says the rollup is readable "from task detail JSON via `designation_rollups`" but, unlike the neighbouring `auto_enqueue_on_ready` note (line 78, which names `agh task inspect <id> -o json`), it never names the command an agent runs; `designation_rollups` is surfaced by `agh task get <id> -o json` (→ `client.GetTask` → `taskDetailPayload`), not by `agh task inspect` (diagnostics payload).
- Suggested fix: Change to "Read aggregated designation results from task detail JSON (`agh task get <id> -o json`, field `designation_rollups`)." mirroring the round-3 N-301 suggestion and the adjacent read-back note.

## N-402 — Worker-facing sentence lives in the Coordinator Loop section

- File: skills/agh/references/tasks-and-orchestration.md
- Line: 72
- Issue: "Workers should follow only their own `designation.brief`." is worker guidance placed in the Coordinator Loop section; the Worker Loop section (lines 82-94) is the canonical home for worker-facing instruction, though co-locating it with the coordinator-invoked fan-out surface is defensible.
- Suggested fix: Optionally move (or cross-reference) the worker sentence into the Worker Loop section; acceptable as-is if the intent is to document the full fan-out flow in one place.

# Evidence

Read in full (first-hand): the round-4 diff (`impl-review-diff-round4.patch`); the changed file
`skills/agh/references/tasks-and-orchestration.md` in full; round-3 findings
(`impl-review-findings-round3.md`); the megaround packet
(`.compozy/tasks/network-token-optimization/codex-megaround-packet.txt`); the megaround plan
(`.codex/plans/20260701T200023-0300*network-token-optimization-megaround.md`); root `CLAUDE.md` and
`internal/CLAUDE.md`.

Git state (first-hand): `git diff --stat HEAD -- skills/agh/references/tasks-and-orchestration.md`
reports `1 file changed, 10 insertions(+)`, and the last commit touching this file is
`a71b6a7ff feat: dependency-driven auto-enqueue`. The 10-line fan-out block (working-tree lines
68-77) is therefore net-new in round 4; no fan-out documentation existed in this file before it. This
means the round-3 N-301 premise ("the skill/docs only describe writing fan-out") was slightly off for
this specific file, and round 4 correctly documents both the write path and the read-back, closing the
megaround C5 documentation requirement that had been missing through the round-3 SHIP.

Truthfulness verification (first-hand source reads — every documented surface maps to real code):

- `agh task fan-out <task-id> --designation "..." --designation "..." -o json`: command defined at
  `internal/cli/task.go:1938-1972`, `Use: "fan-out <task-id>"`, alias `fanout`, `--designation` is a
  repeatable `StringArrayVar` marked required (`mustMarkFlagRequired`), and output flows through
  `writeCommandOutput` (supports `-o json`). ✓
- Shared `designation_group_id`: real field `Run.DesignationGroupID`
  (`internal/task/types.go:316,443,544,830`), stamped per sibling in `internal/api/core/tasks.go:1220-1263`
  and `internal/daemon/native_tools.go:3003-3020`. ✓
- Worker reads its own `designation.brief`: run metadata JSON shape is
  `{"designation": {"index": N, "brief": "..."}}` (`internal/task/designation.go:31-42`), rendered into
  the per-run prompt overlay at `internal/daemon/task_role_runtime.go:614`. The backticked path
  `designation.brief` is accurate. ✓
- Status-back by `agh.runtime`: `RuntimePeerID = "agh.runtime"` (`internal/network/manager.go:29`),
  emitted via `SendFromRuntimePeer` (`manager.go:713-779`) from the status observer
  (`internal/daemon/network_task_status_observer.go:385`). The observer only sends for tasks with a
  network thread origin (`originForTask`, observer:162-169) and, for designation-group runs, only on a
  terminal run status (`IsTerminalRunStatus`, observer:150-155) — matching the doc's "If the task came
  from an AGH Network thread, terminal run state is summarized back." ✓
- `designation_rollups` read surface: `TaskDetailPayload.DesignationRollups []TaskDesignationRollupPayload
\`json:"designation_rollups,omitempty"\`` (`internal/api/contract/tasks.go:222`); each payload carries
an aggregated `Summary json.RawMessage` (`contract/tasks.go:206-212`), populated in
`internal/api/core/tasks.go:267-274`and surfaced by`agh task get <id>` (`internal/cli/task.go:418-433`).
So "aggregated designation results from task detail JSON via `designation_rollups`" is truthful. ✓

AGH Impact Audit:

- Native tools: no impact — doc-only edit to a skill reference; no `agh__*` tool ID, descriptor, schema,
  digest, capability gate, or diagnostic changed. The documented `agh task fan-out` CLI, `agh.runtime`
  status path, and `designation_rollups` field already shipped and received SHIP in round 3.
- Extensibility and hooks: official AGH skill reference (`skills/agh/references/tasks-and-orchestration.md`)
  updated; no extension manifest, hook taxonomy/dispatch, config lifecycle, bundle, registry, or bridge
  SDK touched.
- Workspace data isolation: no impact — no new datum, store query, API route, SSE, cache, or event path;
  the documented rollup read is task-scoped and reachable only after `manager.GetTask` authorizes the
  task view (re-confirmed round-3 assessment). No cross-workspace surface added.
- Official AGH skill: updated — this is the change (fan-out ladder, designation briefs, aggregation via
  `designation_rollups`, and `agh.runtime` status-back register are now documented in the coordinator
  guidance).

Scoped-write-contract conflict (reported per contract §0, not written): repo `CLAUDE.md` mandates a
per-session Memory Ledger under `.claude/ledger/` and updates to the auto-memory `MEMORY.md` index. This
review's scoped-write contract permits writing only this findings file, so no ledger/memory/plan artifact
was created; this is the required conflict report.

Limitations: I did not run `make verify`, `make codegen-check`, or any test suite (read-only review of the
working tree; the prompt states `make verify` passed after this doc change). Findings are static analysis
plus first-hand source reads; every truthfulness claim affecting the verdict was verified by direct source
read of the cited files/lines.

# Deferred Or Follow-Up

- N-401 / N-402: optional doc-clarity polish (name the `agh task get <id> -o json` read command; consider
  relocating the worker sentence to the Worker Loop section). Non-blocking.
- Carried-forward from round 3 (out of this doc-only diff's scope, still open under the round-3 SHIP):
  R-301 (add `task_designation_rollups(task_id, created_at DESC)` index), R-302 (make the task-detail
  rollup enrichment best-effort), R-303 (atomic fan-out enqueue + orphan test), R-304 (transactional
  rollup completion-transition detector), R-305 (two positive-path delivery tests).
