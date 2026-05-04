# Tool Registry — Feature-Level QA Plan

- **Feature:** Tool Registry Foundation (Tasks 01-14)
- **Owner:** AGH Platform
- **Plan author:** task_15 (cy-execute-task)
- **Source artifacts:** `_techspec.md`, `adrs/adr-001..011`, `task_01..task_14`, `task_15` requirements, per-task memory in `memory/`
- **Companion execution task:** `task_16` (`qa-execution`, `real-scenario-qa`)
- **QA output root:** `.compozy/tasks/tools-registry/qa/`

## 1. Executive Summary

The Tool Registry replaces metadata-only tool records with an executable, daemon-owned registry that unifies identity, discovery, availability, policy, dispatch, hooks, telemetry, extension descriptors, MCP adapters, and session-visible exposure. It executes three backend classes (`native_go`, `extension_host`, `mcp`) and exposes session-callable tools through an AGH-hosted local MCP proxy plus shared CLI/HTTP/UDS contracts.

This QA plan covers the full surface from Task 01 contracts through Task 14 site documentation. It must prove that:

1. Every AGH-owned tool call enters `internal/tools.Registry.Call` (Safety Invariant 1).
2. Operator and session projections stay deterministic and intentionally divergent (ADR-006).
3. Policy decisions stay below the ACP `permissions.mode` ceiling (ADR-005).
4. Hosted MCP exposure preserves UDS peer/binary validation, single-use bind nonces, and the approval bridge (ADR-002).
5. Remote MCP credentials, bind nonces, and approval tokens never leak across surfaces (Safety Invariants 12, 16-21, 27).
6. Manifest-authoritative extension descriptors reconcile with runtime `provide_tools` digests (ADR-008).

## 2. Objectives and Key Risks

### Objectives

- Validate executable behavior across `native_go`, `extension_host` (TypeScript and Go), and `mcp` backends.
- Validate canonical `ToolID` grammar, collision rules, and pattern matching across registry, policy, CLI, HTTP, UDS, hooks, telemetry, and hosted MCP.
- Validate dispatch pipeline ordering: schema → policy/availability recheck → hooks → handle → result limiter → telemetry.
- Validate redaction across CLI JSON, HTTP JSON, UDS JSON, MCP responses, SSE/event payloads, logs, settings output, and process diagnostics.
- Validate approval flows: CLI/HTTP/UDS approval-token issuance and consumption, hosted MCP approval bridge with timeout/cancel/unreachable.
- Validate config lifecycle: `[tools]`, `[tools.policy]`, `[tools.hosted_mcp]`, agent `tools`/`toolsets`/`deny_tools`, validation bounds, overlays.
- Validate web diagnostics surface (Task 13) renders truthful daemon-backed state without inventing controls.
- Validate site documentation (Task 14) describes canonical ID, backend kinds, manifest reconciliation, MCP call-through, hosted MCP, approval bridge, and external CLI/HTTP/UDS surfaces.

### Key Risks

| ID | Risk | Probability | Impact | Mitigation |
|----|------|-------------|--------|------------|
| R-01 | Mutating tool mislabeled as `read_only` slips through `approve-reads` | Medium | Critical | TC-SEC-001..004; matrix tests across `native_go`/`extension_host`/`mcp`; dispatch-time recheck (T04) |
| R-02 | Remote MCP OAuth token leaks into descriptors, events, results, logs | Low | Critical | TC-SEC-005..008; redaction sentinel scans across all surfaces |
| R-03 | Hosted MCP bind nonce treated as bearer secret | Low | Critical | TC-SEC-009..010; UDS peer-credential and AGH-binary validation tests |
| R-04 | Approval token replay/leak across surfaces | Medium | High | TC-SEC-011..013; hash-only storage, single-use semantics, redaction |
| R-05 | Manifest/runtime descriptor digest drift between TypeScript/Go SDK/daemon | Medium | High | TC-INT-007/009; shared JCS digest fixtures (T06) |
| R-06 | Hosted MCP `tools/list` diverges from `GET /api/sessions/{id}/tools` | Medium | High | TC-INT-013/014; projection-stream parity |
| R-07 | Canonical ID collision or sanitized-name collision allows shadow tool | Low | High | TC-FUNC-016/017; fail-closed conflict |
| R-08 | Result budget bypass leaks oversized payload | Medium | Medium | TC-FUNC-031..033; truncation metadata |
| R-09 | Web UI invents login or invoke controls not backed by daemon | Medium | Medium | TC-UI-001..006 truthfulness gates |
| R-10 | Docs reference dotted IDs, descriptor-only callable extensions, or removed compatibility paths | Medium | Medium | TC-FUNC-051..053; grep gates |
| R-11 | `mcp_server.transport = "sse"` silently rewritten to `http` | Low | High | TC-FUNC-039 transport preservation tests |
| R-12 | Concurrency: two parallel calls into the same tool corrupt result limiter or hook order | Low | High | TC-PERF-001..003 race coverage |

