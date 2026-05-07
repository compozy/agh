# Refacs 023: `internal/e2elane`

## Scope

- Package: `github.com/pedronauck/agh/internal/e2elane`
- Iteration: 023
- Goal: deep refactoring and performance audit for E2E lane planning and the immediate Mage runner that consumes those plans.
- Subagents:
  - Read-only refactoring/correctness audit for `internal/e2elane`.
  - Read-only performance/concurrency audit for `internal/e2elane`.

## Baseline

Initial package state:

```bash
rtk go test ./internal/e2elane -count=1
rtk proxy go test ./internal/e2elane -cover -count=1
rtk golangci-lint run ./internal/e2elane
rtk env CGO_ENABLED=1 go test -race ./internal/e2elane -count=1
rtk go test -tags integration ./internal/e2elane -count=1
rtk proxy go test ./internal/e2elane -run '^$' -bench . -benchmem -count=3
```

Observed baseline:

- Package tests: `32 passed in 1 packages`.
- Package lint: no issues.
- Package coverage usually reported `91.7% of statements`, but one concurrent baseline run failed before coverage completed because the package test shelling out to `make help` raced Mage's generated `mage_output_file.go`.
- Race package tests: passed.
- Integration-tag package tests: passed.
- Bench command passed but found no benchmarks.
- CPU and memory profiles from the performance subagent showed no useful `PlanForLane` hotspot; the package is too small, and real lane cost is in subprocess build/test/browser execution.

## Findings

### P1: package tests were not hermetic and could race Mage's generated mainfile

`internal/e2elane/command_wiring_test.go` executed `make -n` and `make help`. `make help` delegates to `mage -l`, and Mage generates `mage_output_file.go` in the repo root during compilation, then removes it. Running several package tests concurrently can make one Mage process remove another process's generated file.

Impact: `go test ./internal/e2elane` could fail nondeterministically with `stat mage_output_file.go: no such file or directory`, even though lane mapping was correct.

Root cause: a tiny package-level test depended on an external build tool with shared repo-root state.

### P1: E2E lane binaries built by Mage leaked temp directories

`resolveOrBuildLaneBinary` created `agh-e2e-lane-*` temp directories for the daemon and mock ACP driver when no override env var was provided. The function returned only the binary path, so `runE2ELane` had no cleanup handle.

Impact: every `make test-e2e-*` execution that built lane binaries left one or two temp directories behind.

### P2: binary override paths failed late and far from the configuration error

`AGH_TEST_DAEMON_BIN` and `AGH_TEST_ACPMOCK_DRIVER_BIN` overrides were normalized to absolute paths but not validated. Missing files, directories, and non-executable files failed later inside E2E startup instead of at lane environment preparation.

Impact: operator and CI errors were harder to diagnose, and some Unix symlink paths could defer failure past the actual bad override.

### P2: primary lane mapping tests mirrored production globals

The main `PlanForLane` mapping test used `runtimeGoSuites`, `daemonServedWebSuites`, and `nightlyGoSuites` as expected values. Those are the same package globals that power production behavior.

Impact: deleting a package or changing a run selector in the production globals could update the test expectation automatically. Focused tests caught some cases, but not the whole lane matrix.

### P3: not all lane regex constants had representative compile/match coverage

The tests only compiled and checked the daemon runtime/nightly patterns. HTTP, UDS, harness, and Daytona selectors were only exercised when their lanes ran.

Impact: selector typos could escape the package unit suite and fail later in E2E execution.

## Changes Made

### Hermetic command wiring tests

- Replaced `make -n` subprocess checks with direct Makefile recipe parsing.
- Replaced `make help` subprocess checks with static AST discovery of Mage target functions in `magefile.go`.
- Removed the package test helper that spawned external commands.
- Normalized touched tests to AGH `Should ...` subtest shape.

### Lane contract tests

