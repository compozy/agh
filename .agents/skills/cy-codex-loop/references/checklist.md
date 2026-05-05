# Per-iteration self-audit checklist

Walk this checklist before printing the iteration summary block. Failing
any item means the iteration is **not** complete: do the missing work,
then re-check. Do not print the done-signature until every item passes
on the final iteration.

## Every iteration

- [ ] `.agents/skills/cy-codex-loop/scripts/detect-phase.py` was run as the first action and its output was followed.
- [ ] The dispatched `cy-*` skills or the Compozy/Claude Opus delegation lane for the printed phase were activated **before** any code edits or reviews.
- [ ] If Phase B work was frontend/docs, `references/frontend-docs-delegation.md` was read and the local Codex session stayed in orchestration mode.
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
- [ ] `task_NN.md` frontmatter `type:` was checked before choosing the local lane or the delegation lane.
- [ ] Exactly ONE task was attempted in this iteration.

## Phase B mode=free only

- [ ] The slice picked was small enough to finish in one iteration (≤ ~4 hours).
- [ ] The slice was added to `progress.checklist[]` BEFORE implementation started.
- [ ] If the slice was delegated, its owned paths were explicitly limited to frontend/docs surfaces per `references/frontend-docs-delegation.md`.
- [ ] If `deliverables_complete` was set true: every techspec acceptance criterion has at least one matching `progress.checklist[]` entry with `status=completed`. Self-quote each criterion → its checklist entry in the iteration summary.

## Phase C only

- [ ] `qa-report` was completed before `qa-execution` (do not skip ahead).
- [ ] If `bootstrap-manifest.json` was missing, a QA bootstrap skill (e.g. `agh-qa-bootstrap` in AGH) ran first — or the absence of such a skill in this project was noted before falling through.

## Phase D only

- [ ] Round number was computed from `ls reviews-*` and zero-padded to 3 digits.
- [ ] `.agents/skills/cy-codex-loop/scripts/coderabbit-to-rounds.py` was used to generate `issue_NNN.md`; nothing was hand-written.
- [ ] `.agents/skills/cy-codex-loop/scripts/check-rounds-clean.py` was the source of truth for clean/dirty, not eyeballed counts.
- [ ] If the streak was reset, the iteration summary explicitly names the issue(s) that broke it.

## Phase E only

- [ ] `coderabbit.rounds_clean_streak >= coderabbit.rounds_required` was confirmed via `state.yaml`, not memory.
- [ ] `verify.last_status` is `PASS` and the timestamp is recent (same iteration as Phase E entry).
- [ ] The done-signature from `assets/done-signature.txt` is the LAST line of the message.
