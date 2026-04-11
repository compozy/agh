# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build task 02 foundation only: `[network]` config/defaults/validation, home-path audit surface, embedded NATS transport primitives with in-memory token, audit file + globaldb persistence, and required tests.

## Important Decisions
- Treat the existing PRD, tech spec, and ADRs as the approved design baseline; no separate design branch is needed for this run.
- Preserve unrelated task-tracking edits already present in the worktree until task 02 is fully implemented and verified.
- Use `store.NetworkAuditEntry` plus `globaldb` write/list helpers as the persisted audit shape, with `internal/network/audit.go` normalizing once and mirroring the exact same entry to file and database sinks.
- Keep daemon boot wiring out of task 02; this task stops at reusable config/home/store/network foundations.

## Learnings
- Current repo state includes protocol-level `internal/network` files from task 01, but no `transport.go`, no `audit.go`, no config/home network surface, and no `network_audit_log` schema yet.
- Root `AGENTS.md` is the only backend-scoped AGENTS file for the files in this task.
- Embedded NATS works cleanly with `server.NewServer` + `Start` + `ReadyForConnections`, then `nats.Connect(..., nats.InProcessServer(ns), nats.Token(token))`, and shutdown via connection drain before server shutdown.
- Package coverage after implementation: `internal/config` 84.0%, `internal/store/globaldb` 80.5%, `internal/network` 80.0%.
- `go mod tidy` is needed after adding the transport packages so `github.com/nats-io/nats-server/v2` and `github.com/nats-io/nats.go` land in the direct dependency block before final verification.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/home.go`
- `internal/config/{config_test.go,home_test.go,merge_test.go}`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/{global_db_network_audit.go,global_db_network_audit_test.go}`
- `internal/network/`
- `internal/network/{transport.go,audit.go,transport_test.go,audit_test.go,transport_integration_test.go}`
- `internal/store/{store.go,types.go}`
- `go.mod`
- `go.sum`

## Errors / Corrections
- Self-review found the NATS transport modules still marked indirect after the initial dependency add; corrected with `go mod tidy` and reran the full verification plus task-specific integration/coverage commands.

## Ready for Next Run
- Task 02 is complete. Local code commit: `f9ad1be` (`feat: add network transport and audit foundation`). Task-tracking and memory files remain intentionally unstaged.
