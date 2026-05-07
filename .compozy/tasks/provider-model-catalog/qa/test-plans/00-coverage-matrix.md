# Provider Model Catalog - QA Coverage Matrix

This matrix maps every TechSpec safety invariant, ADR decision, and implementation task to the concrete test cases authored under `qa/test-cases/`. Task 13 must run every case listed here. A blank row is a Task 13 blocker.

## Source Authorities

- TechSpec: `.compozy/tasks/provider-model-catalog/_techspec.md` (Safety Invariants 1-13, Testing Approach, Observability).
- ADRs: `adrs/adr-001-daemon-owned-provider-model-catalog.md`, `adrs/adr-002-provider-model-config-hard-cut.md`, `adrs/adr-003-extension-model-source-contract.md`.
- Tasks: `task_01.md` through `task_11.md`.
- QA tail template: `.agents/skills/cy-tasks-tail-qa-pair/references/hermes-tail-template.md`.

## TechSpec Safety Invariants

| Invariant | Description | Test Cases |
|-----------|-------------|------------|
| SI-1 | Session creation never depends on successful network model discovery. | TC-SCEN-001, TC-FUNC-008, TC-INT-005 |
| SI-2 | Discovery must not create, load, mutate, or stop ACP sessions. | TC-FUNC-009, TC-INT-006 |
| SI-3 | Live discovery uses provider effective auth/home/env policy and explicit timeouts. | TC-FUNC-009, TC-FUNC-014 |
| SI-4 | Source refresh failure records source status and preserves prior stale rows. | TC-FUNC-006, TC-FUNC-013, TC-INT-002 |
| SI-5 | `models.dev` rows never prove account-level availability. | TC-FUNC-005, TC-INT-002 |
| SI-6 | `models.curated` is never an allowlist; manual model IDs remain valid. | TC-FUNC-002, TC-SCEN-002, TC-UI-002 |
| SI-7 | Active ACP `configOptions` override catalog metadata for that session only. | TC-FUNC-010, TC-INT-006, TC-UI-003 |
| SI-8 | Global catalog rows are only written through `internal/modelcatalog.Store`. | TC-FUNC-004, TC-INT-001 |
| SI-9 | No raw secrets, API keys, OAuth data, or credential material in source errors / logs / status / SSE / web / Host API. | TC-SEC-001, TC-FUNC-013, TC-INT-002 |
| SI-10 | SQLite schema changes append a new migration at the registry tail and pass fresh DB plus reopen-after-restart tests. | TC-INT-001 |
| SI-11 | HTTP/UDS request lifetime does not own background refresh; refresh uses `context.WithoutCancel(ctx)` + explicit deadline. | TC-FUNC-014, TC-PERF-002 |
| SI-12 | Live refresh work is serialized/coalesced per `provider_id` before touching `HOME`, native CLI auth state, cache files, or SQLite. | TC-PERF-001, TC-PERF-002 |
| SI-13 | Partial-source success is success; list fails only when every usable source fails and no stale cache exists. | TC-FUNC-007, TC-INT-002 |

## ADR Decisions

| ADR | Decision | Test Cases |
|-----|----------|------------|
| ADR-001 | Daemon-owned catalog with HTTP/UDS/CLI/Host API/web parity. | TC-INT-002, TC-INT-003, TC-INT-004, TC-SCEN-002 |
| ADR-002 | Hard cut of `default_model`/`supported_models`/`supports_reasoning_effort`. | TC-FUNC-001, TC-REG-001 |
| ADR-003 | Extension `model.source` capability + Host API `models/list|refresh|status`. | TC-FUNC-011, TC-FUNC-012, TC-INT-005 |

## Task Coverage

