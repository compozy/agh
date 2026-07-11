---
name: cy-loop-tasks
description: Drives end-to-end execution of a Compozy techspec by detecting the current phase from state.yaml plus .compozy/tasks/<slug>/, running exactly one phase action per iteration, printing the iteration summary, then continuing immediately into the next iteration until Phase E or a blocker — task execution with atomic checkpoint commits, QA, then consecutive cy-impl-peer-review rounds until SHIP. Use when running a Compozy techspec under the codex-loop plugin in goal mode, when shipping a techspec end-to-end in one session, or when frontend tasks must be orchestrated through herdr worker TUIs via the --frontend parameter. Do not use for one-off tasks without a .compozy/tasks/<slug>/ directory, or for techspec authoring (use cy-create-techspec).
---

# Loop Tasks Driver

Drive a Compozy techspec to completion as a **continue** loop: each
iteration detects the current phase, runs exactly one phase action,
writes memory, updates `state.yaml`, and prints the iteration summary —
then **continues** at detect unless the outcome is a blocker or Phase E.
Filesystem state still resumes cleanly if the session ends mid-loop.

The loop is a five-phase state machine:

| Phase | Action | Executor |
|-------|--------|----------|
| 0 | bootstrap | orchestrator |
| B | one task or slice + checkpoint commit | orchestrator, or herdr frontend worker |
| C | `qa_report`, then `qa_execution` | Fable 5 herdr worker, then orchestrator |
| D | `cy-impl-peer-review` rounds until SHIP + checkpoint commit per round | orchestrator |
| E | done-signature | orchestrator |

Compatible with `~/dev/ai/codex-loop-plugin` goal mode; the plugin itself
is never modified. Prefer in-session **continue** over waiting for a
Stop→restart — restarts are a resume safety net, not the driver.

## Inputs

- `<slug>` — directory name under `.compozy/tasks/`.
- `<goal_text>` — verbatim `[[CODEX_LOOP goal="..."]]` text or the manual
  reason for the run. Captured once at bootstrap into
  `state.yaml.goal_signature`.
- `--frontend <claude|cursor>` — optional. Selects the frontend worker agent
  and activates the herdr frontend lane for the whole loop. Captured once at
  bootstrap into `state.yaml.frontend_agent`; when absent, every task runs
  locally. Syntax examples in `references/goal-header-template.md`.
- A pre-authored `.compozy/tasks/<slug>/_techspec.md`. Without it, bootstrap
  stops with a blocker.

## Helper scripts

Bundled under `.agents/skills/compozy/cy-loop-tasks/scripts/` — stdlib-only
Python 3.11+, no network, no model calls. Invoke by the explicit repo-root
paths shown in the workflow steps.

| Script | Role | Phase |
|--------|------|-------|
| `init-state.py` | bootstrap (mutating once) | 0 |
| `detect-phase.py` | read-only | every iteration |
| `update-state.py` | mutating | every iteration |
| `commit-checkpoint.py` | mutating (git commit) | B, D |
| `test_scripts.py` | read-only self-test | skill maintenance only |

## Herdr delegation lanes

Two lanes dispatch work to herdr worker TUIs. Before any dispatch, read
`references/herdr-delegation.md` in full and activate the
`herdr-orchestration` skill it builds on.

- **Frontend lane (Phase B)** — active only when `state.frontend_agent` is
  set. While active, every frontend task or slice is dispatched to the
  selected worker (`claude` → Claude Code Opus at xhigh effort, `cursor` →
  `cursor-agent --yolo --model grok-4.5`); the orchestrator session stays in
  orchestration mode and never implements frontend work itself.
- **QA-report lane (Phase C)** — always active. `qa_report` is produced by a
  Claude Fable 5 worker (`claude --permission-mode auto --model
  claude-fable-5`), launched direct — never plan-first. The orchestrator runs
  `qa_execution` itself.

## Workflow

Each **iteration** is one detect → phase action → memory → state →
summary cycle. After a completed (non-blocked) summary that is not
Phase E, **continue** at Step 1 in the same turn — do not end the
session between rounds.

**Step 1 — Detect.**

1. Print `pwd` and confirm the working directory is the repo root (the
   directory containing `.compozy/tasks/`); on mismatch, stop and report.
2. Activate `cy-workflow-memory` so its protocol is loaded for later use
   (once per session is enough; re-activate only if context was dropped).
3. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/detect-phase.py <slug>`.
   The printed line decides the whole iteration; the output catalog and
   entry/exit conditions are in `references/phase-transitions.md`.

**Step 2 — Run exactly one phase branch.**

### Phase 0 — Bootstrap

1. Confirm `.compozy/tasks/<slug>/_techspec.md` exists. Missing → scaffold
   `memory/MEMORY.md` from `references/memory-protocol.md`, record the
   blocker under `## Open Risks`, print the iteration summary with
   `outcome=blocked`, and stop (`state.yaml` does not exist yet, so there is
   no update-state call).
2. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/init-state.py <slug> --goal "<goal_text>"`,
   adding `--frontend <claude|cursor>` when the invocation text carries the
   parameter. Mode auto-detects: `tasks` when `_tasks.md` plus at least one
   `task_*.md` exist, else `free`.
3. Activate `cy-spec-preflight` for the phase the next iteration enters
   (`tasks`, or `task-body` when a single concrete file is next).
4. Scaffold `.compozy/tasks/<slug>/memory/MEMORY.md` with the canonical
   sections from `references/memory-protocol.md`.
5. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase 0 --action "bootstrap (mode=<mode>)" --outcome completed --memory-written "memory/MEMORY.md"`.

Done when: `state.yaml` exists and `memory/MEMORY.md` has the canonical
sections.

### Phase B mode=tasks — execute one task

1. Take the task printed by detect-phase (`task=<stem>`). Read
   `.compozy/tasks/<slug>/<stem>.md` and confirm frontmatter `status:` is
   `pending` or `in_progress`. Frontmatter wins — on drift, reconcile with
   `update-state.py <slug> --task-completed <stem>` for already-finished
   tasks or `--reconcile-tasks` for a late-authored graph, then re-run
   detect-phase.
2. Activate `cy-spec-preflight` in `task-body` mode for the picked file.
3. Resolve the shared and current memory paths from
   `references/memory-protocol.md` and pass them into the lane that executes
   the work.
4. **Frontend lane** — when detect-phase printed `lane=frontend agent=<x>`:
   dispatch the task to that worker per `references/herdr-delegation.md` and
   skip step 5. The worker owns implementation, memory updates, validation,
   and `cy-final-verify` evidence; it never commits.
5. **Local lane** — activate `cy-execute-task` on the picked file with
   auto-commit disabled, then `cy-final-verify`; capture its PASS/FAIL
   evidence for the iteration summary. Skip any per-task peer-review step the
   task file requests — peer review is Phase D (see Critical Rules).
6. Confirm memory is updated (written locally, or verified from the worker)
   before any state flip.
7. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase B --task-completed <stem> --action "executed <stem>" --outcome completed --memory-written "memory/<stem>.md,memory/MEMORY.md" --verify-pass`.
8. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/commit-checkpoint.py <slug> --task <stem>`.
   Stdout is a commit SHA or `SKIP: no changes`; copy it into the iteration
   summary. On exit 1, record `--verify-fail --blocker
   "checkpoint-commit-failed: <stderr summary>"` via update-state and stop —
   never retry with `--no-verify`.

Done when: task frontmatter, memory, `state.yaml`, and the checkpoint result
all reflect the same completed task.

### Phase B mode=free — execute one slice

1. Re-read `_techspec.md` deliverables and acceptance in full; compare
   against `state.progress.checklist[]`.
2. Pick the smallest coherent slice (≤ ~4 hours) that advances at least one
   acceptance criterion; capture its text exactly.
3. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --add-progress "<slice text>" --action "slice picked" --outcome completed`.
4. Re-read `state.yaml`; the current memory file is
   `memory/free-iter-<NNN>.md`, `<NNN>` = the new checklist entry's
   `iteration`, zero-padded to three digits.
5. **Frontend lane** — when `state.frontend_agent` is set AND the slice's
   owned paths are exclusively frontend surfaces (classification in
   `references/herdr-delegation.md`): dispatch per that reference and skip
   step 6.
6. **Local lane** — implement the slice, record decisions and learnings in
   the current memory file, then run `cy-final-verify`.
7. Acceptance self-check: when every techspec criterion has a completed
   checklist entry, add `--deliverables-complete` to the step 8 call.
8. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase B --complete-progress "<slice text>" [--deliverables-complete] --action "slice <text>" --outcome completed --memory-written "memory/free-iter-<NNN>.md,memory/MEMORY.md" --verify-pass`.
9. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/commit-checkpoint.py <slug> --slice "<slice text>"`
   with the exact step 3 text — same SKIP / exit-1 semantics as mode=tasks
   step 8.

Done when: the slice's checklist entry is `completed` and the checkpoint
result is recorded.

### Phase C — QA

Run only the printed action.

`qa_report` — dispatched, never authored locally:

1. When release-grade runtime scope needs a lab and no active
   `bootstrap-manifest.json` exists, activate the project's QA bootstrap
   skill first (e.g. `agh-qa-bootstrap` in AGH) when installed.
2. Dispatch the Fable 5 worker per `references/herdr-delegation.md`
   (QA-report lane). The worker activates `qa-report` with
   `qa-docs-path=docs/qa` and updates journey flows, `docs/qa/state.csv`
   scenario rows, and cycle charters.
3. Verify the worker evidence (each reported artifact exists, no worker
   commit), then run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase C --qa-report-done --action "qa-report produced" --outcome completed --memory-written "memory/qa-report.md,memory/MEMORY.md"`.

`qa_execution` — local:

1. Activate `qa-execution` with `qa-docs-path=docs/qa`; it writes the dated
   run report at `docs/qa/reports/<YYYY-MM-DD>-<slug>.md` and updates
   `state.csv` verdicts.
2. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase C --qa-execution-done --action "qa-execution produced" --outcome completed --memory-written "memory/qa-execution.md,memory/MEMORY.md"`,
   adding `--verify-fail` when the report's Final Status is "not ready" or
   any Blocks-Completion/Data-Loss bug is open, else `--verify-pass`.

mode=tasks addition: when the printed QA action corresponds to the pending QA
task file, flip that task's frontmatter `status:` to `completed` and add
`--task-completed <stem>` to the same update-state call so `tasks.pending`
drains.

Done when: the printed QA artifact exists on disk and its flag is recorded in
`state.yaml`.

### Phase D — peer-review rounds until SHIP

One round per iteration; detect-phase re-emits `peer_review` until the
verdict is SHIP on a verify-PASS tree.

1. Activate `cy-impl-peer-review` for the round number printed by
   detect-phase, scoped to the loop's full diff (`--base` = the ref the loop
   started from when known, default `main`), passing the spec's
   contract-bearing artifacts via `--context`.
2. The loop is the deciding authority at that skill's user-decision points:
   remediate **every blocker and every nit** from the round's findings in
   this same iteration, then re-run the project verification gate.
3. Update `memory/peer-review.md` (a `## Round <N>` section per round), then
   run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --phase D --review-round-done <SHIP|FIX_BEFORE_SHIP|REWORK> --action "peer-review round <N> (<verdict>)" --outcome completed --memory-written "memory/peer-review.md,memory/MEMORY.md" --verify-pass|--verify-fail`.
   `--review-round-done SHIP` is valid only with `--verify-pass` — a SHIP
   verdict on a failing tree is void.
4. Run `python3 .agents/skills/compozy/cy-loop-tasks/scripts/commit-checkpoint.py <slug> --review-round <N>`
   — same SKIP / exit-1 semantics as Phase B.

Done when: the round's findings artifact exists, every blocker and nit from
it is remediated (or the verdict was SHIP), and `state.yaml` records the
round.

### Phase E — done

1. Run a final `cy-final-verify` and confirm `state.verify.last_status=PASS`.
   Anything else is a regression: re-enter the phase detect-phase points to
   and skip the done-signature.
2. Walk the Phase E section of `references/checklist.md`; every box must
   pass.
3. Print the iteration summary block from
   `assets/iteration-summary.template.md` with `phase_out=E` and checkpoint
   field `n/a (phase != B/D)`.
4. Print the literal contents of `assets/done-signature.txt` on its own line
   — the codex-loop goal-check confirmation scans for it.
5. Stop — Phase E is the only successful terminal.

**Step 3 — Self-audit, summarize, then continue.**

1. Walk `references/checklist.md` for the phase just executed; every box must
   pass before summarizing.
2. Print the iteration summary block from
   `assets/iteration-summary.template.md` (Phase E already printed it and
   adds only the done-signature line).
3. **Continue gate:** if `outcome=blocked` or `phase_out=E`, stop. Otherwise
   re-enter Step 1 immediately — the summary marks the round; it does not
   end the session.

## Memory protocol

Memory goes through the `cy-workflow-memory` skill — the exact paths per
phase are in `references/memory-protocol.md`. Update memory **before**
flipping any tracking field.

## Goal-mode integration

The canonical `[[CODEX_LOOP ...]]` header, the manual invocation text, and
`--frontend` syntax live in `references/goal-header-template.md`.

## Critical Rules

- One phase action per iteration; after a completed summary, **continue** at
  detect until Phase E or a blocker — never idle between rounds waiting for
  a restart or re-invocation.
- `state.yaml` mutates only through `init-state.py` and `update-state.py`;
  hand-edits void resume guarantees. There is no top-level `current_phase` —
  `detect-phase.py` derives it from durable state and filesystem truth every
  run.
- Frontmatter `status:` on `task_NN.md` is the source of truth; reconcile
  `state.yaml` when they disagree.
- Memory updates precede status flips. Always.
- Frontend lane: `state.frontend_agent` set → herdr dispatch is the only way
  frontend work gets implemented; null → every task runs locally. Workers are
  interactive TUIs launched via `rtk herdr agent start` — a pane streaming
  raw JSON is a broken headless delegation: interrupt and relaunch per
  `references/herdr-delegation.md`.
- `qa_report` is always produced by the Fable 5 worker; `qa_execution` always
  runs locally.
- `cy-impl-peer-review` runs only in Phase D. Per-task peer-review
  instructions inside task files or specs are superseded by this loop's phase
  machine — note "deferred to Phase D" in the task memory and move on.
- Phase D repeats in consecutive rounds until SHIP; every non-SHIP round
  remediates all blockers and nits before the next round starts.
- Checkpoint commits (Phases B and D) belong to the orchestrator:
  `cy-execute-task` runs with auto-commit disabled, and every worker packet
  forbids committing. The checkpoint captures code, memory, task frontmatter,
  the master tasks file, and the advanced `state.yaml` in one atomic,
  restorable snapshot.
- Phase E requires `qa.report_done=true`, `qa.execution_done=true`,
  `review.ship=true`, and `verify.last_status=PASS`.
- Never invoke `cy-create-tasks`, `cy-create-techspec`,
  `cy-tasks-tail-qa-pair`, or `cy-web-docs-impact` from this loop — it
  consumes their output.

## Error Handling

- **`_techspec.md` missing at bootstrap** — record the blocker in
  `memory/MEMORY.md` `## Open Risks`, print the iteration summary with
  `outcome=blocked`, stop. No update-state call: `state.yaml` does not exist
  yet.
