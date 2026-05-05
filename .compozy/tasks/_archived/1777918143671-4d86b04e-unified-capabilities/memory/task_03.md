# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Preserve former recipe transfer behavior under `kind:"capability"` in `internal/network`, with explicit regression coverage for broadcast delivery, directed lifecycle progression, mixed `direct` + `capability` interaction flow, terminal handling, and audit labeling.

## Important Decisions

- Treat the current branch as partially ahead of the task spec instead of assuming task 03 was fully unimplemented; validate the actual gap against code and tests first.
- Fix sender-side interaction lifecycle tracking in `router.Send` rather than only adding tests. True multi-router capability flows need outbound lifecycle state so later `trace` / `receipt` messages are not ignored by the sender router.
- Preflight outbound lifecycle messages against current local interaction state and reject post-terminal follow-up sends with `ErrInteractionClosed`.

## Learnings

- Task 02 already removed `recipe` from runtime network kinds. The remaining task-03 gap was not stale recipe naming in code; it was missing sender-side lifecycle bookkeeping plus missing explicit integration/audit coverage.
- Before the fix, distributed capability interactions could diverge from single-router tests because only inbound receive paths updated interaction state. That meant a sender router could ignore later lifecycle updates from another peer.
- `internal/network` still meets the task coverage target after the new tests: `go test -cover ./internal/network` = 81.9%, `go test -cover -tags integration ./internal/network` = 82.3%.

## Files / Surfaces

- `internal/network/router.go`
- `internal/network/router_test.go`
- `internal/network/router_integration_test.go`
- `internal/network/audit_test.go`

## Errors / Corrections

- `make verify` is currently blocked by an unrelated web lint error in `web/src/systems/tasks/components/tasks-multi-agent-panel.tsx`: unused `Section` import. This file is outside task 03 scope and already part of unrelated worktree changes.

## Ready for Next Run

- Task-scoped verification is green:
  - `go test ./internal/network/...`
  - `go test -tags integration ./internal/network/...`
  - `go test -cover ./internal/network`
  - `go test -cover -tags integration ./internal/network`
- Repo-wide completion, tracking updates, and commit are blocked until the unrelated `make verify` failure is resolved or explicitly authorized to fix.
