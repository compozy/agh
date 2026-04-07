# TechSpec: Codebase Refactoring — Duplication Elimination & File Organization

## Executive Summary

This TechSpec defines the implementation plan for a systematic refactoring of the AGH codebase, addressing 74 findings across 15 packages identified in the [refactoring analysis](./20260406-summary.md). The work is organized into four phases of increasing effort and risk, designed to be executed incrementally with `make verify` gating each step.

The primary trade-off is **execution time vs. consolidation depth**: we extract shared packages (`apicore`, `procutil`, `fileutil`, `testutil`) that introduce new import edges in the dependency graph, trading a slightly wider package surface for dramatically reduced duplication (~2,600 lines of duplicated production + test code eliminated). All refactorings preserve existing public APIs — no consumer-facing changes.

The dominant architectural decision is consolidating `httpapi/` and `udsapi/` into a shared `apicore/` layer, which addresses the single largest duplication cluster (~900 production lines, ~1,700 test lines) but requires careful handling of intentional behavioral divergences (error masking, timestamp formats).

## Reference Documents

All findings, code locations, and before/after sketches are in the individual analysis reports:

| Report | Scope | Link |
|--------|-------|------|
| Summary | All 15 packages, 74 findings | [`20260406-summary.md`](./20260406-summary.md) |
| Core | `session/`, `acp/` | [`20260406-core-session-acp.md`](./20260406-core-session-acp.md) |
| Storage | `store/`, `observe/`, `memory/` | [`20260406-storage-observe-memory.md`](./20260406-storage-observe-memory.md) |
| API | `httpapi/`, `udsapi/`, `apisupport/` | [`20260406-api-layer.md`](./20260406-api-layer.md) |
| Infra | `config/`, `daemon/`, `cli/`, `logger/`, `version/` | [`20260406-config-daemon-cli.md`](./20260406-config-daemon-cli.md) |
| New | `skills/`, `workspace/` | [`20260406-skills-workspace.md`](./20260406-skills-workspace.md) |

## System Architecture

### New Packages Introduced

Five new `internal/` packages are proposed. Three are leaf utilities; two are deliberate consolidation layers for the transport and test surfaces:

```
internal/
  procutil/       # ProcessAlive, Signal — used by daemon, cli, memory
  fileutil/       # AtomicWriteFile — used by store, memory
  testutil/       # Context, EqualStringSlices — used by test files across 7+ packages
  apicore/        # Shared API payloads, handlers, SSE, parsers — used by httpapi, udsapi
  apitest/        # Shared API test stubs/helpers — used by httpapi, udsapi tests
```

### Dependency Flow (post-refactoring)

```
daemon/ ──→ httpapi/ ──→ apicore/ ──→ session/, store/, observe/, memory/, workspace/, acp/, config/
         ──→ udsapi/  ──→ apicore/
         ──→ cli/
         ──→ session/ ──→ store/, workspace/, acp/
         ──→ skills/  ──→ workspace/
         ──→ procutil/ (new)
         ──→ fileutil/ (new, indirect via store/memory)
```

No circular dependencies are introduced. `procutil/`, `fileutil/`, and `testutil/` remain leaf utilities. `apicore/` is intentionally not a leaf package: it consolidates existing transport-to-domain dependencies already present in `httpapi/` and `udsapi/` rather than introducing new architectural coupling. `apitest/` is test-only shared infrastructure.

### Core Interfaces

```go
// internal/apicore/handlers.go
// BaseHandlers holds shared dependencies for both HTTP and UDS transports.
type BaseHandlers struct {
    Sessions     SessionManager
    Observer     Observer
    DreamTrigger DreamTrigger
    Workspaces   WorkspaceService
    MemoryStore  *memory.Store
    DreamService *memory.Service
    HomePaths    config.HomePaths
    Logger       *slog.Logger
}

// ErrorResponder controls how errors are surfaced per transport.
type ErrorResponder func(c *gin.Context, status int, err error)
```

