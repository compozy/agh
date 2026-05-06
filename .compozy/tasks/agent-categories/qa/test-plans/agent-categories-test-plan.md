# Agent Categories QA Test Plan

## Executive Summary

This plan validates the AGENT.md `category_path` feature end-to-end as a single canonical, display-only metadata field that flows verbatim from the AGENT.md frontmatter through `internal/config` parse / validate / edit / clone / resource paths, the `internal/api/contract` payloads, OpenAPI codegen, the CLI human/JSON/TOON outputs, and the web UI tree + grouped command pickers. Greenfield-alpha posture is enforced: there is exactly one canonical name (`category_path`), no `categories` alias, no slash-string fallback, no synthetic `Uncategorized` folder, and no `config.toml` knob.

The QA objective is behavior-first evidence that:

- `category_path` parses, normalizes, validates, edits, clones, and round-trips through the resource codec without dropping casing, segment order, or co-located fields like `Skills`.
- The same flat array is exposed by HTTP `/api/agents`, `/api/agents/:name`, `/api/workspaces/:id`, bundle activation payloads, the UDS handlers behind `agh agent list/info/workspace describe`, and the native `agh__workspace_describe` tool.
- CLI human output renders a `Category` column / detail row, TOON output adds a `category` key, and JSON output exposes `category_path` as an array.
- The web sidebar replaces the flat agent list with `AgentCategoryTree` (folders before leaves, deterministic IDs, ancestor-of-active expansion) and the three native `<select>` agent pickers (session-create, settings skills agent scope, network create-channel) are replaced by `AgentCommandSelect` / `AgentCommandMultiSelect` with grouped headings.
- A categorized agent flows from a fresh AGENT.md through the daemon all the way to the web sidebar and the session-create command picker in one observable Playwright run.

Key risks and the cases that cover them:

