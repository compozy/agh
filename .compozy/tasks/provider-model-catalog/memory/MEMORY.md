# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task 01 completed the provider-config hard cut and regenerated contracts. Later tasks should build on `providers.<id>.models.*` and `[model_catalog.sources.models_dev]`, not the removed provider-level model fields.
- Task 02 completed model catalog persistence. Later tasks should use `internal/modelcatalog.Store` and the `GlobalDB` implementation instead of writing model catalog tables directly.
- Task 04 completed live provider discovery sources in `internal/modelcatalog`. Later daemon wiring should register `NewLiveProviderSources` with resolved providers, `HomePaths`, daemon base env, and a real provider secret resolver.
- Task 05 completed daemon-owned catalog wiring: later API/CLI/extension tasks should consume the injected `core.ModelCatalogService` from daemon/runtime handler dependencies instead of composing or locating `modelcatalog.Service` outside `internal/daemon`.
- Task 06 upgraded AGH ACP integration to `github.com/coder/acp-go-sdk` v0.12.2 and exposes active ACP session `configOptions` through `ACPCapsPayload.config_options` / `SessionConfigOptionPayload`.
- Task 07 completed native model catalog HTTP/UDS routes, the HTTP-only OpenAI-compatible `/api/openai/v1/models` projection, `agh provider models list|refresh|status`, generated OpenAPI/web contracts, and generated API-reference navigation support for Gin catch-all route matching.
- Task 08 completed extension model source contracts in local commit `fef35196`: manifests can provide `model.source`, AGH can call extension `models/list`, Host API exposes daemon-backed `models/list|refresh|status`, extension rows are validated through `internal/modelcatalog`, and generated TypeScript contracts include `ModelSource*` plus Host API model methods.
- Task 09 completed the web model catalog experience: new `web/src/systems/model-catalog/` system (adapter, query keys/options, hooks, `deriveActiveSessionOptions` helper); session-create dialog consumes the daemon catalog with stale/error/refresh states; Settings > Providers per-card source status + refresh; curated metadata is snapshot-preserved on save.
- Task 10 completed runtime docs hard cut + generated contracts: `packages/site` provider/config/agent docs use the nested `[providers.<id>.models]` block; new `packages/site/content/runtime/core/agents/model-catalog.mdx` covers native HTTP/UDS catalog endpoints, `/api/openai/v1/models`, refresh lifetime/coalescing rules, and the extension `model.source` contract; `[model_catalog.sources.models_dev]` and provider `models.discovery` are documented in config-toml.mdx; CLI reference `provider/models/{list,refresh,status}` is regenerated; new docs vitest `packages/site/lib/__tests__/provider-model-catalog-docs.test.ts` enforces no remaining flat-field claims outside the hard-cut warning copy.
- Task 11 completed in local commit `7566e79d test: harden provider model catalog regressions`: resolved the daemon refresh deadline regression, added hard-cut residue, redaction, refresh concurrency/SQLite contention, HTTP/UDS/CLI/OpenAI/Host API parity, ACP mock config option, and web fixture regressions, and verified the full monorepo gate before and after commit.
- Task 12 produced the QA program under `.compozy/tasks/provider-model-catalog/qa/`: coverage matrix, master test plan, regression suite, 33 test cases (SMOKE-001 + TC-FUNC-001..015 + TC-INT-001..006 + TC-PERF-001..002 + TC-SEC-001..002 + TC-UI-001..003 + TC-REG-001..002 + TC-SCEN-001..002), bug template, and verification report template. Task 13 must execute the regression suite from an isolated `agh-qa-bootstrap` lab and close out by renaming `verification-report-template.md` to `verification-report.md`.

## Shared Decisions
- ACP/session capability `supported_models` remains a separate runtime capability surface. Do not treat those references as old provider config residue when searching for Task 01 hard-cut leftovers.
- The model catalog schema is global migration v23, after v22 `memv2_memory_events`; future schema changes must append new migrations and preserve v1-v23 identities.
- `model_catalog_sources` is provider-scoped. Cross-provider sources such as `models_dev` should persist one source status per AGH provider and must not use a blank-provider sentinel.
- Live provider discovery is side-effect-free catalog work only: it must not call ACP `session/new`, `session/load`, or `session/set_*`; unavailable safe discovery paths are source status failures/disabled states, not session blockers.
- Catalog refresh lifetime is daemon-owned. Request callers may trigger refreshes, but the daemon wrapper detaches refresh work from request cancellation, applies a duration-based configured timeout, and joins outstanding refresh workers during daemon shutdown.
- Active ACP session config options are session-scoped runtime truth. Prefer ACP `session/set_config_option` for model/reasoning when conservative option IDs and values are advertised; use legacy `session/set_model` only when config options are absent and legacy model state advertises the requested model.
- Source error redaction must happen both at catalog persistence time and at public projection boundaries (HTTP/UDS/OpenAI/Host API/log-visible payloads) so unsanitized in-memory/test source errors cannot leak through alternate surfaces.

## Shared Learnings
- Task 02 stores `default_reasoning_effort` as nullable data and stores reasoning efforts in `model_catalog_reasoning_efforts`, replaced inside the same `BEGIN IMMEDIATE` transaction as source rows/status.
- Task 04 uses `provider_live:<provider_id>` source IDs with priority 110 and coalesces identical same-provider refresh scopes while serializing different same-provider scopes before provider-home/subprocess work.

## Open Risks
- Task 13 reproduced TC-SEC-002 failure: `/api/openai/v1/models?provider_id=codex` returns catalog data with no `Authorization` header and with `Authorization: Bearer bad-token`. Current `internal/api/httpapi` only has loopback/CORS guards and `HTTPConfig` has no generic bearer-token authority, so fixing the accepted bearer-auth contract likely requires an explicit HTTP API auth design decision rather than a narrow model-catalog patch.
- Task 13 reproduced a workspace overlay catalog gap: workspace-scoped provider `models.curated` metadata appears in new-session provider options but is not projected through provider-scoped daemon model catalog APIs, so the model selector falls back to manual entry. Fixing this likely needs a workspace-aware catalog contract across HTTP/UDS/OpenAPI/web query keys/source identity.
- Task 13 real-scenario audit blocks release-grade proof when no live provider-backed ACP session evidence exists. Stub/fake-provider and Host API evidence is still valuable, but must not be reported as live provider proof.

## Handoffs
- Task 03 can extend `internal/modelcatalog` service/source behavior on top of the existing store types; avoid duplicating row/status structs in another package.
- Future in-session AGH UIs should consume the model catalog through `@/systems/model-catalog` (`useProviderModels`, `useRefreshProviderModels`, `useProviderModelStatus`, `deriveActiveSessionOptions`) rather than re-deriving the catalog vs ACP `configOptions` precedence rules.
- Task 12/13 QA should reuse Task 11 regression coverage as baseline evidence, then focus on real-scenario operator flows instead of rediscovering unit-level parity/redaction/concurrency behavior.
- Future docs work should keep `packages/site/content/runtime/core/agents/model-catalog.mdx` truthful to daemon behavior — when adding new sources or refresh semantics, update the merge priority table and refresh-lifetime section there, not in providers.mdx.
