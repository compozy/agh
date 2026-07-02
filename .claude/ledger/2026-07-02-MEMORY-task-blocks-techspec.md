---
name: task-blocks-techspec
task: cy-create-techspec for .compozy/tasks/task-blocks
---

- Goal (incl. success criteria): Advance the `task-blocks` spec. `_techspec.md` already exists (37K, canonical template, 7 ADRs) and passed 3 peer-review rounds (round3 blockers B-001/B-002 + nits N-001/N-002 all incorporated per peer-review-incorporation-round3.md). User's critical instructions describe the TASKS-phase deliverable (loops-style), not techspec edits.
- Constraints/Assumptions:
  - User critical rules: every task file must carry a `<skills>` map, a peer-review gate enforced to `verdict: SHIP` (nits included), a complete test-case list, and 2 final QA tasks — modeled on `.compozy/tasks/loops`.
  - Tasks may be LARGE (long-running specialized models); do NOT over-decompose into tiny tasks.
  - Loops reference structure: `_tasks.md` (graph nodes/edges + MVP boundary + Execution Gates), `task_NN.md` (skills map + Peer Review Gate), `_tests.md` (numbered cases mapped to invariants/ADRs), tail QA pair.
- Key decisions: CONFIRMED — user meant `/cy-create-tasks` (not techspec). Techspec is treated as final input. Generate loops-style task decomposition honoring the <critical> directives.
- State:
- Done: Read \_techspec.md, round3 summary+incorporation, loops \_tasks.md, ran cy-spec-preflight.
- Now: Run cy-create-tasks. Decompose Build Order (steps 1–13) into LARGE loops-style tasks; author \_tasks.md (graph + MVP boundary + Execution Gates), \_tests.md (numbered, mapped to invariants/ADRs), task_NN.md (each with <skills> map + Peer Review Gate + test refs + Web/Docs Impact + Extensibility/Agent-Manageability/Config subitems), tail QA pair (qa-report high + qa-execution critical).
- Next: Explore codebase surfaces per techspec to enrich task bodies; author files; verify with cy-spec-preflight tasks-phase checks.
- Open questions (UNCONFIRMED): none blocking — proceed.
- Working set:
  - .compozy/tasks/task-blocks/\_techspec.md
  - .compozy/tasks/task-blocks/adrs/adr-001..007.md
  - .compozy/tasks/task-blocks/qa/peer-review-\*-round3.md
  - .compozy/tasks/loops/{\_tasks.md,\_tests.md,task_NN.md} (reference structure)
