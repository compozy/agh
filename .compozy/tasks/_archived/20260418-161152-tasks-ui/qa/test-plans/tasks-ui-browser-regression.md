# Tasks UI Browser Regression Handoff

## Purpose

This document narrows the regression plan down to the daemon-served Playwright lane under `web/e2e/` so `task_19` can add durable browser coverage without redefining scope.

## Browser Lane Contract

- Use `web/playwright.config.ts` as the authoritative browser config.
- Use `web/e2e/fixtures/test.ts` as the only browser fixture entrypoint.
- Reuse `web/e2e/fixtures/runtime.ts`, `web/e2e/fixtures/runtime-seed.ts`, and `web/e2e/fixtures/selectors.ts`.
- Keep the execution artifact root aligned with `.compozy/tasks/tasks-ui/qa/` and the runtime artifact collector used by the existing browser harness.

## Required Browser Specs

| Planned Spec | Required Case IDs | Priority | Notes |
| --- | --- | --- | --- |
| `web/e2e/tasks.spec.ts` or `web/e2e/tasks-*.spec.ts` | `TC-FUNC-003`, `TC-FUNC-006`, `TC-FUNC-007`, `TC-FUNC-008` | P0 | minimum Tasks browser proof: sidebar entry, draft creation, publish, detail, run-detail |
| same Tasks spec or companion spec | `TC-FUNC-001` or `TC-FUNC-002` | P0 | one aggregate mode must be validated in the same route family |
| same Tasks spec or companion spec | `TC-FUNC-004`, `TC-FUNC-005`, `TC-FUNC-009` | P1 | expand to kanban, empty state, and multi-agent live/fallback once P0 is stable |
| `web/e2e/settings.spec.ts` or `web/e2e/settings-*.spec.ts` | `TC-REG-010` | P0 | Settings route-presence preflight happens before all other Settings browser work |
| same Settings spec or companion spec | `TC-REG-011`, `TC-REG-012` | P0 if preflight passes | shell navigation + one restart-aware save flow |
| same Settings spec or companion spec | `TC-REG-013`, `TC-REG-014` | P1 if preflight passes | collection CRUD + advanced scoped settings flow |

## Selector Expectations

### Existing Stable Selectors To Reuse

- Sidebar and onboarding:
  - `app-sidebar`
  - `workspace-onboarding`
  - `workspace-use-global`
  - `nav-tasks`
- Tasks route shell:
  - `tasks-mode-list`
  - `tasks-mode-kanban`
  - `tasks-mode-dashboard`
  - `tasks-mode-inbox`
  - `tasks-open-create`
  - `tasks-detail-content`
  - `tasks-run-detail-content`

### Planned Selector Work

- Add a `tasksOperatorSelectors` helper to `web/e2e/fixtures/selectors.ts` if the current raw test IDs are not enough for stable task card, timeline, run, or inbox item targeting.
- Add a `settingsOperatorSelectors` helper only if the Settings surface exists on the execution branch.
- Avoid brittle text-only targeting for shell, panel, table, and form assertions.

## Seed Expectations

- Extend `web/e2e/fixtures/runtime-seed.ts` only where needed to create:
  - a draft task
  - a publishable task
  - a task with a visible run-detail path
  - dashboard/inbox aggregate data
  - multi-agent live data or a legitimate fallback state
- If Settings exists, add deterministic seed helpers only for the chosen restart-aware save, collection CRUD, and advanced-flow prerequisites.
- Do not fork a Tasks- or Settings-specific harness outside the shared runtime fixtures.

## Evidence Expectations

| Artifact | Minimum Expectation |
| --- | --- |
| Screenshots | at least one screenshot per critical Tasks flow group and one per Settings flow group that actually executes |
| Verification report | references the case IDs executed in browser, the spec files added, and any blockers |
| Bug reports | one `BUG-*.md` per reproducible execution failure discovered during task_19 |

## Settings Branch-Readiness Rule

- Before implementing `web/e2e/settings*.spec.ts`, run the `TC-REG-010` preflight.
- If `web/src/routes/_app/settings*.tsx` is still absent, do not fabricate a Settings spec.
- In that case, keep Tasks browser work moving, but report the Settings blocker explicitly in `.compozy/tasks/tasks-ui/qa/verification-report.md`.

## Completion Gate For Task 19

- `make test-e2e-web` passes with the new Tasks coverage included.
- `make verify` passes after the final browser-related fix set.
- The verification report, screenshot paths, and bug paths all point back to the case IDs defined in this task's QA artifacts.
