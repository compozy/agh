# Tasks UI QA Test Plan

## Executive Summary

This plan defines the reusable QA scope for the first-class Tasks operator surface under `/_app/tasks` and the adjacent Settings regression coverage that `task_19` must execute without redefining artifact paths, priorities, or blocker handling.

### Objectives

- Prove every shipped Tasks surface has a traceable QA artifact before execution begins.
- Define the exact P0/P1 routes and behaviors that must land in the repo's daemon-served `web/e2e` lane.
- Pull Settings into the regression matrix as a critical adjacent operator surface instead of leaving it implied.
- Keep all evidence, bugs, screenshots, and execution reporting under `.compozy/tasks/tasks-ui/qa/`.

### Key Risks

- Tasks live/detail flows depend on deterministic task, run, and tree seed state for browser execution.
- Dashboard and inbox correctness depends on observer freshness and mutation invalidation.
- The current branch does not contain `web/src/routes/_app/settings*.tsx`, so Settings browser coverage needs a preflight gate and explicit blocker behavior.
- Figma MCP is unavailable for this run, so visual/manual coverage must derive from the committed Paper exports under `docs/design/paper/tasks/` and `docs/design/paper/settings/`.

## Scope

### In Scope

- Tasks route family:
  - `/_app/tasks`
  - `/_app/tasks/$id`
  - `/_app/tasks/$id/runs/$runId`
- Tasks surfaces:
  - dashboard
  - inbox
  - split-view list
  - kanban
  - empty state
  - create modal
  - detail timeline
  - run detail
  - multi-agent live
- Settings-adjacent regression-critical flows for the shared browser lane:
  - route/surface presence preflight
  - settings shell navigation
  - one restart-aware save flow
  - one collection CRUD flow
  - one advanced settings flow such as workspace-scoped MCP or hooks/extensions
- Execution artifact ownership and evidence paths under `.compozy/tasks/tasks-ui/qa/`

### Out of Scope

- Executing manual, API, or browser flows in this task
- Fixing product bugs or browser harness gaps before `task_19`
- Broad feature testing outside Tasks and the explicitly required Settings regression surfaces
- Inventing a second E2E harness outside `web/e2e/`

## Test Strategy And Approach

### Phase 1: Preflight

- Confirm the repo browser lane is the daemon-served Playwright harness under `web/e2e/`.
- Confirm the Tasks route family renders inside the main app shell and remains the first execution target.
- Run the Settings route-presence preflight before any Settings execution.
- If Settings surfaces are absent on the execution branch, write an explicit blocker in `.compozy/tasks/tasks-ui/qa/verification-report.md` and mark only the downstream Settings cases blocked. Do not silently skip them.

### Phase 2: Tasks P0 Execution

- Use the Tasks sidebar entry as the primary browser entrypoint.
- Cover list browsing, draft creation, publication, task detail inspection, run-detail navigation, and one aggregate mode in the first browser pass.
- Keep screenshots and bug reports aligned to the case IDs listed in this plan.

### Phase 3: Tasks P1 And Aggregate/Live Coverage

- Extend coverage to dashboard + inbox parity, kanban, empty state/template flows, and multi-agent live/fallback behavior.
- Reuse the `@/systems/tasks` route/system ownership model when evaluating regressions; do not accept behaviors that only work through legacy or session-first detours.

### Phase 4: Settings Regression-Critical Coverage

- Execute shell navigation, restart-aware save, collection CRUD, and advanced configuration only if the Settings preflight passes.
- Follow the same `web/e2e` fixture/runtime/selector pattern used by existing Automation, Bridges, and Network specs.
- Treat Settings as a blocker-bound adjacent surface, not optional exploratory coverage.

## Traceability Matrix

