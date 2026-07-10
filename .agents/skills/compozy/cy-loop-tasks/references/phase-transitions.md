# Phase transitions — detect-phase contract

`.agents/skills/compozy/cy-loop-tasks/scripts/detect-phase.py` (read-only) is
the single source of truth for "what phase am I in right now?". It reads
`state.yaml` plus the filesystem under `.compozy/tasks/<slug>/` and prints
exactly one line:

```
phase=0 action=bootstrap
phase=B action=execute_task task=task_NN [lane=frontend agent=claude|cursor]
phase=B action=execute_free_slice
phase=C action=qa_report
phase=C action=qa_execution
phase=D action=peer_review round=N
phase=E action=done
```

The agent runs the printed action (procedures live in `SKILL.md`), records
the iteration via `update-state.py`, and stops. The next restart re-runs
detect-phase and resumes from filesystem truth.

## Entry conditions

| Printed line | Entry condition |
|--------------|-----------------|
| `phase=0 action=bootstrap` | `state.yaml` does not exist. |
| `phase=B action=execute_task task=<stem>` | `mode=tasks` AND head of `tasks.pending` is not a QA task. The `lane=frontend agent=<x>` suffix appears when `frontend_agent` is set AND the head task's frontmatter `type:` is `frontend`. |
| `phase=B action=execute_free_slice` | `mode=free` AND `progress.deliverables_complete=false`. |
| `phase=C action=qa_report` | QA is next (tasks: head of pending is a QA task; free: deliverables complete) AND `qa.report_done=false`. Always precedes `qa_execution`. |
| `phase=C action=qa_execution` | `qa.report_done=true` AND `qa.execution_done=false`. |
| `phase=D action=peer_review round=N` | Both QA flags true AND `review.ship=false` (`N = review.rounds + 1`). Also re-emitted when `review.ship=true` but `verify.last_status != PASS` — a SHIP verdict on a failing tree is void, so review re-enters after the tree is fixed. |
| `phase=E action=done` | Both QA flags true AND `review.ship=true` AND `verify.last_status=PASS`. |

## Exit rules

- Phase 0 exits once `init-state.py` has written `state.yaml`; the next run
  enters B.
- Phase B covers exactly one task or slice per iteration. In free mode,
  `--deliverables-complete` (set only when every techspec acceptance
  criterion has a completed checklist entry) moves the loop to C.
- Phase C produces exactly one QA artifact per iteration, `qa_report` first.
  In mode=tasks the corresponding QA task is also marked completed so
  `tasks.pending` drains.
- Phase D closes one `cy-impl-peer-review` round per iteration via
  `--review-round-done <verdict>`; `SHIP` sets `review.ship=true`.
- Phase E prints the iteration summary plus the literal contents of
  `assets/done-signature.txt`, then stops. The codex-loop verdict prompt
  marks the goal complete from that signature.

## Blocker handling (any phase)

When a step fails irrecoverably (techspec missing, contradictory specs,
verify FAIL with no fix path, two-touch limit hit):

1. Record the blocker in `memory/MEMORY.md` `## Open Risks`.
2. Run `.agents/skills/compozy/cy-loop-tasks/scripts/update-state.py <slug> --blocker "<text>"`
   (skip when `state.yaml` does not exist yet — bootstrap failures record in
   memory and the summary only).
3. Print the iteration summary with `outcome=blocked` and stop **without**
   the done-signature.

The next restart re-detects the same blocker until a human resolves it.