- **Mode disagreement** — `init-state.py` exits 4 when `--mode` contradicts
  the filesystem. Reconcile by adding/removing `_tasks.md` before bootstrap,
  or run `update-state.py <slug> --reconcile-tasks` when the task graph was
  authored after a free-mode bootstrap.
- **`state.yaml` parse failure** — `detect-phase.py` exits 1 with the parse
  error on stderr. Hand-editing is the usual cause; restore from `git diff`
  and resume.
- **`cy-final-verify` FAIL** — record `--verify-fail --action "verify FAIL:
  <summary>" --outcome blocked`, print the summary, and stop (continue gate).
  A later invocation re-detects the same phase. Two consecutive verify
  failures in one phase → declare a blocker (two-touch rule).
- **`commit-checkpoint.py` exit 1** — record `--verify-fail --blocker
  "checkpoint-commit-failed: <stderr summary>"`, print the summary, and stop;
  never bypass the hook. `SKIP: no changes` on stdout is success, not failure.
- **Worker launch or delegation failure** — the pane shows raw JSON instead
  of a TUI banner, `rtk herdr agent list` stays `unknown`, or the status wait
  times out with no progress: interrupt
  (`rtk herdr pane send-keys <pane_id> ctrl+c`) and relaunch once via
  `rtk herdr agent start`; a second failure is a blocker.
- **Delegated run lacks PASS evidence or artifacts, or committed anyway** —
  keep the phase open and record blocked/`--verify-fail` naming the missing
  item. A worker commit is a contract breach: report it, do not advance.
- **Invalid peer-review round** (missing or malformed findings artifact) —
  the round does not count; follow `cy-impl-peer-review` error handling and
  re-run it.
- **Two-touch rule** — a third corrective change to the same task or area
  escalates to a blocker instead of a third patch.
- **Blocker recorded** — print the summary and stop without the done-signature;
  a later invocation re-detects it until a human resolves it.
