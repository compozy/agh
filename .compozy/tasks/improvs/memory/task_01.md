# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the `internal/acp` improvements pass with inventories, report, in-package fixes, fresh verification, tracking updates, and one local commit.
- Status: completed and committed locally as `9b354aba`.

## Important Decisions
- Keep code edits confined to `internal/acp/`; treat report, ledger, and workflow/tracking files as non-code task artifacts.
- Use the existing `internal/acp/acp_bench_test.go` benchmarks as the baseline optimization harness, adding more only if a newly identified hot-path candidate requires it.
- Treat terminal output buffering as the strongest current optimization candidate based on benchmark + pprof evidence.
- Keep tracking-only task files out of the automatic commit; commit only package/report deliverables unless repository requirements change.

## Learnings
- `gocyclo` and `dupl` are installed and usable in this environment.
- Current package coverage baseline is `81.5%`.
- Baseline benchmarks show `BenchmarkManagedTerminalAppendOutputOverflow` at roughly `7.4µs–19.9µs` with `155648 B/op`, and pprof attributes nearly all alloc space to `(*managedTerminal).appendOutput` plus `trimUTF8LeadingBytes`.
- `BenchmarkHandleSessionUpdateAgentMessage` is allocation-heavy (`3544 B/op`, `70 allocs/op`) but pprof shows most cost inside repeated JSON unmarshalling through the ACP SDK, making it a weaker immediate optimization target.
- Reworking terminal output windowing in `handlers.go` eliminated the overflow benchmark allocations (`155648 B/op` -> `0 B/op`) and reduced runtime to roughly `574 ns/op` in the post-fix benchmark average.
- The old UTF-8 trimming logic could drop the entire retained terminal buffer when overflow ended with a partial multibyte rune; new tests now lock that behavior down.
- `make verify` passed after the fix and report updates.
- The deliverable commit for this task is `9b354aba` (`refactor: acp improvements pass`).
- A fresh post-commit `make verify` rerun also passed with exit code `0`.

## Files / Surfaces
- `internal/acp/client.go`
- `internal/acp/handlers.go`
- `internal/acp/permission.go`
- `internal/acp/tool_host.go`
- `internal/acp/types.go`
- `internal/acp/launcher.go`
- `internal/acp/acp_bench_test.go`
- `internal/acp/handlers_test.go`
- `.compozy/tasks/improvs/reports/acp.md`

## Errors / Corrections
- No execution blockers so far; unrelated worktree changes already existed and must be left untouched.
- UBS could not be executed because this environment exposes skill instructions but no dedicated UBS invocation tool; recorded as `not-run` in the report with the literal limitation.

## Ready for Next Run
- If a later task inspects this package, start from `.compozy/tasks/improvs/reports/acp.md`; tracking files and workflow memory were intentionally left out of the commit.
