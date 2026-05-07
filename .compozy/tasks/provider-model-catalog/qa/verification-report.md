# Provider Model Catalog - Verification Report

## Run Metadata

- **Date:** 2026-05-07
- **Operator:** Codex QA agent
- **Branch:** `fix-migrations`
- **Commit:** `2debf0cf` at Task 13 start; working tree is dirty with Task 13 QA fixes/artifacts and unrelated pre-existing changes.
- **Bootstrap manifest path:** `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`
- **Lab root:** `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab`
- **Runtime home (`AGH_HOME`):** `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`
- **HTTP base URL:** `http://127.0.0.1:62444`
- **UDS socket:** `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/aghd.sock`
- **`AGH_WEB_API_PROXY_TARGET`:** `http://127.0.0.1:62444`
- **tmux-bridge socket:** `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/tmux-bridge.sock`
- **Provider home:** `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/provider`
- **Provider Codex home:** `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/provider/.codex`
- **Browser mode:** Browser-use requested, but the callable browser-use runtime was unavailable in this session; manual browser coverage used the installed `agent-browser` fallback and required Playwright E2E used `make test-e2e-web`.

```yaml
QA_BOOTSTRAP:
  manifest_path: /Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json
  lab_root: /Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab
  runtime_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime
  base_url: http://127.0.0.1:62444
  uds_socket: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/aghd.sock
  tmux_bridge_socket: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime/tmux-bridge.sock
  agh_web_api_proxy_target: http://127.0.0.1:62444
  provider_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/provider
  provider_codex_home: /var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/provider/.codex
  qa_output_path: /Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts
  browser_mode: agent-browser-fallback-plus-playwright-e2e
  provider_attempt: blocked-no-live-provider-credentials
```

## Smoke Readiness

| Step | Command | Result | Evidence |
|------|---------|--------|----------|
| 1 | `make build` | PASS | `qa/logs/make-build.log` |
| 2 | `make codegen-check` | PASS | `qa/logs/make-codegen-check.log` |
| 3 | `make bun-typecheck` | PASS | `qa/logs/make-bun-typecheck.log` |
| 4 | `make bun-test` | PASS | `qa/logs/make-bun-test.log` |
| 5 | Focused Go race gate with `-parallel=4` | PASS | `qa/logs/focused-go-gates-parallel4.log` |
| 6 | `./bin/agh daemon start --foreground` in isolated lab | PASS | `qa/logs/daemon.log`, `qa/logs/daemon-status.json` |
| 7 | Seed `providers.codex.models.curated` through Settings API and restart daemon | PASS | `qa/logs/http-put-settings-provider-codex.txt`, `qa/logs/daemon-restarted.log` |

## Test Case Results

| TC | Title | Priority | Result | Notes / BUG IDs |
|----|-------|----------|--------|-----------------|
| SMOKE-001 | Daemon startup and baseline catalog readiness | P0 | PASS | Isolated daemon served HTTP/UDS with seeded catalog state. |
| TC-FUNC-001 | Provider config hard cut | P0 | PASS | Covered by Task 11 regressions and final `make verify`. |
| TC-FUNC-002 | Curated validation rules | P1 | PASS | Covered by config/model catalog test suites in final `make verify`. |
| TC-FUNC-003 | Builtin source priority 10 | P1 | PASS | Covered by modelcatalog tests in final `make verify`. |
| TC-FUNC-004 | Merge determinism | P0 | PASS | Covered by modelcatalog tests in final `make verify`. |
| TC-FUNC-005 | `models.dev` TTL/disable/aliases | P1 | PASS | Covered by modelcatalog live-source tests in final `make verify`. |
| TC-FUNC-006 | Stale fallback | P1 | PASS | Covered by modelcatalog tests and source status checks. |
| TC-FUNC-007 | Partial vs all-source failure | P0 | PASS | Covered by modelcatalog service tests in final `make verify`. |
| TC-FUNC-008 | Live provider timeout | P1 | PASS | Covered by fake provider/source timeout tests; live-provider annex blocked. |
| TC-FUNC-009 | No ACP session calls from discovery | P1 | PASS | Covered by live discovery tests in final `make verify`. |
| TC-FUNC-010 | ACP `set_config_option` precedence | P0 | PASS | Covered by ACP/session tests in final `make verify`. |
| TC-FUNC-011 | Extension manifest validation | P1 | PASS | Host API focused regression passed. |
| TC-FUNC-012 | Extension capability denial | P1 | PASS | Host API focused regression passed. |
| TC-FUNC-013 | Source error redaction | P0 | PASS | Covered by modelcatalog/API/Host API regressions in final `make verify`. |
| TC-FUNC-014 | Detached refresh deadline | P1 | PASS | Covered by daemon/modelcatalog tests in final `make verify`. |
| TC-FUNC-015 | Codegen + docs drift | P1 | CLEARED | BUG-001; reran generators and proved idempotence with no remaining generated artifact diff. |
| TC-INT-001 | Migration v23 fresh + reopen | P0 | PASS | Covered by globaldb/modelcatalog tests in final `make verify`. |
| TC-INT-002 | HTTP/UDS handler payloads | P0 | PASS | Native HTTP/UDS list/status/refresh exercised against daemon. |
| TC-INT-003 | Canonical JSON parity | P0 | PASS | `http-provider-models-list-codex.json` and `uds-provider-models-list-codex.json` matched after canonical `jq -S`. |
| TC-INT-004 | OpenAI projection HTTP-only | P0 | FAIL | HTTP-only registration and provider filter were observed; auth contract failed. See BUG-002. |
| TC-INT-005 | Extension success/denial | P0 | PASS | `host-api-models-go-test.log`. |
| TC-INT-006 | ACP SDK upgrade flows | P0 | FIXED | BUG-003; runtime E2E compile failures fixed and `make test-e2e-runtime` passed. |
| TC-PERF-001 | Refresh concurrency + coalesce | P0 | PASS | Covered by final `make verify`. |
| TC-PERF-002 | Detached refresh + shutdown join | P0 | PASS | Covered by daemon/modelcatalog tests in final `make verify`. |
| TC-SEC-001 | Secret redaction across surfaces | P0 | PASS | Covered by API/Host API regressions in final `make verify`. |
| TC-SEC-002 | OpenAI auth + envelope | P0 | FAIL | Missing and invalid bearer tokens returned HTTP 200 catalog data. See BUG-002. |
| TC-UI-001 | Settings source status + refresh | P1 | PASS | Agent-browser screenshot plus Playwright E2E. |
| TC-UI-002 | Manual entry + curated edit | P1 | PASS | Settings API/manual catalog seed plus new-session manual entry covered. |
| TC-UI-003 | New session ACP override | P1 | PASS WITH RISK | Manual entry path passed; workspace catalog projection gap remains BUG-005. |
| TC-REG-001 | Hard-cut residue scan | P1 | PASS | Covered by docs/config regressions in final `make verify`. |
| TC-REG-002 | Generated docs + CLI sync | P1 | CLEARED | BUG-001; `make codegen`, `make cli-docs`, and `make codegen-check` idempotence passed with no remaining generated artifact diff. |
| TC-SCEN-001 | Operator real journey | P0 | PARTIAL | Browser Settings and new-session flows exercised; live provider-backed session proof blocked by missing credentials. |
| TC-SCEN-002 | Agent real journey | P0 | FAIL | CLI/HTTP/UDS/Host API parity passed, but OpenAI auth failed and live provider-backed session proof blocked. |

