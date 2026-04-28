# Settings UI Test Plan

**Feature:** Settings UI
**QA output path:** `.compozy/tasks/settings-ui/qa/`
**Planning date:** 2026-04-17
**Planned execution task:** `task_16.md`
**Primary sources:** `task_15.md`, `_techspec.md`, ADR-001..004, `task_10.md`..`task_14.md`
**Design references:** `docs/design/paper/settings/*.png` (10 local Paper exports, each `2880x1800`)
**Figma status:** Not configured for this run; visual planning uses the local Paper exports

## Executive Summary

The settings feature is one operator-facing surface composed of a shared shell plus 10 route-level screens:

- `general`
- `providers`
- `mcp-servers`
- `environments`
- `memory`
- `skills`
- `automation`
- `network`
- `observability`
- `hooks-extensions`

This plan defines the reusable QA artifacts that `task_16` must execute without changing scope, priorities, or artifact paths. The primary objective is to prove that route navigation, restart-aware saves, applied-now behavior, collection CRUD semantics, workspace-scoped MCP behavior, and the Hooks & Extensions hybrid interaction model work as designed.

### Test Objectives

- Validate every settings route under the shared `/settings` shell with explicit traceability back to the TechSpec and implementation tasks.
- Prove operator-visible mutation semantics: `applied_now`, `restart_required`, and `action_trigger`.
- Validate high-risk persistence and transport behaviors: restart polling, collection fallback, workspace scope, write-target messaging, and loopback-only HTTP mutation restrictions.
- Seed `task_16` with a stable regression model that can drive manual execution, daemon-served browser E2E coverage, and defect documentation.

### Key Risks

- Restart status can mislead operators if the persisted operation record and banner state diverge.
- Workspace-scoped MCP flows can silently mutate the wrong target or hide effective-source changes.
- Mixed mutation semantics can confuse users if immediate actions and restart-required saves share the same status language.
- Collection delete behavior can obscure builtin fallback or shadowed lower-precedence sources.
- HTTP mutation restrictions on non-loopback binds can look like broken UI unless the product messaging is explicit.
- Visual drift can accumulate because the feature spans 10 Paper artboards plus multiple dialog and banner states.

## Scope

### In Scope

- Shared `/settings` shell and index placeholder behavior.
- Route-by-route verification for:
  - `/settings/general`
  - `/settings/providers`
  - `/settings/mcp-servers`
  - `/sandbox`
  - `/settings/memory`
  - `/settings/skills`
  - `/settings/automation`
  - `/settings/network`
  - `/settings/observability`
  - `/settings/hooks-extensions`
- Restart-required messaging, restart trigger flow, restart-status polling, and reconnect/refresh continuity.
- Applied-now behavior for disabled skills and immediate-action behavior for extension enable/disable flows.
- Collection CRUD semantics for providers, environments, and MCP servers.
- Workspace-scoped MCP behavior, `target=auto|config|sidecar`, precedence metadata, and fallback visibility.
- HTTP loopback-only mutation restrictions and operator-facing messaging when HTTP is bound non-loopback.
- Visual/manual validation against the local Paper exports at desktop, tablet, and mobile viewports.
- Artifact and evidence layout under `.compozy/tasks/settings-ui/qa/`.

### Out of Scope

- Remote authenticated settings administration beyond ADR-004.
- Performance/load characterization beyond identifying blocking UX regressions during execution.
- Deep operational workflows already owned by `/skills`, `/automation`, and `/network`, except for linked navigation and return-path behavior.
- New feature work, contract redesign, or implementation fixes not required by defects found during `task_16`.
- Figma-specific pixel inspection workflows, because Figma MCP is not configured for this task.

## Test Strategy and Approach

### Coverage Model

| Layer | Goal | Primary Artifacts | Execution Owner |
|------|------|-------------------|-----------------|
| Functional route coverage | Prove each route loads, mutates, and communicates status correctly | `TC-FUNC-*` | `task_16` |
| Integration / transport coverage | Prove restart polling, scope/precedence, and HTTP restriction behavior | `TC-INT-*` | `task_16` |
| UI / visual coverage | Compare the shipped routes to the Paper exports and breakpoint behavior | `TC-UI-*` | `task_16` |
| Regression gating | Define smoke, targeted, full, and post-fix sanity execution order | `settings-ui-regression.md` | `task_16` |

