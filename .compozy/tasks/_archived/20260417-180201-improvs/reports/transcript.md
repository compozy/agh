# Improvements Report — internal/transcript

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/transcript/transcript_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/transcript | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 17 | `buildToolResult` | `internal/transcript/transcript.go:453` |
| 16 | `TestAssembleLegacyACPEvents` | `internal/transcript/transcript_test.go:12` |
| 10 | `parseLegacyEvent` | `internal/transcript/transcript.go:377` |
| 10 | `TestMarshalAgentEventPreservesRawToolResultShape` | `internal/transcript/transcript_test.go:352` |
| 9 | `extractLegacyContentText` | `internal/transcript/transcript.go:522` |
| 9 | `TestUnmarshalAgentEventRoundTrip` | `internal/transcript/transcript_test.go:424` |
| 9 | `TestParseLooseEventBuildsToolResultFromLoosePayload` | `internal/transcript/transcript_test.go:274` |
| 9 | `TestAssembleReadsCanonicalEnvelopeAndStableOrdering` | `internal/transcript/transcript_test.go:131` |
| 8 | `parseEvent` | `internal/transcript/transcript.go:329` |
| 8 | `applyToolResult` | `internal/transcript/transcript.go:265` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/transcript/transcript.go` | 798 | Single production file still mixes transcript assembly, legacy/loose payload parsing, tool-result normalization, and canonical marshal/unmarshal responsibilities. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/transcript`:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/transcript/transcript_test.go:154-172` | `internal/transcript/transcript_test.go:192-210` | Duplicate canonical fixture setup blocks in the ordering test. The duplication is test-only and small enough that extracting a helper would not materially simplify the package. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `Assemble` | `internal/transcript/transcript.go:109` | Public replay path used by `internal/session/transcript.go:34` after loading persisted session events. It loops over every event, sorts, parses payloads, and materializes message slices. | `BenchmarkAssembleMixedTranscript` |
| `buildToolResult` | `internal/transcript/transcript.go:453` | Allocation-heavy normalization helper used by legacy/loose transcript parsing and `MarshalAgentEvent` when tool-result payloads include `rawOutput` objects. | `BenchmarkBuildToolResultObjectRawOutput` |
| `MarshalAgentEvent` | `internal/transcript/transcript.go:714` | Per-event canonical serialization path used by `internal/session/manager_prompt.go:320` and `internal/extension/bridge_delivery_notifier.go:87`. | `BenchmarkMarshalAgentEventToolResult` |
| `UnmarshalAgentEvent` | `internal/transcript/transcript.go:769` | Per-event decode path used by `internal/extension/host_api.go:1670` when rebuilding prompt projection seed events from stored transcript content. | `BenchmarkUnmarshalAgentEventCanonical` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/transcript/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/transcript/...`

Values below use the median of 5 runs from `/tmp/transcript-bench-before.txt` and `/tmp/transcript-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkAssembleMixedTranscript` | 15411 | 17204 | 14769 | 16475 | fixed-with-benchmark |
| `BenchmarkBuildToolResultObjectRawOutput` | 3192 | 3059 | 1492 | 1433 | fixed-with-benchmark |
| `BenchmarkMarshalAgentEventToolResult` | 6675 | 6105 | 6202 | 5503 | fixed-with-benchmark |
| `BenchmarkUnmarshalAgentEventCanonical` | 3841 | 2152 | 3885 | 2152 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/transcript/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/transcript/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/transcript/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/transcript/`.

### Security — Threat Model

- Trust boundaries:
  - `internal/transcript` is an internal serialization package reached from session persistence (`internal/session/transcript.go:29-34`), session event recording (`internal/session/manager_prompt.go:319-324`), extension bridge delivery fingerprinting (`internal/extension/bridge_delivery_notifier.go:87-89`), and host API replay/projection seeding (`internal/extension/host_api.go:1670-1691`).
  - The package never opens files, executes commands, performs network I/O, or crosses a process boundary itself; it only reshapes already-captured event payloads into structured Go values or JSON strings.
- Attacker capabilities:
  - A user prompt or tool output can influence persisted `store.SessionEvent.Content` and `acp.AgentEvent.Raw` through upstream agent/runtime components.
  - A same-user local actor with direct database access could tamper with stored event payload strings before callers pass them into `Assemble` or `UnmarshalAgentEvent`.
- In-scope assets:
  - Canonical transcript message integrity and stable event ordering.
  - Safe JSON-based serialization/deserialization of transcript payloads without interpreting input as code, paths, or commands.
  - Preservation of tool-result data shape when normalizing raw outputs.
- Out-of-scope:
  - Trustworthiness of the ACP agent subprocess or upstream event recorder beyond the serialized payloads handed to this package.
  - Authorization or validation decisions in higher-level session/extension code before they choose which events to store or replay.
  - Direct database tampering by an operator who already controls the AGH home directory and session store files.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/transcript/transcript.go:329-358` | `store.SessionEvent.Content` values loaded by `internal/session/transcript.go:29-34`. | `strings.TrimSpace`, `json.Unmarshal`, schema/type branching, and fallback to inert text extraction for user/assistant/thought chunks. | `Message` fields returned by `Assemble` at `internal/transcript/transcript.go:109-125`. | LOW — rejected; this path only parses stored JSON/text into in-memory transcript messages and never executes commands, dereferences paths, or crosses a privilege boundary. |