## Daemon-Served Scenario Evidence

- CLI refresh/list/status for `codex` config source: `qa/logs/cli-provider-models-refresh-codex-config.json`, `qa/logs/cli-provider-models-list-codex-after-config-refresh.json`, `qa/logs/cli-provider-models-status-codex-after-restart.json`.
- HTTP native catalog list/status/refresh: `qa/logs/http-provider-models-list-codex.json`, `qa/logs/http-provider-models-status-codex.json`, `qa/logs/http-provider-models-refresh-codex-config.json`.
- UDS native catalog list/status: `qa/logs/uds-provider-models-list-codex.json`, `qa/logs/uds-provider-models-status-codex.json`.
- OpenAI-compatible projection:
  - No auth: `qa/logs/http-openai-models-codex-no-auth.txt` (BUG-002).
  - Bad bearer: `qa/logs/http-openai-models-codex-bad-token.txt` (BUG-002).
  - Bad origin OpenAI-shaped denial: `qa/logs/http-openai-models-codex-bad-origin.txt`.
  - Unknown provider filter: `qa/logs/http-openai-models-unknown-provider.txt`.
  - UDS route absence: `qa/logs/uds-openai-models-codex.txt`.
- Host API model source/grant checks: `qa/logs/host-api-models-go-test.log`.
- Browser/manual workflow evidence:
  - Settings > Providers screenshot: `qa/logs/browser-settings-providers.png`.
  - New-session model options snapshot: `qa/logs/browser-new-session-model-options.txt`.
  - Playwright E2E final pass: `qa/logs/make-test-e2e-web-rerun-3.log`.
- Real-scenario charter and journey log:
  - `qa/behavioral-scenario-charter.yaml`
  - `qa/scenario-contract.json`
  - `qa/journey-log.jsonl`
  - `qa/provider-attempt.json`

## Verification Commands Executed