| Task | Title | Test Cases |
|------|-------|------------|
| 01 | Provider Config and Builtin Model Hard Cut | TC-FUNC-001, TC-FUNC-002, TC-REG-001 |
| 02 | Model Catalog Persistence | TC-INT-001 |
| 03 | Catalog Service and Catalog Sources | TC-FUNC-003, TC-FUNC-004, TC-FUNC-005, TC-FUNC-006, TC-FUNC-007 |
| 04 | Live Provider Discovery Sources | TC-FUNC-008, TC-FUNC-009 |
| 05 | Daemon Catalog Wiring | TC-INT-002, TC-PERF-001, TC-PERF-002 |
| 06 | ACP SDK Upgrade and Config Options | TC-FUNC-010, TC-INT-006, TC-UI-003 |
| 07 | HTTP, UDS, CLI, OpenAI Model Projection | TC-INT-002, TC-INT-003, TC-INT-004, TC-SEC-002 |
| 08 | Extension Model Source Contract | TC-FUNC-011, TC-FUNC-012, TC-INT-005 |
| 09 | Web Model Catalog Experience | TC-UI-001, TC-UI-002, TC-UI-003, TC-SCEN-001, TC-SCEN-002 |
| 10 | Generated Contracts and Runtime Docs | TC-FUNC-015, TC-REG-002 |
| 11 | Cross-Surface Regression Hardening | TC-FUNC-013, TC-FUNC-014, TC-PERF-001, TC-PERF-002 |

## Public Surface Coverage

