# Improvements Report — internal/cli

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/cli/perf_bench_test.go`, before/after numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo $(rg --files internal/cli -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 20 | `(commandDeps).withDefaults` | `internal/cli/root.go:145` |
| 17 | `readSkillResource` | `internal/cli/skill_workspace.go:343` |
| 17 | `buildAutomationTriggerUpdateRequest` | `internal/cli/automation.go:1511` |
| 16 | `listSkillResources` | `internal/cli/skill_workspace.go:270` |
| 15 | `newBridgeUpdateCommand` | `internal/cli/bridge.go:194` |
| 15 | `decodeSSE` | `internal/cli/client.go:1489` |
| 14 | `fenceIndentedBlocks` | `internal/cli/docpost/docpost.go:385` |
| 14 | `buildTaskUpdateRequest` | `internal/cli/task.go:250` |
| 13 | `streamSessionEvents` | `internal/cli/session.go:337` |
| 13 | `resolveMemoryLocation` | `internal/cli/memory.go:339` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/cli/automation.go` | 1641 | Job, trigger, and run command wiring, parsing, validation, and rendering are concentrated in one monolith. |
| `internal/cli/bridge.go` | 663 | Bridge CRUD, route rendering, JSON parsing, lifecycle actions, and delivery helpers all live in one file. |
| `internal/cli/client.go` | 1845 | The daemon transport interface, shared aliases, every API wrapper, SSE decoding, and query-value helpers are concentrated in one unit. |
| `internal/cli/daemon.go` | 561 | Daemon lifecycle commands, polling/wait logic, process spawn, status fallback, and log attachment helpers share one file. |
| `internal/cli/docpost/docpost.go` | 523 | Output-tree cleanup, markdown transforms, link remapping, and meta generation are bundled together. |
| `internal/cli/extension.go` | 632 | Offline/online extension commands, local install prep, status rendering, and bundle helpers are mixed in one file. |
| `internal/cli/extension_marketplace.go` | 958 | Marketplace registry loading, staging, rollback, update flows, and output helpers are concentrated in one file. |
| `internal/cli/format.go` | 309 | Format parsing, JSON/human/toon rendering, time helpers, and generic list rendering sit in a single shared formatter unit. |
| `internal/cli/hooks.go` | 381 | Hook list/info/events/runs commands and all hook rendering helpers share one file. |
| `internal/cli/install.go` | 365 | Install command orchestration, wizard state, provider/model suggestions, and output rendering are grouped together. |
| `internal/cli/memory.go` | 620 | Memory CRUD commands, scope resolution, stdin/content handling, YAML formatting, and render helpers are concentrated here. |
| `internal/cli/network.go` | 521 | Network status/send/inbox commands, JSON parsing, and bundle rendering all live in one unit. |
| `internal/cli/session.go` | 678 | Session CRUD/stream commands, follow-mode rendering, workspace resolution, and bundle helpers are concentrated in one file. |
| `internal/cli/skill_commands.go` | 365 | Skill list/view/info/create/search/install/remove/update commands are wired in one file. |
| `internal/cli/skill_marketplace.go` | 703 | Skill marketplace registry, install/update/remove workflows, path validation, and filesystem moves are concentrated in one file. |
| `internal/cli/skill_workspace.go` | 585 | Workspace skill resolution, resource listing, safe file reads, XML rendering, and name normalization are bundled together. |
| `internal/cli/task.go` | 1502 | Task/task-run command tree, validation, parsing, request builders, and rendering helpers are concentrated in one monolith. |
| `internal/cli/workspace.go` | 472 | Workspace CRUD commands, edit merging, and detail rendering are combined in one file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/cli` notable production findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/cli/client.go:474-482` | `internal/cli/client.go:511-522` | Repeated `doJSON` wrapper shape across extension and bridge client methods. |
| `internal/cli/client.go:951-969` | `internal/cli/client.go:990-1008` | Memory read/delete wrappers mirror each other with only HTTP verb/path differences. |
| `internal/cli/client.go:1087-1116` | `internal/cli/client.go:1156-1182` | Automation job/trigger run-list wrappers repeat the same request/response flow. |
| `internal/cli/automation.go:189-207` | `internal/cli/automation.go:334-352` | Repeated command handlers that fetch one record then render a bundle. |
| `internal/cli/bridge.go:272-310` | `internal/cli/bridge.go:292-330` | Enable/disable/restart bridge command handlers differ only by the invoked action. |
| `internal/cli/task.go:1144-1183` | `internal/cli/task.go:1185-1224` | Human/toon task detail rendering mirrors the same section layout twice. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `renderHumanTable` | `internal/cli/format.go:159` | Shared allocation-heavy table renderer used by list/status commands across the CLI. | `BenchmarkRenderHumanTableLarge` |
| `renderToonArray` | `internal/cli/format.go:211` | Shared TOON array renderer repeatedly allocates per row and runs on every non-JSON list command in toon mode. | `BenchmarkRenderToonArrayLarge` |
| `decodeSSE` | `internal/cli/client.go:1489` | Shared message-stream loop for session/observe SSE handling with per-event data assembly. | `BenchmarkDecodeSSELargeStream` |
| `(*unixSocketClient).doRequest` | `internal/cli/client.go:1444` | Shared transport entry point for every daemon request, including request JSON marshalling and header setup. | `BenchmarkDoRequestPostJSON` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/cli/...` before any production fix, followed by the same command on the final end-state:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkRenderHumanTableLarge` | 134818.0 | 309199.2 | 135048.6 | 309199.4 | not-hot-confirmed-by-benchmark — no shared table-path change was justified in this pass. |
| `BenchmarkRenderToonArrayLarge` | 85385.8 | 130891.8 | 52362.4 | 84752.0 | fixed-with-benchmark |
| `BenchmarkDecodeSSELargeStream` | 113787.4 | 246216.2 | 114733.6 | 246216.0 | not-hot-confirmed-by-benchmark — the attempted fast path was reverted after measurement showed no improvement. |
| `BenchmarkDoRequestPostJSON` | 1224.2 | 2642.0 | 1231.0 | 2642.0 | not-hot-confirmed-by-benchmark — no transport-layer change was justified in this pass. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/cli/daemon.go:214` | `waitForDaemonStart` | The goroutine exits when `child.Wait()` returns; otherwise it is bounded by the short-lived CLI process lifetime. | Used only to surface detached-daemon early-exit failures while polling readiness. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/cli/daemon.go:214` | 1 | `waitForDaemonStart` | none | `waitForDaemonStart` select loop | Buffered channel transports the `child.Wait()` result back to the readiness poller. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `sync.Mutex` or `sync.RWMutex` fields are declared under `internal/cli/`. |