```go
// internal/procutil/procutil.go
package procutil

// Alive reports whether a process with the given PID is running.
func Alive(pid int) bool

// Signal sends a signal to a process by PID.
func Signal(pid int, sig syscall.Signal) error
```

```go
// internal/fileutil/atomic.go
package fileutil

// AtomicWriteFile writes content to path via temp-file-and-rename.
// Always calls Sync before rename for durability.
func AtomicWriteFile(path string, content []byte, perm os.FileMode) error
```

## Implementation Design

### Phase 1: Quick Wins (trivial effort, no structural changes)

Mechanical extractions and deduplication within existing files. Each item is an independent commit.

| # | Action | Files Affected | Report Ref |
|---|--------|---------------|------------|
| 1.1 | Create `internal/procutil/` with `Alive` + `Signal` | `daemon/lock.go`, `daemon/daemon.go`, `cli/root.go`, `memory/lock.go` | Infra F1 |
| 1.2 | Create `internal/fileutil/` with `AtomicWriteFile` (with `Sync`) | `store/meta.go`, `memory/store.go` | Storage F3 |
| 1.3 | Create `internal/testutil/` with `Context` + `EqualStringSlices` | 7 `_test.go` files across packages | Storage F7 |
| 1.4 | Move `userAgentsSkillsDir` to `config.ResolveUserAgentsSkillsDir()` | `daemon/daemon.go`, `cli/skill.go`, `config/home.go` | Infra F2 |
| 1.5 | Merge `cleanupFailedCreate`/`cleanupFailedResume` → `cleanupFailedStart` | `session/manager.go` | Core F3 |
| 1.6 | Extract `processSkill` method (3x loop dedup) | `skills/registry.go` | New F3 |
| 1.7 | Replace `reflect.DeepEqual` with snapshot comparison | `skills/registry.go` | New F4 |
| 1.8 | Merge `startingDaemonStatus`/`stoppedDaemonStatus` → parameterized | `cli/daemon.go` | Infra F10 |
| 1.9 | Fix typo `defaultReadHeaderTimout` | `udsapi/server.go` | API Phase 1 |
| 1.10 | Remove custom `max()` (use Go builtin) | `cli/format.go` | Infra F14 |

### Phase 2: File-Level Splits (moderate effort, zero API changes)

Pure file reorganization within packages. Methods stay on the same receiver types, no import changes for consumers. Each package split is an independent commit.

| # | Source File | Target Split | Report Ref |
|---|------------|-------------|------------|
| 2.1 | `daemon/daemon.go` (1,495 LOC) | `daemon.go`, `boot.go`, `dream.go`, `orphan.go`, `boundary.go`, `notifier.go` | Infra F3 |
| 2.2 | `session/manager.go` (1,205 LOC) | `manager.go`, `manager_lifecycle.go`, `manager_prompt.go`, `manager_workspace.go`, `manager_helpers.go` | Core F1 |
| 2.3 | `store/global_db.go` (1,099 LOC) | `global_db.go`, `global_db_workspace.go`, `global_db_session.go`, `global_db_observe.go`, `global_db_permission.go` | Storage F1 |
| 2.4 | `workspace/resolver.go` (1,069 LOC) | `resolver.go`, `resolver_crud.go`, `scanner.go`, `clone.go`, `helpers.go` | New F2 |
| 2.5 | `udsapi/handlers.go` (1,084 LOC) | `sessions.go`, `agents.go`, `observe.go`, `prompt.go`, `daemon.go`, `stream.go`, `payloads.go` (match current `httpapi` layout where applicable) | API F2 |
| 2.6 | `store/schema.go` (734 LOC) | `schema.go`, `sqlite.go`, `migrate_workspace.go` | Storage F4 |
| 2.7 | `store/store.go` (568 LOC) | `types.go`, `store.go`, `sql_helpers.go` | Storage F5 |

### Phase 3: API Layer Consolidation (significant effort, highest impact)

The `apicore/` extraction is the most complex change. It is executed as a sequence of sub-steps, each independently compilable and testable.

