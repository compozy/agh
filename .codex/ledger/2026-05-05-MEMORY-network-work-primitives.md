Goal (incl. success criteria):

- Implement Task 03: work lifecycle and direct-room identity primitives in `internal/network`; success requires deterministic direct-room IDs, work/container binding, lifecycle idempotency/terminal rules, tests, tracking updates, clean `make verify`, and one local commit.

Constraints/Assumptions:

- Must not run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) without explicit permission.
- Current task scope is runtime primitives only; durable SQLite/store work is deferred by `task_03.md` despite older TechSpec task-numbering text.
- Missing installed skills: `nats`, `agh-code-guidelines`, and `agh-test-conventions`; using available project guidance plus `golang-pro`, `testing-anti-patterns`, `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify`.
- Full monorepo `make verify` is mandatory before completion and before commit.

Key decisions:

- Treat `task_03.md` as the scope authority and TechSpec/ADRs as semantic authority for direct-room identity and work lifecycle rules.

State:

- Task 03 implementation is complete and committed locally.
- Tracking/memory files were updated but kept out of the automatic code commit per tracking-only guidance.

Done:

- Read workflow memory and current task memory.
- Read root/internal AGH guidance, Task 03, `_tasks.md`, TechSpec excerpts, ADR-001, ADR-002, ADR-003, and relevant design lifecycle sections.
- Implemented work/container mismatch error handling, terminal timestamp validation, and stricter post-terminal rejection.
- Implemented direct peer normalization, same-peer rejection, collision signaling, and direct-room binding validation.
- Updated network tests for deterministic direct IDs, collision signaling, work binding, cross-container rejection, duplicate-before-lifecycle idempotency, and terminal rejection.
- Ran `go test -count=1 ./internal/network` successfully.
- Ran `go test -count=1 -cover ./internal/network` successfully with 82.0% coverage.
- Ran pre-commit `make verify` successfully after tracking updates.
- Created local commit `78a714be feat: add network work lifecycle primitives`.
- First post-commit `make verify` hit an unrelated SDK integration timeout; targeted SDK integration rerun passed; final full post-commit `make verify` passed exit 0.

Now:

- Final response handoff.

Next:

- No further Task 03 work remains in this run.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- PRD dir: `/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads`
- Task memory: `/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads/memory/task_03.md`
- Shared memory: `/Users/pedronauck/Dev/compozy/agh2/.compozy/tasks/network-threads/memory/MEMORY.md`
- Code surfaces: `internal/network/lifecycle.go`, `internal/network/validate.go`, `internal/network/helpers_test.go`, `internal/network/lifecycle_test.go`, `internal/network/router.go`, `internal/network/router_test.go`, `internal/network/validate_test.go`, plus mechanical active naming cleanup in `internal/network/*_test.go`.