| Surface | Entry Route / Preflight | Design Reference | Manual Case IDs | Browser Priority | Owner |
| --- | --- | --- | --- | --- | --- |
| Tasks dashboard | `/_app/tasks` with dashboard mode | `docs/design/paper/tasks/AGH Tasks — Dashboard@2x.png` | `TC-FUNC-001` | P0 | `task_19` |
| Tasks inbox | `/_app/tasks` with inbox mode | `docs/design/paper/tasks/AGH Tasks — Inbox@2x.png` | `TC-FUNC-002` | P0 | `task_19` |
| Tasks list split view | `/_app/tasks` with list mode | `docs/design/paper/tasks/AGH Tasks — List (Split View)@2x.png` | `TC-FUNC-003` | P0 | `task_19` |
| Tasks kanban | `/_app/tasks` with kanban mode | `docs/design/paper/tasks/AGH Tasks — Kanban View@2x.png` | `TC-FUNC-004` | P1 | `task_19` |
| Tasks empty state | `/_app/tasks` with zero-task seed | `docs/design/paper/tasks/AGH Tasks — Empty State@2x.png` | `TC-FUNC-005` | P1 | `task_19` |
| Tasks create modal | `/_app/tasks` modal flow | `docs/design/paper/tasks/AGH Tasks — Create Modal@2x.png` | `TC-FUNC-006` | P0 | `task_19` |
| Tasks detail timeline | `/_app/tasks/$id` | `docs/design/paper/tasks/AGH Tasks — Detail (Events SSE)@2x.png` | `TC-FUNC-007` | P0 | `task_19` |
| Tasks run detail | `/_app/tasks/$id/runs/$runId` | `docs/design/paper/tasks/AGH Tasks — Run Detail@2x.png` | `TC-FUNC-008` | P0 | `task_19` |
| Tasks multi-agent live | `/_app/tasks/$id` Agents panel | `docs/design/paper/tasks/AGH Tasks — Multi-Agent Live@2x.png` | `TC-FUNC-009` | P1 | `task_19` |
| Settings preflight | `web/src/routes/_app/settings*.tsx` presence + UI entry check | `docs/design/paper/settings/*.png` | `TC-REG-010` | P0 | `task_19` |
| Settings shell navigation | `/_app/settings*` if present | `docs/design/paper/settings/AGH Settings — General@2x.png`, `docs/design/paper/settings/AGH Settings — Providers@2x.png`, `docs/design/paper/settings/AGH Settings — Automation@2x.png`, `docs/design/paper/settings/AGH Settings — Network@2x.png`, `docs/design/paper/settings/AGH Settings — Observability@2x.png`, `docs/design/paper/settings/AGH Settings — Memory@2x.png`, `docs/design/paper/settings/AGH Settings — Skills@2x.png`, `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png`, `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png`, `docs/design/paper/settings/AGH Settings — Environments@2x.png` | `TC-REG-011` | P0 | `task_19` |
| Settings restart-aware save | first shipped restart-aware settings form | `docs/design/paper/settings/AGH Settings — General@2x.png` | `TC-REG-012` | P0 | `task_19` |
| Settings collection CRUD | first shipped collection editor, prefer Environments then Providers | `docs/design/paper/settings/AGH Settings — Environments@2x.png`, `docs/design/paper/settings/AGH Settings — Providers@2x.png` | `TC-REG-013` | P1 | `task_19` |
| Settings advanced flow | first shipped advanced scoped surface, prefer MCP Servers then Hooks & Extensions | `docs/design/paper/settings/AGH Settings — MCP Servers@2x.png`, `docs/design/paper/settings/AGH Settings — Hooks & Extensions@2x.png` | `TC-REG-014` | P1 | `task_19` |

## Environment Matrix

| Sandbox ID | Purpose | Runtime / Browser | Viewports | Notes |
| --- | --- | --- | --- | --- |
| `ENV-WEB-01` | Browser gate | daemon-served `web/e2e` + Playwright `Desktop Chrome` | desktop only | Matches `web/playwright.config.ts` and is the blocking browser lane |
| `ENV-WEB-02` | Manual responsive/visual spot checks | local daemon-served web app in Chrome devtools | `1280`, `768`, `375` | Derived from the `qa-report` responsive standard and local Paper exports |
| `ENV-WEB-03` | Aggregate/live data checks | daemon + seeded runtime helpers | desktop | Must support deterministic task draft, run-detail, dashboard/inbox, and tree-live states |
| `ENV-WEB-04` | Settings preflight | repo file tree + running app shell | desktop | Required before any Settings case moves past `TC-REG-010` |

## Entry Criteria

- `task_11`, `task_14`, `task_15`, `task_16`, and `task_17` are complete on the execution branch.
- The Tasks QA artifacts under `.compozy/tasks/tasks-ui/qa/` exist and match this plan.
- The daemon-served browser harness under `web/e2e/` is operational.
- The local Paper exports under `docs/design/paper/tasks/` and `docs/design/paper/settings/` are available for visual/manual reference.
- Task runtime seeds or equivalent fixtures can create:
  - a draft task
  - a publishable task
  - a task with run detail
  - a dashboard/inbox aggregate state
  - a task tree with descendant activity or a valid fallback state
