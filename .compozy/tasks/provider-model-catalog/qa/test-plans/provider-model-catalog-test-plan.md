# Provider Model Catalog - Master QA Plan

## Executive Summary

The provider model catalog program (Tasks 01-11) replaces the flat provider model fields with a daemon-owned, persisted, refreshable, agent-manageable catalog. It hard-cuts `default_model`, `supported_models`, and `supports_reasoning_effort`; introduces nested `[providers.<id>.models]`, `[model_catalog.sources.models_dev]`, and discovery config; persists rows in three new SQLite tables; exposes HTTP/UDS/CLI/Host API/web/`/api/openai/v1/models` projections; and upgrades ACP to `coder/acp-go-sdk@v0.12.2` with `session/set_config_option` semantics.

This plan defines the QA contract that Task 13 must execute. Every TechSpec safety invariant (SI-1..SI-13), every ADR decision, every public surface, every failure mode, and every cross-surface parity boundary has a concrete test case in `qa/test-cases/`.

### Objectives

- Prove the hard cut is complete: no production code reads `default_model`, `supported_models`, or `supports_reasoning_effort`; old TOML keys fail with deterministic errors.
- Prove the catalog merge policy is deterministic: priority ordering, freshness tie-break, source-id tie-break, lower-priority enrichment, merged availability states, partial success.
- Prove HTTP, UDS, CLI, Host API, web, and the OpenAI projection serve the same persisted catalog state.
- Prove refresh stays correct under concurrency, request cancellation, SQLite write contention, and source failure.
- Prove redaction is enforced at persistence, projection, and log boundaries.
- Prove ACP sessions respect `configOptions` and only fall back to `session/set_model` when config options are absent.
- Prove operator and agent can manage the catalog without web UI through CLI/HTTP/UDS/Host API.

### Out of Scope

- Droid discovery.
- Fake ACP sessions for discovery.
- `models.dev` as account-level availability proof.
- `models.curated` as an allowlist.
- Real-provider `models.dev`, OpenAI, Anthropic, Gemini, OpenRouter, Vercel, Ollama, OpenCode HTTP calls — opt-in only via env tags, not gated by `make verify`.

## Scope

### In-Scope Surfaces