**Step 3.1 — Shared interfaces** (`apicore/interfaces.go`):
- Move `SessionManager`, `Observer`, `DreamTrigger`, `WorkspaceService` interface definitions from both packages into `apicore/`. Both transport packages import from `apicore/`.
- Resolve the `ApprovePermission` gap: add it to the shared interface (udsapi can return 501 until implemented).

**Step 3.2 — Shared payloads** (`apicore/payloads.go`):
- Move all request/response structs: `sessionPayload`, `agentPayload`, `agentEventPayload`, `tokenUsagePayload`, `observeEventPayload`, `daemonStatusPayload`, `errorPayload`, `sseMessage`, memory payloads, workspace payloads.
- Intentional divergence: `agentEventPayload.Timestamp` uses `string` in httpapi (AI SDK SSE protocol) vs `time.Time` in udsapi. Resolution: define the base payload with `time.Time`, add `agentEventPayloadAISdk` in `httpapi/` only for the SSE stream endpoint.

**Step 3.3 — Shared conversions** (`apicore/conversions.go`):
- Move all `*FromInfo`, `*FromEvent`, `*FromDef` conversion functions.

**Step 3.4 — Shared parsers** (`apicore/parsers.go`):
- Move `parseSessionEventQuery`, `parseObserveEventQuery`, `parseOptionalTime`, `parseOptionalInt`, `parseOptionalInt64`, `parseObserveCursor`.

**Step 3.5 — Shared SSE** (`apicore/sse.go`):
- Move `prepareSSE`, `writeSSE`, `writeSSERaw`, `emitObserveEvents`, `observeEventAfterCursor`, `observeEventID`, `flushWriter`.

**Step 3.6 — Shared error handling** (`apicore/errors.go`):
- Move `respondError` with a `maskInternalErrors bool` parameter. `httpapi` calls with `true`, `udsapi` calls with `false`. This preserves the intentional divergence (httpapi masks 5xx details for security, udsapi exposes them for debugging).

**Step 3.7 — Shared handlers** (`apicore/handlers.go`):
- Move handler methods to `BaseHandlers`: `listSessions`, `createSession`, `getSession`, `stopSession`, `resumeSession`, `sessionEvents`, `sessionHistory`, `sessionTranscript`, `listAgents`, `getAgent`, `observeEvents`, `daemonStatus`, `health`.
- Transport-specific handlers remain in their packages: `httpapi` keeps `streamAISdkPrompt`, static file serving, CORS; `udsapi` keeps socket lifecycle.

**Step 3.8 — Shared memory/workspace handlers** (`apicore/memory.go`, `apicore/workspaces.go`):
- Move verbatim — these are byte-identical between packages.

**Step 3.9 — Shared test infrastructure** (`internal/apitest/`):
- Move `stubSessionManager`, `stubObserver`, `stubWorkspaceService`, `sseRecord`, `parseSSE`, `performRequest`, `decodeJSONResponse`, `newTestHomePaths`, `writeAgentDef`, `newSessionInfo`, `newSession`, `discardLogger`.
- Each transport package keeps only transport-specific test helpers (`mustStaticFS` for httpapi, `shortSocketPath`/`newUnixClient` for udsapi).

### Phase 4: Domain-Level Deduplication (moderate effort, opportunistic)

These are applied as part of ongoing work, not as a dedicated sprint.

