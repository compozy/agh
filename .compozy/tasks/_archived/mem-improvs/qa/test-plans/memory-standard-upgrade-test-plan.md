# Memory Standard Upgrade Test Plan

## Executive Summary

This plan covers the QA strategy for the memory-system improvements implemented on April 17, 2026 and captured in `.codex/plans/2026-04-17-memory-standard-upgrade.md`. The change set introduces a derived SQLite FTS5 catalog, resilient prompt-index synthesis when `MEMORY.md` is missing or stale, public search and reindex surfaces in HTTP/UDS/CLI, bounded pre-dispatch prompt recall, and expanded health plus observability reporting.

The primary risks are silent data drift between Markdown memory files and the derived catalog, incorrect workspace scoping, prompt mutation leaking into persisted transcripts, and operational blind spots where health or observe surfaces do not reflect the true catalog state. QA therefore prioritizes source-of-truth integrity, transport parity, workspace isolation, and operator visibility.

## Scope

### In Scope

- Derived memory catalog behavior backed by the global SQLite database.
- Global and workspace memory search via store, HTTP, UDS, and CLI surfaces.
- Explicit memory reindex flows and recovery from catalog drift.
- `LoadPromptIndex` synthesis when `MEMORY.md` is missing, stale, or references missing files.
- Prompt recall injection before ACP dispatch, including bounds, scoring threshold, and transcript preservation.
- Health payload additions: memory enabled/config values, indexed file counts, orphaned file counts, last reindex timestamp.
- Global observe/event summary exposure for `memory.write`, `memory.delete`, `memory.search`, `memory.reindex`, and consolidation events.
- Regression coverage for workspace-root handling under `.agh/memory`.

### Out of Scope

- Web UI rendering and browser flows; no memory-specific UI changes were found in the current diff.
- Figma validation; there is no design artifact or UI surface involved in this backend-focused improvement.
- Dream-consolidation business rules beyond verifying that memory events remain observable and health metadata remains accurate.
- Vector or graph backends deferred to later phases.

## Requirements Traceability

| Requirement | Description | Primary Test Cases |
| --- | --- | --- |
| REQ-MEM-001 | Markdown files, `MEMORY.md`, and the derived catalog stay synchronized on write/delete. | `TC-FUNC-001`, `TC-FUNC-004`, `SMOKE-002` |
| REQ-MEM-002 | Prompt index loading synthesizes safe output when `MEMORY.md` is missing or stale. | `TC-FUNC-002`, `TC-REG-003` |
| REQ-MEM-003 | Search returns ranked, scope-aware results across transport surfaces. | `SMOKE-001`, `TC-FUNC-003`, `TC-INT-001`, `TC-INT-002` |
| REQ-MEM-004 | Reindex rebuilds the derived catalog and exposes completion metadata. | `SMOKE-002`, `TC-FUNC-004`, `TC-INT-001`, `TC-INT-002` |
| REQ-MEM-005 | Prompt recall is bounded and does not mutate the stored raw user message. | `TC-INT-003` |
| REQ-MEM-006 | Health payload exposes memory config and catalog stats. | `TC-REG-001` |
| REQ-MEM-007 | Observe summaries include memory operation events. | `TC-REG-002` |
| REQ-MEM-008 | Invalid scope/workspace/limit inputs are rejected consistently. | `TC-SEC-001` |
| REQ-MEM-009 | Workspace-root resolution uses the current `.agh/memory` layout rather than legacy assumptions. | `TC-REG-003` |
| REQ-MEM-010 | Search and reindex remain operational at realistic corpus size without pathological degradation. | `TC-PERF-001` |

## Test Strategy and Approach

### Strategy

1. Validate source-of-truth integrity first.
   - Confirm that write/delete/reindex flows preserve consistency between Markdown files, `MEMORY.md`, and the derived catalog.
2. Validate public interfaces second.
   - Exercise HTTP, UDS, and CLI search/reindex flows using the same underlying corpus and compare results.
3. Validate prompt-path behavior third.
   - Confirm recall injection occurs only on driver dispatch and never mutates the persisted user event.
4. Validate operator visibility last.
   - Confirm health and global observe surfaces reflect new catalog metadata and memory operation logs.

### Test Levels

- Smoke:
  - Quick confirmation that search and reindex work on a mixed global/workspace corpus.
- Functional:
  - Store-level synchronization, stale-index recovery, ranking, and reindex rebuild behavior.
- Integration:
  - API/UDS/CLI parity plus session-prompt augmentation behavior.
- Regression:
  - Health payload, observe summaries, and workspace-root derivation.
- Security:
  - Validation and isolation behavior for user-controlled scope/workspace/limit inputs.
- Performance:
  - Corpus-scale sanity check on search/reindex latency and allocation behavior.

### Execution Notes

