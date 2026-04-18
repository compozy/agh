# Tasks UI Regression Suite

## Purpose

This suite defines the smoke, targeted, and full execution priorities for Tasks plus the required Settings-adjacent regression coverage. It is the primary execution-order document for `task_19`.

## Suite Inventory

| Suite | Duration Target | When To Run | Stop Conditions |
| --- | --- | --- | --- |
| Smoke | 15-30 min | before deeper execution, after each significant fix | stop immediately on any P0 failure |
| Targeted | 30-60 min | after Tasks/Settings changes or browser harness changes | continue only after Smoke passes |
| Full | 2-4 hours | release readiness, branch closeout, or broad fix batches | requires Smoke + Targeted green or documented blockers |

## Execution Order

1. Run the Settings preflight `TC-REG-010`.
2. Run the Tasks smoke path in sidebar-to-run-detail order.
3. If Smoke passes, execute the remaining P0 cases.
4. Run P1 targeted Tasks and Settings cases.
5. Finish with the full matrix and exploratory follow-up only after all scripted priorities are accounted for.

## Smoke Suite

| Order | Case ID | Priority | Why It Runs In Smoke |
| --- | --- | --- | --- |
| 1 | `TC-REG-010` | P0 | prevents fake Settings execution when the route family is absent |
| 2 | `TC-FUNC-003` | P0 | proves the Tasks sidebar entry and split-view route are usable |
| 3 | `TC-FUNC-006` | P0 | proves draft creation and publish-aware entry flow |
| 4 | `TC-FUNC-007` | P0 | proves task-native detail deep links and timeline rendering |
| 5 | `TC-FUNC-008` | P0 | proves run-detail deep links and linked-session drill-down entry |
| 6 | `TC-FUNC-001` | P0 | proves one aggregate Tasks mode beyond list/detail |
| 7 | `TC-REG-011` | P0 if Settings preflight passes | proves the Settings shell is reachable in the operator app |
| 8 | `TC-REG-012` | P0 if Settings preflight passes | proves at least one restart-aware Settings save path |

## Targeted Regression

Run this suite after any Tasks UI, Tasks API contract, aggregate view, live-state, or Settings-shell change.

| Area | Case IDs | Priority | Notes |
| --- | --- | --- | --- |
| Tasks aggregates | `TC-FUNC-001`, `TC-FUNC-002` | P0 | dashboard/inbox should be treated as separate read-model validations |
| Tasks browse/create | `TC-FUNC-003`, `TC-FUNC-004`, `TC-FUNC-005`, `TC-FUNC-006` | P0/P1 | list and create stay P0; kanban and empty-state remain P1 |
| Tasks detail/live | `TC-FUNC-007`, `TC-FUNC-008`, `TC-FUNC-009` | P0/P1 | run-detail stays P0; multi-agent live remains P1 but cannot be forgotten |
| Settings shell/save | `TC-REG-010`, `TC-REG-011`, `TC-REG-012` | P0 | preflight is mandatory before deeper Settings work |
| Settings collection/advanced | `TC-REG-013`, `TC-REG-014` | P1 | required when the surface exists; blocked if preflight fails |

## Full Regression

Run the complete matrix below before declaring the feature ready after browser execution:

- `TC-FUNC-001`
- `TC-FUNC-002`
- `TC-FUNC-003`
- `TC-FUNC-004`
- `TC-FUNC-005`
- `TC-FUNC-006`
- `TC-FUNC-007`
- `TC-FUNC-008`
- `TC-FUNC-009`
- `TC-REG-010`
- `TC-REG-011`
- `TC-REG-012`
- `TC-REG-013`
- `TC-REG-014`

## P0 And P1 Rules

### P0

- `TC-FUNC-001`
- `TC-FUNC-002`
- `TC-FUNC-003`
- `TC-FUNC-006`
- `TC-FUNC-007`
- `TC-FUNC-008`
- `TC-REG-010`
- `TC-REG-011`
- `TC-REG-012`

### P1

- `TC-FUNC-004`
- `TC-FUNC-005`
- `TC-FUNC-009`
- `TC-REG-013`
- `TC-REG-014`

## Pass / Fail Criteria

### PASS

- All P0 cases pass.
- No critical bug remains open.
- The verification report references the executed case IDs and screenshot/bug paths.
- If Settings is present, the required Settings flows run in the same browser lane.

### FAIL

- Any P0 case fails.
- A task create/publish, detail, run-detail, dashboard, inbox, or Settings preflight/save flow regresses.
- The browser lane cannot produce screenshots or reproducible evidence for the critical flows.

### CONDITIONAL

- One or more P1 cases fail with a documented workaround and follow-up fix plan.
- Settings cases `TC-REG-011` through `TC-REG-014` are blocked only because `TC-REG-010` proved the settings route family is absent on the execution branch, and the blocker is explicitly documented.

## Blocker Expectations

- `TC-REG-010` failing is a **branch blocker**, not a skipped test.
- If `TC-REG-010` fails, mark `TC-REG-011` through `TC-REG-014` as blocked, continue Tasks execution, and include the Settings blocker in `.compozy/tasks/tasks-ui/qa/verification-report.md`.
- Missing selectors, seeds, or evidence paths for P0 flows are also blockers until corrected in the shared `web/e2e` pattern.

## Evidence Requirements

- Store screenshots under `.compozy/tasks/tasks-ui/qa/screenshots/`.
- Store structured bug reports under `.compozy/tasks/tasks-ui/qa/issues/`.
- Store the final outcome in `.compozy/tasks/tasks-ui/qa/verification-report.md`.
- Keep the suite names and case IDs unchanged between planning and execution.
