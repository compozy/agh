# Session Provider Override QA Plan

**Feature:** Per-session ACP provider override
**QA output path:** `.compozy/tasks/session-driver-override/qa/`
**Planned for execution by:** `task_08.md` via `/qa-execution` with `qa-output-path=.compozy/tasks/session-driver-override`
**Created:** 2026-04-21
**Last Updated:** 2026-04-21

## Executive Summary

This plan defines the execution-ready QA matrix for session provider override across config resolution, session lifecycle, persistence, migration and repair, explicit transport surfaces, workspace provider discovery, and the web dialog and resume-failure UX.

The core objective is to prove deterministic runtime identity. A session that starts with an explicit provider must persist and resume with that exact provider, invalid selections must fail before side effects, legacy blank-provider state must converge once, and every explicit surface must expose the same provider semantics. The follow-up execution task must use this artifact set without redefining scope, output paths, priorities, or evidence standards.

## Scope

### In Scope

- Task 01 behavior: session-aware provider resolution and coherent provider-owned runtime field replacement.
- Task 02 behavior: provider persistence in runtime and on-disk `session.json`, plus validation-before-persistence ordering.
- Task 03 behavior: global SQLite `sessions.provider` migration, global index round-trip, and one-time legacy blank-provider repair.
- Task 04 behavior: HTTP, UDS, CLI, and Host API create/read parity for `provider`.
- Task 05 behavior: workspace provider catalog discovery through `WorkspaceDetailPayload.providers` and automatic internal creators explicitly using empty provider to stay on agent defaults.
- Task 06 behavior: dialog-driven web creation flow, provider picker, provider badge visibility, and dedicated inline resume-failure state.
- Evidence collection requirements for backend logs/errors, session metadata, SQLite state, API payloads, CLI output, and browser-visible UI.

### Out of Scope

- Live execution of the feature or repository gates in this task.
- New product requirements or scope beyond tasks 01-06 and the accepted ADRs.
- Per-event provider duplication in session or agent event payloads.
- New provider-list endpoints or alternate frontend discovery logic.
- Broad exploratory UX polish unrelated to provider override behavior.

## Test Strategy and Approach

### Coverage Model

The execution pass in task 08 must validate the feature through the same seams that operators and downstream integrations use:

1. Config resolution seam: prove the provider override changes only runtime/provider-owned state, not agent identity.
2. Lifecycle seam: prove create and resume validate before persistence or startup side effects.
3. Persistence seam: prove `provider` survives `session.json`, global index storage, stop/resume, and query surfaces.
4. Migration and repair seam: prove old DB and blank-provider metadata converge safely and explicitly.
5. Contract seam: prove HTTP, UDS, CLI, and Host API stay aligned on request/response semantics.
6. Workspace seam: prove provider options come from the resolved workspace payload and are suitable for UI rendering.
7. Web seam: prove every create entrypoint opens the dialog, submits the chosen provider, and exposes resume failures inline.

### Evidence Standard

Every executed P0/P1 case must name the evidence that counts as proof. Evidence must be captured under `.compozy/tasks/session-driver-override/qa/` during task 08 and should include the relevant subset of:

- Backend logs with `session_id`, `agent_name`, `provider`, `workspace_id`, and `phase` where applicable.
- On-disk session metadata showing `provider` in `session.json`.
- SQLite inspection of the global `sessions.provider` column and affected rows.
- HTTP/UDS request and response payloads.
- CLI command output showing the effective provider.
- Host API request/response payloads where applicable.
- Browser screenshots, network captures, and visible UI state for dialog and resume-failure flows.

### Execution Principles

- Start with smoke. If smoke fails, stop and fix the blocking issue before broader execution.
- Treat P0 cases as release-blocking for task 08.
- Use the same `qa-output-path` for planning and execution artifacts.
- Prefer fixtures that make provider differences obvious by varying provider-owned `command`, `default_model`, or provider MCP configuration.
- Do not accept fallback behavior when the persisted or requested provider is invalid or unavailable.
- Keep the automatic internal-creator empty-provider contract in the full-lane rerun set whenever provider plumbing changes, even though the manual cases focus on operator-visible explicit surfaces.

## Environment Requirements

