# Improvements Report — internal/extensiontest

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/extensiontest/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/extensiontest | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 20 | `TestHarnessIntegrationTelegramReferenceConformance` | `internal/extensiontest/bridge_adapter_harness_integration_test.go:25` |
| 14 | `validateStateConformance` | `internal/extensiontest/bridge_adapter_harness.go:466` |
| 12 | `TestHarnessHelperUtilities` | `internal/extensiontest/bridge_adapter_harness_test.go:146` |
| 10 | `validateDeliveryRecord` | `internal/extensiontest/bridge_adapter_harness.go:569` |
| 10 | `(*ScriptedPromptDriver).Prompt` | `internal/extensiontest/bridge_adapter_harness.go:752` |
| 9 | `validateOwnershipConformance` | `internal/extensiontest/bridge_adapter_harness.go:374` |
| 9 | `validateConformanceMatrixEntry` | `internal/extensiontest/bridge_conformance_matrix.go:292` |
| 9 | `seedManagedInstanceOutcomes` | `internal/extensiontest/bridge_conformance_matrix.go:212` |
| 9 | `TestSummarizeConformanceReportBuildsStableMultiInstanceMatrixRow` | `internal/extensiontest/bridge_conformance_matrix_test.go:11` |
| 8 | `validateDeliveryConformance` | `internal/extensiontest/bridge_adapter_harness.go:532` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/extensiontest/bridge_adapter_harness.go` | 1705 | The harness file mixes contract validation, prompt-driver behavior, observer/runtime construction, filesystem helpers, and polling utilities in one unit. |
| `internal/extensiontest/bridge_conformance_matrix.go` | 456 | Matrix summarization, validation, normalization, and merge helpers are concentrated in one file. |

### Refactoring — Duplication

Output from `dupl -plumbing -t 60 internal/extensiontest`:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/extensiontest/bridge_adapter_harness.go:172-181` | `internal/extensiontest/bridge_conformance_matrix.go:81-90` | `ConformanceError.Error` and `ConformanceMatrixError.Error` still build identical `code: message` strings. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `BuildConformanceMatrix` | `internal/extensiontest/bridge_conformance_matrix.go:115` | The package’s main allocation-heavy normalization/merge path; it trims, clones, sorts, and merges summary rows before validation. | `BenchmarkBuildConformanceMatrix` |
| `(*ScriptedPromptDriver).Prompt` | `internal/extensiontest/bridge_adapter_harness.go:752` | The package’s only goroutine entry point and event emission loop; every scripted prompt call replays the configured event sequence. | `BenchmarkScriptedPromptDriverPrompt` |
| `readJSONLinesFile` | `internal/extensiontest/bridge_adapter_harness.go:1552` | The package’s repeated file-polling/read loop for state, delivery, and ingest markers. | `BenchmarkReadJSONLinesFileStateRecord` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/extensiontest/...` before production fixes.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/extensiontest/...` after all `internal/extensiontest/` changes land.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkBuildConformanceMatrix` | 26548.2 | 30632 | 25533.6 | 31144 | fixed-with-benchmark |
| `BenchmarkScriptedPromptDriverPrompt` | 868.7 | 1168 | 811.3 | 992 | fixed-with-benchmark |
| `BenchmarkReadJSONLinesFileStateRecord` | 390961.4 | 316344 | 373551.0 | 204344 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/extensiontest/bridge_adapter_harness.go:768` | `(*ScriptedPromptDriver).Prompt` | The goroutine closes `events` on exit and returns when the script completes or `ctx.Done()` fires. | Deterministic prompt replay loop used by the in-process harness driver. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/extensiontest/bridge_adapter_harness.go:713` | 0 | `scriptedPromptProcess` | `(*ScriptedPromptDriver).Stop` via `sync.Once` | `session.AgentProcess.Wait` / `Wait` closure | Process-lifecycle done channel for the fake session process. |
| `internal/extensiontest/bridge_adapter_harness.go:767` | `len(script)` | `(*ScriptedPromptDriver).Prompt` | The prompt goroutine closes it on return. | Benchmark/test consumers range until close. | Buffered event stream for deterministic prompt replay. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/extensiontest/bridge_adapter_harness.go:707` | write-heavy | `ScriptedPromptDriver.processes` and `ScriptedPromptDriver.prompts` | Serializes prompt bookkeeping and fake process registration. |

### Concurrency — Select Audit

- `internal/extensiontest/bridge_adapter_harness.go:773` and `:792` are `ctx.Done()`-aware prompt-loop selects.
- `internal/extensiontest/bridge_adapter_harness.go:777` is the timer-drain select used only after `timer.Stop()` fails.
- `internal/extensiontest/bridge_adapter_harness_integration_test.go:328` is a timer-bounded test assertion select; it is input-bounded rather than long-lived orchestration.

### Security — Threat Model

- Trust boundaries:
  - Unit and integration tests call exported harness and matrix helpers in `internal/extensiontest/`.
  - The reference adapter subprocess writes marker files that this package reads back for conformance assertions.
  - `buildHarnessTelegramReferenceAdapter` shells out to `go build` using a repo-root path derived from the checked-in test file location.
- Attacker capabilities:
  - A malformed test fixture or adapter subprocess can emit arbitrary JSON into marker files and arbitrary provider/platform/instance identifiers into reports.
  - A test caller can provide arbitrary `HarnessConfig` values including `ExtensionDir`, `ManagedInstances`, `ProviderConfig`, and `ExtraEnv`.
  - There is no network-facing production endpoint in this package; all inputs are local test/runtime inputs.
- In-scope assets:
  - Deterministic harness behavior, marker parsing, and correct conformance/matrix summarization.
  - Isolation of test-scoped env vars, temp files, and repo-local build commands.
  - Integrity of provider/platform/instance grouping used by matrix validation.