| Surface | Endpoints / Commands | Test Cases |
|---------|----------------------|------------|
| HTTP native catalog | `GET /api/providers/models`, `GET /api/providers/{provider_id}/models`, `POST /api/providers/models/refresh`, `POST /api/providers/{provider_id}/models/refresh`, `GET /api/providers/models/status`, `GET /api/providers/{provider_id}/models/status` | TC-INT-002, TC-INT-003 |
| HTTP-only OpenAI projection | `GET /api/openai/v1/models`, `GET /api/openai/v1/models?provider_id=` | TC-INT-004, TC-SEC-002 |
| UDS native catalog | Same path family registered on UDS group, **never** the OpenAI projection. | TC-INT-002, TC-INT-003, TC-INT-004 |
| CLI | `agh provider models list [provider]`, `agh provider models refresh [provider]`, `agh provider models status [provider]`, with `--source`, `--refresh`, `--include-stale`, `-o json`. | TC-INT-002, TC-INT-003, TC-SCEN-002 |
| Extension Host API | `models/list`, `models/refresh`, `models/status` | TC-FUNC-011, TC-FUNC-012, TC-INT-005 |
| AGH -> extension | `models/list` request shape, capability gate. | TC-FUNC-011, TC-FUNC-012 |
| Web (Settings > Providers) | `web/src/routes/_app/settings/providers.tsx`, source status cards, refresh button, curated/default editor. | TC-UI-001, TC-UI-002 |
| Web (Session create dialog) | `web/src/systems/session/components/session-create-dialog.tsx`, model picker pulled from catalog, manual entry fallback. | TC-UI-003, TC-SCEN-001 |
| Web TanStack adapter | `web/src/systems/model-catalog/` query keys, hooks, adapter, `deriveActiveSessionOptions`. | TC-UI-003 |
| Generated contracts | `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, extension TS types. | TC-FUNC-015, TC-REG-002 |
| Docs | `packages/site/content/runtime/core/agents/model-catalog.mdx`, `providers.mdx`, `config-toml.mdx`, `cli/provider/models/*.mdx`, extension authoring docs. | TC-FUNC-015 |
| `config.toml` | `[providers.<id>.models]` (default, curated, discovery), `[model_catalog.sources.models_dev]`. | TC-FUNC-001, TC-FUNC-002, TC-INT-001 |
| Observability | Structured logs/events with `refresh_request_id`, `provider_id`, `source_id`, `source_kind`, `model_id`, `extension_name`. | TC-FUNC-013, TC-INT-005, TC-PERF-002 |
| Persistence | `model_catalog_sources`, `model_catalog_rows`, `model_catalog_reasoning_efforts` tables (global migration v23). | TC-INT-001 |

## Failure-Mode Coverage

| Failure / Edge Case | Cases |
|---------------------|-------|
| Old TOML keys present (`default_model`, `supported_models`, `supports_reasoning_effort`). | TC-FUNC-001, TC-REG-001 |
| Curated default not in curated list. | TC-FUNC-002 |
| Curated duplicate IDs / blank reasoning efforts / `default_reasoning_effort` not in list. | TC-FUNC-002 |
| `models.dev` HTTP 5xx, network timeout, JSON malformed, legacy field aliases. | TC-FUNC-005, TC-FUNC-006, TC-FUNC-013 |
| `models.dev` disabled via config. | TC-FUNC-005 |
| Live provider source timeout, subprocess failure, missing auth. | TC-FUNC-008, TC-FUNC-009 |
| Live provider source attempts ACP `session/new`/`set_*`. | TC-FUNC-009 |
| Stale source rows preserved across daemon restart. | TC-FUNC-006, TC-INT-001 |
| All sources fail, no stale cache exists. | TC-FUNC-007 |
| Source error contains API key / OAuth token / env secret. | TC-SEC-001, TC-FUNC-013 |
| Source error shape leaks beyond redaction at HTTP/UDS/Web/Host API. | TC-SEC-001, TC-INT-002 |
| Concurrent same-provider refresh. | TC-PERF-001 |
| Concurrent cross-provider refresh storm. | TC-PERF-001 |
| Repeated coalesced refresh returns same status batch. | TC-PERF-001 |
| Request cancellation during refresh detaches refresh lifetime. | TC-PERF-002, TC-FUNC-014 |
| SQLite `BUSY` write contention. | TC-PERF-001 |
| Extension capability missing or revoked. | TC-FUNC-012 |
| Extension manifest declares non-normalizable `model.source` slug. | TC-FUNC-011 |
| Extension `models/list` returns invalid rows. | TC-FUNC-011 |
| `/api/openai/v1/models` registered on UDS by mistake. | TC-INT-004 |
| `/api/openai/v1/models` unauthenticated request. | TC-SEC-002 |
| `/api/openai/v1/models?provider_id=unknown`. | TC-INT-004 |
| ACP `session/set_config_option` succeeds; `session/set_model` fallback only when config option absent. | TC-FUNC-010, TC-INT-006 |
| ACP session exposes no model option; reasoning never sent. | TC-FUNC-010 |
| Web: Settings > Providers refresh button surfaces stale state and last error. | TC-UI-001 |
| Web: New session dialog uses ACP `configOptions` after creation. | TC-UI-003 |
| Web: Manual model entry remains valid when curated empty. | TC-UI-002, TC-SCEN-001 |
| Generated docs / OpenAPI / TS types drift. | TC-FUNC-015, TC-REG-002 |

## Real-Scenario Mapping (TC-SCEN)

| TC-SCEN | Operator Journey | Surfaces | TechSpec Anchors |
|---------|-------------------|----------|-------------------|
| TC-SCEN-001 | Operator opens Settings > Providers, edits curated metadata, refreshes models, then creates a session and selects a model. | Web + HTTP + SQLite + ACP | SI-1, SI-6, SI-7 |
| TC-SCEN-002 | Agent driving CLI/HTTP/UDS lists, refreshes, and inspects model status without using the web UI. | CLI + HTTP + UDS + SQLite | SI-4, SI-12, SI-13 |

## Auditor Coverage

The TC-SCEN cases must satisfy:

- C4 actor/role coverage: operator + agent both exercise catalog surfaces.
- C5 channels: HTTP, UDS, CLI, web, Host API.
- C6 task tree: TC-SCEN cases reference Tasks 01-11.
- C8 cross-surface truth: TC-INT-003 / TC-SCEN-002 compare CLI/HTTP/UDS/Host API/web payloads.
- C9 live provider: TC-FUNC-008 documents the live discovery boundary; real-provider runs are opt-in.
- C10 artifact reuse: catalog rows produced in TC-SCEN-001 are reused by TC-SCEN-002.
- C11 disruption probes: stale, timeout, redaction, denial, and SQLite contention.
- C14 final verification: TC-SCEN-001 and TC-SCEN-002 require `make verify` evidence in `qa/verification-report.md`.