| Command | Exit | Result | Evidence |
|---------|------|--------|----------|
| `make build` | 0 | PASS | `qa/logs/make-build.log` |
| `make codegen-check` | 0 | PASS | `qa/logs/make-codegen-check.log` |
| `make bun-typecheck` | 0 | PASS | `qa/logs/make-bun-typecheck.log` |
| `make bun-test` | 0 | PASS | `qa/logs/make-bun-test.log` |
| `go test -race -count=1 -parallel=4 ./internal/config ./internal/store/globaldb ./internal/modelcatalog/... ./internal/acp ./internal/api/... ./internal/cli ./internal/extension/...` | 0 | PASS | `qa/logs/focused-go-gates-parallel4.log` |
| `make codegen` | 0 | PASS | `qa/logs/make-codegen.log` |
| `make cli-docs` | 0 | PASS | `qa/logs/make-cli-docs.log` |
| `cd packages/site && bun run test -- provider-model-catalog-docs` | 0 | PASS | `qa/logs/site-provider-model-catalog-docs.log` |
| `CGO_ENABLED=1 go test -race -count=1 -parallel=4 ./internal/extension -run "TestHostAPIModels|TestModelSource|TestCapabilityModel"` | 0 | PASS | `qa/logs/host-api-models-go-test.log` |
| `make test-e2e-runtime` | non-zero first, 0 after BUG-003 fix | FIXED/PASS | `qa/logs/make-test-e2e-runtime.log`, `qa/logs/make-test-e2e-runtime-rerun.log` |
| `make test-e2e-web` | non-zero first, 0 after BUG-004 fix and BUG-005 classification | FIXED/PASS WITH OPEN BUG | `qa/logs/make-test-e2e-web.log`, `qa/logs/make-test-e2e-web-rerun-2.log`, `qa/logs/make-test-e2e-web-rerun-3.log` |
| `make codegen` idempotence | 0 | PASS | `qa/logs/make-codegen-idempotence.log` |
| `make cli-docs` idempotence | 0 | PASS | `qa/logs/make-cli-docs-idempotence.log` |
| `make codegen-check` final artifacts | 0 | PASS | `qa/logs/make-codegen-check-final-artifacts.log` |
| `make lint` after `configDefaultKey` fix | 0 | PASS | `qa/logs/make-lint-after-config-default-const.log` |
| `make verify` first final attempt | non-zero | FAIL | `qa/logs/final-make-verify.log`; blocked by `goconst` in `internal/cli/config.go`. |
| `make verify` rerun | 0 | PASS | `qa/logs/final-make-verify-rerun.log`; 9400 Go tests in 74.913s and package boundaries passed. |
| Real-scenario QA audit | 2 | BLOCKED AS EXPECTED | `qa/logs/real-scenario-audit-rerun.log`; C9 blocks because provider-backed live behavior is absent. |

## Filed Bugs

| BUG | Severity | TC | Status | Fix Commit |
|-----|----------|----|--------|------------|
| BUG-001: Generated contracts and CLI reference drift | Medium | TC-FUNC-015, TC-REG-002 | Cleared; no remaining generated diff | None. |
| BUG-002: `/api/openai/v1/models` returns catalog data without bearer auth | Critical | TC-SEC-002, TC-INT-004, TC-SCEN-002 | Open | None; requires explicit HTTP auth design decision or contract revision. |
| BUG-003: Runtime E2E ACP test agents missing v0.12.2 interface methods | High | TC-INT-006 | Fixed in working tree | None; no commit created because QA decision remains blocked. |
| BUG-004: Web E2E provider override flow expected native select | Medium | TC-UI-003 | Fixed in working tree | None; no commit created because QA decision remains blocked. |
| BUG-005: Workspace provider models not projected into session catalog | High | TC-UI-003, TC-SCEN-001 | Open | None; requires workspace-aware catalog contract design. |

## Live-Provider Annex

Live-provider annex was not executed. The isolated QA run used daemon-served config/source fixtures, Host API grant/denial tests, ACP mock/fake-provider E2E lanes, and manual browser coverage. No live provider credentials were available in the lab, and no provider-backed live ACP session IDs were produced.

Provider attempt evidence: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/provider-attempt.json`.

Real-scenario audit evidence:

- Report: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/qa-audit-report.md`
- JSON: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/qa-audit-report.json`
- Verdict: FAIL with blocker `C9`, which is the correct release-grade outcome when live provider-backed behavior is absent.

## Residual Risk

- **BUG-002 remains a blocking security/contract finding.** The OpenAI-compatible catalog endpoint returns model data without bearer auth, while the accepted QA contract expects 401/403 OpenAI-shaped errors. Current runtime code does not expose a generic HTTP API bearer-token authority, so the next step is a deliberate auth design decision or contract revision.
- **BUG-005 remains a blocking product/API finding for workspace overlays.** Workspace-scoped providers appear in the new-session provider list, but their curated model metadata is not available through provider-scoped catalog APIs. A fix affects API shape, OpenAPI/types, web query keys, and source identity semantics.
- **Live provider-backed session proof is blocked.** The scenario contract requires at least one provider-backed session for release-grade proof, but the isolated lab had no live credentials. The correct result is BLOCKED, not PASS.
- **Browser-use tool availability was blocked in this session.** Manual browser checks used `agent-browser` fallback and the required Playwright E2E lane passed.

## Final Verification

- `make verify` exit code: 0 on rerun.
- Log path: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/final-make-verify-rerun.log`
- Summary: `make verify` passed after a minimal `internal/cli/config.go` lint fix. The gate result proves the working tree builds/tests/lints, but does not clear the open QA findings.
- Cleanup: isolated daemon stopped cleanly; final status evidence is `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/daemon-status-final.json`.

## Sign-Off

- Reporter: Codex QA agent
- Date: 2026-05-07
- Decision: FAIL / BLOCKED
- Rationale: The technical verification gate is green, but Task 13 cannot be marked complete while BUG-002, BUG-005, and the live-provider-backed session boundary remain open.