### Approach

1. Run the smoke lane first to prove the shell, one restart-required flow, one applied-now flow, one collection CRUD flow, one workspace-scoped MCP flow, and the Hooks & Extensions hybrid flow.
2. If smoke passes, continue with the targeted lane that matches the change set or discovered defect cluster.
3. Run the full lane before completion of `task_16`, including visual/manual validation and transport restriction checks.
4. Capture screenshots in `.compozy/tasks/settings-ui/qa/screenshots/` and defects in `.compozy/tasks/settings-ui/qa/issues/`.
5. Publish execution evidence in `.compozy/tasks/settings-ui/qa/verification-report.md`.

### Required Semantic Checks

- `restart_required`
  - Save returns restart-required messaging.
  - Restart banner becomes visible when appropriate.
  - Restart trigger starts polling and survives refresh/reconnect.
- `applied_now`
  - Success state is visible without showing a restart banner.
  - Refetch/invalidation updates visible data on the current route.
- `action_trigger`
  - Action progress/result is shown separately from save-state messaging.
  - Manual actions do not masquerade as config saves.

## Environment Requirements

| Axis | Baseline | Variants | Why it matters |
|------|----------|----------|----------------|
| OS | macOS local dev machine | Linux if available | Covers the primary local workflow plus a second host class for daemon/runtime behavior |
| Browsers | Chromium desktop | Firefox, Safari; responsive checks at `1280`, `768`, `375` | Matches the planned Playwright lane plus manual responsive checks |
| Daemon runtime | Detached daemon started from the local repo build | Fresh start before restart tests | Restart helper and restart-status polling require a real daemon lifecycle |
| HTTP bind | Loopback host | Non-loopback host for restriction testing | Required to validate ADR-004 behavior |
| Transport | HTTP for web UI | UDS/CLI only when validating fallback for non-loopback mutation restrictions | Confirms read parity and privileged local mutation boundaries |
| Workspace fixtures | At least one workspace fixture, recommended id `ws-polybot` | Optional second workspace for cache isolation checks | Required for scoped MCP tests |
| Seeded data | At least one provider, one environment, one MCP server, one hook declaration, one installed extension | Builtin provider present for fallback coverage | Needed for CRUD, fallback, and hybrid behavior |
| Logs / restart artifacts | Writable `~/.agh/log` and `~/.agh/restarts/` state | Clean restart records before execution | Required for observability and restart verification |

## Entry Criteria

- `task_10` through `task_14` are already completed and the settings routes exist.
- The repo builds and starts in a state that `task_16` can execute against.
- `.compozy/tasks/settings-ui/qa/test-plans/` and `.compozy/tasks/settings-ui/qa/test-cases/` contain the planning artifacts from this task.
- The executor has a loopback-bound environment for positive mutation tests.
- The executor has a non-loopback-bound environment or config variant for ADR-004 negative-path validation.
- A workspace fixture exists for workspace-scoped MCP coverage.
- Screenshot capture and bug-report paths under `.compozy/tasks/settings-ui/qa/` are available.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass.
- No open `Critical` or `High` bugs remain without an accepted workaround and follow-up plan.
- `task_16` publishes `.compozy/tasks/settings-ui/qa/verification-report.md` with executed case IDs, sandbox details, evidence links, and rerun results.
- Browser E2E coverage lands in the standard daemon-served lane and the repo verification gates pass after the final fix set.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Restart helper succeeds in persisting settings but fails before replacement boot | Medium | Critical | Treat restart flow as P0, capture `operation_id`, verify terminal status, and require screenshot plus verification-report evidence |
| Restart polling loses continuity after refresh | Medium | High | Include refresh/reconnect steps in the restart case and in the E2E plan |
| MCP scope or target selection edits the wrong source | Medium | Critical | Run explicit global and workspace cases with visible write-target assertions |
| Shadowed or builtin fallback is not obvious after delete | Medium | High | Include delete/refetch checks in providers and MCP cases |
| Immediate extension actions incorrectly surface a restart banner | Medium | High | Keep extension action validation separate from policy-save validation in Hooks & Extensions |
| HTTP mutation restriction is not explained when non-loopback bind returns `403` | Medium | High | Run a dedicated restriction case and require operator-facing messaging evidence |
| Visual drift from the Paper artboards hides route-specific regressions | Medium | Medium | Use dedicated UI cases tied to the 10 local Paper exports and capture screenshots at standard breakpoints |

