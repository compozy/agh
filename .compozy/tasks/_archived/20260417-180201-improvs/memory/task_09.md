# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the `internal/config` improvements pass with mandatory inventories, co-located benchmarks, in-scope fixes, report evidence, and a clean `make verify`.

## Important Decisions
- Replace `godotenv.Load` with a scoped dotenv lookup so workspace `.env` data can influence only the active config load.
- Keep public home-resolution APIs stable by adding internal env-aware helpers (`resolveHomeDir`, `resolveHomePaths`) and routing package-internal validation through them.
- Collapse provider/agent/global MCP server merging into one internal multi-layer merge helper instead of nested `MergeMCPServers` calls.
- Record `ubs` as `not-run` because this session exposes skill instructions but no runnable UBS skill interface.

## Learnings
- `internal/config` has no production goroutines, channels, mutexes, or `select` statements, so the concurrency audit is inventory-only for this package.
- `go test -cover ./internal/config` reached `85.0%` statement coverage after the new dotenv regression tests.
- Benchmark signal was concentrated in `ResolveAgent`; the MCP merge collapse improved `BenchmarkResolveAgentMergedMCPServers` from `39147.4 ns/op, 91590.4 B/op` to `28127.4 ns/op, 60623.0 B/op`.
- The other benchmarked candidates (`LoadForHome`, `ParseMCPServersJSON`, `HookDeclarations`) stayed effectively flat in this pass.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/home.go`
- `internal/config/automation.go`
- `internal/config/bootstrap.go`
- `internal/config/provider.go`
- `internal/config/config_test.go`
- `internal/config/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/config.md`

## Errors / Corrections
- New regression tests first proved the bug: `Load` mutated process `AGH_HOME`, and `LoadForHome` let webhook secret resolution bleed from one workspace into another.
- The first full `make verify` run tripped a non-reproducing race in `extensions/bridges/teams`; repeated package-local reruns and a clean detached-`HEAD` reproduction passed, and the next full `make verify` run succeeded.

## Ready for Next Run
- No task-local follow-up is required once the verified report, tracking files, and commit are in place.
