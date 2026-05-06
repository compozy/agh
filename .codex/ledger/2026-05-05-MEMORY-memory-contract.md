Goal (incl. success criteria):

- Execute `.compozy/tasks/mem-v2/task_01.md`: extract Memory v2 shared contracts into `internal/memory/contract`, hard-delete `internal/memory/types.go`, update callers, add focused tests, run full verification, update tracking, and create one local commit.

Constraints/Assumptions:

- Work in `/Users/pedronauck/dev/compozy/agh3`; `/Users/pedronauck/dev/compozy/looper` does not contain `internal/memory/types.go`.
- Must use `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `golang-pro`, `agh-code-guidelines`, `agh-test-conventions`, `testing-anti-patterns`, and `no-workarounds`.
- `brainstorming` approval gate is not applicable because this is execution of an already approved PRD/TechSpec/ADR task, not open-ended feature design.
- Do not run destructive git commands. Automatic local commit only after clean verification, self-review, memory/tracking updates.
- Current repo has untracked `.compozy/tasks/mem-v2/memory/` before implementation; treat carefully as workflow/task tracking state.

Key decisions:

- Use `/Users/pedronauck/dev/compozy/agh3/.compozy/tasks/mem-v2` as the source of truth for Task 01.

State:

- Complete. Task 01 code was committed locally as `4d372f70 refactor: extract memory contract package`; post-commit `env -u NO_COLOR make verify` passed.

Done:

- Read workflow memory files for mem-v2.
- Read root/internal AGENTS and CLAUDE guidance.
- Loaded required workflow, Go, test, no-workarounds, and contract-codegen-related skills.
- Scanned existing ledgers; relevant prior mem-v2 task generation ledger says TechSpec/ADRs/task pack were validated.
- Created `internal/memory/contract`, moved contract DTOs/enums/provider interfaces there, deleted `internal/memory/types.go`, and removed `internal/memory/interfaces_test.go`.
- Updated memory store/catalog/recall, API, CLI, extension contract/host API, codegen metadata, and daemon callers to import contract types directly.
- Hard-cut `contract.Header` to canonical YAML `agent` while keeping JSON/API `agent_name`; no legacy YAML `agent_name` compatibility field remains.
- Made `contract.Header.Provenance` optional and refreshed generated OpenAPI/TypeScript consumers for `agent_tier`, optional provenance, and `agent` scope.
- `go test ./internal/memory/...` passed.
- `go test ./internal/api/... ./internal/cli/... ./internal/extension/... ./internal/codegen/... ./internal/daemon/...` passed.
- `go test ./internal/memory/contract -cover` passed with 98.8% coverage.
- `go test ./internal/memory/... ./internal/api/... ./internal/cli/... ./internal/extension/...` passed after final hardening.
- `env -u NO_COLOR make verify` passed after final hardening: frontend format/lint/typecheck/test/build passed, Go lint reported `0 issues`, Go tests reported `DONE 8081 tests`, and package boundaries passed.
- Updated task tracking: `.compozy/tasks/mem-v2/task_01.md` is `completed`, all Task 01 subtasks/tests are checked, and `_tasks.md` marks Task 01 complete.
- Created local commit `4d372f70 refactor: extract memory contract package` with code/generated changes only; memory/tracking files remain uncommitted by policy.
- Post-commit `env -u NO_COLOR make verify` passed. A transient earlier site metadata timeout was rerun directly and passed before the final full verify rerun.

Now:

- Final response.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None blocking.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-05-MEMORY-memory-contract.md`
- `.compozy/tasks/mem-v2/memory/MEMORY.md`
- `.compozy/tasks/mem-v2/memory/task_01.md`
- `.compozy/tasks/mem-v2/task_01.md`
- `.compozy/tasks/mem-v2/_techspec.md`
- `.compozy/tasks/mem-v2/_tasks.md`
- `.compozy/tasks/mem-v2/adrs/adr-001.md` through `adr-012.md`
- `internal/memory/contract/*`
- `internal/memory/{store,catalog,document,assembler,recall}*.go`
- `internal/api/**`, `internal/cli/**`, `internal/extension/**`, `internal/codegen/sdkts/generate.go`, `internal/daemon/**`
- Commit: `4d372f70 refactor: extract memory contract package`
