# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Refac task 01 is complete in local commit `2d0405e`, and post-commit repository-wide verification passed.
- Refac task 02 is complete in local commit `5a582c4`, and the committed tree passed `make verify` afterward.
- Refac task 03 is complete in local commit `1542208`, and the committed tree passed `make verify` afterward.
- Refac task 04 is complete in local commit `5a60b8a`, and the committed tree passed `make verify` afterward.

## Shared Decisions
- Shared home/path resolution for daemon and CLI should go through exported helpers in `internal/config/home.go` instead of duplicating path normalization or user agents/skills directory logic in consumers.
- Shared low-level helpers introduced by the refactor live in dedicated packages: process helpers in `internal/procutil`, atomic file writes in `internal/fileutil`, and common test helpers in `internal/testutil`.
- Shared filesystem snapshot helpers now live in `internal/filesnap`; later tasks should reuse `filesnap.Snapshot`, `FromPath`, `Equal`, and `Clone` instead of creating local snapshot structs or comparison helpers in domain packages.
- `internal/udsapi` now mirrors `internal/httpapi` naming for route-domain handlers: `sessions.go`, `agents.go`, `observe.go`, `prompt.go`, `daemon.go`, `stream.go`, with payload adapters in `payloads.go` and base handler wiring in `server.go`.
- Shared API transport logic now lives in `internal/apicore`; later tasks should extend `apicore` for cross-transport handlers, payloads, parsing, SSE, and shared error behavior instead of reintroducing `httpapi`/`udsapi` duplication.
- Shared API transport test scaffolding now lives in `internal/apitest`; if transport tests still need local package names, keep those aliases in `*_test.go` wrappers instead of production files.
- CLI list-style output should reuse the generic `listBundle[T]` helper in `internal/cli/format.go` instead of open-coding repeated human/toon row bundle loops.

## Shared Learnings
- Atomic writes that replace persisted metadata must `Sync` the temp file before rename to preserve the durability guarantees already relied on by the store layer.
- Snapshot comparison is the safe registry no-op check for skills reloads; later tasks should avoid reintroducing `reflect.DeepEqual` over mutable skill maps.
- Fresh verification can expose non-functional regressions late in the flow: this task needed an extra command-path CLI test to satisfy the 80% package coverage target and a small nil-context test adjustment for staticcheck before `make verify` would pass.

## Open Risks
- None currently.

## Handoffs
- Later refac tasks can depend on the new utility packages instead of adding fresh local copies of process, atomic-write, or common test helpers.
- Later refac tasks should target the new focused files in `internal/daemon`, `internal/session`, `internal/store`, `internal/workspace`, and `internal/udsapi` instead of recreating monolithic package files.
