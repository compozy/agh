# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Daytona provider package for task_06: SSH transport with token refresh, SDK-backed sandbox lifecycle/file ops, tar sync, env/network policy, unit/integration tests, tracking, and verification.

## Important Decisions
- Do not mark task_06 complete or create the completion commit unless required verification evidence is available. Current run has a dependency blocker: task_05 is still pending and live Daytona SSH validation cannot run because `DAYTONA_API_KEY` is missing.
- Keep the implemented Daytona provider changes in the worktree, but do not update task tracking to complete until credentialed Daytona E2E tests run successfully.

## Learnings
- Shared workflow memory says task_05 added a Daytona SSH non-PTY validation harness, but the live credentialed gate remains blocked until `DAYTONA_API_KEY=... go test -tags integration ./internal/sandbox/daytona -run TestDaytonaSSHNonPTYValidation -count=1 -v` passes.
- `_tasks.md` currently marks task_05 as pending even though task_06 depends on it.
- Local verification passed after implementation: `make verify` exited 0 and reported `DONE 4254 tests in 24.242s`; `go test -cover ./internal/sandbox/daytona` reported 80.0% statement coverage.
- Tagged Daytona integration tests currently skip live cases because `DAYTONA_API_KEY` is missing: `TestDaytonaProviderIntegrationFullLifecycle` and `TestDaytonaSSHNonPTYValidation` both skipped under `go test -tags integration -v ./internal/sandbox/daytona`.
- The first `make verify` run failed because `modernize` applied mechanical Go fixes and requested a rerun; the second `make verify` run passed.

## Files / Surfaces
- Implemented surface: `internal/sandbox/daytona/`, Daytona provider registry wiring in `internal/daemon/boot.go`, network required policy propagation in config/environment types, and `go.mod`/`go.sum` Daytona SDK + SSH dependencies.
- Existing state includes `internal/sandbox/daytona/doc.go`, `ssh_validation_test.go`, and `VALIDATION.md` from task_05.

## Errors / Corrections
- `golangci-lint run ./internal/sandbox/daytona` initially flagged test formatting/noctx/misspell issues; fixed with gofmt, `net.ListenConfig`, and spelling correction.
- `make verify` modernize rewrote four Daytona Go files during the first run; rerunning the full gate passed.

## Ready for Next Run
- If credentials become available, run the task_05 credentialed validation before claiming task_06 completion.
- Then run the task_06 Daytona E2E suite with real credentials/config (`DAYTONA_API_KEY` plus image or snapshot inputs), update task tracking, and only commit after a fresh full `make verify`.
