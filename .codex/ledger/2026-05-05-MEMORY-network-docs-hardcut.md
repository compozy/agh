# Goal (incl. success criteria):

- Complete `.compozy/tasks/network-threads/task_01.md`: rewrite active RFC/glossary protocol docs so `public_thread`, `direct_room`, `surface`, `thread_id`, `direct_id`, and `work_id` are normative.
- Success criteria: RFC 003/RFC 004/glossary no longer teach active `interaction_id`, `kind:"direct"`, or direct-as-message-kind semantics; focused docs tests and `make verify` pass; task memory/tracking updated; one local commit created after clean verification.

# Constraints/Assumptions:

- Do not run destructive git commands (`restore`, `checkout`, `reset`, `clean`, `rm`) without explicit permission.
- Existing dirty worktree entries are user/other-agent work; do not revert or clobber them.
- Artifacts and commit message must be English.
- Active protocol docs follow `.compozy/tasks/network-threads/_techspec.md` plus ADR-001/002/003. No compatibility aliases or invented behavior.
- Automatic commit is enabled only after clean verification, self-review, and tracking updates.
- Repo-local skills `documentation-writer`, `copywriting`, and `cy-spec-preflight` were found under `.agents/skills/` and loaded even though absent from the session-level skill list.

# Key decisions:

- Add focused Vitest coverage under `packages/site/lib/` for active RFC/glossary scans because the repo already keeps docs truth checks there and `make bun-test` includes the site project.
- Keep shared workflow memory unchanged unless a new durable cross-task finding appears.
- Avoid editing unrelated modified/deleted QA artifacts and untracked task files in `.compozy/tasks/network-threads/`.

# State:

- in_progress

# Done:

- Loaded `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `testing-anti-patterns`, `golang-pro`, `vitest`, repo-local `documentation-writer`, `copywriting`, `cy-spec-preflight`, and `agh-test-conventions` context.
- Read root `AGENTS.md`/`CLAUDE.md`, `packages/site/AGENTS.md`/`CLAUDE.md`, `COPY.md`, workflow memory files, task file, `_tasks.md`, `_techspec.md`, ADR-001/002/003, spec authoring playbook, standing directives, and glossary.
- Captured pre-change signal: active RFCs still contain many `interaction_id` references and RFC 003 still lists `direct` as a message kind.
- Rewrote RFC 003, RFC 004, and glossary around `surface`, `thread_id`, `direct_id`, and `work_id`; added focused docs scan tests in `packages/site/lib/protocol-rfc-hard-cut.test.ts`.
- Targeted checks passed: active RFC/glossary stale-term scan, focused Vitest docs scan, oxfmt check, and site typecheck.
- First full `make verify` attempt failed once in unrelated `internal/session TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail`; targeted `go test -race ./internal/session -run TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail -count=20 -parallel=4` passed.
- Full `make verify` rerun passed with `DONE 8066 tests` and `OK: all package boundaries respected`.
- Updated task-local workflow memory, shared workflow memory, Task 01 tracking, and master task table for Task 01 completion.

# Now:

- Final self-review and commit staging.

# Next:

- Create one local commit for implementation docs/test changes, then run post-commit `make verify`.

# Open questions (UNCONFIRMED if needed):

- Master `_tasks.md` was already modified by another workflow and listed only task 18/19; added Task 01 completed row without changing those existing rows. Tracking/memory files are expected to remain out of the implementation commit unless staging policy changes.

# Working set (files/ids/commands):

- Candidate docs: `docs/rfcs/003_agh-network-v0.md`, `docs/rfcs/004_agh-network-v1.md`, `docs/_memory/glossary.md`.
- Candidate test: `packages/site/lib/protocol-rfc-hard-cut.test.ts`.
- Workflow memory: `.compozy/tasks/network-threads/memory/task_01.md`.
- Required final gate: `make verify`.