- Go runtime: `internal/config`, `internal/store/globaldb`, `internal/modelcatalog`, `internal/acp`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli`, `internal/extension`, `internal/daemon`.
- Generated contracts: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, extension TS types.
- Web app: `web/src/systems/model-catalog/`, `web/src/systems/session/`, `web/src/systems/settings/`, `web/src/routes/_app/settings/providers.tsx`, web E2E fixtures.
- Docs: `packages/site/content/runtime/core/agents/{providers.mdx,model-catalog.mdx,extensions/*.mdx}`, `packages/site/content/runtime/core/configuration/config-toml.mdx`, generated CLI docs under `packages/site/content/runtime/cli/provider/models/*.mdx`.

### Out-of-Scope

- Real-provider live discovery validation (covered as opt-in scenario with explicit boundaries).
- Pricing/cost rendering changes outside the catalog payload.
- AGH Network protocol changes.

## Behavioral Scenario Charter

- **Startup situation**: Greenfield AGH alpha (no production users). Operator runs daemon locally with isolated `AGH_HOME`, custom ports, and a tmux-bridge socket. Provider env may include real or stubbed credentials per scenario.
- **Operator intent**: Add or refine a provider, see which models AGH knows about, refresh catalog state, and start a session against a chosen model with optional reasoning effort.
- **Expected business outcome**: The operator sees a coherent, deterministic, source-attributed catalog; manual model entry remains valid; sessions start without depending on network discovery; agent and operator perceive the same catalog state across surfaces.
- **AGH surfaces used**: HTTP (`/api/providers/...`, `/api/openai/v1/models`), UDS, CLI (`agh provider models {list|refresh|status}`), web Settings > Providers, web new-session dialog, extension Host API, ACP `session/set_config_option`.
- **Real provider/LLM expectation**: The daemon must function with stubbed live discovery (default in `make verify`); opt-in real-provider runs (`MODELCATALOG_LIVE=1`) document a single end-to-end refresh against `models.dev` and one configured ACP provider.
- **Blocked live-provider boundary**: `make verify` and CI runs use stub HTTP servers and fake subprocesses. Real-provider runs are opt-in; missing credentials are reported as source status, not failures.
- **Scenario contract minimums covered**: TC-SCEN-001 + TC-SCEN-002 collectively satisfy operator and agent journeys, cross-surface parity, manual entry, refresh under stress, and stale-state observation.

## Test Strategy

1. **Smoke readiness (entry criteria only)**: SMOKE-001 verifies daemon starts, web build succeeds, codegen is clean, focused Go gates compile. Smoke is not release-grade evidence.
2. **Unit tests** cover pure logic per package: config validation, schema migrations, catalog merge, redaction, source parsing, conversion helpers, ACP config option capture/apply.
3. **Integration tests** cover daemon-served HTTP/UDS handlers, CLI client, Host API capability gating, deterministic JSON byte parity, and migration boot reconciliation against a real `globaldb` instance.
4. **E2E (runtime + browser)** cover operator journeys end-to-end through `make test-e2e-runtime` and `make test-e2e-web` with fresh QA labs created via `agh-qa-bootstrap`.
5. **Failure / chaos** cover stale fallback, all-source failure, SQLite contention, request cancellation, concurrent refresh coalescing, and credential redaction.
6. **Codegen and docs** are gated through `make codegen-check`, `make bun-typecheck`, and the `provider-model-catalog-docs` vitest suite.

Each test case in `qa/test-cases/` declares Audit Coverage IDs that map back to `qa/test-plans/00-coverage-matrix.md`.

## Environment Requirements

- Go 1.23.x with `CGO_ENABLED=1` (`-race` parity).
- Bun and Node toolchain compatible with the repo `.tool-versions` / `.nvmrc`.
- macOS 15+ or Linux x86_64; SQLite 3.45+.
- `coder/acp-go-sdk@v0.12.2` available through `go mod`.
- Isolated lab via `agh-qa-bootstrap`: unique `AGH_HOME`, daemon ports, `AGH_WEB_API_PROXY_TARGET`, tmux-bridge socket.
- Browser: Chromium under Playwright; `browser-use:browser` primary, `agent-browser` fallback.
- Provider env: synthetic credentials by default; opt-in real credentials only under `MODELCATALOG_LIVE=1`.

## Entry Criteria

- `git status` clean for production code under test (only QA artifacts may be uncommitted).
- `make verify` passed at the previous commit.
- `agh-qa-bootstrap` produced a fresh `bootstrap-manifest.json` for the run.
- Unique `AGH_HOME`, ports, and `tmux-bridge` socket allocated per worktree.
- Bootstrap manifest exports `AGH_WEB_API_PROXY_TARGET` for any web QA.

## Exit Criteria

- All P0 cases pass.
- ≥90% of P1 cases pass; remaining failures have `qa/issues/BUG-NNN.md` with root-cause + fix.
- Cross-surface parity test (TC-INT-003) shows byte-equal canonical JSON between native HTTP and UDS, and structurally equivalent CLI / Host API rows.
- Redaction tests (TC-SEC-001, TC-FUNC-013) show no API key, OAuth token, or env-shaped secret in any logged or projected payload.
- `make verify` passes after any QA-driven fixes.
- `qa/verification-report.md` records bootstrap manifest path, lab root, runtime home, base URL, commands, results, bug links, and residual risk.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Hard-cut residue silently rehydrates old fields. | Medium | High | TC-FUNC-001 + TC-REG-001 + repository scan. |
| Refresh under concurrency corrupts SQLite rows or status. | Medium | Critical | TC-PERF-001 + per-provider serialization assertions. |
| Refresh request cancellation cancels detached work. | Medium | High | TC-FUNC-014 + TC-PERF-002 + `context.WithoutCancel` assertions. |
| Source error leaks credentials into logs/UI/Host API. | Low | Critical | TC-SEC-001 + redaction at persistence and projection. |
| Generated contracts drift from runtime payload. | Medium | High | TC-FUNC-015, `make codegen-check`. |
| ACP `session/set_config_option` regresses to legacy `set_model`. | Low | High | TC-FUNC-010 + TC-INT-006 fixtures from upgraded SDK. |
| `/api/openai/v1/models` accidentally registered on UDS. | Low | High | TC-INT-004 explicit registration check. |
| `models.dev` becomes account availability proof under UI label drift. | Low | Medium | TC-FUNC-005 + TC-UI-001 stale label assertions. |
| Browser E2E flake on slow runners. | Medium | Medium | Use Playwright retries, deterministic seed via `web/e2e/fixtures/runtime-seed.ts`. |

## Timeline and Deliverables

- Day 1: Bootstrap fresh lab, run focused gates, replay TC-FUNC and TC-INT cases.
- Day 2: TC-PERF, TC-SEC, TC-UI cases; file BUGs as discovered.
- Day 3: TC-SCEN cases, fix loops with regression tests, finalize `verification-report.md`, commit.

Deliverables are listed in Task 12 / Task 13 specs and in `qa/verification-report.md`.

## Scenario Contract

The following minimums must collectively be satisfied by the P0/P1 real-scenario cases (`TC-SCEN-001`, `TC-SCEN-002`):

- Agents: operator (human) + remote agent (CLI/HTTP/Host API consumer).
- Roles: catalog editor, catalog reader, session creator, extension model source provider.
- Channels: HTTP, UDS, CLI, web, Host API, generated docs, generated TS types.
- Task tree: every public surface that Tasks 07-09 touched.
- Provider-backed sessions: at least one ACP-backed session uses `session/set_config_option` semantics (mock SDK fixture acceptable when real provider is blocked).
- Cross-surface objects: catalog row, source status, refresh request id, model availability state, source error.
- Artifacts used later: catalog row written via Settings > Providers (TC-SCEN-001) is read by CLI in TC-SCEN-002.
- Disruption probes: stale fallback, refresh coalescing, redaction, extension denial, request cancellation.
- Required surfaces: HTTP, UDS, CLI, web, Host API, OpenAI projection.

## Auditor Mapping

- C4 actor/role coverage → TC-SCEN-001 (operator) + TC-SCEN-002 (agent).
- C5 channels → TC-INT-002, TC-INT-003, TC-INT-004, TC-INT-005, TC-UI-001..003.
- C6 task tree → TC-FUNC + TC-INT cover Tasks 01-11.
- C8 cross-surface truth → TC-INT-003.
- C9 live provider → TC-FUNC-008 (stub) + opt-in `MODELCATALOG_LIVE=1` annex.
- C10 artifact reuse → TC-SCEN-001 → TC-SCEN-002 catalog row hand-off.
- C11 disruption probes → TC-PERF-001, TC-PERF-002, TC-FUNC-013, TC-FUNC-014.
- C14 final verification → `qa/verification-report.md` records `make verify` output.

## Verification Commands (Required)

Task 13 must run all of the following from a clean isolated lab. Substitute paths with the bootstrap manifest output where applicable.

```bash
# 1. Activate isolated lab
.agents/skills/agh-qa-bootstrap/scripts/bootstrap.sh \
  --scenario provider-model-catalog \
  --output .compozy/tasks/provider-model-catalog/qa/lab
export AGH_HOME=$(jq -r '.runtime_home' .compozy/tasks/provider-model-catalog/qa/lab/bootstrap-manifest.json)
export AGH_WEB_API_PROXY_TARGET=$(jq -r '.web_api_proxy_target' .compozy/tasks/provider-model-catalog/qa/lab/bootstrap-manifest.json)

# 2. Codegen + docs gates
make codegen
make codegen-check
cd packages/site && bun run test -- provider-model-catalog-docs && cd -

# 3. Focused Go gates
go test -race ./internal/config ./internal/store/globaldb ./internal/modelcatalog/... \
  ./internal/acp ./internal/api/... ./internal/cli ./internal/extension/...

# 4. Bun gates
make bun-typecheck
make bun-test
make web-build

# 5. E2E lanes
make test-e2e-runtime
make test-e2e-web

# 6. Optional live-provider annex (opt-in)
MODELCATALOG_LIVE=1 go test -tags=live ./internal/modelcatalog/... -run TestLive

# 7. Repo-wide gate
make verify
```

`make verify` is the final blocking gate. It must run last and pass with zero warnings.

## Bug Report Template

Every reproduced defect must use `assets/issue-template.md` (see `qa/issues/BUG-NNN-template.md`). Each bug records reproduction, root cause, fix, verification, and links the failing TC-ID.

## Verification Report Template

Task 13 closes the run by writing `qa/verification-report.md` (template at `qa/verification-report-template.md`) with:

- Bootstrap manifest path.
- Lab root, runtime home, base URL, ports, tmux socket.
- Commands executed (verbatim) with results and durations.
- Test case index with pass/fail/blocked status.
- Bug links and root-cause summaries.
- Residual risk + recommended follow-up.
- Final `make verify` evidence.
