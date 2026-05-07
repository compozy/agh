# Refacs 019: `internal/config`

## Scope

- Package: `github.com/pedronauck/agh/internal/config`
- Iteration: 019
- Goal: deep refactoring and performance audit for AGH configuration loading, merging, validation, sidecars, and persistence helpers.
- Subagents:
  - Read-only refactoring audit for `internal/config`.
  - Read-only performance audit for `internal/config`.

## Baseline

Commands run before changes:

```bash
rtk go test ./internal/config -count=1
rtk golangci-lint run ./internal/config
rtk proxy go test ./internal/config -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/config -count=1
rtk go test -tags integration ./internal/config -count=1
rtk proxy go test ./internal/config -run '^$' -bench . -benchmem -count=5
```

Observed baseline:

- Package tests passed: `493` tests.
- Package lint passed before edits.
- Coverage: `79.4% of statements`, below the package target floor.
- Race package tests passed.
- Integration-tag package tests passed: `503` tests.
- Full benchmark suite failed because stale benchmark fixtures still placed `TOKEN` under regular MCP `env`, while production validation requires secret-shaped keys under `secret_env`.
- Usable subset baseline:
  - `BenchmarkResolveAgentMergedMCPServers`: about `45-47 us/op`, `93020 B/op`, `510 allocs/op`.
  - `BenchmarkHookDeclarationsNormalization`: about `66-69 us/op`, `124640 B/op`, `484 allocs/op`.

## Findings

### P1: Adjacent config sidecars followed symlinks inconsistently

`.env`, `mcp.json`, and capability catalog single-file paths were loaded through `os.Stat` / `os.ReadFile` paths that followed symlinks. `.env` repair already rejected symlinks, so load and repair semantics diverged on the same file.

Impact: workspace-controlled sidecars could point outside the expected adjacent-file surface, and load/inspect/repair behavior was inconsistent.

### P1: TOML MCP overlay merge used raw names while provider/MCP JSON paths used normalized names

`MergeMCPServers` and `OverrideMCPServers` indexed names with `normalizeMCPServerName`, but `applyMCPServerOverlays` indexed base TOML servers by raw `server.Name`. A base entry named `"  github  "` and an overlay named `"github"` could fail to collide in TOML while colliding correctly elsewhere.

### P1: `EditAgentDefFile` used direct `os.WriteFile`

Config persistence used temp file, fsync, rename, and directory sync through `writePersistedFile`, but agent definition edits rewrote `AGENT.md` directly. A crash or interrupted write could leave a partially written agent file.

### P1: Benchmarks were stale against current MCP secret validation

The package benchmark generator used `"env": {"TOKEN": ...}` even though current validation rejects secret-shaped regular env keys. This made the benchmark suite fail before measuring the package.

### P2: Hook declaration normalization allocated redundant deep clones

`HookDeclarations` built a cloned `raw` slice and then called `hooks.NormalizeHookDecl`, which already sanitizes and clones returned declaration data. This doubled slice/map copy work before normalization.

### P2: MCP collision merge cloned already-cloned base entries

`mergeMCPServerLayers` begins by deep-cloning the base slice, then collision merges called `mergeMCPServer`, which cloned the already cloned entry again before applying overlay fields.

### P2: Persistence helpers were buried in the TOML editor file

`persistence.go` mixed target selection, TOML AST editing, rendering, optional reads, and atomic file writes. This package iteration did not do a full decomposition, but the reusable file I/O helpers were extracted to reduce the most immediate coupling.

### P2: Cleanup errors were discarded in production writers

Failure paths in `.env` repair and persisted config writes ignored close/remove/directory-close errors. This violated AGH production-code discipline and could hide real filesystem failures.

## Changes Made

### Correctness and hardening

- Added regular-file/no-symlink reads for optional sidecars:
  - `.env` load now uses `os.Lstat`, rejects symlinks, rejects directories, and returns structured `ErrDotEnvUnsupported` diagnostics for unsupported `.env` paths.
  - `mcp.json` load now uses the same regular-file helper and treats missing files as absent.
  - Capability catalog file and directory layout detection now uses `os.Lstat` and rejects symlinked reserved paths.
  - Capability catalog file reads and capability definition reads now refuse symlinks at read time too.
- Removed the extra `os.Stat` from `.env` lookup so load behavior is centralized in `readDotEnvFile`.
- Changed `EditAgentDefFile` to write through `writePersistedFile`, giving agent definition edits the same atomic temp-file, fsync, rename, directory-sync, and private-mode behavior as config writes.
- Fixed TOML MCP overlay merge to index and update by normalized MCP server names.
- Fixed stale benchmark fixtures by moving benchmark `TOKEN` data from MCP regular `env` to `secret_env`.
- Removed production `_ =` cleanup discards in config writers:
  - `.env` repair now closes temp files on chmod/write errors and joins close errors.
  - persisted file writes now report temp removal errors when relevant.
  - persisted file writes now close temp files on chmod/write/sync errors and join close errors.
  - directory sync now joins close errors.
