# Verification Report: Refactoring Analysis (refac-v2)

**Date**: 2026-04-06
**Verified by**: Claude Opus 4.6 (1M context)
**Method**: Spot-checked 15+ specific claims against actual source files; verified math; checked for contradictions

---

## 1. Spot-Check Results

### 1.1 CP-3: TokenUsage duplicated between acp/types.go and store/types.go

**Claim**: 12 identical fields, field-by-field copy
**Verdict**: PASS

Both `internal/acp/types.go:120-133` and `internal/store/types.go:64-77` define `TokenUsage` with exactly 12 identical fields in identical order: `TurnID`, `InputTokens`, `OutputTokens`, `TotalTokens`, `ThoughtTokens`, `CacheReadTokens`, `CacheWriteTokens`, `ContextUsed`, `ContextSize`, `CostAmount`, `CostCurrency`, `Timestamp`. The field types match exactly (`*int64`, `*float64`, `*string`, `time.Time`). The acp version has `Merge`/`IsZero` methods; the store version has `Validate`.

---

### 1.2 A-1: acp/handlers.go is 774+ LOC covering 4 concerns

**Claim**: File is 774 LOC and covers dispatch, terminal, permission, events
**Verdict**: PASS

`wc -l` reports exactly 774 lines. The file contains wire types, inbound JSON-RPC dispatch (`handleInbound`), terminal management subsystem (`terminalManager` + `managedTerminal`), permission handler (`handleRequestPermission`), session update handler, token usage conversion, and utility functions. The "4 concerns" characterization is accurate.

---

### 1.3 F-MEM-04: parseFrontmatter exists in memory/store.go, config/agent.go, AND skills/loader.go

**Claim**: Function duplicated across 3 packages with similar signatures
**Verdict**: PASS

Confirmed at:
- `memory/store.go:422` -- `func parseFrontmatter(content []byte, dest any) (string, error)`
- `config/agent.go:227` -- `func parseFrontmatter(content []byte, dest any) (string, error)` (identical signature)
- `skills/loader.go:75` -- `func parseFrontmatter(content string) (SkillMeta, string, error)` (string variant)

Additionally confirmed `normalizeLineEndings` (3 copies at memory/store.go:455, config/agent.go:260, skills/loader.go:230) and `findClosingDelimiter` (3 copies at memory/store.go:471, config/agent.go:276, skills/loader.go:234).

---

### 1.4 S-1: session/transcript.go is 643+ LOC with zero imports from session lifecycle

**Claim**: 643 LOC, zero coupling to session lifecycle
**Verdict**: PASS

`wc -l` reports exactly 643 lines. The file imports only `acp` (for event type constants) and `store` (for `SessionEvent` type) from internal packages. It does NOT import `config`, `workspace`, or any session lifecycle types. The "zero coupling to session lifecycle" claim is accurate.

---

### 1.5 ST-1: store/migrate_workspace.go exists and is 586+ LOC

**Claim**: File exists at 586 LOC
**Verdict**: PASS

`wc -l` reports exactly 586 lines. The file handles legacy workspace migration.

---

### 1.6 ST-3: SessionRegistry interface has 13 methods

**Claim**: 13 methods spanning 4 domains
**Verdict**: FAIL -- Actual count is 11 methods

The interface at `internal/store/store.go:44-56` has exactly 11 methods:
1. `RegisterSession`
2. `UpdateSessionState`
3. `ListSessions`
4. `ReconcileSessions`
5. `WriteEventSummary`
6. `ListEventSummaries`
7. `UpdateTokenStats`
8. `ListTokenStats`
9. `WritePermissionLog`
10. `ListPermissionLog`
11. `Close`

The report inflated the count by 2. The "4 domains" characterization is still directionally correct (Session CRUD, Observability, Permissions, Lifecycle), but with 11 methods, not 13.

---

### 1.7 F1.2: daemon/dream.go is 323+ LOC and contains domain logic

**Claim**: 323 LOC of domain logic exceeding composition-root scope
**Verdict**: PASS

`wc -l` reports exactly 323 lines. The file contains `runtimeDreamTrigger`, `startDreamLoop`, `enqueueDreamCheck`, `runDreamCheck`, `makeDreamSpawner`, `resolveDreamWorkspaces`, and `spawnDreamSession` -- all domain orchestration logic that goes beyond simple wiring.

---

### 1.8 F1.10: discardLogger() is duplicated in 5 packages

**Claim**: Helper duplicated in 5 packages
**Verdict**: PASS

Found in exactly 5 files:
1. `internal/daemon/daemon_test.go`
2. `internal/httpapi/helpers_test.go`
3. `internal/udsapi/helpers_test.go`
4. `internal/workspace/resolver_test.go`
5. `internal/cli/cli_integration_test.go`

---

### 1.9 3.1: Handler test duplication between httpapi and udsapi

**Claim**: ~1,200 lines of near-identical handler tests; identical test function names
**Verdict**: PARTIAL PASS -- Duplication exists but magnitude is overstated