| # | Action | Files Affected | Report Ref |
|---|--------|---------------|------------|
| 4.1 | Extract `activateAndWatch` method (Create/Resume dedup) | `session/manager.go` | Core F2 |
| 4.2 | Extract `emitPermissionEvent` (3x event emission dedup) | `acp/handlers.go` | Core F4 |
| 4.3 | Extract `requireField` + `requirePositiveLimit` validation helpers | `store/store.go` | Storage F2 |
| 4.4 | Extract `checkReady(ctx)` nil-guard helper on GlobalDB | `store/global_db.go` | Storage F6 |
| 4.5 | Extract shared `fileSnapshot` helper set across skills/workspace | `skills/types.go`, `workspace/resolver.go` | New F1 |
| 4.6 | Move `slog.Warn` from `ParseSkillFile` to callers | `skills/loader.go` | New F5 |
| 4.7 | Consolidate `cloneRawMessage`/`cloneRawJSON` within session package | `session/transcript.go`, `acp/handlers.go` | Core F5 |
| 4.8 | Extract generic `listBundle[T]` for CLI output bundles | `cli/format.go` | Infra F6 |
| 4.9 | Evaluate removal of legacy transcript parsers (zero-compat policy) | `session/transcript.go` | Core F8 |
| 4.10 | Make `timeNowUTC` injectable in `acp/` | `acp/handlers.go` | Core F16 |

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/procutil/` | new | Shared process utilities. Low risk — pure functions, no state. | Create package, update 4 consumers |
| `internal/fileutil/` | new | Shared atomic write. Low risk — fixes latent Sync bug. | Create package, update 2 consumers |
| `internal/testutil/` | new | Shared test helpers. Zero production risk. | Create package, update 7+ test files |
| `internal/apicore/` | new | Shared API layer. Medium risk — largest change, most files touched. | Create package, refactor httpapi + udsapi |
| `internal/apitest/` | new | Shared API test stubs. Zero production risk. | Create package, refactor 2 test files |
| `internal/httpapi/` | modified | Reduced to transport binding + httpapi-specific handlers. Medium risk. | Remove duplicated code, import apicore |
| `internal/udsapi/` | modified | Reduced to transport binding. Medium risk. | Remove duplicated code, import apicore |
| `internal/daemon/` | modified | File split only + procutil import. Low risk. | Split files, update imports |
| `internal/session/` | modified | File split + method extraction. Low risk. | Split files, extract methods |
| `internal/store/` | modified | File split + fileutil import + validation helpers. Low risk. | Split files, update imports |
| `internal/workspace/` | modified | File split only. Low risk. | Split files |
| `internal/skills/` | modified | Method extraction + reflect removal. Low risk. | Extract processSkill, update imports |
| `internal/config/` | modified | Add exported path resolver. Low risk. | Add ResolveUserAgentsSkillsDir |
| `internal/memory/` | modified | fileutil + procutil imports. Low risk. | Update imports |
| `internal/cli/` | modified | procutil import + minor dedup. Low risk. | Update imports |

## Testing Approach

### Unit Tests

- Every new package (`procutil`, `fileutil`, `testutil`, `apicore`) ships with its own unit tests.
- `procutil`: test `Alive` with current PID (should return true) and PID 0 (should return false).
- `fileutil`: test `AtomicWriteFile` with `t.TempDir()` — verify content, permissions, and that partial writes don't corrupt the target.
- `apicore`: test payload conversion functions, query parsers, and SSE formatting independently of Gin context where possible.
- Existing tests for refactored packages must continue to pass unchanged (the refactoring does not change behavior).

### Integration Tests

- After Phase 3 (apicore extraction), run the full `httpapi` and `udsapi` test suites to verify transport-level behavior is preserved.
- `make verify` (fmt → lint → test → build) is the gate for every commit across all phases.
- The `-race` flag is already enforced by `make test` — no additional configuration needed.

### Coverage

- 80% minimum per package (project standard).
- New packages (`procutil`, `fileutil`) should target >95% — they are small and pure.

## Development Sequencing

### Build Order

1. **Phase 1.1–1.3: Create utility packages** (`procutil`, `fileutil`, `testutil`) — no dependencies on other phases. These are leaf packages.
2. **Phase 1.4–1.10: Inline deduplication** — no dependencies. Each is an independent commit. Can be parallelized.
3. **Phase 2.1–2.7: File-level splits** — depends on Phase 1 being complete (to avoid merge conflicts with moved code). Each split is independent of other splits.
4. **Phase 3.1: Shared interfaces** — depends on Phase 2.5 (udsapi handlers split) to reduce diff noise.
5. **Phase 3.2–3.6: Shared payloads, conversions, parsers, SSE, errors** — depends on 3.1. Can be done in any order within this group.
6. **Phase 3.7–3.8: Shared handlers** — depends on 3.2–3.6 (handlers use payloads, conversions, parsers, SSE).
7. **Phase 3.9: Test consolidation** — depends on 3.7–3.8 (tests must match the new handler locations).
8. **Phase 4: Domain deduplication** — depends on Phase 2 (file splits). Each item is independent. Execute opportunistically.

### Technical Dependencies

- **No external dependencies** — all work is internal refactoring.
- **No infrastructure changes** — no database migrations, no config format changes.
- **No API contract changes** — HTTP and UDS endpoints preserve identical request/response shapes.
- **Go 1.25** confirmed — generics fully available for `listBundle[T]` and any typed helpers.

## Technical Considerations

### Key Decisions

**1. Expand `apisupport/` vs. create new `apicore/`**
- Decision: Create new `apicore/` package rather than expanding `apisupport/`.
- Rationale: `apisupport` currently has a narrow scope (workspace/session validation). The shared handler layer is a fundamentally different responsibility. A new package with a clear name avoids overloading `apisupport` and signals the architectural intent.
- Trade-off: One more package in `internal/`. Acceptable given it eliminates ~900 lines of duplication.

**2. File-level splits vs. package splits for bloated files**
- Decision: File-level splits within the same package (no new sub-packages).
- Rationale: Go packages are the encapsulation boundary, not files. File splits preserve all existing import paths and require zero consumer changes. Consistent with CLAUDE.md: "File-level organization within packages — sub-packages only when complexity justifies it."
- Trade-off: The package itself remains large in terms of exported API surface. Acceptable — the files provide navigability without architectural overhead.

**3. `respondError` divergence handling**
- Decision: Single `RespondError(c, status, err, maskInternalErrors bool)` function in `apicore/`.
- Rationale: The divergence between httpapi (masks 5xx details) and udsapi (exposes raw errors) is intentional and must be preserved. A boolean parameter makes the choice explicit at each call site rather than relying on implicit copy-paste behavior.
- Trade-off: Slightly more verbose call sites. Worth it for explicitness.

**4. `agentEventPayload.Timestamp` type divergence**
- Decision: Base payload uses `time.Time`. httpapi defines a separate `agentEventPayloadAISdk` with `string` timestamps only for the AI SDK SSE stream endpoint.
- Rationale: `time.Time` is the correct Go representation. The string format is an AI SDK protocol requirement specific to one httpapi endpoint.
- Trade-off: httpapi has one extra type. Minimal overhead.

**5. Shared `fileSnapshot` location**
- Decision: If extracted, keep exactly one canonical snapshot implementation, either in a dedicated helper package (for example `internal/fsnap/`) or by promoting one existing package helper into a shared owner. Do not force it into `fileutil/` if that package would become a grab bag.
- Rationale: Both packages need identical file-metadata comparison for cache staleness detection. A single canonical helper eliminates divergence risk without over-constraining package layout.
- Trade-off: Adds one more shared dependency edge if a dedicated package is chosen. That is acceptable only if it reduces net complexity compared with keeping parallel implementations.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Merge conflicts with concurrent feature work | Medium | Execute Phase 1–2 first (smaller, isolated commits). Coordinate Phase 3 timing with team. |
| apicore extraction introduces subtle behavioral regression | Low | Existing test suites for both httpapi and udsapi provide strong regression coverage. Run `make verify` after each sub-step. |
| New packages proliferate beyond what's needed | Low | Hard cap: only `procutil`, `fileutil`, `testutil`, `apicore`, `apitest` are created. No speculative packages. |
| Legacy transcript parser removal breaks stored sessions | Medium | Audit `~/.agh/sessions/` for pre-canonical events before removing. If found, write a one-time migration script rather than keeping runtime compat code. |

## Monitoring and Observability

No monitoring changes required. This is a pure refactoring — no new runtime behavior, no new metrics, no new log events. The existing `make verify` pipeline (fmt → lint → test → build with `-race`) is the sole quality gate.

Post-refactoring, verify:
- `golangci-lint` reports zero issues (confirms no import cycles, unused code, or style violations).
- Test coverage per package remains ≥80%.
- Binary size does not meaningfully change (new packages add no new dependencies).
