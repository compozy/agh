# Memory Ledger — loops-editor-design-review

- Goal (incl. success criteria): Assess whether the loop editor design in `docs/design/opendesign` (software-delivery fork & edit screen) supports the goal of replacing compozy (github.com/compozy/compozy) task execution. Deliver explicit layout/design change recommendations. Assessment only — no code changes.
- Constraints/Assumptions:
  - User context: after Loop Engineering ships in AGH, AGH merges with compozy; current compozy users must be able to use Loops for task execution.
  - Conversation in BR-PT; artifacts in English.
  - Read-only exploration via subagents.
- Key decisions:
  - Verified against source: requirements.md:40 (C5 — authoring stays OUTSIDE the Loop) and task_17.md:39 (software-delivery MUST fan-out over the task collection, gates, failed-only, cap 50).
  - Verdict: editor chrome is fine; the software-delivery example graph in loop-editor.html/loop-detail.html contradicts the spec (invents plan/plan-gate planner, fans out over plan.subtasks instead of loaded task files; missing load-tasks/file-import head, post-collect review gate, human-approval node).
  - Compozy parity gaps beyond graph content: worktree isolation + merge/conflict-resolution semantics absent from design (and apparently techspec); run-form slug should be a task-dir picker; run-detail should group fan-out branches by dependency wave; configure sheet lacks concurrency limit.
- State: analysis delivered to user (assessment only, no code changes)
- Done: 3 Explore agents (opendesign, loops spec, compozy repo at ~/Dev/compozy/compozy); verified load-bearing quotes; delivered explicit change list
- Now: awaiting user direction (apply changes to design artboards? open spec gap for worktree/merge?)
- Next: —
- Open questions (UNCONFIRMED if needed):
  - Whether \_techspec.md covers worktree/merge semantics anywhere (subagents found none — UNCONFIRMED gap)
- Working set (files/ids/commands): docs/design/opendesign, .compozy/tasks/loops, screenshot of Fork & Edit editor (plan-gate inspector, node palette: run-agent, call-tool, channel-post, fan-out, collect, branch, gate, sub-loop, watch-source, file-import, input)