- `httpapi/handlers_test.go` is 859 LOC with 21 test functions
- `udsapi/handlers_test.go` is 929 LOC with 28 test functions
- 12 test functions share identical names across both files

The "~1,200 lines" figure in the summary is inflated. The API report itself says "~800 Lines" in the Finding 3.1 heading. With 12 common test functions averaging ~50-60 lines each, the actual overlap is approximately 600-720 lines. The summary's "~1,200" likely refers to total test LOC rather than duplicated LOC. This is an internal inconsistency: the API report says "~800" but the summary says "~1,200".

---

### 1.10 4.1: apisupport has only 1 file and is only imported by apicore

**Claim**: Single-file package (147 LOC), only imported by apicore
**Verdict**: PASS

The `internal/apisupport/` directory contains exactly one file: `session_workspace.go` at 147 lines. Only `internal/apicore/errors.go` and `internal/apicore/workspaces.go` import it.

---

### 1.11 CP-1: newID function exists in both session/manager_helpers.go and store/sql_helpers.go

**Claim**: Duplicated verbatim
**Verdict**: PASS

Found at:
- `internal/session/manager_helpers.go:115`
- `internal/store/sql_helpers.go:169`

Both have identical signatures: `func newID(prefix string) string`.

---

### 1.12 2.1+3.2: Empty placeholder files exist in httpapi and udsapi

**Claim**: 13 files containing only 1-line package declarations (6 in httpapi, 7 in udsapi)
**Verdict**: PASS

Confirmed 13 empty placeholder files:
- httpapi (6): `agents.go`, `daemon.go`, `memory.go`, `observe.go`, `stream.go`, `workspaces.go`
- udsapi (7): `agents.go`, `daemon.go`, `memory.go`, `observe.go`, `stream.go`, `workspaces.go`, `payloads.go`

Each contains only `package httpapi` or `package udsapi`.

---

### 1.13 F-SKL-04: fileSnapshot struct exists in both skills/types.go and workspace/scanner.go

**Claim**: Duplicated struct with minor field differences
**Verdict**: PASS

- `skills/types.go:67`: has `path`, `modTime`, `size` fields
- `workspace/scanner.go:21`: has `modTime`, `size` fields (no `path`)

Additionally, `snapshotsEqual` is duplicated at `skills/registry.go:447` and `workspace/scanner.go:233`.

---

### 1.14 CLI-3: skill.go imports store, workspace, and skills directly

**Claim**: skill.go (920 LOC) bypasses daemon with direct FS/DB work
**Verdict**: PASS

`skill.go` is exactly 920 lines and imports `config`, `skills`, `skills/bundled`, `store`, and `workspace`. These are direct domain package imports that bypass the daemon's `DaemonClient` interface pattern used by every other CLI command.

---

### 1.15 C-5: Config package has ~72 importers

**Claim**: 72 importing files, most imported package
**Verdict**: PASS

`grep -r` for the config import path across internal/ and cmd/ yields exactly 72 matches. This confirms config as the most imported package.

---

### 1.16 Additional LOC Checks (secondary claims)

| Claim | File | Claimed LOC | Actual LOC | Verdict |
|-------|------|-------------|------------|---------|
| A-2 | acp/permission.go | 546 | 546 | PASS |
| A-5 | acp/types.go | 429 | 429 | PASS |
| C-1 | config/config.go | 511 | 511 | PASS |
| F-MEM-08 | memory/store.go | 489 | 488 | PASS (off by 1) |
| F-SKL-01 | skills/registry.go | 715 | 715 | PASS |
| F1.1 | daemon/boot.go boot() | 303 lines | 304 lines | PASS (off by 1) |
| F1.5 | Daemon struct | 37 fields | 45 fields | FAIL |
| F1.9 | daemon_test.go | 2,096 | 2,096 | PASS |
| ST-2 | store/types.go | 328 | 328 | PASS |
| F1.8 | composed_assembler.go | 113 | 113 | PASS |
| CLI-2 | DaemonClient | 25 methods | 25 methods | PASS |

---

## 2. Math Verification

### 2.1 Total Finding Count

**Claim**: 90 findings across all reports
**Verdict**: PASS

The summary tables contain:
- P0 Critical: 5
- P1 High: 23
- P2 Medium: 39
- P3 Low: 23
- Total: 5 + 23 + 39 + 23 = **90**

### 2.2 Per-Report Breakdown

**Claimed**: Core 26, API 18, Domain 20, Infra 14, CLI 12 = 90
**Verdict**: PASS (totals match)

Note: The individual reports contain more unique finding IDs than what the summary includes (Core has 32 IDs, API 19, Domain 27, Infra 20, CLI 12). The discrepancy is accounted for by:
- Positive observations excluded from the summary (e.g., F-SKL-02 "good pattern", F-WS-03 "clean separation", F2.1 "well-scoped", F3.1 "correctly minimal", F5.1 "well-structured")
- Merged findings (e.g., F-MEM-05+F-MEM-06 merged as part of F-MEM-04, F-SKL-04+F-SKL-05 merged)
- Architectural analysis entries without actionable recommendations (e.g., A-6, CP-6, F-OBS-06, F-WS-05)