- Prefer isolated temp workspaces and temp `AGH_HOME` roots.
- Use real SQLite files, not mocks, for reindex/catalog scenarios.
- Use the already-added repo tests as implementation evidence, but execute from public surfaces where practical during later `qa-execution`.
- When transport parity is checked, compare:
  - result count
  - top-hit ordering
  - `scope`, `workspace`, `score`, and `snippet` fields
  - reindex `indexed_files` and `completed_at`

## Environment Requirements

| Area | Requirement |
| --- | --- |
| OS | macOS 14+ or Linux with filesystem semantics compatible with repo tests |
| Go Toolchain | Repository-default Go toolchain required by `make verify` |
| Database | Local SQLite with FTS5 enabled |
| Runtime Paths | Writable temp directories for `AGH_HOME`, workspace roots, and memory files |
| Interfaces | HTTP daemon surface, UDS surface, and `agh` CLI binary or `go run ./cmd/agh` |
| Test Data | Mixed corpus of global and workspace memories with distinct types and timestamps |

## Entry Criteria

- Current branch contains the memory-standard-upgrade diff described in the persisted plan/ledger.
- QA artifact directory exists at `.compozy/tasks/mem-improvs/qa/`.
- A deterministic temp workspace can be created with both global and workspace memory directories.
- HTTP/UDS/CLI transport surfaces are buildable and can address the same memory corpus.
- Real SQLite database files can be created for catalog and health-stat verification.

## Exit Criteria

- All smoke cases pass.
- All P0 cases pass.
- At least 90% of P1 cases pass.
- No open Critical or High defects remain in source-of-truth integrity, workspace isolation, or transcript preservation.
- Traceability matrix is complete for REQ-MEM-001 through REQ-MEM-010.
- Any blocked scenario is documented with exact prerequisite and impact.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Catalog drifts from Markdown source after write/delete/reindex | Medium | Critical | Prioritize `TC-FUNC-001` and `TC-FUNC-004`; compare file state, `MEMORY.md`, and search results. |
| Search leaks wrong workspace results or ranks global above workspace unexpectedly | Medium | High | Run smoke plus parity tests with mixed corpora and explicit workspace filters. |
| Recall block changes persisted transcript content or duplicates user text | Low | High | Validate event storage and ACP dispatch separately in `TC-INT-003`. |
| Health payload reports stale or partial catalog stats | Medium | Medium | Reindex before health check and validate `indexed_files`, `orphaned_files`, and `last_reindex`. |
| Observe summaries omit memory events, reducing operator visibility | Medium | Medium | Include `TC-REG-002` in every targeted and full regression run. |
| Legacy workspace-root assumption (`.compozy/memory`) reappears | Low | High | Lock regression with `TC-REG-003` against `.agh/memory` workspaces. |
| Search/reindex degrades on a realistic corpus size | Medium | Medium | Run `TC-PERF-001` against a seeded corpus and compare against baseline expectations. |

## Timeline and Deliverables

### Planned Deliverables

- Test plan:
  - `qa/test-plans/memory-standard-upgrade-test-plan.md`
- Regression suite:
  - `qa/test-plans/memory-standard-upgrade-regression.md`
- Manual test cases:
  - `qa/test-cases/SMOKE-001.md`
  - `qa/test-cases/SMOKE-002.md`
  - `qa/test-cases/TC-FUNC-001.md`
  - `qa/test-cases/TC-FUNC-002.md`
  - `qa/test-cases/TC-FUNC-003.md`
  - `qa/test-cases/TC-FUNC-004.md`
  - `qa/test-cases/TC-INT-001.md`
  - `qa/test-cases/TC-INT-002.md`
  - `qa/test-cases/TC-INT-003.md`
  - `qa/test-cases/TC-REG-001.md`
  - `qa/test-cases/TC-REG-002.md`
  - `qa/test-cases/TC-REG-003.md`
  - `qa/test-cases/TC-SEC-001.md`
  - `qa/test-cases/TC-PERF-001.md`

### Suggested Execution Order

1. Smoke suite (`SMOKE-001`, `SMOKE-002`)
2. Functional integrity (`TC-FUNC-*`)
3. Integration parity and prompt behavior (`TC-INT-*`)
4. Regression visibility and workspace-root checks (`TC-REG-*`)
5. Validation hardening (`TC-SEC-001`)
6. Corpus-scale sanity (`TC-PERF-001`)

## Coverage Gaps and Planned Follow-Up

- No dedicated Web UI or Figma validation is planned because the current diff does not expose new memory UI.
- No bug reports were created during this planning pass because `qa-report` did not execute flows; `qa-execution` should create `BUG-*.md` files if failures are discovered.
- Performance thresholds should be calibrated against the host running `qa-execution`; this plan defines relative acceptance, not hard-coded machine-specific timings.
