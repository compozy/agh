# Task Memory: task_20.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute the five-skill improvements pass for `internal/network/`.
- Produce `.compozy/tasks/improvs/reports/network.md` with all mandatory inventories, benchmark evidence, findings triage, and final `make verify` excerpt.
- Keep production code changes inside `internal/network/`; treat report/tracking/memory files as required task artifacts outside the package.

## Important Decisions
- Reused the established per-package report structure from prior improvements tasks to reduce failure-mode risk.
- Selected three initial hot-path candidates for benchmarking before any fixes: `formatNetworkMessage`, `(*PeerRegistry).ListPeers`, and `networkLogFields`.
- If no dedicated UBS skill runner appears in this session, record UBS as `not-run` with the shared tooling-limitation wording rather than substituting a manual review.
- Left the `manager.go` audit-helper duplicate as `wontfix` because the two short helpers diverge only at the event/log label level and a shared abstraction would add indirection without eliminating a broader pattern.

## Learnings
- `internal/network/` already exceeds the 80% package coverage target (`81.1%` for the main package; `rules` remains uncovered because it is a tiny regex helper package).
- The package has five production goroutine entry points: two heartbeat loops, the delivery worker, the retry waiter, and the transport shutdown waiter.
- `(*PeerRegistry).ListPeers` currently scans all locals/remotes even when a single channel filter is supplied, despite the registry already maintaining per-channel indexes.
- Baseline benchmark medians: `formatNetworkMessage` `22593 ns/op / 96367 B/op / 220 allocs`, filtered `ListPeers` `22673 ns/op / 85560 B/op / 172 allocs`, `networkLogFields` `1174 ns/op / 1945 B/op / 28 allocs`.
- Post-fix benchmark medians: `formatNetworkMessage` `6794 ns/op / 11088 B/op / 47 allocs`, filtered `ListPeers` `10493 ns/op / 8504 B/op / 172 allocs`, `networkLogFields` `1173 ns/op / 1945 B/op / 28 allocs`.
- `make verify` exited `0` after the package changes; the run still emitted the repo-wide `NO_COLOR` and macOS `-bind_at_load` toolchain warnings already seen in prior package tasks.

## Files / Surfaces
- Production files: `router.go`, `manager.go`, `envelope.go`, `peer.go`, `rules/channel.go`, `validate.go`, `lifecycle.go`, `stats.go`, `delivery.go`, `tasks.go`, `audit.go`, `transport.go`.
- Key external production callers: `internal/daemon/boot.go`, `internal/api/core/network.go`, `internal/api/core/network_details.go`, `internal/api/core/interfaces.go`, `internal/api/core/errors.go`, `internal/api/udsapi/routes.go`, `internal/cli/task.go`, `internal/extension/host_api_tasks.go`, `internal/daemon/task_runtime.go`.
- Security-sensitive ingress surfaces: `Router.Receive`, `Router.Send`, `formatNetworkMessage`, task-ingress helpers in `tasks.go`, and audit persistence in `audit.go`.
- Touched implementation surfaces: `internal/network/peer.go`, `internal/network/delivery.go`, and new benchmark coverage in `internal/network/perf_bench_test.go`.

## Errors / Corrections
- No ADR directory exists under `.compozy/tasks/improvs/adrs`; treated as absent context rather than a blocker.
- The session exposes UBS skill instructions only; there is no dedicated skill invocation tool available so far.

## Ready for Next Run
- Next step is task tracking, self-review of the final diff, and one local commit after confirming only task-relevant files are staged.