| Risk | Why it matters | Primary coverage |
| --- | --- | --- |
| `category_path` is silently dropped by a conversion seam (`AgentPayloadFromDef`, bundle materialization, resource codec, workspace clone) | Cross-surface drift makes operators distrust the data model | TC-FUNC-001, TC-INT-001, TC-INT-002, TC-REG-002 |
| `EditAgentDefFile` rewrites disk without preserving `category_path` on unrelated mutations | Corrupts authored intent on every skill toggle | TC-FUNC-002 |
| Validation accepts an unsafe segment (`""`, `.`, `..`, `/`, `\`) or revives an alias (`categories:`, `"Marketing/Sales"`) | Greenfield invariant broken; future migration code becomes inevitable | TC-FUNC-003, TC-REG-001 |
| OpenAPI / generated TS drifts from the contract | Web compiles against stale types and loses the field | TC-INT-003 |
| CLI human/TOON/JSON disagree on category presence or formatting | Agents and operators see different truths from the same source | TC-FUNC-004 |
| Sidebar tree breaks active-agent indication, keyboard navigation, or test IDs | Existing automation and operator muscle memory regress | TC-UI-001 |
| Command pickers regress to flat lists or drop existing `data-testid`s | Session-create, settings, and network dialogs lose grouped semantics | TC-UI-002 |
| Live, end-to-end browser scenario fails for a categorized agent | The whole feature is unusable from the user perspective even when units pass | TC-SCEN-001 |
| Casing or order is mutated anywhere in the pipeline | Authors lose intent (`Marketing` vs `marketing`, parent-before-child is meaningful) | TC-REG-003 |

## Scope Definition

In scope:

- `internal/config` parse, normalization, validation, `EditAgentDefFile` round-trip, `CloneAgentDef`, `validateAgentResourceSpec`.
- `internal/workspace.cloneAgentDefs` delegation to `aghconfig.CloneAgentDef` and the `Skills` regression that fix exposes.
- `internal/api/contract.AgentPayload` and `BundleAgentPayload` `category_path` shape (`omitempty`, defensive copy, diagnostic exclusion).
- `internal/api/core/conversions.AgentPayloadFromDef` and `AgentPayloadFromDiagnostic`.
- `internal/cli` agent and workspace commands across `human`, `toon`, and `json` formats.
- `internal/testutil/e2e/config_seed.go` `AgentSeed.CategoryPath` plumbing for runtime E2E fixtures.
- `internal/extension/manager.go` clone-of-clone removal (the round-3 review fix).
- OpenAPI / generated TS surface (`openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`).
- `packages/ui/src/components/reui/tree.tsx` re-exports plus the new `tree.test.tsx` proving optional-feature guards.
- Web sidebar `AgentCategoryTree`, `AgentCommandSelect`, `AgentCommandMultiSelect`, `AgentCommandList`, agent-category lib, view-model hook, and stories.
- The three command-picker call sites: session-create dialog, settings skills agent scope, network create-channel dialog.
- Playwright `web/e2e/agent-categories.spec.ts` end-to-end coverage.
- Documentation under `packages/site/content/runtime/core/...` (definitions and configuration AGENT.md pages).

Out of scope:

- Behavior branching on `category_path` in ACP, scheduling, autonomy, permissions, or workspace partitioning (the feature is display-only by contract).
- Backend tree/group endpoints, denormalized category tables, schema migrations.
- A `config.toml` toggle for category UI; tree-expansion persistence is intentionally local UI state.
- Compatibility shims, aliases, slash-string fallbacks, or `Uncategorized` synthetic folders.
- Marketing site / `packages/site` landing copy.
- Performance characterization beyond what `make verify` covers.

## Behavioral Scenario Charter

Startup situation:

- A fresh QA lab seeded via `agh-qa-bootstrap` with at least one categorized AGENT.md (multi-segment), at least one root-level AGENT.md (no `category_path`), and at least one bundle activation that ships a categorized agent.
- Daemon, HTTP, UDS, and Web surfaces all point at the same isolated `AGH_HOME` and unique daemon port; web dev server reads `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest.
- Provider home / env policy follows the manifest contract: bound-secret providers use `PROVIDER_HOME`/`PROVIDER_CODEX_HOME`; `native_cli` providers preserve operator `HOME`.

Operator intent:

- Author a categorized agent in AGENT.md, observe it in CLI human/JSON/TOON outputs, in the daemon REST/UDS payloads, in the native `agh__workspace_describe` tool, and in the Web sidebar tree and session-create command picker, then route from a sidebar leaf to `/agents/:name` and start a session — without ever seeing a `categories` alias, slash-string, or `Uncategorized` bucket.

Expected business outcome:

- The operator can group agents into a hierarchy via metadata only, run agents normally regardless of category, and see consistent grouping/folder/leaf state across CLI, HTTP, UDS, native tools, and the web UI.
- Agents can introspect `category_path` purely through agent-manageable surfaces (`agh agent list -o json`, `agh agent info -o json`, native `workspace_describe`).

Agent roles:

| Actor / Agent | Role | Expected behavior | Evidence source |
| --- | --- | --- | --- |
| Operator | Scenario driver | Edits AGENT.md, runs CLI, opens Web, starts a session through the categorized agent. | CLI transcript, browser screenshot, API/UDS responses. |
| Categorized agent | Provider-backed work peer | Runs a session normally; behavior is unchanged by category. | Session events, transcript or blocked-provider boundary. |
| Reviewer / observer agent | Cross-surface verifier | Calls `agh__workspace_describe` and `agh agent info -o json` to confirm `category_path` is present and identical to disk. | Native tool transcript, JSON evidence files. |
| QA harness | Disruption prober | Toggles `Skills.Disabled`, restarts daemon, retries with invalid segments, regenerates codegen. | `make verify`, `make codegen-check`, `make test-e2e-runtime`, `make test-e2e-web`. |

Live provider / LLM expectations:

- Release-grade execution should run a provider-backed AGH session with a categorized agent when credentials and local prerequisites are reachable.
- If live provider execution is blocked, QA execution must record the exact provider, credential, binary, or account boundary and still validate every local runtime / CLI / API / UDS / Web / E2E harness surface.
- Mock / `acpmock` evidence remains readiness or regression evidence only; it is not counted as live provider proof.

Expected artifacts:

- `.compozy/tasks/agent-categories/qa/verification-report.md`
- CLI / API / Web / UDS / native-tool transcripts under `.compozy/tasks/agent-categories/qa/`
- Browser screenshots showing the sidebar tree and the session-create command picker grouping under `.compozy/tasks/agent-categories/qa/screenshots/`
- A scenario contract file at `.compozy/tasks/agent-categories/qa/scenario-contract.json` listing the minimum agents, channels, surfaces, artifacts-used-later, and disruption probes.
- A behavioral charter at `.compozy/tasks/agent-categories/qa/behavioral-scenario-charter.yaml` consumed by `qa-execution`.
- Bug reports under `.compozy/tasks/agent-categories/qa/issues/BUG-*.md` whenever a journey fails.

Disruption probes:

- Toggle `Skills.Disabled` via `EditAgentDefFile` and confirm `category_path` survives on disk.
- Restart the daemon and confirm the categorized agent is still tree-grouped and bundle-activatable.
- Submit a malformed `category_path` (`""`, `"."`, `".."`, `"a/b"`, `"a\\b"`, scalar `"Marketing"`, `categories:` alias) and confirm the daemon emits an `agent_diagnostic` with `category_path: nil` rather than a permissive fallback.
- Resolve the same agent's `category_path` from CLI JSON, native tool, and Web simultaneously; the values must agree byte-for-byte.

## Test Strategy and Approach

Smoke readiness checks (entry criteria only):

- `make verify` is green on the branch HEAD.
- `make codegen-check` confirms `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` are in sync.
- The bootstrap manifest exposes a daemon port and `AGH_WEB_API_PROXY_TARGET`.
- Smoke checks must not be reported as release-grade proof.

Release-grade behavioral evidence:

- Execute P0 functional + integration cases first to lock the contract: `internal/config` parse / edit / clone / resource (TC-FUNC-001..004), workspace clone regression (TC-REG-002), conversion seam parity (TC-INT-001..002), codegen drift gate (TC-INT-003), CLI surface coverage (TC-FUNC-004).
- Execute P0 UI cases: sidebar tree behavior (TC-UI-001), command picker grouping across session/settings/network (TC-UI-002).
- Execute the P0 real-scenario case (TC-SCEN-001): a categorized agent appears in the sidebar tree, groups inside the session-create picker, routes to `/agents/:name`, and starts a session whose state agrees across CLI, API/UDS, and Web.
- Execute P1 regression cases: alias / slash-string rejection (TC-REG-001) and casing / order preservation (TC-REG-003).
- Every P0 case names CLI, API/UDS, and Web evidence side-by-side. Mock/`acpmock` runs are captured separately as harness evidence.
- Every P0 journey runs at least one realistic disruption probe.

Regression evidence:

- Re-run `make verify` after the last code or fixture change.
- Re-run `make test-e2e-runtime` and `make test-e2e-web` as targeted behavior harnesses.
- Re-run TC-SCEN-001 after the full gate passes.

## Environment Requirements

| Requirement | Expected value |
| --- | --- |
| OS | macOS developer workstation or CI-equivalent Linux runner with AGH prerequisites. |
| Runtime | Go toolchain, Bun workspace dependencies, SQLite with race-test support. |
| Browser | Browser plugin / browser-use for local web validation; approved fallback is `agent-browser`. |
| Daemon isolation | Fresh QA lab by default; unique `AGH_HOME`, daemon ports, and tmux bridge socket paths when concurrency is signaled. |
| Provider homes | `PROVIDER_HOME` / `PROVIDER_CODEX_HOME` for bound-secret providers; preserve operator home for `native_cli`. |
| Web proxy | Export `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest before `make web-dev`. |
| Output root | `.compozy/tasks/agent-categories/qa/` |

## Entry Criteria

- TechSpec is approved and the implementation has shipped (Opus round-4 verdict SHIP, 0 blockers / 0 risks / 0 nits).
- Local `make verify` is green on `agent-categories` branch HEAD.
- A QA bootstrap manifest exists or `qa-execution` records why bootstrap could not be created.
- The QA execution task has access to this plan, the test cases, the scenario contract, and the behavioral charter.

## Exit Criteria

- All P0 cases pass or produce bug reports with exact reproduction and evidence.
- 90%+ of P1 cases pass with no critical or high bug left unresolved.
- CLI, HTTP, UDS, native-tool, and Web UI agree on `category_path` for the same agent.
- Live provider-backed behavior is exercised with at least one categorized agent OR the exact blocked provider/tool/credential boundary is documented.
- `make verify`, `make test-e2e-runtime`, and `make test-e2e-web` all pass after the last fix.
- `.compozy/tasks/agent-categories/qa/verification-report.md` includes a QA bootstrap block when a healthy reusable lab remains.
- The strict QA auditor (per `qa-execution` checklist) reports 0 blockers across C4, C5, C8, C9, C10, C11, C14.

## Execution Matrix

| ID | Priority | Class | Primary surfaces | Must run before |
| --- | --- | --- | --- | --- |
| SMOKE-001 | P0 | Smoke readiness | `make verify`, `make codegen-check`, manifest sanity | Any P0 case |
| TC-FUNC-001 | P0 | Functional / Go | `internal/config` parse + validate | TC-FUNC-002 |
| TC-FUNC-002 | P0 | Functional / Go | `internal/config.EditAgentDefFile` round-trip | TC-INT-001 |
| TC-FUNC-003 | P0 | Functional / Go | Validation negative cases | TC-INT-001 |
| TC-FUNC-004 | P0 | Functional / CLI | `agh agent list/info`, `agh workspace info` (human/toon/json) | TC-SCEN-001 |
| TC-INT-001 | P0 | Integration / Contract | `AgentPayloadFromDef`, bundle activation, native tools, UDS/HTTP parity | TC-INT-002 |
| TC-INT-002 | P0 | Integration / Resource codec | `validateAgentResourceSpec`, daemon resource sync | TC-SCEN-001 |
| TC-INT-003 | P0 | Integration / Codegen | OpenAPI + generated TS drift gate | TC-UI-001 |
| TC-UI-001 | P0 | UI / Web | Sidebar `AgentCategoryTree` behavior, ancestor expansion, test IDs | TC-SCEN-001 |
| TC-UI-002 | P0 | UI / Web | `AgentCommandSelect`/`MultiSelect` in session, settings, network | TC-SCEN-001 |
| TC-SCEN-001 | P0 | Real Scenario / Playwright | Sidebar + session-create grouping + routing + live session | Final verification |
| TC-REG-001 | P1 | Regression / Validation | Alias and slash-string rejection (parser strict) | Final verification |
| TC-REG-002 | P1 | Regression / Workspace | `cloneAgentDefs` preserves Skills + CategoryPath | Final verification |
| TC-REG-003 | P1 | Regression / Casing-order | Author intent preserved across all surfaces | Final verification |

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Live provider credentials unavailable for the categorized session | Medium | High | Record exact boundary; continue all local surfaces; do not claim live provider proof. |
| Browser dev server points at the default daemon port | Medium | High | Derive `AGH_WEB_API_PROXY_TARGET` from bootstrap manifest. |
| QA lab reuses stale state | Medium | Medium | Fresh lab by default; reuse only the same-session healthy manifest. |
| Hidden conversion seam silently nils `category_path` | Medium | High | TC-INT-001/002 cover each named seam (HTTP, UDS, bundle, native tool, resource codec, diagnostic). |
| Strict-yaml decode regresses to permissive on `categories:` alias or scalar string | Low | High | TC-REG-001 locks the contract with explicit negative tests. |
| OpenAPI codegen drift slips into PR | Medium | High | TC-INT-003 reruns `make codegen` and `make codegen-check` and compares hashes. |
| Existing `data-testid`s removed when replacing native `<select>` | Medium | High | TC-UI-002 asserts `session-create-agent-select`, `settings-agent-select`, `network-agent-option-${name}` survive. |
| Tree expansion default surprises operators | Low | Low | TC-UI-001 covers ancestor-of-active expansion AND no-active default. |

## Timeline and Deliverables

| Phase | Deliverable | Output |
| --- | --- | --- |
| Planning | Feature test plan | `qa/test-plans/agent-categories-test-plan.md` |
| Planning | Scenario contract minimums (machine-readable) | `qa/scenario-contract.json` |
| Planning | Behavioral charter (machine-readable) | `qa/behavioral-scenario-charter.yaml` |
| Planning | Execution-ready cases | `qa/test-cases/SMOKE-001.md`, `qa/test-cases/TC-*.md` |
| Execution | Baseline verification, behavioral evidence, bug reports | `qa/verification-report.md`, `qa/issues/`, `qa/screenshots/` |

## Scenario Contract

The minimums below are the machine-readable contract `qa-execution` will check against `qa/scenario-contract.json`. They MUST be satisfied by P0/P1 cases collectively before this feature can ship.

- Minimum agents: at least one categorized agent (multi-segment `category_path`), at least one root-level agent (no `category_path`), at least one categorized agent shipped via a bundle activation.
- Minimum surfaces: CLI (`human`, `toon`, `json`), HTTP `/api/agents`, HTTP `/api/agents/:name`, HTTP `/api/workspaces/:id`, UDS equivalents, native tool `agh__workspace_describe`, Web sidebar tree, Web session-create command picker, Web settings skills agent scope picker, Web network create-channel multi-select.
- Minimum task tree: at least one task that creates a session for a categorized agent and observes the session in CLI + Web simultaneously.
- Minimum provider-backed sessions: at least one provider-backed session for a categorized agent, OR an explicit blocked-provider boundary file at `qa/provider-attempt.json`.
- Minimum cross-surface objects: same `category_path` on the same `agent.name` across CLI JSON, HTTP JSON, UDS JSON, and the rendered Web tree.
- Minimum artifacts-used-later: AGENT.md edited by `EditAgentDefFile` (e.g., toggling `Skills.Disabled`) is later read by the daemon parse + Web payload, proving disk → daemon → web reuse.
- Minimum disruption probes: invalid-segment rejection at parse, daemon restart preserves grouping, agent without `category_path` lands as a root-level leaf (no `Uncategorized` synthetic folder).
- Minimum required surfaces: `make verify`, `make test-e2e-runtime`, `make test-e2e-web`.

## Auditor Mapping

| TC ID | C4 actors / roles | C5 channels / surfaces | C6 task tree | C8 cross-surface truth | C9 live provider | C10 artifact reuse | C11 disruption | C14 final verification |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| TC-FUNC-001 | Operator + parser | AGENT.md → `AgentDef` | Parse-only | n/a | n/a | n/a | Trim segments | `go test ./internal/config -run CategoryPath` |
| TC-FUNC-002 | Operator + writer | AGENT.md disk round-trip | Edit-only | Disk vs `parsedAgentDef` | n/a | Edited AGENT.md reused by next parse | Skills toggle preserves category | `go test ./internal/config -run TestEditAgentDef` |
| TC-FUNC-003 | Parser | Validation pipeline | n/a | n/a | n/a | n/a | Negative segments rejected | `go test ./internal/config -run validateAgentCategoryPath` |
| TC-FUNC-004 | Operator | CLI human/toon/json | Read-only | CLI vs disk | n/a | JSON output reused by agents | Empty category renders dash | `go test ./internal/cli -run AgentList\|AgentInfo\|Workspace` |
| TC-INT-001 | Daemon + agents | HTTP + UDS + native tool | Read-only | All four agree | Optional | Bundle payload reused for activation | Diagnostic agent has nil `category_path` | `go test ./internal/api/...` |
| TC-INT-002 | Daemon + extensions | Resource codec + sync | n/a | Resource vs `AgentDef` | n/a | Codec output reused on next read | Invalid segments rejected via `errors.Is(...,resources.ErrValidation)` | `go test ./internal/config -run AgentResource` |
| TC-INT-003 | Build | OpenAPI + TS codegen | n/a | `agh.json` vs `agh-openapi.d.ts` | n/a | Generated TS reused by web | Drift fails the gate | `make codegen && make codegen-check` |
| TC-UI-001 | Operator | Web sidebar | Active route | DOM vs payload | n/a | Tree renders Web payload | Active-ancestor expansion | `make web-test -- agent-category` |
| TC-UI-002 | Operator | Session / Settings / Network dialogs | Read-only | Picker vs payload | n/a | Same payload, three pickers | Empty search renders empty state | `make web-test -- agent-command-select\|agent-command-multi-select` |
| TC-SCEN-001 | Operator + agent | Sidebar + session-create + agent route + session | Session start | All surfaces agree on category | Required (or boundary file) | Sidebar agent → session-create → /agents/:name → live session | Daemon restart preserves grouping | `make test-e2e-web -- agent-categories` |
| TC-REG-001 | Parser | Strict YAML decode | n/a | n/a | n/a | n/a | Alias + slash-string rejected | `go test ./internal/config` |
| TC-REG-002 | Workspace clone | `cloneAgentDefs` | n/a | Clone vs source | n/a | Cloned agent reused by daemon | Skills + CategoryPath preserved | `go test ./internal/workspace` |
| TC-REG-003 | All surfaces | CLI + HTTP + UDS + Web | Read-only | Casing/order parity | n/a | Authored intent preserved | Mixed casing not normalized | Manual cross-surface diff |

## Notes

- This plan governs only the `agent-categories` feature shipped on the `agent-categories` branch. It does not authorize changes to runtime ACP, scheduling, autonomy, permissions, or workspace partitioning behavior — those would require a separate TechSpec and QA pass.
- All non-Go test execution should run inside an isolated `AGH_HOME` per the AGH worktree-isolation rule whenever concurrency is signaled.
