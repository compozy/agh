# Memory Standard Upgrade Regression Suite

## Objective

Protect the new memory catalog, recall, and observability behavior from regressions after follow-up fixes, refactors, or adjacent daemon/API/CLI work. This suite assumes the durable source of truth remains Markdown memory files and that the SQLite catalog is derived and rebuildable.

## Regression Tiers

### Smoke Suite

**Duration:** 15-30 minutes  
**Frequency:** per build, before deeper QA  
**Pass Gate:** all smoke tests must pass before continuing

| Order | Test Case | Priority | Coverage |
| --- | --- | --- | --- |
| 1 | `SMOKE-001` | P0 | Search returns the expected workspace hit from a mixed corpus |
| 2 | `SMOKE-002` | P0 | Reindex succeeds and health reflects indexed files and last reindex |

### Targeted Suite

**Duration:** 30-60 minutes  
**Frequency:** every change touching `internal/memory`, `internal/session`, `internal/api/*/memory*`, `internal/cli/memory.go`, or `internal/store/globaldb/*`

| Order | Test Case | Priority | Coverage |
| --- | --- | --- | --- |
| 1 | `TC-FUNC-001` | P0 | Write/delete synchronization between file store, `MEMORY.md`, and catalog |
| 2 | `TC-FUNC-002` | P0 | Safe prompt-index synthesis for missing/stale `MEMORY.md` |
| 3 | `TC-FUNC-003` | P1 | Ranking, scope awareness, and limit handling |
| 4 | `TC-FUNC-004` | P0 | Explicit reindex rebuild after catalog drift |
| 5 | `TC-INT-001` | P1 | HTTP API search/reindex contract |
| 6 | `TC-INT-002` | P1 | UDS and CLI parity |
| 7 | `TC-INT-003` | P0 | Recall augmentation versus stored raw user message |
| 8 | `TC-REG-001` | P1 | Health payload memory stats |
| 9 | `TC-REG-002` | P1 | Global observe summaries include memory operations |
| 10 | `TC-REG-003` | P0 | `.agh/memory` workspace-root regression guard |
| 11 | `TC-SEC-001` | P1 | Validation failures for bad scope/workspace/limit |

### Full Regression Suite

**Duration:** 2-4 hours  
**Frequency:** weekly or before release candidate cut

Run everything in the targeted suite plus:

| Order | Test Case | Priority | Coverage |
| --- | --- | --- | --- |
| 12 | `TC-PERF-001` | P2 | Corpus-scale search/reindex sanity and allocation trend |

### Sanity Suite

**Duration:** 10-15 minutes  
**Frequency:** after hotfixes that claim to fix memory search, reindex, or recall

| Order | Test Case | Priority | Coverage |
| --- | --- | --- | --- |
| 1 | `SMOKE-001` | P0 | Search still finds the intended workspace memory |
| 2 | `TC-INT-003` | P0 | Prompt recall still augments dispatch only |
| 3 | `TC-REG-001` | P1 | Health still reports catalog stats |

## Priority Model

### P0

- Source-of-truth integrity
- Workspace isolation
- Transcript preservation
- Legacy workspace-root regression

### P1

- Transport parity
- Operator visibility
- Validation and contract quality

### P2

- Corpus-scale performance sanity

## Execution Order

1. Run Smoke.
   - Stop immediately if any smoke case fails.
2. Run all P0 cases.
   - Fail the build or release candidate if any P0 case fails.
3. Run all P1 cases.
   - Continue only if failures are documented and non-critical.
4. Run P2 cases.
   - Use results to detect trend regressions and investigate if materially worse.
5. Perform exploratory follow-up around any failing surface.

## Pass / Fail Criteria

### PASS

- All P0 tests pass.
- At least 90% of P1 tests pass.
- No Critical or High bugs remain open in memory integrity, workspace scoping, recall injection, or operator visibility.
- No data-loss scenario is observed during write/delete/reindex flows.

### FAIL

- Any P0 test fails.
- Search returns cross-workspace leakage or wrong-scope dominant results for the same corpus.
- Stored raw user message contains the injected recall block.
- Reindex completes but `indexed_files`/`last_reindex` are absent or inconsistent with the corpus.
- Health or observe surfaces regress enough to hide memory activity or catalog drift.

### CONDITIONAL PASS

- A P1 test fails but a safe workaround exists, the risk is documented, and a fix plan is scheduled.
- `TC-PERF-001` misses a soft target on a noisy host but shows no user-visible regression in smoke/targeted flows.

## Regression Coverage Map

| Risk Area | Tests |
| --- | --- |
| Markdown/catalog synchronization | `SMOKE-002`, `TC-FUNC-001`, `TC-FUNC-004` |
| Missing/stale `MEMORY.md` behavior | `TC-FUNC-002`, `TC-REG-003` |
| Search ranking and workspace isolation | `SMOKE-001`, `TC-FUNC-003`, `TC-INT-001`, `TC-INT-002` |
| Public transport parity | `TC-INT-001`, `TC-INT-002` |
| Prompt recall behavior | `TC-INT-003` |
| Health metadata | `SMOKE-002`, `TC-REG-001` |
| Observe summaries | `TC-REG-002` |
| Input validation | `TC-SEC-001` |
| Corpus-scale performance | `TC-PERF-001` |

## Prerequisites for `qa-execution`

- Seed a temp global memory directory and a temp workspace under the current `.agh/memory` layout.
- Start daemon/API/UDS surfaces against that corpus when transport cases run.
- Use real SQLite files to verify catalog rebuilds and health stats.
- Capture verification evidence under `qa/screenshots/` only if a UI surface is unexpectedly introduced during execution.