### Concurrency — Select Audit

All production `select` statements under `internal/cli/daemon.go` include `ctx.Done()` or a derived timeout context.

### Security — Threat Model

- Trust boundaries:
  - Local operators and automation invoke this package through CLI args, env vars, stdin, and the hidden docs command.
  - The package crosses into the local daemon over a Unix-socket HTTP API, and into the local filesystem for doc generation, skill reads, extension installs, and marketplace staging.
  - The `docpost` subpackage writes to arbitrary filesystem locations passed by the CLI command.
- Attacker capabilities:
  - A local caller controls positional args, flags, stdin payloads, and environment variables such as session/agent identity.
  - A local caller can supply filesystem paths, relative resource names, JSON payloads, workspace refs, and output directories.
  - Remote network attackers are out of scope unless their data is already presented to this package as local CLI or daemon payload input.
- In-scope assets:
  - Integrity of local filesystem contents touched by doc generation, extension install/remove, skill reads, and skill/extension staging directories.
  - Integrity of requests forwarded to the daemon API.
  - Confidentiality boundaries around skill file reads and marketplace-managed install paths.
- Out-of-scope:
  - Authentication and authorization in the daemon/API layers after the request leaves the CLI transport.
  - Correctness of remote registries or external marketplace content once downloaded and validated by other packages.
  - Malicious code already executing with the same local filesystem permissions as the operator.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/cli/doc.go:36-41`, `internal/cli/docpost/docpost.go:47-50`, `262-282` | User-controlled `--output-dir` for the hidden `agh doc` command. | `filepath.Abs` canonicalizes the path, and the current code now requires the target to be empty or look like a managed CLI docs root before cleanup. | `docpost.Process` calls `cleanOutput`, which recursively deletes existing generated entries under the accepted target directory before rewriting docs. | HIGH — fixed by rejecting non-empty unmanaged output roots before cleanup. |
| `internal/cli/skill_commands.go:90-103`, `internal/cli/skill_workspace.go:343-423` | User-controlled `agh skill view --file <relative-path>`. | Relative-path cleaning plus bundled-path rejection, filesystem path normalization, `EvalSymlinks`, and root-relative containment checks. | Reads a file inside the selected skill directory and prints it to stdout. | LOW — traversal and symlink-escape paths are rejected before file reads. |
| `internal/cli/extension.go:408-440`, `443-469` | User-controlled local extension install path. | Path is made absolute, must exist, must be a directory, then manifest + checksum validation run before installation. | Reads manifest and computes a directory checksum for install staging. | LOW — local filesystem reads stay inside the supplied directory and fail closed on invalid manifests or missing directories. |
| `internal/cli/network.go:120-146`, `internal/cli/client.go:1444-1486` | User-controlled `--body`, `--ext`, and envelope routing flags for `agh network send`. | JSON object parsing plus string trimming and `url.PathEscape` in downstream client methods. | Marshalled JSON request is forwarded to the daemon’s network API. | LOW — this package parses and forwards structured data; it does not execute or interpolate the payload locally. |
| `internal/cli/memory.go:161-190`, `490-507` | User-controlled memory filename, description, scope, and content via flags/stdin. | Filename/content presence checks, type/scope normalization, and YAML frontmatter generation through structured marshaling. | `WriteMemory` daemon API request plus formatted memory document body. | LOW — the package only formats and forwards the document; no local file write occurs here. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | security-review | high | `internal/cli/doc.go:36-41` -> `internal/cli/docpost/docpost.go:47-50,83-145` | The hidden `agh doc --output-dir` flow treated any existing target directory as safe output state, so `cleanOutput` could recursively remove unrelated contents under an arbitrary caller-selected path. | fixed |
| 02 | extreme-software-optimization | medium | `internal/cli/format.go:201-238` | TOON object/array rendering built temporary slices and joined them per row, making shared toon-mode list rendering slower and allocation-heavy. | fixed |
| 03 | refactoring-analysis | medium | `internal/cli/client.go:474-522,951-1008,1087-1182` | The API client still repeats the same `doJSON` / decode wrapper shape across extension, bridge, memory, and automation endpoints. | deferred |
| 04 | refactoring-analysis | medium | `internal/cli/task.go:1144-1224` | Task detail human/toon rendering still duplicates section assembly logic, so output-format changes must be kept in sync manually. | deferred |

## Per-Skill Notes

### refactoring-analysis

- Production complexity is concentrated in `client.go`, `task.go`, `automation.go`, and `skill_workspace.go`; no single cyclomatic hotspot inside `internal/cli/` justified a structural split during this pass.
- The duplication scan points at two recurring maintainability seams: repeated transport wrappers in `client.go` and mirrored detail rendering in `task.go`.
- I left those structural refactors deferred because this pass could land higher-value, lower-risk correctness and performance fixes without widening package churn.

### extreme-software-optimization

- Added `internal/cli/perf_bench_test.go` so every selected hot-path candidate now has a co-located benchmark.
- CPU profiling on `BenchmarkRenderToonArrayLarge` showed avoidable work in TOON row assembly, especially temporary slice creation and `strings.Join` usage.
- Fixed the TOON renderer by streaming directly into a `strings.Builder` via `writeToonValues`, preserving output semantics while removing per-row temporary slices.
- `BenchmarkRenderToonArrayLarge` improved from `85385.8 ns/op, 130891.8 B/op, 1551 allocs/op` to `52362.4 ns/op, 84752.0 B/op, 19 allocs/op`.
- I benchmarked an SSE decoder fast path separately and reverted it because the measured end-state regressed slightly (`113787.4 ns/op` -> `114733.6 ns/op`), so no `decodeSSE` production change shipped.

### ubs

`not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No production deadlock or goroutine-leak finding was confirmed after auditing the package-owned goroutine/channel usage in `daemon.go`.
- The only package-owned goroutine is the short-lived `child.Wait()` watcher inside `waitForDaemonStart`, and its two `select` sites are context-aware.
- No production mutexes were found under `internal/cli/`; the concurrency surface here is intentionally small.