| Area | Requirement | Notes |
| --- | --- | --- |
| Repository baseline | Clean checkout of `/Users/pedronauck/dev/compozy/agh` | Task 08 must start from a clean, known branch state. |
| Verification gates | `make verify` and `make codegen-check` available | Required by project policy and task 08. |
| Backend runtime | Local daemon + SQLite storage | Needed for create/resume, global DB inspection, and logs. |
| Workspace fixture A | Workspace with one agent and at least two visible providers | Used for default vs override comparisons. |
| Workspace fixture B | Same agent after default provider change | Used to prove persisted provider wins on resume. |
| Workspace fixture C | Same persisted session after override provider removal | Used for explicit resume-failure behavior. |
| Legacy session fixture | Stopped session metadata with blank `provider` | Used for one-time repair coverage. |
| Legacy global DB fixture | `sessions` table without `provider` column | Used for migration coverage. |
| Transport surfaces | HTTP, UDS, CLI, and Host API access | Used for explicit-surface parity. |
| Browser | Chromium-class browser or `agent-browser` execution | Used for dialog and resume-failure evidence. |
| Viewports | Desktop `1280x800`, tablet `768x1024`, mobile `375x812` | Required for dialog/resume-failure UI validation. |

## Test Data and Fixture Matrix

| Fixture ID | Purpose | Required Characteristics | Primary Cases |
| --- | --- | --- | --- |
| `WS-PROVIDER-MATRIX` | Default vs override behavior | One agent with default provider `A` and alternate provider `B`; provider-owned command/model differ visibly | `TC-FUNC-001`, `TC-FUNC-002`, `TC-UI-010` |
| `WS-DEFAULT-DRIFT` | Persisted resume after agent default change | Session created with provider `B`; workspace later changes agent default to `A` or `C` | `TC-INT-004` |
| `WS-PROVIDER-REMOVED` | Resume failure after provider removal | Persisted provider is removed from workspace-visible config | `TC-INT-005`, `TC-UI-011` |
| `LEGACY-META-BLANK` | One-time repair | Stopped session metadata lacks `provider`; stored agent still resolves | `TC-INT-007` |
| `LEGACY-DB-NO-PROVIDER` | SQLite migration | Existing `sessions` table without `provider` column | `TC-INT-006` |
| `TRANSPORT-PARITY` | Explicit surface parity | One canonical session scenario accessible through HTTP, UDS, CLI, and Host API | `TC-INT-008` |
| `WORKSPACE-CATALOG` | Provider discovery | Workspace detail returns sorted visible provider options | `TC-INT-009`, `TC-UI-010` |

## Entry Criteria

- Tasks 01-06 are marked complete and their code is present in the branch under test.
- `.compozy/tasks/session-driver-override/qa/` exists with this plan, regression suite, and manual test cases.
- Task 08 activates `/qa-execution` with `qa-output-path=.compozy/tasks/session-driver-override`.
- Required fixtures for default, override, removed-provider, migration, and legacy-repair scenarios are prepared or documented before execution starts.
- Generated contracts are current enough to begin execution; task 08 will re-check with `make codegen-check`.
- Backend log capture, SQLite inspection, API payload capture, CLI output capture, and browser screenshot capture are available.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass, and any P1 failure has a documented issue file plus fix plan or follow-up.
- No open Critical or High-severity issues remain for provider override behavior.
- Task 08 publishes `.compozy/tasks/session-driver-override/qa/verification-report.md`.
- Any execution-discovered bug is captured under `.compozy/tasks/session-driver-override/qa/issues/BUG-*.md` and linked to its originating test case.
- Final execution reruns repository gates after the last fix, including `make verify` and `make codegen-check`.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Provider override reuses stale command/model/MCP state from the original provider | Medium | Critical | Use a fixture with visibly different provider-owned runtime markers and require backend/runtime evidence in `TC-FUNC-002`. |
| Invalid provider creates partial session state before failing | Medium | Critical | Require absence checks for `session.json`, global DB row, and list/status visibility in `TC-FUNC-003`. |
| Persisted resume silently drifts to a new agent default | Medium | Critical | Force a post-create default change and verify persisted provider wins in `TC-INT-004`. |
| Removed persisted provider falls back silently instead of failing | High | Critical | Require explicit error payload/log text naming session id and missing provider in `TC-INT-005` and `TC-UI-011`. |
| Migration leaves `sessions.provider` missing or blank after reconcile | Medium | High | Inspect schema and rows directly in `TC-INT-006` and `TC-INT-007`. |
| HTTP, UDS, CLI, and Host API drift on `provider` shape or output | Medium | High | Run explicit parity checks in `TC-INT-008` and keep task 08 evidence side-by-side. |
| Workspace provider picker drifts from backend-visible providers | Medium | High | Treat `WorkspaceDetailPayload.providers` as the source of truth in `TC-INT-009` and `TC-UI-010`. |
| Web create flow still bypasses the dialog from some entrypoint | Medium | High | Require route- and sidebar-based dialog evidence in `TC-UI-010`. |
| Resume failure becomes a toast-only transient error | Medium | High | Require a screenshot and DOM-visible inline failure panel in `TC-UI-011`. |

## Manual Test Case Inventory

