# Memory Ledger — loops-docs-review

- Goal (incl. success criteria): Full review of `.compozy/tasks/loops/` spec corpus for gaps/errors + apply fixes. DONE.
- Constraints/Assumptions: Docs-only edits; Bash denied; subagents read-only; artifacts EN.
- Key decisions:
  - Pinned 2 previously-unspecified semantics (flagged for operator veto): (1) `Stop` → terminal `failed` + `TransitionCause=operator_stop` (no 5th outcome); (2) versions v1 model = daemon-owned `meta.version` bump on publish + projection metadata history; drafts editor-session only.
  - Appended unit-35 (start bindings) / unit-36 (stop) / SDK-§6 to `_tests.md` (never renumbered existing cases).
- State: COMPLETE — review + fixes applied across `_techspec.md`, `_tests.md`, `_tasks.md`, task_01..05/08/09/11/13/14/18..23, adr-001.
- Done: verified all 17 ADRs + 9 design files exist; QA round-6 verdict READY; impl not started (no internal/loop, no migrations); fixed test-ownership mismatches (tasks 01/03/19/20/21/22 wrong case numbers), graph edges (06→17, 16→17, 16→23), missing Runs route in task_18, glossary/RFC-003 ownership → task_23, handoff.txt→.md, ADR count 16→17, agent_skill_resources :62→:67, no-progress two-axes clarification, queued/ready truthful-UI note.
- Done (round 2 — surface audit docs/tools/ext/sdk/hooks/http-uds/cli): gate `extension` check added to DSL + GateEvaluator + task_08; §9.10 input types enumerated in DSL (string|number|boolean|file|agent|ref, pickers ride existing list endpoints); CLI gains `runs`+`delete`, native tools gain `agh__loop_runs`+`agh__loop_delete` (no `agh__loop_edit` — FS/PATCH is the agent edit path) in techspec + task_15 + E2E-runtime-3 + Impact Audit; task_23 gains explicit reference-page MUSTs (DSL ref, guardrails, hooks family, extension authoring guide) + subtask 23.6.
- Now: —
- Next: operator review of pinned decisions (stop→failed/operator_stop; versions v1 model; new runs/delete verbs); ledger can be deleted after user ack.
- Open questions (UNCONFIRMED if needed): none blocking.
- Working set: `.compozy/tasks/loops/**`
