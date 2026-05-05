# Peer Review Summary Round 2

## Verdict

`READY`

Claude/Opus found that all eight round 1 blockers are materially addressed. The remaining findings are nits: mostly spec clarifications, one boundary-ownership cleanup, and implementation-task guardrails.

## Counts

- Blockers: 0
- Nits: 13

## Blockers

None.

## Nits

- `N-001` — `Architectural Boundaries / Implementation Design — ConversationStore`: `ConversationStore` currently exposes `network.ConversationRef`, which would force `globaldb` to import `internal/network` despite the boundary rule.
- `N-002` — `Data Models — SQLite Migration`: require implementers to verify the next free `globalSchemaMigrations` version at task start instead of blindly using version 16.
- `N-003` — `Data Models — Fresh Schema DDL`: explicitly confirm or assert `PRAGMA foreign_keys = ON` for `network_thread_participants` cascades.
- `N-004` — `Implementation Design — Same-Transaction Write Strategy step 8`: define or remove the "allowed terminal duplicate" carve-out.
- `N-005` — `Implementation Design — Direct Room Resolution Algorithm`: define the zero-row-after-`INSERT OR IGNORE` direct-ID collision case.
- `N-006` — `Implementation Design — Same-Transaction Write Strategy step 4`: pin whether first send to a fresh `thread_id` opens a thread, and define `thread_id` grammar.
- `N-007` — `Extensibility — Hooks`: state post-commit hook delivery semantics explicitly.
- `N-008` — `Implementation Design — Validators and Reason Codes`: add explicit symmetric rejection invariants for container IDs without matching `surface`.
- `N-009` — `Data Models — network_work DDL`: prevent dangling `network_work` rows with per-surface FKs or constraint triggers.
- `N-010` — `Implementation Design — Work Lifecycle and task_runs`: name a canonical task metadata key linking task runs to `work_id`.
- `N-011` — `API, CLI, Tooling — Web Route Map`: define channel-level composer behavior when no thread/direct is selected.
- `N-012` — `MVP Boundary — task split`: state that `cy-create-tasks` appends the required `qa-report` + `qa-execution` pair.
- `N-013` — `Implementation Design — Wire Model`: add JCS/JSON round-trip tests for absent vs zero-valued nullable fields.

## Main Themes

- Boundary precision: move shared conversation reference types out of `internal/network`, or use primitive arguments in store-facing interfaces.
- SQLite safety: confirm foreign-key enforcement, prevent dangling work rows, and make rare direct-ID collision behavior explicit.
- Runtime semantics: pin thread-opening, terminal duplicate, hook-delivery, and task-run correlation rules before implementation.
- QA/task hygiene: make generated-task QA split and JCS canonicalization tests explicit.

## Artifacts

- Prompt: `.compozy/tasks/network-threads/qa/peer-review-prompt-round2.md`
- Raw event stream: `.compozy/tasks/network-threads/qa/peer-review-result-round2.json`
- Clean final JSON: `.compozy/tasks/network-threads/qa/peer-review-final-round2.json`
- Stderr: `.compozy/tasks/network-threads/qa/peer-review-result-round2.err`

## Notes

`peer-review-result-round2.err` contains the same non-fatal workspace extension discovery warning seen in round 1 for `.compozy/extensions/cy-qa-workflow` using an unknown hook event `plan.pre_resolve_task_runtime`. The `compozy exec` run completed successfully with exit code 0.