| `internal/transcript/transcript.go:714-765` | `acp.AgentEvent` fields and `event.Raw` supplied by `internal/session/manager_prompt.go:319-324`. | `json.Valid`, best-effort `json.Unmarshal`, legacy field extraction, and final canonical `json.Marshal`. | Stored canonical payload string returned by `MarshalAgentEvent`. | LOW — rejected; attacker-influenced text is serialized as JSON and stored/fingerprinted, not interpreted as code or shell input. |
| `internal/transcript/transcript.go:714-765` | Same `MarshalAgentEvent` entry point used by `internal/extension/bridge_delivery_notifier.go:87-89` for bridge delivery fingerprints. | Same canonical JSON serialization and raw-payload normalization as above. | `DeliveryProjectionEvent.Fingerprint` string in extension bridge delivery projections. | LOW — rejected; the output is a deterministic JSON fingerprint string, not a trusted control channel. |
| `internal/transcript/transcript.go:769-798` | Stored event payload strings from `internal/extension/host_api.go:1670-1691`. | `strings.TrimSpace`, strict `json.Unmarshal`, and host-API fallback to stored `Type`, `TurnID`, and `Timestamp` if decoded values are blank. | `bridgepkg.DeliveryProjectionEvent` seed data returned to host API replay logic. | LOW — rejected; malformed payloads fail closed with an error or blank-field fallback and are not used in a command, path, or network sink inside this package. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `TRANSCRIPT-REF-001` | refactoring-analysis | medium | `internal/transcript/transcript.go:1` | The production package still lives in a single 798-line file that mixes transcript assembly, legacy parsing, and canonical serialization concerns. | deferred |
| `TRANSCRIPT-REF-002` | refactoring-analysis | low | `internal/transcript/transcript_test.go:154` | The stable-ordering test duplicates two canonical fixture blocks. | wontfix |
| `TRANSCRIPT-OPT-001` | extreme-software-optimization | low | `internal/transcript/transcript.go:453` | `buildToolResult` round-tripped `map[string]any` raw outputs through extra JSON marshal/unmarshal work and used string conversion for empty-object checks, inflating allocations in replay and marshal hot paths. | fixed |

## Per-Skill Notes

### refactoring-analysis

- The package remains structurally dense: one 798-line production file plus tests/benchmarks.
- I recorded the large-file concern as deferred because splitting the file cleanly would be a package-local reorganization with no measured correctness or performance gain in this task, and the package API is already stable.
- The duplication scan only found test-fixture setup overlap. I left it as `wontfix` because a helper extraction would mostly trade two inline fixtures for test indirection.

### extreme-software-optimization

- Added `internal/transcript/transcript_bench_test.go` before the production change so the package had co-located evidence for all chosen hot-path candidates.
- Targeted CPU profiles for `BenchmarkBuildToolResultObjectRawOutput` and `BenchmarkMarshalAgentEventToolResult` showed JSON marshal/unmarshal work dominating the selected paths, which justified a single-lever optimization in `buildToolResult`.
- The landed fix reuses already-decoded `map[string]any` tool outputs and only decodes `json.RawMessage` payloads when they are object-shaped, which cut the direct helper benchmark by more than half and improved both replay assembly and marshal throughput/allocation counts.
- `UnmarshalAgentEvent` did not improve under the full benchmark command, so no further decode refactor was justified.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package contains no goroutines, channels, mutexes, or `select` statements, so there is no package-local deadlock or goroutine-leak path to fix.

### security-review

- The package is a pure serialization/normalization layer with no direct I/O or execution sinks.
- No HIGH-confidence or MEDIUM-confidence source-to-sink vulnerability survived the threat-model review; every inspected surface terminates in structured JSON encoding/decoding or in-memory message assembly.
- The primary trust risk remains upstream: if a trusted caller stores maliciously large or malformed event payloads, this package will spend CPU/memory decoding them, but that is a low-severity resource-governance concern owned by session/extension boundaries rather than a package-local vulnerability.

## Deferred Items (carry forward)

- `TRANSCRIPT-REF-001` — Split `internal/transcript/transcript.go` into smaller file-level units once the surrounding session improvements settle; doing it safely would require a dedicated package-shaping pass rather than a cosmetic extraction during this evidence-driven improvements task.

## `make verify`

Final command: `make verify`

```text
(node:73098) Warning: The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
DONE 4513 tests in 1.113s
OK: all package boundaries respected
```