- Out-of-scope:
  - Production daemon/package security after requests leave this test helper package.
  - A malicious local operator with direct control of the repository checkout, filesystem, or process environment.
  - The security posture of the example adapter binary itself beyond what this harness validates.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/extensiontest/bridge_adapter_harness.go:859-924` | Test-controlled `HarnessConfig` (`ExtensionDir`, managed instances, provider config, env) | `ExtensionDir` must be non-empty; temp-home/temp-workspace isolation keeps writes under test-owned directories. | Manifest install, env setup, bridge-instance creation, and manager/session wiring. | LOW — rejected as a test-only local input surface, not a production trust boundary. |
| `internal/extensiontest/bridge_adapter_harness.go:1283-1286` | Test-controlled fake inbound update payload | JSON encoding only; no semantic validation in this package. | Marker file append for the reference adapter to ingest. | LOW — rejected because it writes only to temp marker files under the harness root. |
| `internal/extensiontest/bridge_adapter_harness.go:1545-1574` | Adapter-written JSON / JSONL marker files | `json.Unmarshal` validation returns errors to the polling/report helpers on malformed input. | Conformance report assembly and wait predicates. | LOW — rejected because malformed data causes local test failures, not privilege or data-boundary escalation. |
| `internal/extensiontest/bridge_conformance_matrix.go:115-150` | Provider/platform/instance identifiers from conformance reports | String trimming plus target normalization. | Matrix merge/grouping used by `ValidateConformanceMatrix`. | LOW — rejected because this is a local test-matrix integrity surface rather than a production security boundary; structured keys now keep distinct provider/platform pairs separate. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `EXTTEST-REF-001` | refactoring-analysis | low | `internal/extensiontest/bridge_adapter_harness.go:1305-1329` | The three JSONL wait helpers duplicated the same poll/read/predicate logic. | fixed |
| `EXTTEST-REF-002` | refactoring-analysis | medium | `internal/extensiontest/bridge_adapter_harness.go:34-1705` | `bridge_adapter_harness.go` still concentrates contract validation, runtime wiring, polling helpers, and file utilities in one 1.7k LOC unit. | deferred — splitting the harness by concern would widen this low-complexity pass into a larger test-utility redesign. |
| `EXTTEST-REF-003` | refactoring-analysis | low | `internal/extensiontest/bridge_adapter_harness.go:172-181` | `ConformanceError.Error` still duplicates the matrix error formatter. | wontfix — the remaining 10-line duplication is too small to justify an abstraction in a test helper package. |
| `EXTTEST-OPT-001` | extreme-software-optimization | medium | `internal/extensiontest/bridge_conformance_matrix.go:109-139` | `BuildConformanceMatrix` used a delimiter-encoded string key, which could merge distinct provider/platform pairs containing `|` and performed extra key allocations on every row. | fixed |
| `EXTTEST-OPT-002` | extreme-software-optimization | low | `internal/extensiontest/bridge_adapter_harness.go:760-761` | `(*ScriptedPromptDriver).Prompt` copied the already-immutable script slice on every prompt invocation. | fixed |
| `EXTTEST-OPT-003` | extreme-software-optimization | low | `internal/extensiontest/bridge_adapter_harness.go:1552-1568` | `readJSONLinesFile` converted the whole payload to a string, split it, then converted each line back to bytes before unmarshalling. | fixed |

## Per-Skill Notes

### refactoring-analysis
- Removed the duplicated JSONL polling logic by routing `WaitForStates`, `WaitForDeliveries`, and `WaitForIngests` through `waitForJSONLinesCondition`.
- The remaining dominant structural issue is file concentration: `bridge_adapter_harness.go` still combines multiple concerns in a single 1.7k LOC unit.
- The only surviving `dupl` hit is the shared 10-line error-string formatter; I left that as `wontfix` because a helper would add indirection without meaningful maintenance payoff here.

### extreme-software-optimization
- Added `internal/extensiontest/perf_bench_test.go` before production changes and re-ran the required benchmark command after every optimization change.
- `BuildConformanceMatrix` now uses a structured `providerPlatformKey`, fixing the `|` delimiter collision bug while improving mean runtime from `26548.2 ns/op` to `25533.6 ns/op` and cutting allocs from `232` to `184` per call. `B/op` rose slightly (`30632` -> `31144`) because the map now stores a larger two-field key, but the change is still a net win on latency, allocation count, and correctness.
- `(*ScriptedPromptDriver).Prompt` now reuses the immutable pre-cloned script slice held by the driver, improving mean runtime from `868.7 ns/op` to `811.3 ns/op` and reducing per-call memory from `1168 B/op` / `4 allocs/op` to `992 B/op` / `3 allocs/op`.
- `readJSONLinesFile` now keeps the payload in bytes and splits on newline bytes directly, improving mean runtime from `390961.4 ns/op` to `373551.0 ns/op` while reducing memory from `316344 B/op` / `2441 allocs/op` to `204344 B/op` / `2311 allocs/op`.

### ubs
- `not-run` due missing skill-runner interface in this session; no manual substitute will be used.

### deadlock-finder-and-fixer
- Inventory complete; no confirmed deadlock or goroutine-leak path is present in the current harness code.

### security-review
- The package has no external production exposure; all reviewed surfaces are local test/runtime inputs. No HIGH or MEDIUM source-to-sink security finding survived the threat-model review.

## Deferred Items (carry forward)

- None yet.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
✓ built in 375ms
0 issues.
✓  internal/extensiontest (1.077s)
✓  internal/extension (7.6s)
✓  internal/daemon (8.435s)
DONE 4467 tests in 9.567s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:
- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final rerun after all package changes.
