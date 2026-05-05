# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Rewrite active AGH Network RFC/glossary vocabulary for Task 01 so later implementation tasks use `surface:"thread"|"direct"`, `thread_id`, `direct_id`, and `work_id`.
- Acceptance requires active docs/tests to stop teaching `interaction_id`, `kind:"direct"`, or direct-as-message-kind behavior.

## Important Decisions

- Use the approved TechSpec and ADR-001/002/003 as the source of protocol truth; do not invent compatibility behavior.
- Add focused docs scan coverage in `packages/site/lib/` because site Vitest already hosts documentation truth tests and is included in the monorepo Bun test gate.
- Keep direct-room language as restricted two-party visibility, not cryptographic privacy.

## Learnings

- Pre-change signal: RFC 003 and RFC 004 actively mention `interaction_id`; RFC 003 still lists `direct` as a message kind.
- The target repo contains local `.agents/skills/documentation-writer`, `.agents/skills/copywriting`, and `.agents/skills/cy-spec-preflight` even though they were absent from the session-level skill index.
- Post-edit active RFC/glossary scan is clean for the removed wire terms.
- Repo-wide scan still finds old vocabulary in implementation code and site/runtime docs that the TechSpec assigns to later tasks; Task 01 scope remains RFC/glossary plus tests.
- First full `make verify` attempt reached the Go test phase and failed once in unrelated `internal/session` test `TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail`; targeted `go test -race ./internal/session -run TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail -count=20 -parallel=4` passed, pointing to full-suite timing sensitivity rather than a Task 01 docs regression.
- Final pre-commit `make verify` rerun passed with `DONE 8066 tests` and `OK: all package boundaries respected`.

## Files / Surfaces

- Touched active docs: `docs/rfcs/003_agh-network-v0.md`, `docs/rfcs/004_agh-network-v1.md`, `docs/_memory/glossary.md`.
- Touched test: `packages/site/lib/protocol-rfc-hard-cut.test.ts`.
- Touched tracking/memory: `.compozy/tasks/network-threads/task_01.md`, `.compozy/tasks/network-threads/_tasks.md`, `.compozy/tasks/network-threads/memory/task_01.md`, `.compozy/tasks/network-threads/memory/MEMORY.md`.

## Errors / Corrections

- Oxfmt initially reported format issues in RFC 003 and the new test; ran `bunx oxfmt ...` and rechecked clean.
- Initial site typecheck exposed an overly wide JSON-example helper type in `packages/site/lib/protocol-rfc-hard-cut.test.ts`; tightened the helper return type instead of weakening assertions.

## Ready for Next Run

- If interrupted, continue from the session ledger `.codex/ledger/2026-05-05-MEMORY-network-docs-hardcut.md` and the planned working set above.
- Current scoped evidence: `rg -n 'interaction_id|kind:"direct"|...` over RFC 003, RFC 004, and glossary returned no matches; targeted Vitest docs scan passed 1 file / 4 tests; oxfmt check passed for changed docs/test.
- Task tracking has been updated to completed after clean full verification. Keep tracking/memory files out of the automatic implementation commit unless repository policy requires staging them.