- Replaced expected values that mirrored production globals with explicit package/run/script literals for runtime, web, combined, and nightly lanes.
- Added representative regex compile/match coverage for:
  - `RuntimeE2EPattern`
  - `NightlyRuntimeE2EPattern`
  - `HTTPTransportE2EPattern`
  - `UDSTransportE2EPattern`
  - `HarnessRuntimeE2EPattern`
  - `DaytonaNightlyE2EPattern`
- Kept defensive-copy coverage for returned Go suite package slices.

### Mage E2E lane cleanup

- Added an `e2eLaneEnv` value object carrying both env values and a cleanup callback.
- Changed `runE2ELane` to defer cleanup after successful lane env preparation and to preserve cleanup errors with `errors.Join`.
- Changed `resolveOrBuildLaneBinary` to return a cleanup function:
  - no-op cleanup for caller-provided override paths;
  - `os.RemoveAll(buildDir)` cleanup for binaries built into `agh-e2e-lane-*` temp dirs.
- Added cleanup-on-build-failure for partially prepared temp dirs.

### Override validation

- Added early validation for lane binary overrides:
  - resolves relative paths to absolute paths;
  - rejects missing paths with the underlying `os.ErrNotExist`;
  - rejects directories with a typed sentinel;
  - rejects non-executable files on non-Windows platforms with a typed sentinel.
- Added Mage-tagged tests for executable override paths, missing overrides, directory overrides, non-executable overrides, and generated temp-dir cleanup.
- Kept the `t.Setenv` tests in a separate serial Mage test file to preserve Go's process-wide environment contract.

## Performance Results

Final focused benchmark command:

```bash
rtk proxy go test ./internal/e2elane -run '^$' -bench . -benchmem -count=3
```

Observed final result:

- The command passed, but the package still has no benchmarks.

Interpretation:

- No production performance optimization was made inside `internal/e2elane`; the performance subagent found no `PlanForLane` hotspot and no score-worthy optimization. `PlanForLane` is called once per Mage lane execution before much heavier build/test/browser subprocesses.
- The implemented work improves correctness, determinism, and resource lifecycle. Any future micro-optimization of defensive slices should first add package-local benchmarks and preserve returned-plan mutation isolation.

## Deferred / Cross-Package Notes

- `internal/testutil/acpmock.DefaultDriverPath` also builds into a temp directory without a cleanup handle. That belongs to the `internal/testutil/acpmock` package iteration.
- E2E binary env var names are duplicated across Mage, Go testutil, and web E2E fixtures. A later harness-contract cleanup could centralize or generate these constants, but it crosses Go/TypeScript package boundaries.
- Parallelizing `runE2ELane` suites was intentionally rejected for this iteration. The current sequential execution is conservative for daemon ports, runtime homes, browser state, and credentialed nightly isolation.

## Validation

Final validation commands:

```bash
rtk go test ./internal/e2elane -count=1
rtk go test ./internal/e2elane -count=20
rtk proxy go test ./internal/e2elane -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/e2elane -count=1
rtk go test -tags integration ./internal/e2elane -count=1
rtk golangci-lint run ./internal/e2elane
rtk proxy go test ./internal/e2elane -run '^$' -bench . -benchmem -count=3
rtk go test -tags mage . -count=1
rtk env CGO_ENABLED=1 go test -race -tags mage . -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/e2elane/command_wiring_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/e2elane/lanes_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py magefile_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py magefile_lane_binary_test.go
rtk make test-e2e-runtime
rtk make verify
```

Observed final results:

- Package tests: `53 passed in 1 packages`.
- Package stress rerun: `1060 passed in 1 packages` with `-count=20`.
- Package coverage: `91.7% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `53 passed in 1 packages`.
- Package lint: no issues.
- Benchmark command: passed; no benchmarks registered.
- Mage-tagged root tests: `18 passed in 1 packages`.
- Mage-tagged race tests: passed.
- AGH test-shape checks: passed for all touched test files.
- Runtime E2E lane:
  - `internal/daemon`: `24 tests` passed.
  - `internal/api/httpapi`: `8 tests` passed.
  - `internal/api/udsapi`: `14 tests` passed.
  - `internal/testutil/e2e`: `6 tests` passed.
- `make verify`: passed.