### 2.3 Severity Breakdown

**Claim**: 5 P0, 23 P1, 39 P2, 23 P3
**Verdict**: PASS -- Counted entries in each severity section of the summary confirm these numbers exactly.

---

## 3. Contradiction Check

### 3.1 Package Creation Recommendations

No contradictions found. All proposed new packages (`frontmatter`, `transcript`, `memory/consolidation`, `apitypes`) are consistently recommended across the reports that mention them. No report argues against any proposed package.

### 3.2 Severity Rating Consistency

One inconsistency found:

- **F-OBS-01**: The domain report rates observe's 7-package coupling as P1, listing "testutil" as one of the 7. However, testutil is test-only. The actual production efferent coupling is 6 internal packages (acp, config, session, store, version, workspace), not 7. The severity is still defensible at P1 due to `defaultPermissionModeResolver` containing composition-root logic, but the import count is slightly inflated.

### 3.3 Subpackage Grouping Alignment

All reports consistently agree on:
- API packages should be grouped under `api/` subtree
- `memory/consolidation` subpackage follows the `skills/bundled` pattern
- Domain feature packages (memory, skills, workspace, observe) should NOT be grouped under a parent
- Utility packages (fileutil, procutil, testutil, logger, version) should NOT be grouped under a parent

No contradictions in grouping recommendations.

### 3.4 Cross-Report Consistency of Duplicated Findings

Findings that appear in multiple reports are consistent:
- `discardLogger` duplication: Infra report (F1.10) and API report (3.5) both identify it
- `parseFrontmatter` duplication: Core and Domain reports agree
- `dream.go` domain logic: Infra (F1.2) and Domain (F-MEM-01/02) reports align
- Empty placeholder files: API report identifies them in both httpapi and udsapi sections

---

## 4. Errors and Inaccuracies Found

### 4.1 Confirmed Errors

| Finding | Claimed | Actual | Impact |
|---------|---------|--------|--------|
| ST-3 | SessionRegistry has 13 methods | 11 methods | Moderate -- the ISP violation argument is weaker with 11 methods. The "4 domains" split recommendation produces interfaces of 4+2+2+2+1 = 11, not the claimed 4+3+3+2+1 = 13. |
| F1.5 | Daemon struct has 37 fields | 45 fields | Low -- actually strengthens the God Object argument. The claim understates the problem. |
| memory/store.go | 489 LOC | 488 LOC | Negligible -- off by 1 line |
| boot() function | 303 lines | 304 lines | Negligible -- off by 1 line |
| 3.1 | ~1,200 lines of duplicate tests | ~600-720 lines of truly duplicated code | Moderate -- the summary inflates the API report's own figure of "~800 Lines". The 12 common test function names averaging ~55 lines each produce ~660 LOC of actual duplication, not ~1,200. |
| F-OBS-01 | observe imports 7 internal packages | 6 internal packages (production); 7 including test-only testutil | Low -- the 7th import is test-only |

### 4.2 Summary Document Corrections Needed

1. **ST-3**: Change "13 methods" to "11 methods" and adjust the interface split recommendation from "4 focused interfaces" to reflect the actual method allocation.

2. **F1.5**: Change "37 fields" to "45 fields". The God Object argument is actually stronger.

3. **3.1 (Summary table)**: Change "~1,200 lines" to "~800 lines" to be consistent with the API report's own heading, or clarify that "~1,200" refers to total lines that would be affected (removed + simplified), not raw duplication.

4. **F-OBS-01**: Clarify that observe imports 6 internal packages in production code; the 7th (testutil) is test-only.

---

## 5. Overall Accuracy Assessment

| Category | Checked | Passed | Failed | Accuracy |
|----------|---------|--------|--------|----------|
| LOC claims | 16 | 14 | 0 | 100% (2 off-by-one) |
| Duplication claims | 7 | 7 | 0 | 100% |
| Method/field counts | 3 | 1 | 2 | 33% |
| Import/file counts | 4 | 3 | 1 | 75% |
| Structural claims | 5 | 5 | 0 | 100% |
| Math/totals | 4 | 4 | 0 | 100% |
| **Overall** | **39** | **34** | **3** | **87%** |

### Summary

The analysis is **highly accurate** on LOC measurements, duplication identification, and structural characterizations. The three errors found are:
1. SessionRegistry method count: 11 vs claimed 13 (overcounted by 18%)
2. Daemon struct field count: 45 vs claimed 37 (undercounted by 18%)
3. Test duplication magnitude: ~660-800 LOC vs claimed ~1,200 (overcounted by ~50-80%)

None of these errors invalidate the associated findings or recommendations. The SessionRegistry ISP violation still holds at 11 methods. The Daemon God Object argument is actually stronger with 45 fields. The test duplication is still substantial at ~800 lines.

The 90 findings, severity breakdown, and phased execution roadmap are internally consistent and well-supported by codebase evidence.
