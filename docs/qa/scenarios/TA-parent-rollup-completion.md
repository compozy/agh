---
id: TA-parent-rollup-completion
area: TA
title: Complete a parent when every child completes
persona: Bruno
journey: J-complete-task-tree
expected: Completing the final child transitions the parent task to completed exactly once, while completing an earlier child leaves the parent non-terminal; the rollup is visible after refresh through Web and structured surfaces.
entry_points: web /tasks; task detail modal; CLI agh task list
qa_status: pass
bug_ids: BUG-20260713-parent-task-rollup-missing
fix_status: fixed
retest_status: pass
fix_commits:
evidence: /Users/pedronauck/dev/qa-labs/agh-automation-features-20260713-20260713-044543-173594-lab/qa-artifacts/qa/screenshots/agh71-parent-after-first-child.dom.txt;/Users/pedronauck/dev/qa-labs/agh-automation-features-20260713-20260713-044543-173594-lab/qa-artifacts/qa/screenshots/agh71-all-children-completed-parent-stuck.dom.txt;/Users/pedronauck/dev/qa-labs/agh-automation-features-post-onboarding-fix-20260713-20260713-203513-816377-lab/qa-artifacts/qa/screenshots/agh71-faithful-parent-run.dom.txt;/Users/pedronauck/dev/qa-labs/agh-automation-features-post-onboarding-fix-20260713-20260713-203513-816377-lab/qa-artifacts/qa/screenshots/agh71-faithful-parent-children.dom.txt
last_report: docs/qa/reports/2026-07-13-automation-features.md
overlaps: LP-042;TA-012
---

Linear issue AGH-71 is the named regression target.

2026-07-13: Owner clearing, recovery, task-role activation, and exactly-once child completion are fixed and passed. An earlier child correctly left the parent non-terminal, but after the final two real Cursor completions all three children are Completed while the parent remains Ready / Needs Attention. This is the direct AGH-71 failure.

2026-07-14 retest: fresh parent `task-a2b46ce593b5e75b` had no bound session and stayed nonterminal after child A. Child B's one real completion atomically settled the existing parent run and Task. Reload plus the Children tab showed one Completed parent run and both children Completed. AGH-71 passes; the separately tracked matching-Loop wake remains pending.