## Route and Traceability Matrix

| Surface | Route | Source tasks | Paper export | Primary cases | Priority |
|--------|-------|--------------|--------------|---------------|----------|
| Settings shell and index | `/settings` | `task_08`, `task_09` | Shared shell, no dedicated artboard | `TC-FUNC-001`, `TC-UI-014` | P0 |
| General | `/settings/general` | `task_10` | `AGH Settings — General@2x.png` | `TC-FUNC-002`, `TC-UI-014` | P0 |
| Memory | `/settings/memory` | `task_10` | `AGH Settings — Memory@2x.png` | `TC-FUNC-003`, `TC-UI-014` | P1 |
| Observability | `/settings/observability` | `task_10` | `AGH Settings — Observability@2x.png` | `TC-FUNC-004`, `TC-UI-014` | P1 |
| Skills | `/settings/skills` | `task_11` | `AGH Settings — Skills@2x.png` | `TC-FUNC-005`, `TC-UI-014` | P0 |
| Automation | `/settings/automation` | `task_11` | `AGH Settings — Automation@2x.png` | `TC-FUNC-006`, `TC-UI-014` | P1 |
| Network | `/settings/network` | `task_11` | `AGH Settings — Network@2x.png` | `TC-FUNC-007`, `TC-UI-014`, `TC-INT-013` | P1 |
| Providers | `/settings/providers` | `task_12` | `AGH Settings — Providers@2x.png` | `TC-FUNC-008`, `TC-UI-015` | P0 |
| Environments | `/sandbox` | `task_12` | `AGH Settings — Environments@2x.png` | `TC-FUNC-009`, `TC-UI-015` | P1 |
| MCP Servers | `/settings/mcp-servers` | `task_13` | `AGH Settings — MCP Servers@2x.png` | `TC-FUNC-010`, `TC-INT-011`, `TC-UI-015` | P0 |
| Hooks & Extensions | `/settings/hooks-extensions` | `task_14` | `AGH Settings — Hooks & Extensions@2x.png` | `TC-FUNC-012`, `TC-INT-013`, `TC-UI-015` | P0 |

## Artifact and Evidence Contract

### Expected Artifact Layout

- `.compozy/tasks/settings-ui/qa/test-plans/settings-ui-test-plan.md`
- `.compozy/tasks/settings-ui/qa/test-plans/settings-ui-regression.md`
- `.compozy/tasks/settings-ui/qa/test-cases/TC-*.md`
- `.compozy/tasks/settings-ui/qa/issues/BUG-*.md`
- `.compozy/tasks/settings-ui/qa/screenshots/<TC-ID>-*.png`
- `.compozy/tasks/settings-ui/qa/verification-report.md`

### Evidence Rules for `task_16`

- Every executed case must be referenced by case ID in the verification report.
- Every discovered bug must reference the originating case ID.
- Restart-flow executions must capture the `operation_id`, the terminal status, and one screenshot of the visible restart banner state.
- Visual checks must capture one screenshot per route family at the relevant breakpoint.
- If a targeted or full suite is shortened, the verification report must explicitly list the skipped case IDs and why.

## Timeline and Deliverables

| Phase | Owner | Output |
|------|-------|--------|
| Planning | `task_15` | Test plan, regression plan, manual cases, stable QA directory layout |
| Execution | `task_16` | Verification report, screenshots, bug reports, committed E2E coverage |
| Rerun / closeout | `task_16` | Fresh `make test-e2e-web` and `make verify` evidence after final fixes |

## Handoff Notes for `task_16`

- Do not rename or relocate the artifacts in this directory; the execution task expects these exact paths.
- Treat the manual cases in `qa/test-cases/` as the seed matrix for both exploratory/manual execution and durable Playwright coverage selection.
- Preserve the priority model from `settings-ui-regression.md`; if execution scope changes, document the deviation instead of silently reprioritizing.
