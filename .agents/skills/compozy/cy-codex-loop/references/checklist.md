# Per-iteration self-audit checklist

Walk this checklist before printing the iteration summary block. Failing
any item means the iteration is **not** complete: do the missing work,
then re-check. Do not print the done-signature until every item passes
on the final iteration.

## Every iteration

- [ ] `.agents/skills/cy-codex-loop/scripts/detect-phase.py` was run as the first action and its output was followed.
- [ ] The dispatched `cy-*` skills (or, when the opt-in delegation lane is active, the Compozy/Claude Opus dispatch) for the printed phase were activated **before** any code edits or reviews.
- [ ] The delegation-lane gate (`delegation=frontend-docs` in `state.goal_signature`) was evaluated. When inactive, all Phase B work ran locally regardless of task `type:`. When active and the picked task/slice qualified, the local Codex session stayed in orchestration mode and dispatched via `compozy exec`.
- [ ] Memory was updated via `cy-workflow-memory` in the execution lane before any state-flipping operation (`cy-execute-task` step 5 sequence: memory → checkboxes → status → master → commit).
- [ ] `.agents/skills/cy-codex-loop/scripts/update-state.py` was called with the right flags so `state.yaml` reflects the new reality.
- [ ] `cy-final-verify` ran for any iteration that produced code or fixes; if the work was delegated, the delegated PASS/FAIL evidence was captured and cited in the iteration summary's `verify_evidence`.
- [ ] The iteration summary block (from `assets/iteration-summary.template.md`) is the LAST thing in the assistant message — nothing else after it except, in Phase E only, the done-signature line.

## Phase 0 (bootstrap) only

- [ ] `_techspec.md` existence was confirmed before writing `state.yaml`.
- [ ] `mode` was decided by filesystem, not by guess (`tasks` if `_tasks.md` AND a `task_*.md` exist, else `free`).
- [ ] `goal_signature` was copied verbatim from the user's prompt (CODEX_LOOP `goal=` value or manual reason).

## Phase B mode=tasks only

- [ ] `task_NN.md` frontmatter `status:` was checked and trusted as source of truth (state.yaml reconciled if it disagreed).
- [ ] If the delegation lane is active for this loop, `task_NN.md` frontmatter `type:` was checked before deciding to dispatch via `compozy exec` versus running locally. When the lane is inactive, this check is skipped.
- [ ] Exactly ONE task was attempted in this iteration.
- [ ] `.agents/skills/cy-codex-loop/scripts/commit-checkpoint.py <slug> --task <stem>` ran after `update-state.py` and either printed a commit SHA or the literal `SKIP: no changes`. The result is captured in the iteration summary's checkpoint commit field.

## Phase B mode=free only

- [ ] The slice picked was small enough to finish in one iteration (≤ ~4 hours).
- [ ] The slice was added to `progress.checklist[]` BEFORE implementation started.
- [ ] If the slice was delegated, its owned paths were explicitly limited to frontend/docs surfaces per `references/frontend-docs-delegation.md`.
- [ ] If `deliverables_complete` was set true: every techspec acceptance criterion has at least one matching `progress.checklist[]` entry with `status=completed`. Self-quote each criterion → its checklist entry in the iteration summary.
- [ ] `.agents/skills/cy-codex-loop/scripts/commit-checkpoint.py <slug> --slice "<slice text>"` ran after `update-state.py` and either printed a commit SHA or the literal `SKIP: no changes`. The result is captured in the iteration summary's checkpoint commit field.

## Phase C only

- [ ] `qa-report` was completed before `qa-execution` (do not skip ahead).
- [ ] If `bootstrap-manifest.json` was missing, a QA bootstrap skill (e.g. `agh-qa-bootstrap` in AGH) ran first — or the absence of such a skill in this project was noted before falling through.

## Phase E only

- [ ] `qa.report_done=true` AND `qa.execution_done=true` confirmed via `state.yaml`, not memory.
- [ ] `verify.last_status` is `PASS` and the timestamp is recent (same iteration as Phase E entry).
- [ ] The done-signature from `assets/done-signature.txt` is the LAST line of the message.
