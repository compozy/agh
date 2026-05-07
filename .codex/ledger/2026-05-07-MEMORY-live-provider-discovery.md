Goal (incl. success criteria):

- Implement provider-model-catalog Task 04: side-effect-free live provider discovery sources for core providers, configured/fail-closed discovery for OpenClaw/Hermes/Pi, safe timeout/env/home/coalescing behavior, required tests, tracking updates, and one local commit after clean verification.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory files before edits and before finish.
- Must follow `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- Must read AGENTS/CLAUDE guidance, provider-model-catalog PRD docs, `_techspec.md`, `_tasks.md`, task_04, and ADRs before implementation.
- No destructive git commands without explicit permission.
- `make verify` is required before completion and before/after commit.
- Conversation in BR-PT; code/docs/artifacts in English.

Key decisions:

- Task 04 depends on Task 03; current worktree already contains uncommitted `internal/modelcatalog` service/source code from Task 03. Treat it as existing branch state and work with it, not against it.

State:

- Implementation, focused verification, full pre-commit verification, self-review, workflow memory updates, task tracking updates, scoped local commit, and post-commit verification are complete.

Done:

- Read RTK.
- Scanned ledger directory for cross-agent awareness.
- Read workflow shared memory and task_04 memory.
- Read relevant Task 02/03/spec ledgers for cross-task context.
- Read required workflow, Go, and test skill entrypoints plus canonical references.
- Read root/internal AGENTS/CLAUDE guidance, Task 04, `_tasks.md`, full `_techspec.md`, and ADR-001..003.
- Captured pre-change signal: `rtk go test ./internal/modelcatalog ./internal/modelcatalog/... -count=1 -cover` passed with 32 tests, but Task 04 live adapters were not present.
- Printed cy-execute-task working checklist.
- Added `internal/modelcatalog/live_sources.go` with live provider source adapters, HTTP/subprocess discovery wrappers, timeout/redaction handling, effective auth/env/home policy use, configured discovery, fail-closed unsafe providers, and source registration helpers.
- Updated `internal/modelcatalog/service.go` to coalesce same-provider refresh work with in-flight sharing before source execution.
- Added `internal/modelcatalog/live_sources_test.go` covering core provider parsing/status behavior, configured discovery, timeouts, redaction, env/home policy, source registration, real command executor behavior, and refresh coalescing.
- Self-review found and fixed a coalescing bug: same-provider refreshes with different `SourceID` scopes now serialize without sharing the wrong statuses.
- Ran focused verification:
  - `rtk go test ./internal/modelcatalog -count=1` passed.
  - `rtk proxy go test ./internal/modelcatalog/... -count=1 -cover` passed with 83.4% coverage after the coalescing correction.
  - `rtk go test ./internal/modelcatalog -run 'TestLiveProvider|TestLiveDiscovery|TestLiveProviderParsing' -count=1 -race` passed.
  - `rtk go test ./internal/modelcatalog/... -count=1 -race` passed.
  - `rtk go test ./internal/store/globaldb -run 'TestGlobalDBModelCatalog|TestOpenGlobalDBFailsOnSchemaMigrationIntegrityMismatch' -count=1` passed.
  - `rtk make fmt` passed.
  - `rtk make lint` passed.
- Ran `rtk make verify` three times; all passed, including the required post-commit run.
- Updated `.compozy/tasks/provider-model-catalog/task_04.md` and `_tasks.md` to mark Task 04 complete.
- Updated workflow shared memory and task_04 memory.
- Created local commit `ca4f350e feat: add live provider discovery sources`.
- `scripts/check-test-conventions.py` is absent in this checkout; test-conventions script check could not be run.

Now:

- Final status check and response.

Next:

- None for Task 04.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_04.md`
- `.compozy/tasks/provider-model-catalog/task_04.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `.compozy/tasks/provider-model-catalog/_techspec.md`
- `.compozy/tasks/provider-model-catalog/adrs/`
- `internal/modelcatalog/service.go`
- `internal/modelcatalog/live_sources.go`
- `internal/modelcatalog/live_sources_test.go`
- `.compozy/tasks/provider-model-catalog/memory/task_04.md`
- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
