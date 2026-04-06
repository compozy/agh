# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement `agh workspace` add/list/info/edit/remove over the daemon UDS client and extend `agh session` with `--workspace`, `--cwd`, default CWD auto-register, and list filtering.
- Leave task tracking files updated after clean verification; keep tracking files out of the auto-commit because the PRD directory is untracked.

## Important Decisions
- Reused the existing UDS `/api/workspaces` and workspace-aware `/api/sessions` contract by extending `internal/cli/client.go` instead of adding any local filesystem workspace logic.
- Enforced the session input contract in CLI code: `--workspace` and `--cwd` are mutually exclusive; when neither is supplied, `session new` sends the caller CWD as `workspace_path`.
- Did not add `--force-refresh`: the resolver cache bust hook exists only as `workspace.Resolver.Invalidate` and is not exposed through the current transport surface.
- Expanded `internal/cli/install_test.go` only to raise package-wide `internal/cli` coverage above the task target without changing production behavior.

## Learnings
- `workspace edit` must fetch the current workspace first because the transport PATCH shape only accepts the full `add_dirs` array, while the CLI UX is additive/removal-based.
- `go test -cover -tags integration ./internal/cli` reached 81.1% coverage after adding workspace command tests, a daemon-backed workspace/session integration test, and the existing-install wizard unit coverage.

## Files / Surfaces
- `internal/cli/root.go`
- `internal/cli/session.go`
- `internal/cli/client.go`
- `internal/cli/workspace.go`
- `internal/cli/helpers_test.go`
- `internal/cli/session_test.go`
- `internal/cli/workspace_test.go`
- `internal/cli/client_test.go`
- `internal/cli/install_test.go`
- `internal/cli/format_test.go`
- `internal/cli/cli_integration_test.go`

## Errors / Corrections
- Initial CLI package coverage stayed below the 80% target; corrected by adding human/toon workspace renderer tests plus install wizard model/formatter tests instead of weakening the target.

## Ready for Next Run
- Verification evidence: `go test ./internal/cli`, `go test -tags integration ./internal/cli`, `go test -cover -tags integration ./internal/cli` (81.1%), and `make verify` all passed after the final edits.