### security-review

- The highest-risk local boundary in this package was the hidden docs generator because it crossed from caller-controlled filesystem paths into recursive cleanup.
- Fixed that boundary in `docpost.Process` by requiring an existing output directory to be empty or look like a managed CLI docs root before `cleanOutput` is allowed to remove anything.
- Added regression coverage in `internal/cli/docpost/docpost_test.go` for both refusal of an unmanaged directory and successful rerun into a generated docs tree.

## Deferred Items (carry forward)

- **03** — Consolidate repeated `doJSON` client wrappers in `internal/cli/client.go` only when a future task can absorb a wider API-surface refactor safely.
- **04** — Unify the duplicated task detail human/toon section assembly in `internal/cli/task.go` when a follow-up task is ready to touch both output modes together.
- **OPT-02** — Leave `decodeSSE` unchanged unless future profiling shows stream decoding dominates real CLI latency; the attempted fast path did not benchmark positively.

## `make verify`

Command: `make verify`

Exit code: `0`

Excerpt from the clean pass:

```text
0 issues.
✓  internal/store/globaldb (cached)
✓  internal/task (cached)
✓  internal/config (cached)
✓  internal/memory (cached)
✓  internal/session (cached)
✓  internal/cli/docpost (1.03s)
✓  internal/skills/bundled (1.113s)
✓  internal/hooks (1.593s)
✓  internal/acp (4.453s)
✓  internal/daemon (7.301s)
✓  internal/extension (8.175s)
✓  internal/cli (8.205s)

DONE 4434 tests in 9.854s
OK: all package boundaries respected
```