- Settings execution beyond `TC-REG-010` may begin only if the settings route family is present on the execution branch.

## Exit Criteria

- All P0 manual cases are executed or explicitly blocked with evidence.
- All P0 browser-regression flows are implemented or exercised in the `web/e2e` lane.
- No open Critical or High bug remains against the Tasks P0 flow set without an explicit blocker note.
- If Settings routes are absent, the blocker is captured explicitly in `.compozy/tasks/tasks-ui/qa/verification-report.md`; if routes are present, `TC-REG-011` through `TC-REG-014` are executed.
- `.compozy/tasks/tasks-ui/qa/verification-report.md` references the same case IDs and suite names defined here.
- Screenshots and bug reports, if any, are stored under the planned QA artifact root.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Settings route family is still absent on the execution branch | High | High | Run `TC-REG-010` first, write a blocker immediately if it fails, and do not fake Settings browser coverage |
| Task live/tree flows are not seedable in a deterministic way | High | High | Task `19` should extend `web/e2e/fixtures/runtime-seed.ts` only for draft/run/live prerequisites and keep those helpers reusable |
| Dashboard or inbox data is stale after create/publish/triage mutations | Medium | High | Execute aggregate cases after mutation flows and treat stale counts or lanes as blocking regressions |
| Sidebar/workspace onboarding blocks route access during browser runs | Medium | Medium | Use the existing onboarding selectors and global-workspace flow before asserting feature-specific states |
| Stable selectors are missing for Tasks or future Settings surfaces | Medium | Medium | Extend `web/e2e/fixtures/selectors.ts` with durable test IDs before writing broad text-based selectors |
| Paper export and implementation drift on layout or copy | Medium | Medium | Use the committed exports as the visual source of truth and capture screenshot-backed discrepancies under `.compozy/tasks/tasks-ui/qa/` |

## Timeline And Deliverables

| Phase | Producer | Deliverable | Output Path |
| --- | --- | --- | --- |
| Planning | `task_18` | feature test plan | `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-test-plan.md` |
| Planning | `task_18` | regression suites | `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-regression.md`, `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-browser-regression.md` |
| Planning | `task_18` | route-by-route manual cases | `.compozy/tasks/tasks-ui/qa/test-cases/TC-*.md` |
| Execution | `task_19` | browser specs and selector/seed extensions | `web/e2e/` and `web/e2e/fixtures/` |
| Execution | `task_19` | screenshots and bug reports | `.compozy/tasks/tasks-ui/qa/screenshots/`, `.compozy/tasks/tasks-ui/qa/issues/` |
| Execution | `task_19` | verification report | `.compozy/tasks/tasks-ui/qa/verification-report.md` |

## Artifact Ownership

| Artifact | Producer | Consumer | Purpose |
| --- | --- | --- | --- |
| `qa/test-plans/tasks-ui-test-plan.md` | `task_18` | `task_19` | source of truth for scope, priorities, environments, and blocker policy |
| `qa/test-plans/tasks-ui-regression.md` | `task_18` | `task_19` | smoke/targeted/full execution order and pass/fail rules |
| `qa/test-plans/tasks-ui-browser-regression.md` | `task_18` | `task_19` | `web/e2e`-specific mapping for specs, fixtures, selectors, and screenshots |
| `qa/test-cases/TC-*.md` | `task_18` | `task_19` | route-by-route execution matrix with expected results |
| `qa/screenshots/` | `task_19` | reviewers | captured browser/manual evidence keyed back to cases |
| `qa/issues/BUG-*.md` | `task_19` if needed | reviewers and follow-up tasks | structured discrepancy tracking |
| `qa/verification-report.md` | `task_19` | reviewers and tracking | final evidence summary and blocker accounting |

## Current Branch Notes

- The current branch exposes the Tasks route family and `web/e2e` infrastructure needed for planning.
- The current branch does **not** expose `web/src/routes/_app/settings*.tsx`.
- Because `task_19` explicitly requires Settings browser coverage when the surface exists, this plan treats Settings route presence as a P0 execution preflight rather than assuming the UI is available.