## 3. Scope

### In Scope

- `internal/tools` runtime registry, policy evaluator, dispatch pipeline, projections, hooks, result limiting, telemetry seams.
- `internal/tools` MVP `native_go` providers: `agh__tool_list/search/info`, `agh__skill_list/search/view`, `agh__network_peers/send`, `agh__task_list/read/create/child_create/update/cancel/run_list`.
- `internal/extension` manifest tool metadata, schema digests, runtime reconciliation, `tool.provider` capability, `provide_tools`, `tools/call`.
- `sdk/typescript` `extension.tool(...)` registration and digest helpers.
- `sdk/go` public Go SDK and `go-tool-provider` template.
- `internal/mcp` `MCPCallExecutor`, hosted MCP proxy (`agh tool mcp --session --bind-nonce`), bind/UDS peer/binary validation, projection stream, approval bridge.
- `internal/api/contract`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi` routes for `/api/tools[...]`, `/api/sessions/{id}/tools[...]`, `/api/toolsets[...]`.
- `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` codegen parity.
- `internal/cli` `agh tool` and `agh toolsets` commands; preservation of `agh mcp auth`.
- `web/src/systems/tools/**` operator diagnostics surface (Task 13).
- `packages/site` runtime/core/extension/MCP/sessions/configuration docs and generated CLI/API references (Task 14).

### Out of Scope (Deferred per `_techspec.md` MVP boundary)

- Direct ACP/driver-specific tool injection beyond hosted MCP.
- Full shell/browser/file tool replacement for ACP runtimes.
- Remote peer tool execution over AGH Network.
- Provider-specific deferred schema loading (e.g. Anthropic `tool_reference`).
- Marketplace signing/trust overhaul.
- Skill install/remove/update tools, bridge SDK executable adapters, in-process plugin loading.
- Client-supplied ACP `mcpServers` as session-scoped registry sources.
- Excluded `agh__task_*` lifecycle tools (`claim`, `release`, `complete`, `fail`, `run_start`, `run_complete`, `run_cancel`).

## 4. Test Strategy

### Approach

- **Layered coverage.** Unit + integration tests in repo are the implementation gate (`make verify`). This QA plan adds explicit manual cases and regression suites that prove behavior across surfaces and catch interaction defects that single-package tests cannot.
- **Real backends in final pass.** Final confidence requires real daemon, real subprocess extensions (TypeScript and Go), real local stdio MCP server, real OAuth-protected MCP fixture, and real Playwright/`browser-use:browser` flow. Mocks are acceptable only for unit isolation.
- **Surface parity matrix.** For each public verb (list, search, info, invoke, projection, approval), CLI, HTTP, UDS, and (where relevant) hosted MCP must agree on the same persisted state.
- **Negative-first ordering.** Each suite begins with the deny/conflicted/unauthorized/approval-required path before the happy path so silent-allow regressions surface first.

### Automation Strategy

| Lane | Driver | Owner |
|------|--------|-------|
| Go unit/integration tests | `make test` (`go test ./... -race`) | Existing test suite |
| Go E2E daemon tests | `make test-e2e-runtime` | `internal/testutil/acpmock`, daemon harness |
| Web TypeScript checks | `make bun-typecheck`, `make bun-test`, `make web-build` | `web/src/test` + MSW fixtures |
| Web E2E | `make test-e2e-web` (Playwright); critical UI flow via `browser-use:browser` | Task 16 |
| OpenAPI/codegen drift | `make codegen-check` | CI gate |
| CLI docs regeneration | `make cli-docs` | Task 14 / Task 16 verification |
| Boundaries | `make boundaries` | Existing Mage check |
| Final gate | `make verify` (fmt + lint + test + build) | Task 16 |

Manual-only cases are limited to: hosted MCP threat-model verification (real OS user / binary identity), MCP OAuth token leak proofs across diagnostics, doc copy review for descriptor-only/dotted-ID ghosts, and visual diagnostic-state spot checks.

### Mock vs Real Discipline

- **Mocks acceptable** for: unit isolation of provider I/O boundaries, ACP permission fixtures via `acpmock`, MSW-backed web tests, generated TypeScript contract tests.
- **Real required** for: final E2E suite, redaction leak proofs, MCP auth lifecycle, hosted MCP bind/peer-credential validation, browser flow.

## 5. Environment Matrix

| Layer | Required configuration |
|-------|------------------------|
| OS | macOS arm64 + Linux x86_64 (CI race lane) |
| Go | `go 1.25.5` (mcp-go v0.49.0 requires it) |
| MCP library | `github.com/mark3labs/mcp-go v0.49.0` pinned (verified by focused test) |
| Daemon | Fresh isolated `AGH_HOME` per QA pass; daemon ports/UDS unique per lab |
| Browsers | Chromium (Playwright + `browser-use:browser`); Firefox if available |
| Web viewports | 375 (mobile), 768 (tablet), 1280 (desktop) |
| Bun | Project-pinned via `package.json` lockfile |
| Extensions | TypeScript fixture + Go fixture published with manifest-authoritative `resources.tools` and `tool.provider` capability |
| MCP fixtures | Local stdio MCP server, local streamable-HTTP MCP server, local SSE MCP server, fake OAuth issuer for refresh path |

## 6. Entry Criteria

- All commits implementing tasks 01-14 are present on the QA branch.
- `make verify` passes baseline before any QA executes.
- `make codegen-check` passes (no drift between OpenAPI, generated TS, and source contracts).
- `make boundaries` passes.
- TypeScript and Go SDK fixtures exist under `sdk/typescript/test-fixtures/digest/` and `sdk/go/test-fixtures/digest/` and match `internal/extension/testdata/digest/`.
- `agh-qa-bootstrap` artifact (Task 16 input) is reachable and `AGH_HOME` is fresh.

## 7. Exit Criteria

- All P0 cases pass.
- ≥ 90% of P1 cases pass; any failing P1 has a documented workaround/fix plan and a `BUG-NNN.md` issue under `.compozy/tasks/tools-registry/qa/issues/`.
- No critical (Severity = Critical) defect remains open.
- No redaction sentinel (`AGH_TEST_TOKEN_*`, `mcp:test:bearer:*`, `BIND_NONCE_*`, `APPROVAL_TOKEN_*`) appears in any QA artifact, log, JSON payload, or browser trace.
- `make verify`, `make test-e2e-runtime`, and `make test-e2e-web` all pass on a fresh isolated lab.
- Per-package coverage ≥ 80%; race-sensitive packages run under `-race` with `CGO_ENABLED=1`.
- `make codegen-check` and `make cli-docs` show zero drift after Task 14 regeneration.
- Site `bun run typecheck` and `bun run build` pass under `packages/site`.
- Final `qa/verification-report.md` is written by Task 16 with manifest path, lab root, runtime home, base URL, provider homes, and command outputs.

## 8. Artifact Layout

Stable directories reserved for `qa-execution` (Task 16) inputs and outputs. Task 16 must consume this layout without redefining paths.

```
.compozy/tasks/tools-registry/qa/
├── test-plans/
│   ├── tool-registry-test-plan.md         # this file
│   ├── smoke-regression.md                # P0 daily lane
│   ├── targeted-regression.md             # per-change lane
│   ├── full-regression.md                 # release lane
│   ├── security-redaction-regression.md   # redaction/security lane
│   └── traceability-matrix.md             # P0/P1 → tasks/ADRs/invariants
├── test-cases/
│   ├── TC-FUNC-NNN.md                     # functional cases
│   ├── TC-INT-NNN.md                      # integration / cross-surface
│   ├── TC-SEC-NNN.md                      # security / redaction
│   ├── TC-UI-NNN.md                       # web visual / behavior
│   ├── TC-PERF-NNN.md                     # concurrency / latency
│   └── TC-REG-NNN.md                      # regression-only
├── issues/
│   └── BUG-NNN.md                         # defects discovered during QA
├── screenshots/
│   └── <flow>/<viewport>/<state>.png      # browser evidence (Task 16)
├── logs/
│   └── <component>/<timestamp>.log        # daemon, CLI, MCP, extension, web logs
├── traces/
│   └── <flow>/<run>.har|.trace            # Playwright/browser-use traces
├── fixtures/
│   └── <fixture-set>/                     # MCP servers, OAuth issuer, extensions
└── verification-report.md                 # written by Task 16
```

Task 16 must:

- Copy or reference the `agh-qa-bootstrap` manifest into `qa/bootstrap-manifest.json`.
- Append CLI/HTTP/UDS command outputs into `qa/logs/` with timestamps.
- Write Playwright traces under `qa/traces/<flow>/`.
- Write screenshots under `qa/screenshots/<flow>/<viewport>/<state>.png`.

## 9. Test Case Index Conventions

- IDs follow the qa-report scheme: `TC-FUNC-NNN`, `TC-INT-NNN`, `TC-SEC-NNN`, `TC-UI-NNN`, `TC-PERF-NNN`, `TC-REG-NNN`.
- Each case file lists Priority, Objective, Preconditions, Steps with `**Expected:**` per step, Edge Cases, Automation Target/Status/Command/Notes, and Trace (task / TechSpec section / ADR / Safety Invariant).
- Priority scale matches qa-report: P0 critical/security/blocking; P1 major flow; P2 minor/edge; P3 cosmetic.

## 10. Timeline and Deliverables

| Phase | Owner | Output |
|-------|-------|--------|
| Plan freeze | Task 15 | `tool-registry-test-plan.md` + 4 regression suites + traceability matrix + ≥ 50 test cases |
| Lab bootstrap | Task 16 / `agh-qa-bootstrap` | Fresh `AGH_HOME`, daemon, ports, manifest |
| Smoke + P0 backend | Task 16 | `smoke-regression.md` results in `verification-report.md` |
| P1 cross-surface | Task 16 | `targeted-regression.md` + `full-regression.md` results |
| Security/redaction | Task 16 | `security-redaction-regression.md` results, sentinel scan |
| Web E2E | Task 16 | Playwright + `browser-use:browser` traces |
| Docs verification | Task 16 | `bun run typecheck` + `bun run build` evidence |
| Final gate | Task 16 | `make verify` evidence in `verification-report.md` |
| Defect handling | Task 16 | `BUG-NNN.md` per defect; root-cause fix; rerun |

## 11. Roles and Responsibilities

- **Task 15 (planner):** This document, regression suites, test-case files, traceability matrix.
- **Task 16 (executor):** Lab bootstrap, execution, defect filing, root-cause fixes, final verification report.
- **Implementation owners (tasks 01-14):** Investigate and root-cause fix any defect filed against their surface.

## 12. References

- `_techspec.md` — Test Strategy, Safety Invariants, Implementation Steps, MVP Boundary
- `adrs/adr-001..011` — Decision context and constraints
- `task_01..task_14` — Implementation scope, verification evidence
- `memory/MEMORY.md` — Cross-task durable context and handoffs
- `web/CLAUDE.md`, `packages/site/CLAUDE.md` — Per-surface verification rules
- `.agents/skills/qa-report/SKILL.md`, `.agents/skills/qa-execution/SKILL.md`
