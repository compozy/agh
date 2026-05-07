# Provider Model Catalog - Regression Suite

This suite drives the existing test cases in `qa/test-cases/` through tiered execution that Task 13 follows.

## Tiered Execution

| Suite | Duration | Frequency | Cases |
|-------|----------|-----------|-------|
| Smoke | ≤15 min | Per change | SMOKE-001 (daemon start, focused gates compile, web build, docs vitest, codegen-check) |
| Targeted | 30-60 min | Per task PR | All TC-FUNC-* + relevant TC-INT-* for changed surfaces |
| Full Release | 2-3 h | Release / Task 13 | All TC-FUNC, TC-INT, TC-PERF, TC-SEC, TC-UI, TC-REG, TC-SCEN |
| Sanity | 10-15 min | After hotfix | TC-FUNC-001, TC-INT-002, TC-SCEN-001 happy path only |

## P0 Cases (must always pass)

- TC-FUNC-001: Old TOML keys rejected.
- TC-FUNC-004: Catalog merge + tie-break determinism.
- TC-FUNC-007: Partial-source success / all-source failure.
- TC-FUNC-010: ACP `session/set_config_option` precedence over `session/set_model`.
- TC-FUNC-013: Source error redaction at projection boundary.
- TC-INT-001: Global migration v23 fresh DB + reopen-after-restart.
- TC-INT-002: HTTP/UDS native catalog handlers serve daemon-owned projection.
- TC-INT-003: HTTP/UDS canonical JSON byte equality + CLI parity.
- TC-INT-004: `/api/openai/v1/models` HTTP-only registration + auth + provider filter.
- TC-INT-005: Extension `model.source` + Host API `models/list|refresh|status`.
- TC-INT-006: ACP fixtures from upgraded SDK keep create/load/resume covered.
- TC-PERF-001: Per-provider refresh serialization + coalescing under concurrency.
- TC-PERF-002: Detached refresh lifetime survives request cancellation.
- TC-SEC-001: No raw secrets across logs / status / API / SSE / web / Host API.
- TC-SEC-002: `/api/openai/v1/models` rejects unauthenticated calls with OpenAI-shaped error.
- TC-SCEN-001: Operator real journey through web → HTTP → SQLite → ACP.
- TC-SCEN-002: Agent real journey through CLI → HTTP/UDS → Host API.

## P1 Cases (≥90% pass required)

- TC-FUNC-002: Curated config validation rules.
- TC-FUNC-003: Builtin source converts defaults to priority-10 rows.
- TC-FUNC-005: `models.dev` source TTL, disable, legacy alias parsing.
- TC-FUNC-006: Stale fallback when refresh fails after prior success.
- TC-FUNC-008: Live provider source timeout + per-provider env/home policy.
- TC-FUNC-009: Live discovery never calls ACP `session/*` mutators.
- TC-FUNC-011: Extension manifest validation + invalid row rejection.
- TC-FUNC-012: Extension capability missing/revoked is treated as denial.
- TC-FUNC-014: Refresh deadline detached from request context.
- TC-FUNC-015: Codegen drift gate for OpenAPI / TS contracts / docs.
- TC-REG-001: Hard-cut residue scan.
- TC-REG-002: Generated docs and CLI reference stay in sync.
- TC-UI-001: Settings > Providers source status + refresh state.
- TC-UI-002: Settings > Providers manual entry + curated edit.
- TC-UI-003: New session dialog uses ACP `configOptions` post-creation.

## P2 / Exploratory

- Manual exploratory probes documented in `qa/verification-report.md`:
  - Toggle `[model_catalog.sources.models_dev].enabled = false` and verify status reflects disabled state without outbound HTTP.
  - Disable extension grant mid-run and observe CLI/Host API states.
  - Force `models.dev` 5xx and observe stale rows persisted.

## Pass/Fail Criteria

- **PASS**: All P0 cases pass; ≥90% P1 pass; remaining P1 failures have BUGs filed with root cause + fix; `make verify` clean.
- **FAIL**: Any P0 fails; secret material leaks anywhere; cross-surface parity diverges; SQLite contention causes BUSY errors that escape coalescing; ACP regression fallback path executes when config option exists.
- **CONDITIONAL**: P1 failure only with documented workaround AND scheduled fix in `qa/verification-report.md`.

## Execution Order

1. Smoke (SMOKE-001) — block on failure.
2. P0 unit + integration cases (TC-FUNC + TC-INT).
3. P0 perf + security cases (TC-PERF, TC-SEC).
4. P1 cases.
5. UI cases (TC-UI) under Playwright.
6. Real-scenario cases (TC-SCEN).
7. Final `make verify`.

## Reporting

- Update `qa/verification-report.md` after each case batch.
- File `qa/issues/BUG-NNN.md` for every reproduced defect with TC-ID linkage.
- Update `qa/test-cases/<TC-ID>.md` execution history table.