| Case ID | Priority | Area | What It Proves |
| --- | --- | --- | --- |
| `TC-FUNC-001` | P1 | Resolution baseline | No-override create path uses the resolved agent default provider and persists it coherently. |
| `TC-FUNC-002` | P0 | Resolution override | Explicit provider override re-resolves provider-owned runtime fields without changing agent identity. |
| `TC-FUNC-003` | P0 | Validation ordering | Invalid provider create fails before metadata/global DB writes or driver start. |
| `TC-INT-004` | P0 | Persistence and resume | Persisted provider wins on resume even after the agent default changes. |
| `TC-INT-005` | P0 | Explicit failure | Removed persisted provider fails explicitly and does not fall back. |
| `TC-INT-006` | P1 | Migration | Existing SQLite session indexes gain `provider` safely and preserve provider data on rebuild paths. |
| `TC-INT-007` | P0 | Legacy repair | Blank-provider legacy metadata is repaired once, persisted immediately, and then behaves like a normal session. |
| `TC-INT-008` | P0 | Surface parity | HTTP, UDS, CLI, and Host API agree on create/read provider semantics. |
| `TC-INT-009` | P1 | Workspace catalog | Workspace detail exposes sorted provider options suitable for the web picker. |
| `TC-UI-010` | P0 | Dialog flow | Every create entrypoint opens the dialog, preselects the default provider, and submits the chosen provider. |
| `TC-UI-011` | P0 | Resume failure UX | The SPA renders an actionable inline resume-failure state with session id and missing provider. |

## Traceability Matrix

| Case ID | Priority | Source of Truth |
| --- | --- | --- |
| `TC-FUNC-001` | P1 | Task 01 requirements on `ResolveSessionAgent` default path; TechSpec "Resolution helper semantics". |
| `TC-FUNC-002` | P0 | Task 01 requirements on provider-owned `command`, `default_model`, and provider MCP replacement; ADR-001; TechSpec "Core Interfaces" and "Testing Approach". |
| `TC-FUNC-003` | P0 | Task 02 requirements on validation before `writeMeta`, global registration, and `driver.Start`; TechSpec create flow step 4. |
| `TC-INT-004` | P0 | Task 02 success criteria; ADR-003; TechSpec resume flow steps 3-4. |
| `TC-INT-005` | P0 | Task 02 and Task 06 explicit-failure requirements; ADR-003; TechSpec resume flow step 5 and "Monitoring and Observability". |
| `TC-INT-006` | P1 | Task 03 migration requirements; ADR-005; TechSpec "Global DB schema". |
| `TC-INT-007` | P0 | Task 03 repair requirements; ADR-003 and ADR-005; TechSpec "Legacy metadata repair". |
| `TC-INT-008` | P0 | Task 04 contract requirements; ADR-004; TechSpec "API Endpoints" and CLI/Host API notes. |
| `TC-INT-009` | P1 | Task 05 requirements on `WorkspaceDetailPayload.providers`; ADR-004; TechSpec `GET /api/workspaces/{id}`. |
| `TC-UI-010` | P0 | Task 06 dialog requirements; ADR-004; TechSpec "web surfaces" impact analysis and UI integration tests. |
| `TC-UI-011` | P0 | Task 06 resume-failure requirements; ADR-003 and ADR-004; TechSpec "Known Risks" and UI integration tests. |

## Timeline and Deliverables

### Task 07 Deliverables

- `qa/test-plans/session-provider-override-test-plan.md`
- `qa/test-plans/session-provider-override-regression.md`
- `qa/test-cases/TC-*.md`
- Stable `qa/issues/` and `qa/screenshots/` directories for task 08 evidence

### Task 08 Responsibilities

- Activate `/qa-execution` with the same `qa-output-path`
- Derive the execution matrix from this plan and the regression suite
- Capture fresh evidence in `qa/verification-report.md`
- Create issue files and screenshots as needed
- Fix root-cause regressions and rerun the full repo gates

## Handoff Notes for Task 08

- Do not change the artifact root. All evidence, bugs, screenshots, and the verification report must remain under `.compozy/tasks/session-driver-override/qa/`.
- Treat smoke as the initial stop/go gate. If any smoke case fails, stop and fix before expanding scope.
- When a backend or transport failure also has a web-visible manifestation, capture both layers and cross-link them in the verification report.
- When the removed-provider scenario is executed, reuse the same persisted session across backend, CLI, and web surfaces so the evidence set stays comparable.
- When a bug is found, link the bug report to the originating `TC-*` case and rerun the affected lane plus the full repo verification contract.
- During the full lane, keep the existing repo regression coverage for task 05 automatic creator defaults in scope so empty-provider behavior does not drift silently.
