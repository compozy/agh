# Phase Transitions

`.agents/skills/cy-codex-loop/scripts/detect-phase.py` (read-only) is the single source of truth for
"what phase am I in right now?". The script reads `state.yaml` and the
filesystem under `.compozy/tasks/<slug>/`, then prints exactly one of:

```
phase=0 action=bootstrap
phase=B action=execute_task task=task_07            # mode=tasks
phase=B action=execute_free_slice                    # mode=free
phase=C action=qa_report                             # or qa_execution
phase=E action=done
```

The agent runs the action for the printed phase, then `.agents/skills/cy-codex-loop/scripts/update-state.py`
records the iteration. The next agent restart re-runs `.agents/skills/cy-codex-loop/scripts/detect-phase.py`
and resumes from wherever the filesystem now indicates.

## Phase 0 — Bootstrap

**Entry**: `state.yaml` does not exist.

**Action**:
1. Confirm `_techspec.md` exists. If not → stop with a blocker (`techspec_missing`); write the blocker to `memory/MEMORY.md` `## Open Risks`.
2. Decide `mode`:
   - If `_tasks.md` AND at least one `task_*.md` exist → `mode=tasks`.
   - Otherwise → `mode=free`.
3. Run `.agents/skills/cy-codex-loop/scripts/init-state.py <slug> --mode <mode> --goal "<goal_text>"`.
4. Scaffold `memory/MEMORY.md` with the canonical sections (see `memory-protocol.md`).
5. `.agents/skills/cy-codex-loop/scripts/update-state.py` records iteration 1 with `phase=0 outcome=completed`.

**Exit**: state.yaml now exists. The next `.agents/skills/cy-codex-loop/scripts/detect-phase.py` run enters `B`.

## Phase B — Execution (mode=tasks)

**Entry**: `mode=tasks` AND `tasks.pending` is non-empty AND the head of `tasks.pending` is **not** a QA task (qa-report / qa-execution).

**Action**:
1. Pick the head of `tasks.pending`. Confirm `task_NN.md` frontmatter `status: pending` or `status: in_progress` (frontmatter wins; if it disagrees with state.yaml, trust frontmatter and reconcile state).
2. Activate `cy-spec-preflight` in phase=task-body for the picked file.
3. The frontend/docs delegation lane is opt-in. It is active only when `state.goal_signature` contains the literal token `delegation=frontend-docs` (case-insensitive). When inactive, skip the delegation decision entirely. When active, read `references/frontend-docs-delegation.md` and apply its classification to decide whether this task qualifies (`type:` is `frontend`/`docs`, or owned paths are exclusively frontend/docs surfaces).
4. Pass the shared/current memory paths from `memory-protocol.md` into the lane that will execute the work.
5. If the delegation lane applies (active AND task qualifies): write the temp prompt and run `compozy exec --ide claude --model opus --prompt-file /tmp/cy-codex-loop-<slug>-<task_NN>.md`. The delegated Claude run owns `cy-execute-task`, memory updates, validation, and `cy-final-verify`.
6. Else: activate `cy-execute-task` passing the picked `task_NN.md`, then run `cy-final-verify`.
7. `.agents/skills/cy-codex-loop/scripts/update-state.py --task-completed task_NN` advances state only after the execution lane reports PASS and the expected memory/status artifacts exist.

**Exit**: One iteration covers exactly one task. The agent prints the iteration summary and stops. Next iteration re-evaluates.

## Phase B — Execution (mode=free)

**Entry**: `mode=free` AND `progress.deliverables_complete=false`.

**Action**:
1. Read `_techspec.md` deliverables / acceptance section in full.
2. Compare against `progress.checklist[]`. Identify the smallest coherent slice (≤ ~4 hours of focused work) that moves a deliverable forward.
3. Append the slice to `progress.checklist[]` with `status: in_progress` (via `.agents/skills/cy-codex-loop/scripts/update-state.py --add-progress "<text>"`).
4. Re-read `state.yaml`, then resolve the shared + current-task memory paths from `memory-protocol.md`. The current memory file is `memory/free-iter-<NNN>.md`, where `<NNN>` equals the `iteration` value on the checklist item created in step 3.
5. The frontend/docs delegation lane is opt-in (active only when `state.goal_signature` contains `delegation=frontend-docs`, case-insensitive). When inactive, skip ahead to local implementation. When active, read `references/frontend-docs-delegation.md` and apply its classification: the slice qualifies only when its owned paths are exclusively frontend/docs surfaces.
6. If the delegation lane applies (active AND slice qualifies): write the temp prompt and run `compozy exec --ide claude --model opus --prompt-file /tmp/cy-codex-loop-<slug>-free-iter-<NNN>.md`. The delegated Claude run owns implementation, memory updates, validation, and `cy-final-verify`.
7. Else: implement the slice locally and run `cy-final-verify`.
8. `.agents/skills/cy-codex-loop/scripts/update-state.py --complete-progress "<text>"` flips the slice to `completed` only after the execution lane reports PASS and the expected memory/status artifacts exist.
9. **Self-check before claiming deliverables_complete**: re-read `_techspec.md` acceptance section verbatim. If every criterion has at least one matching `progress.checklist[]` entry with `status=completed`, set `--deliverables-complete`. Otherwise leave false and let the next iteration continue.

**Exit**: Either one slice is now complete (more iterations to come) OR `deliverables_complete=true` (Phase C next).

## Phase C — QA

**Entry**:
- `mode=tasks`: head of `tasks.pending` is a QA task (qa-report or qa-execution).
- `mode=free`: `progress.deliverables_complete=true` AND (`qa.report_done=false` OR `qa.execution_done=false`).

**Action** (one artifact per iteration):
1. If `qa.report_done=false`: when `.compozy/tasks/<slug>/qa/bootstrap-manifest.json` is missing AND a QA bootstrap skill is installed (e.g. `agh-qa-bootstrap` in AGH), activate it first; otherwise skip and let `qa-report` create what it needs. Then activate `qa-report`.
2. Else (`qa.execution_done=false`): activate `qa-execution`.
3. `.agents/skills/cy-codex-loop/scripts/update-state.py --qa-report-done` or `--qa-execution-done` accordingly.

**Exit**: Both QA flags true AND `verify.last_status=PASS` → Phase E next. If QA flags are true but `verify.last_status != PASS`, `detect-phase.py` re-emits `phase=C action=qa_execution` so the loop refreshes verify status instead of hanging.

## Phase E — Done

**Entry**: `qa.report_done=true` AND `qa.execution_done=true` AND `verify.last_status=PASS`.

**Action**:
1. Print the iteration summary block (as for every iteration).
2. On a separate line, print the **literal contents** of `assets/done-signature.txt`. This is the evidence the codex-loop goal-check confirmation prompt looks for.
3. Stop. Do not start another action.

The plugin's verdict interpretation will then mark `completed=true` and stop emitting continuations.

## Blocker handling (any phase)

If any step fails irrecoverably (techspec missing, contradictory specs, verify FAIL with no clear fix path, two-touch limit hit), the agent:

1. Records the blocker in `memory/MEMORY.md` `## Open Risks`.
2. Calls `.agents/skills/cy-codex-loop/scripts/update-state.py --blocker "<text>"` to add it to the iteration log.
3. Prints the iteration summary with `outcome=blocked` and stops **without** the done-signature.

Human attention is then required. The next agent restart will re-detect the same blocker until it is resolved.