- Kept `AutomationTrigger.validateWithEnv` behavior unchanged while making the unused private env-lookup parameter explicit.

### Refactoring

- Extracted reusable file read/write helpers to `file_io.go`.
- Added `closeFileAfterError` for consistent close-on-failure behavior.
- Split hook declaration normalization into a direct iterator plus `appendNormalizedHookDecl`, removing the intermediate cloned `raw` slice while preserving declaration indices and error messages.
- Split MCP collision mutation into `mergeMCPServerInto`, so `mergeMCPServerLayers` mutates the already cloned destination entry instead of deep-cloning it again.

### Tests

Added `config_refac_test.go` covering:

- TOML MCP overlay name normalization and alias isolation.
- MCP collision merge alias isolation for args, env, secret env, and auth scopes.
- Hook declaration ordering, disabled filtering, and input alias isolation after direct normalization.
- `EditAgentDefFile` private-mode rewrite through the atomic writer.
- `.env`, `mcp.json`, and capability catalog symlink rejection without reading target contents.

The new tests raised package coverage from `79.4%` to `80.7%`.

## Performance Results

The full benchmark suite now runs successfully after fixture repair:

```bash
rtk proxy go test ./internal/config -run '^$' -bench . -benchmem -count=5
```

Final focused benchmark command after the file split:

```bash
rtk proxy go test ./internal/config -run '^$' -bench 'Benchmark(ResolveAgentMergedMCPServers|HookDeclarationsNormalization)$' -benchmem -count=5
```

Final focused results:

- `BenchmarkResolveAgentMergedMCPServers`: about `43.7-64.4 us/op`, `89811-89817 B/op`, `486 allocs/op`.
- `BenchmarkHookDeclarationsNormalization`: about `49.9-50.9 us/op`, `57704-57706 B/op`, `337 allocs/op`.

Compared with usable baseline:

- MCP merge allocations dropped from about `93020 B/op` and `510 allocs/op` to about `89815 B/op` and `486 allocs/op`.
- Hook normalization allocations dropped from about `124640 B/op` and `484 allocs/op` to about `57705 B/op` and `337 allocs/op`.
- Timing is noisy on the local machine, but the allocation reductions match the pprof evidence from the performance subagent.

## Deferred / Cross-Package Notes

- `internal/hooks` matcher validation still allocates on the valid path while validating hook matchers. It is on the `HookDeclarations` hot path but belongs to `internal/hooks`, so it was deferred to that package's iteration.
- Full decomposition of `config.go`, `merge.go`, and the remaining TOML editor logic in `persistence.go` is still warranted, but the immediate behavior-carrying refactors landed here. A full domain split should be a later structural pass if more config lifecycle work accumulates.
- `internal/api/spec` still has the hard-coded `hookEventFamilyValues` list recorded in iteration 018. It remains a cross-package deferred item before the overall refacs goal is complete.

## Validation

Final validation commands:

```bash
rtk go test ./internal/config -count=1
rtk golangci-lint run ./internal/config
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/config/config_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/config/perf_bench_test.go
rtk proxy go test ./internal/config -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/config -count=1
rtk go test -tags integration ./internal/config -count=1
rtk proxy go test ./internal/config -run '^$' -bench . -benchmem -count=5
rtk proxy go test ./internal/config -run '^$' -bench 'Benchmark(ResolveAgentMergedMCPServers|HookDeclarationsNormalization)$' -benchmem -count=5
rtk go test ./internal/config ./internal/skills ./internal/daemon ./internal/cli -count=1
rtk go test -tags integration ./internal/config ./internal/skills -count=1
rtk rg -n "_\\s*=\\s*.*(Close|Remove|Write|Sync|Read|Run|Do|Wait)|_\\s*=\\s*[^,]+" internal/config --glob '*.go' --glob '!*_test.go'
rtk make verify
```

Observed final results:

- Package tests: `505 passed in 1 packages`.
- Package lint: no issues.
- New AGH test-shape check: passed for `internal/config/config_refac_test.go`.
- Existing benchmark test-shape check: passed for `internal/config/perf_bench_test.go`.
- Package coverage: `80.7% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `515 passed in 1 packages`.
- Full benchmark suite: passed.
- Focused final benchmarks: passed with the allocation reductions listed above.
- Direct dependent package set (`config`, `skills`, `daemon`, `cli`): `1989 passed in 4 packages`.
- Direct integration dependent set (`config`, `skills`): `698 passed in 2 packages`.
- Production `_ =` cleanup discard scan for `internal/config`: no matches.
- `make verify`: passed after the final file split.
