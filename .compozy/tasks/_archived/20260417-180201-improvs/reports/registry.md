# Improvements Report — internal/registry

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/registry/*_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/registry | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 21 | `TestExtractArchive_EnforcesLimitsAndRejectsUnsafeEntries` | `internal/registry/extract_test.go:77` |
| 17 | `(*Client).doRequest` | `internal/registry/clawhub/client.go:250` |
| 15 | `(*Client).doRequest` | `internal/registry/github/client.go:462` |
| 14 | `TestMoveInstalledDir` | `internal/registry/extract_test.go:322` |
| 14 | `(*MultiRegistry).resolveSource` | `internal/registry/multi.go:236` |
| 14 | `TestClientInfoFetchesLatestAndVersions` | `internal/registry/github/client_test.go:38` |
| 14 | `TestClientSearchParsesListingsAndLimit` | `internal/registry/clawhub/client_test.go:22` |
| 13 | `extractArchive` | `internal/registry/extract.go:67` |
| 13 | `(*MultiRegistry).Search` | `internal/registry/multi.go:54` |
| 13 | `(*Client).fetchRequestedRelease` | `internal/registry/github/client.go:341` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/registry/extract.go` | 431 | Archive extraction, path-safety enforcement, and install move helpers live in one file; the responsibilities are related but broad for a single unit. |
| `internal/registry/installer.go` | 644 | Install orchestration, manifest parsing, verification, and temp-dir cleanup remain co-located even after moving checksum helpers into `installer_checksum.go`. |
| `internal/registry/multi.go` | 392 | Search fan-out, source resolution, and merge helpers share one file, which keeps the package readable today but pushes past the repository’s 300-LOC heuristic. |
| `internal/registry/clawhub/client.go` | 570 | The ClawHub client mixes request orchestration, retry helpers, response decoding, and download spooling in one file. |
| `internal/registry/github/client.go` | 1001 | The GitHub client is the largest surface in the package tree and combines request/retry logic, release selection, response handling, and download spooling. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/registry`:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/registry/clawhub/client.go:465-493` | `internal/registry/github/client.go:925-953` | Download spooling logic is duplicated across both remote-source clients. |
| `internal/registry/clawhub/client.go:516-541` | `internal/registry/github/client.go:972-997` | Temporary download-file cleanup and idle-connection helpers are duplicated across both remote-source clients. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*MultiRegistry).Search` | `internal/registry/multi.go:54` | Marketplace search fans out across sources and merges all listings on every CLI marketplace query (`internal/cli/skill_marketplace.go:93`, `internal/cli/extension_marketplace.go:90`). This is the package’s main goroutine fan-out path. | `BenchmarkMultiRegistrySearch` |
| `(*MultiRegistry).resolveSource` | `internal/registry/multi.go:236` | Detail, download, and update checks all resolve through this concurrent fan-out path (`internal/registry/multi.go:132`, `internal/registry/multi.go:147`, `internal/registry/multi.go:181`). | `BenchmarkMultiRegistryResolveSource` |
| `ExtractArchive` | `internal/registry/extract.go:63` | Archive extraction is the package-owned filesystem write loop used by marketplace installs and updates (`internal/cli/skill_marketplace.go:619`). | `BenchmarkExtractArchive` |
| `computeInstallChecksum` | `internal/registry/installer_checksum.go:17` | Every registry-backed install and update computes a full post-install checksum after the move (`internal/registry/installer.go:347`). This is the package’s heaviest deterministic filesystem read/hash path. | `BenchmarkComputeInstallChecksum` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/registry/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/registry/...`

Values below use the median of 5 runs from `/tmp/registry-bench-before.txt` and `/tmp/registry-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkMultiRegistrySearch` | 203503 | 1005118 | 100609 | 478597 | fixed-with-benchmark |
| `BenchmarkMultiRegistryResolveSource` | 1031 | 960 | 1039 | 960 | not-hot-confirmed-by-benchmark |
| `BenchmarkExtractArchive` | 4016292 | 1771563 | 4071655 | 1771144 | not-hot-confirmed-by-benchmark |
| `BenchmarkComputeInstallChecksum` | 4667151 | 8691820 | 4686818 | 8691800 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/registry/multi.go:80` | `(*MultiRegistry).Search` | WaitGroup join plus caller-provided `context.Context`; source implementations are expected to honor `ctx` and the parent checks `ctx.Err()` after `wg.Wait()`. | One goroutine per searchable source to fan out marketplace search. |
| `internal/registry/multi.go:254` | `(*MultiRegistry).resolveSource` | WaitGroup join plus caller-provided `context.Context`; source implementations are expected to honor `ctx` and the parent checks `ctx.Err()` after `wg.Wait()`. | One goroutine per source to resolve detail priority before `Info`/`Download`/`CheckUpdate`. |

### Concurrency — Channel Inventory

No production channels are declared in `internal/registry/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/registry/`.

### Concurrency — Select Audit

All production `select` statements are context-aware retry waits:

| File:Line | Notes |
| --- | --- |
| `internal/registry/clawhub/client.go:442` | `sleepContext` waits on `ctx.Done()` or timer expiry. |
| `internal/registry/github/client.go:902` | `sleepContext` waits on `ctx.Done()` or timer expiry. |

### Security — Threat Model

- Trust boundaries:
  - `internal/registry` sits between operator/CLI input and remote registry services (`internal/cli/skill_marketplace.go:67-93`, `internal/cli/extension_marketplace.go:67-90`).
  - It also sits between remote registry archives and the local filesystem during install/update flows (`internal/registry/installer.go:227-357`).
- Attacker capabilities:
  - A local operator or CLI caller can influence marketplace queries, slugs, requested versions/assets, and configured registry base URLs.
  - A compromised or malicious remote registry can control HTTP payloads, JSON metadata, archive bytes, archive entry names, manifest contents, and download headers.
  - An attacker cannot directly bypass local filesystem permissions or the package’s explicit archive/path validation through this package alone.
- In-scope assets:
  - Safe HTTP request construction for remote registry lookups.
  - Safe archive extraction and install-path movement without path traversal or symlink escape.
  - Integrity of installed package contents and post-install checksum calculation.
  - Rejection of marketplace content that attempts prompt-injection style instruction override.
- Out-of-scope:
  - Authorization decisions in CLI commands before they choose which registry source, slug, or target directory to use.
  - A hostile local operator who already controls the configured install target path or the process account.
  - Vulnerabilities in third-party registry infrastructure outside the package’s request/validation boundaries.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/cli/skill_marketplace.go:67-93` | Operator-configured skill-registry base URL and CLI search/install arguments flow into `clawhub.NewClient`, `NewMultiRegistry`, and `Installer`. | `strings.TrimSpace` on base URLs/slugs, `url.Values.Encode`, `url.PathEscape`, and explicit empty-query/empty-slug rejection in `internal/registry/clawhub/client.go:125-243`. | Remote HTTP GETs in `internal/registry/clawhub/client.go:250-314` plus archive install flow in `internal/registry/installer.go:227-357`. | LOW — rejected; request components are path-escaped or query-encoded before use, and the package never shells out or interpolates raw input into filesystem paths. |
| `internal/cli/extension_marketplace.go:67-90` | Operator-configured GitHub registry base URL plus CLI slug/version/asset arguments flow into `github.NewClient`, `NewMultiRegistry`, and `Installer`. | `parseRepoSlug` enforces `owner/repo`, `url.PathEscape` protects path segments, and `selectReleaseDownload` only accepts exact asset-name matches in `internal/registry/github/client.go:341-797`. | Remote GitHub API/download requests in `internal/registry/github/client.go:462-601` and the shared install path in `internal/registry/installer.go:227-357`. | LOW — rejected; the package constrains request construction to encoded path segments and validated asset names, leaving no confirmed injection sink. |
| `internal/registry/installer.go:227-357` | Remote `.tar.gz` archive bytes and manifest contents returned by any registry source. | Compressed/decompressed size caps, file-count cap, `CleanArchiveEntryPath`, `PathWithinRoot`, symlink traversal rejection, root-manifest validation, and instruction-pattern verification in `internal/registry/extract.go:67-243` and `internal/registry/installer.go:441-603`. | Filesystem extraction under the install temp root, manifest parsing, final install move, and checksum calculation. | LOW — rejected; the package explicitly defends against traversal, symlink escape, oversized payloads, unsupported tar entry types, and manifest-root confusion before any final install move. |
| `internal/registry/installer_checksum.go:17-72` | Installed payload paths derived from extracted archive contents after the final move. | `filepath.Abs`, `os.Stat`, `filepath.WalkDir`, `filepath.Rel`, `slices.Sort`, `os.Lstat`, and regular-file/symlink type checks constrain hashing to the install root. | SHA-256 checksum generation returned in `InstallResult.Checksum`. | LOW — rejected; the checksum path reads local files only after the install root has passed archive validation, and unsupported file types fail closed. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `REGISTRY-REF-001` | refactoring-analysis | medium | `internal/registry/installer.go:227` | The installer flow mixed checksum internals into the main install file, which made the download/extract/move path harder to scan and test in isolation. | fixed |
| `REGISTRY-REF-002` | refactoring-analysis | medium | `internal/registry/clawhub/client.go:465` | Download temp-file spooling and cleanup helpers are duplicated between the ClawHub and GitHub source clients. | deferred |
| `REGISTRY-OPT-001` | extreme-software-optimization | low | `internal/registry/multi.go:294` | Search normalized each source result into a fresh slice and merged into unbounded containers, causing avoidable allocation churn on marketplace search. | fixed |
| `REGISTRY-OPT-002` | extreme-software-optimization | low | `internal/registry/installer_checksum.go:17` | Checksum generation remains dominated by filesystem IO; the full-suite benchmark rerun did not justify an algorithmic speed tweak in this pass. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- Extracted checksum helpers into `internal/registry/installer_checksum.go`, which narrowed `installer.go` from 766 LOC to 644 LOC and pulled hashing concerns into a dedicated unit.
- The package still has several files above the 300-LOC heuristic (`extract.go`, `installer.go`, `multi.go`, `clawhub/client.go`, `github/client.go`), but only the checksum split was worth landing inside this pass without expanding scope.
- The remaining production duplication is isolated to the two registry clients’ temp-file/download helpers; I deferred that until there is appetite for a shared cross-subpackage helper rather than adding a one-off abstraction here.

### extreme-software-optimization

- Added `internal/registry/perf_bench_test.go` before changing production code so the optimization pass started from measured baselines rather than assumptions.
- `BenchmarkMultiRegistrySearch` improved from `203503 ns/op, 1005118 B/op, 43 allocs/op` to `100609 ns/op, 478597 B/op, 16 allocs/op` after reusing source-result slices in place and pre-sizing the merge containers.
- `resolveSource`, `ExtractArchive`, and `computeInstallChecksum` remained effectively filesystem/runtime bound in the full-suite benchmark run, so no further optimization was justified for those paths.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute will be used.

### deadlock-finder-and-fixer

- Inventory complete; production concurrency is limited to the two `MultiRegistry` goroutine fan-out loops, with no package-local channels or mutexes.
- Both production `select` statements live in the retry sleeps for the two HTTP clients and include `ctx.Done()`, so there is no package-local deadlock or leaked-wait path to fix.

### security-review

- The high-risk surface in this package is remote archive ingestion, and the current guardrail stack already blocks traversal, symlink escape, oversize payloads, unsupported tar entry types, and missing manifest roots before the final move.
- No HIGH-confidence or MEDIUM-confidence vulnerability survived the threat-model review because request construction is encoded, archive extraction is root-bounded, and checksum generation only reads already-validated local payloads.
- Coverage remains steady at `80.8%` for `internal/registry`, with subpackages at `77.0%` (`clawhub`) and `79.5%` (`github`).

## Deferred Items (carry forward)

- **REGISTRY-REF-002** — The duplicated temp-file spooling and idle-connection helpers across `clawhub` and `github` should move into a shared helper only when the package is ready for a deliberate cross-subpackage consolidation, not as an exported or ad-hoc abstraction in this pass.
- Remaining large files in `extract.go`, `installer.go`, `multi.go`, `clawhub/client.go`, and `github/client.go` still warrant future decomposition, but only the checksum split fit cleanly inside this task’s scope.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/registry/github (1.063s)
✓  internal/registry (1.094s)
✓  internal/registry/clawhub (1.137s)
DONE 4489 tests in 10.997s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0`.
