Goal (incl. success criteria):

- Implement provider-model-catalog Task 03: `internal/modelcatalog` service/sources over Task 02 store, deterministic merge, stale fallback, `builtin`/`config`/`models_dev` sources, required tests, tracking updates, and one local commit after clean verification.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory files before edits and before finish.
- Must follow `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- Must read AGENTS/CLAUDE guidance, PRD `_techspec.md`, `_tasks.md`, task_03, and ADRs before implementation.
- No destructive git commands without explicit permission.
- `make verify` is required before completion and before/after commit.

Key decisions:

- Reuse Task 02 `internal/modelcatalog.Store` and GlobalDB implementation; do not duplicate row/status contracts.

State:

- Implementation planning complete; pre-change signal confirms package has no Task 03 behavior/tests yet.

Done:

- Read RTK.
- Read workflow shared memory and task_03 memory.
- Read required workflow, Go, and test skill entrypoints.
- Read root AGENTS/CLAUDE and `internal/CLAUDE.md`.
- Scanned relevant model-catalog ledgers from prior Task 01/02/spec work.
- Read Task 03, `_tasks.md`, `_techspec.md`, and ADR-001..003.
- Captured pre-change signal: `rtk go test ./internal/modelcatalog ./internal/modelcatalog/... -count=1 -cover` reports no tests found.
- Printed cy-execute-task working checklist.

Now:

- Implementing scoped service/source/test changes.

Next:

- Build visible checklist, capture pre-change signal, then implement scoped service/source/test changes.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_03.md`
- `.compozy/tasks/provider-model-catalog/task_03.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `.compozy/tasks/provider-model-catalog/_techspec.md`
- `.compozy/tasks/provider-model-catalog/adrs/`
