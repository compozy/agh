# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Wire daemon boot to construct a resolver from the global registry, inject it into `session.Manager`, switch dream consolidation to workspace IDs/refs, and keep daemon-package verification green.

## Important Decisions
- `internal/daemon` now builds the resolver from the already-open global registry plus daemon `HomePaths`, logger, and a config loader that calls `config.LoadForHome(..., WithWorkspaceRoot(rootDir))`.
- Dream consolidation treats workspace refs as durable IDs. Explicit refs/paths are normalized through the resolver, while recent-workspace selection reads only `SessionInfo.WorkspaceID`.
- Dream session creation now passes resolved workspace IDs through `session.CreateOpts.Workspace` and leaves `WorkspacePath` empty.

## Learnings
- Resolver-backed daemon tests need canonical root comparison because macOS temp directories may resolve under `/private/...`.
- The daemon boot test surface needs a registry fake that implements both `store.SessionRegistry` and `workspace.WorkspaceStore`.

## Files / Surfaces
- `internal/config/config.go`
- `internal/daemon/daemon.go`
- `internal/daemon/daemon_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/session/manager.go`
- `internal/session/session.go`

## Errors / Corrections
- The first `LoadForHome` refactor resolved daemon home too early and broke `.env`-driven `AGH_HOME`; fixing it required preserving the original `.env` load order before final home-path resolution.
- Full verification exposed a real `Prompt()`/`Stop()` race in `internal/session`; the final fix serializes in-flight prompt setup against stop instead of weakening the test.

## Ready for Next Run
- Verification passed with `go test ./internal/daemon -count=1`, `go test -tags integration ./internal/daemon -count=1`, `go test ./internal/daemon -cover -count=1` (`80.5%`), and `make verify`.
- Local code commit: `3fa0601` (`feat: wire daemon workspace resolver`).
- Post-commit `make verify` also passed cleanly.
