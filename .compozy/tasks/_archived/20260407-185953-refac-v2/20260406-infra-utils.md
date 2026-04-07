# Refactoring Analysis: Infrastructure & Utilities

**Date**: 2026-04-06
**Scope**: `internal/daemon`, `internal/logger`, `internal/version`, `internal/fileutil`, `internal/procutil`, `internal/testutil`
**Analyzer**: Claude Opus 4.6 (refactoring-analysis skill)
**Total LOC (non-test)**: ~1,528 production, ~2,730 test, ~1,019 integration test

---

## Table of Contents

1. [Package: daemon](#1-package-daemon)
2. [Package: logger](#2-package-logger)
3. [Package: version](#3-package-version)
4. [Package: fileutil](#4-package-fileutil)
5. [Package: procutil](#5-package-procutil)
6. [Package: testutil](#6-package-testutil)
7. [Group-Level Summary](#7-group-level-summary)
8. [Subpackage Grouping Recommendations](#8-subpackage-grouping-recommendations)

---

## 1. Package: daemon

**Files**: 9 production, 3 test | **LOC**: ~1,504 production, ~2,468 test
**Role**: Sole composition root -- boots, wires, and shuts down the daemon

### 1.1 Findings

#### F1.1: `boot()` is a 300-line orchestration monolith (Bloater)
**Severity**: P1 (high) | **Action**: (A) File-level split / (D) Inline fix
**Location**: `internal/daemon/boot.go:20-323`

The `boot()` method is 303 lines of sequential wiring that performs config loading, logger creation, memory store init, skills registry init, lock acquisition, orphan cleanup, registry opening, workspace resolver creation, dream service creation, session manager creation, observer creation, HTTP/UDS server creation, info file writing, reconciliation, and boundary verification. While each step is individually simple, the cumulative length makes it hard to trace which resources have cleanup obligations at each failure point.

**Recommendation**: Extract named sub-phases (e.g. `bootConfig()`, `bootInfra()`, `bootServers()`) that return their cleanup functions. The method should read as a short pipeline of phase calls. This is purely a file-level readability improvement -- the composition-root role belongs here.

---

#### F1.2: `dream.go` contains domain logic that exceeds composition-root scope (SRP / Boundary)
**Severity**: P1 (high) | **Action**: (B) Package-level split
**Location**: `internal/daemon/dream.go` (323 lines)

This file contains:
- `runtimeDreamTrigger` -- a full interface implementation with business logic (gate checks, lock evaluation)
- `startDreamLoop()`, `enqueueDreamCheck()`, `runDreamCheck()` -- a background goroutine scheduler
- `makeDreamSpawner()`, `resolveDreamWorkspaces()` -- workspace resolution logic with session filtering, dedup, and time-based sorting
- `resolveDreamWorkspaceRef()`, `isPathLikeWorkspaceRef()` -- path classification utilities
- `spawnDreamSession()` -- session lifecycle orchestration

This is far beyond "wiring" -- it's the entire dream consolidation orchestration layer. The daemon package should call `dreamService.Run()`, not implement the scheduling, workspace resolution, and session spawning logic itself.

**Recommendation**: Move dream scheduling and workspace resolution into `internal/memory` (which already owns the `Service`, `SessionSpawner`, `ConsolidationLock` types). The `daemon` package should only retain the thin `startDreamLoop` goroutine that delegates to the memory package for all decisions.

---

#### F1.3: `WriteInfo()` in `info.go` duplicates `fileutil.AtomicWriteFile` (DRY)
**Severity**: P2 (medium) | **Action**: (D) Inline fix
**Location**: `internal/daemon/info.go:58-101`

`WriteInfo()` manually implements temp-file + fsync + rename, which is exactly what `fileutil.AtomicWriteFile` provides. Meanwhile, `store/meta.go` correctly uses `fileutil.AtomicWriteFile` for the same pattern.

```go
// daemon/info.go -- hand-rolled atomic write:
file, err := os.CreateTemp(filepath.Dir(cleanPath), filepath.Base(cleanPath)+".tmp-*")
// ... write, sync, close, rename ...

// store/meta.go -- correctly uses shared helper:
fileutil.AtomicWriteFile(cleanPath, payload, 0o644)
```

**Recommendation**: Replace the hand-rolled implementation in `WriteInfo()` with `fileutil.AtomicWriteFile()`.

---

#### F1.4: `syncDir()` duplicated between daemon and store (DRY)
**Severity**: P2 (medium) | **Action**: (C) Extraction
**Location**: `internal/daemon/info.go:117-130` and `internal/store/meta.go:63-76`

Two identical `syncDir`/`syncDirectory` functions exist, differing only in error prefix string. This is a textbook Extract Function refactoring.

```go
// daemon/info.go:117
func syncDir(path string) error {
    dir, err := os.Open(path)
    // ...
    err := dir.Sync()
}

// store/meta.go:63
func syncDirectory(path string) error {
    dir, err := os.Open(path)
    // ...
    err := dir.Sync()
}
```

**Recommendation**: Move `SyncDir()` into `fileutil` package alongside `AtomicWriteFile`. It could also be integrated directly into `AtomicWriteFile` as an optional step if all callers want it.

---

#### F1.5: `Daemon` struct has 37 fields -- God Object smell (Bloater)
**Severity**: P2 (medium) | **Action**: (A) File-level split
**Location**: `internal/daemon/daemon.go:111-158`

The `Daemon` struct contains 45 fields mixing: runtime state (`sessions`, `observer`, `registry`), factory functions (`openRegistry`, `newSessionManager`, `httpFactory`), configuration (`homePaths`, `config`, `orphanGraceWait`), lifecycle state (`booting`, `readyCh`, `readyClosed`), test seams (`pid`, `now`, `getenv`, `listProcesses`, `signalProcess`, `processAlive`), and subsystem state (`dreamService`, `dreamSpawner`, `dreamCheckCh`, `dreamCancel`, `dreamWG`, `skillsRegistry`, `skillsCancel`, `skillsDone`).

As the sole composition root, some field count is expected. However, the dream and skills subsystem fields (12 of 37) represent runtime orchestration state that could be encapsulated in their own types.

**Recommendation**: Extract `dreamLoop` struct owning `{dreamService, dreamSpawner, dreamCheckCh, dreamCancel, dreamWG}` and `skillsWatcher` struct owning `{skillsRegistry, skillsCancel, skillsDone}`. These become internal implementation types in daemon, reducing the main struct to ~25 fields.

---

#### F1.6: `notifier.go` is a thin fan-out with a callback hook (Potential Design Smell)
**Severity**: P3 (low) | **Action**: Keep (justified)
**Location**: `internal/daemon/notifier.go` (43 lines)

The `notifierFanout` struct is a simple multiplexer for `session.Notifier`. It has a special `onSessionStopped` callback hook for dream-check enqueueing. This is a daemon-specific composition concern, so it legitimately belongs here. The only concern is that `onSessionStopped` is a function pointer rather than just adding another `Notifier` to the list -- but this is because the dream hook needs different context (workspace filtering) than a normal notifier.

**Recommendation**: Keep as-is. This is a legitimate composition-root concern.

---

#### F1.7: `boundary.go` implements a static-analysis tool inside the daemon (Cohesion)
**Severity**: P2 (medium) | **Action**: (B) Package-level split (optional)
**Location**: `internal/daemon/boundary.go` (116 lines)

This file uses `go/parser` and `go/token` to parse Go source files and verify import boundaries at boot time. This is a development-time lint check, not a runtime daemon concern. It imports `go/parser` and `go/token` which increases the daemon's import surface for a feature that is:
- Only active in dev environments (behind `AGH_DEV_VERIFY_BOUNDARIES` env var)
- Could be a standalone `make lint-boundaries` check instead

**Recommendation**: Consider extracting to a standalone tool or build-time check. However, since this is a greenfield alpha with no production users, the current placement is acceptable -- it runs at boot for developer convenience. Mark as P3 unless binary size matters.

---

#### F1.8: `composed_assembler.go` in daemon (Cohesion)
**Severity**: P2 (medium) | **Action**: Keep (reviewed)
**Location**: `internal/daemon/composed_assembler.go` (113 lines)

> **DECISION (2026-04-06)**: After review (including Codex GPT-5.4 feedback), `ComposedAssembler` stays in `daemon`. It is a composition-root concern -- assembling prompt providers during boot. Moving it to `session` would only be justified if prompt assembly becomes a first-class session boundary, which it currently is not.

---

#### F1.9: Test file `daemon_test.go` is 2,096 lines with embedded fakes (Bloater)
**Severity**: P2 (medium) | **Action**: (A) File-level split
**Location**: `internal/daemon/daemon_test.go`

This single test file contains 2,096 lines including:
- 20+ test functions covering boot, shutdown, locks, info, orphans, boundaries, dream, skills
- 8 fake/stub implementations (`fakeSessionManager`, `fakeObserver`, `fakeServer`, `recordingRegistry`, `recordingNotifier`, `fakeDreamService`, `portReportingServer`)
- 15+ test helper functions (`waitForCondition`, `testHomePaths`, `testConfig`, `newTestDaemon`, `discardLogger`, etc.)

**Recommendation**: Split into:
- `daemon_test.go` -- core boot/shutdown/run tests
- `dream_test.go` -- dream loop/spawner tests
- `helpers_test.go` -- fakes, stubs, and test helpers

---

#### F1.10: Test helpers duplicated across 5+ packages (DRY -- cross-package)
**Severity**: P1 (high) | **Action**: (C) Extraction to `testutil`
**Location**: Multiple packages

The following test helpers are duplicated verbatim or near-identically across packages:

| Helper | Duplicated In | Count |
|--------|--------------|-------|
| `discardLogger()` | daemon, httpapi, udsapi, cli, workspace | 5x |
| `freeTCPPort(t)` | daemon, httpapi | 2x |
| `shortSocketPath(t)` | daemon, udsapi, cli | 3x |
| `waitForCondition(t, ...)` | daemon, session, cli | 3x (with signature variance) |

```go
// Identical in 5 packages:
func discardLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

**Recommendation**: Add `DiscardLogger()`, `FreeTCPPort(t)`, `ShortSocketPath(t)`, and `WaitForCondition(t, label, fn)` to `internal/testutil`. This package already exists and is imported by 23 test files across the codebase.

---

### 1.2 Coupling Analysis

**Efferent coupling (daemon imports)**: 15 packages -- `acp`, `config`, `httpapi`, `memory`, `observe`, `procutil`, `session`, `skills`, `skills/bundled`, `store`, `testutil`, `udsapi`, `workspace`, `logger`, `gofrs/flock`

This is expected and correct for the composition root. The daemon is *designed* to import everything. The key question is whether it's *doing too much* beyond wiring.

**Verdict**: Yes, it's doing too much. The dream orchestration logic (F1.2) and prompt composition logic (F1.8) are domain behaviors implemented in daemon rather than delegated to domain packages. If these were extracted, daemon would have fewer lines of logic while maintaining the same import count (which is fine for a composition root).

**Afferent coupling (who imports daemon)**: Only `cmd/agh/main.go` and `internal/cli`. This is correct -- the architectural boundary is maintained.

---

## 2. Package: logger

**Files**: 1 production, 1 test | **LOC**: 101 production, 65 test
**Role**: Structured logging factory

### 2.1 Findings

#### F2.1: Package is well-scoped and clean
**Severity**: None | **Action**: Keep

The `logger` package is a focused factory for `slog.Logger`. It provides:
- Functional options pattern (`WithLevel`, `WithFile`, `WithMirrorToStderr`)
- A `New()` constructor returning `(*slog.Logger, func() error, error)` -- the close function pattern is clean
- A standalone `ParseLevel()` utility

No code smells detected. The package is minimal, cohesive, and has only stdlib dependencies.

#### F2.2: `logger` has only 1 consumer -- could be inlined (Lazy Element?)
**Severity**: P3 (low) | **Action**: Keep (justified)
**Location**: Only imported by `daemon/boot.go`

While only daemon imports this package, it serves a purpose: isolating slog configuration so that daemon doesn't mix logging setup with boot orchestration. The package is tiny (101 LOC) but provides clear encapsulation. In the future, CLI or integration tests may also need standalone logger construction.

**Recommendation**: Keep as-is. Consolidation into a `shared/` parent (see section 8) would give it a home without eliminating it.

---

## 3. Package: version

**Files**: 1 production, 1 test | **LOC**: 32 production, 27 test
**Role**: Build metadata (ldflags injection)

### 3.1 Findings

#### F3.1: Package is correctly minimal
**Severity**: None | **Action**: Keep

The `version` package is the canonical Go pattern for build metadata. It exports three `var` values set via `-ldflags`, a `Current()` snapshot function, and a `String()` method. This is clean and follows widespread Go conventions.

#### F3.2: Very small package -- candidate for consolidation
**Severity**: P3 (low) | **Action**: (C) Extraction (optional)
**Location**: 32 production lines

At 32 lines, this could be a single file inside a shared `infra/` or `shared/` parent package. However, the ldflags injection pattern benefits from being in a dedicated package since the `-ldflags` path (`internal/version.Version`) is embedded in the build system.

**Recommendation**: Keep as standalone. The build system coupling to the package path makes consolidation more trouble than it's worth.

---

## 4. Package: fileutil

**Files**: 1 production, 1 test | **LOC**: 61 production, 119 test
**Role**: Atomic file write helper

### 4.1 Findings

#### F4.1: Missing `SyncDir()` utility that others need (Gap)
**Severity**: P2 (medium) | **Action**: (C) Extraction
**Location**: `internal/fileutil/atomic.go`

The `fileutil` package provides `AtomicWriteFile` but does not export a `SyncDir()` function, even though this operation is duplicated in `daemon/info.go:syncDir()` and `store/meta.go:syncDirectory()`. Both callers need to sync the parent directory after an atomic write for crash durability.

**Recommendation**: Add `SyncDir(path string) error` to `fileutil`. Optionally, make `AtomicWriteFile` call it internally.

#### F4.2: Package is underutilized -- daemon doesn't use it
**Severity**: P2 (medium) | **Action**: (D) Inline fix
**Location**: Consumers: only `memory/store.go` and `store/meta.go`

Despite being a shared utility, `fileutil` is used by only 2 packages (`memory` and `store`). Meanwhile, `daemon/info.go` hand-rolls the same atomic write pattern (F1.3). This suggests the package was created after `daemon/info.go` and the daemon was never updated.

**Recommendation**: After adding `SyncDir()` and updating daemon to use `fileutil`, the package would have 3 consumers -- a healthy adoption level.

---

## 5. Package: procutil

**Files**: 1 production, 1 test | **LOC**: 29 production, 54 test
**Role**: Process liveness check and signal delivery

### 5.1 Findings

#### F5.1: Package is correctly scoped but minimally sized
**Severity**: P3 (low) | **Action**: Keep
**Location**: 29 production lines

The package exports exactly 2 functions: `Alive(pid int) bool` and `Signal(pid int, sig syscall.Signal) error`. Both are clean, well-tested wrappers around `syscall.Kill`. Used by 4 files across `daemon` and `memory`.

#### F5.2: `daemon/orphan.go` contains process management logic that arguably extends procutil's domain (Boundary)
**Severity**: P3 (low) | **Action**: Keep (current placement justified)
**Location**: `daemon/orphan.go:99-125` -- `listProcesses()` shells out to `ps -axo`

The `listProcesses()` function in `daemon/orphan.go` shells out to `ps` to list processes. This is a process utility, but it's specific to daemon's orphan cleanup. Moving it to `procutil` would make sense only if other consumers appeared.

**Recommendation**: Keep in daemon. The `ps`-based implementation is platform-specific and tightly coupled to orphan cleanup semantics.

---

## 6. Package: testutil

**Files**: 1 production, 1 test | **LOC**: 32 production, 59 test
**Role**: Shared test helpers

### 6.1 Findings

#### F6.1: `testutil` is too thin -- missing widely-duplicated helpers (Gap / DRY)
**Severity**: P1 (high) | **Action**: (C) Extraction
**Location**: `internal/testutil/testutil.go` -- only 2 exported functions

The package currently exports only:
- `Context(t testing.TB) context.Context` -- test-scoped context with timeout
- `EqualStringSlices(left, right []string) bool` -- slice comparison

Meanwhile, 4-5 test helper functions are duplicated across 3-5 packages (see F1.10). The `testutil` package is the natural home for these, but they were never centralized.

**Missing helpers that should be added**:
```go
func DiscardLogger() *slog.Logger           // duplicated in 5 packages
func FreeTCPPort(t testing.TB) int          // duplicated in 2 packages
func ShortSocketPath(t testing.TB) string   // duplicated in 3 packages
func WaitForCondition(t testing.TB, label string, fn func() bool) // duplicated in 3 packages
```

**Recommendation**: Centralize these helpers in `testutil`. This is the highest-impact DRY fix in this package group -- it affects the most files across the most packages.

#### F6.2: `EqualStringSlices` may be replaceable with `slices.Equal` (Simplification)
**Severity**: P3 (low) | **Action**: (D) Inline fix
**Location**: `internal/testutil/testutil.go:22-32`

Go 1.21+ provides `slices.Equal()` in the standard library. The custom `EqualStringSlices` is a pre-generics artifact.

**Recommendation**: Replace with `slices.Equal[[]string]` and remove the custom function. Check if any callers rely on the nil-vs-empty distinction.

---

## 7. Group-Level Summary

### Finding Counts by Severity

| Severity | Count | Findings |
|----------|-------|----------|
| P0 (critical) | 0 | -- |
| P1 (high) | 3 | F1.2 (dream logic in daemon), F1.10 (test helper duplication), F6.1 (testutil gaps) |
| P2 (medium) | 6 | F1.1 (boot monolith), F1.3 (WriteInfo duplication), F1.4 (syncDir duplication), F1.5 (God Object fields), F1.8 (ComposedAssembler placement), F4.1 (missing SyncDir) |
| P3 (low) | 5 | F1.6 (notifier OK), F1.7 (boundary tool), F1.9 (test file size), F2.2 (logger singleton), F6.2 (slices.Equal) |

### Top 5 Highest-Impact Opportunities

1. **Centralize test helpers in `testutil`** (F1.10 + F6.1) -- touches 5+ packages, eliminates ~120 lines of duplication, trivial effort
2. **Extract dream orchestration from daemon** (F1.2) -- 323 lines of domain logic moved to `memory`, daemon becomes a thin scheduler, moderate effort
3. **Move `ComposedAssembler` to session** (F1.8) -- enforces architectural boundary, trivial effort
4. **Use `fileutil.AtomicWriteFile` in daemon and extract `SyncDir`** (F1.3 + F1.4 + F4.1) -- eliminates 3 duplicated functions, trivial effort
5. **Decompose `boot()` into phases** (F1.1) -- improves readability of the most complex function, moderate effort

### Suggested Refactoring Order

| Priority | Finding | Effort | Impact |
|----------|---------|--------|--------|
| 1 | F1.10 + F6.1: Add missing helpers to testutil | Trivial | High (5+ packages) |
| 2 | F1.3 + F1.4 + F4.1: Unify atomic write + SyncDir | Trivial | Medium (3 packages) |
| ~~3~~ | ~~F1.8: Move ComposedAssembler to session~~ | ~~Trivial~~ | **DECISION: Keep in daemon** (composition-root concern) |
| 4 | F1.9: Split daemon_test.go | Trivial | Medium (maintainability) |
| 5 | F1.5: Extract dreamLoop/skillsWatcher sub-structs | Moderate | Medium (readability) |
| 6 | F1.2: Extract dream orchestration to memory | Moderate | High (SRP) |
| 7 | F1.1: Decompose boot() into phases | Moderate | Medium (readability) |
| 8 | F6.2: Replace EqualStringSlices with slices.Equal | Trivial | Low |

---

## 8. Subpackage Grouping Recommendations

### Should the utility packages be consolidated?

The user asked whether `fileutil`, `procutil`, `testutil`, `logger`, and `version` should be grouped under a `pkg/` or `shared/` parent.

**Analysis**:

| Package | LOC | Consumers | Verdict |
|---------|-----|-----------|---------|
| `fileutil` | 61 | 2 (memory, store) + 1 pending (daemon) | Keep standalone |
| `procutil` | 29 | 2 (daemon, memory) | Keep standalone |
| `testutil` | 32 (growing to ~80) | 23 test files | Keep standalone |
| `logger` | 101 | 1 (daemon) | Keep standalone |
| `version` | 32 | 4 (cli, observe, main) | Keep standalone |

**Recommendation: Do NOT create a `pkg/` or `shared/` parent.**

Reasons:
1. **Go convention**: Go prefers flat package names over nested hierarchies. `internal/fileutil` is idiomatic; `internal/shared/fileutil` adds a meaningless namespace.
2. **Import paths get longer**: Every consumer would change from `internal/procutil` to `internal/shared/procutil` for zero semantic benefit.
3. **These packages are correctly small**: In Go, a package with 1-2 files and a focused purpose is normal and good. `procutil` having 29 LOC is fine -- it's a leaf package with no dependencies.
4. **testutil is special**: It's a test-only package imported by 23 files. Moving it would be high-churn for no gain.
5. **version has build-system coupling**: The `-ldflags` path would need updating.

The packages are already well-organized. The real problem is not that they're small -- it's that `testutil` is *too* small (missing helpers that are duplicated elsewhere) and that `daemon` is too large (contains domain logic that belongs elsewhere).

### What SHOULD be split/moved

| Source | Destination | Rationale |
|--------|-------------|-----------|
| ~~`daemon/composed_assembler.go`~~ | ~~`session/composed_assembler.go`~~ | **DECISION: Keep in daemon** -- composition-root concern per Codex review |
| `daemon/dream.go` (bulk) | `memory/dream_orchestrator.go` | Domain logic for dream workspace resolution, session spawning |
| `daemon/dream.go` (thin loop) | Keep in daemon | Just the `startDreamLoop` goroutine and channel wiring |
| `daemon/info.go:syncDir()` | `fileutil/sync.go` | Duplicated utility function |
| Test helpers (5 funcs) | `testutil/testutil.go` | Duplicated across 5 packages |

### Daemon's legitimate scope after refactoring

After extracting the above, `daemon/` would contain:
- **`daemon.go`**: Struct, options, `Run()`, `Shutdown()`, `signalSource()` -- lifecycle
- **`boot.go`**: Phase-based boot orchestration -- composition wiring
- **`lock.go`**: File lock acquisition -- singleton enforcement
- **`info.go`**: daemon.json read/write (using `fileutil`) -- discovery record
- **`orphan.go`**: Stale process cleanup -- boot-time housekeeping
- **`notifier.go`**: Fan-out multiplexer -- composition glue
- **`boundary.go`**: Import boundary verification -- dev-time lint

This is a clean composition root: lifecycle management, resource wiring, and infrastructure concerns. No domain logic.
