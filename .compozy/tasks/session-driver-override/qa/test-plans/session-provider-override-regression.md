# Session Provider Override Regression Suite

**Feature:** Per-session ACP provider override
**QA output path:** `.compozy/tasks/session-driver-override/qa/`
**Created:** 2026-04-21
**Last Updated:** 2026-04-21

## Purpose

This regression suite gives task 08 a fixed execution order for smoke, targeted, and full validation. It is intentionally feature-scoped: every lane proves provider override behavior across backend, storage, transport, CLI, and browser seams instead of relying on generic smoke checks.

## Preflight for Every Lane

1. Activate `/qa-execution` with `qa-output-path=.compozy/tasks/session-driver-override`.
2. Read `qa/test-plans/session-provider-override-test-plan.md` and all `qa/test-cases/TC-*.md`.
3. Confirm the required fixtures exist for:
   - default vs override provider matrix
   - agent default drift on resume
   - removed persisted provider
   - legacy blank-provider metadata
   - legacy SQLite schema without `sessions.provider`
4. Capture the current repo health state before scenario execution.
5. Identify the repo verification lanes that cover task 05 automatic internal creators defaulting to empty provider.
6. Publish all evidence, screenshots, and issues back into the same QA root.

## Global Stop Conditions

- Any smoke case fails.
- Any P0 case fails.
- A Critical or High-severity bug is discovered with no immediate fix.
- Repository verification gates fail after a code fix.
- The execution environment cannot provide one of the required surfaces or fixtures.

## Pass / Fail Rules

**PASS**

- All P0 cases pass.
- 90% or more of P1 cases pass.
- No Critical or High-severity open issues remain.
- `make verify` and `make codegen-check` pass after the last fix.

**FAIL**

- Any P0 case fails.
- Any required browser-visible create/resume case cannot be completed.
- Any explicit-surface parity check disagrees on the effective provider.
- Migration or legacy repair leaves blank/incorrect provider state in storage.

**CONDITIONAL PASS**

- One or more P1 cases fail, but the failure has a documented issue, a contained workaround, and no P0 seam is degraded.

## Smoke Lane

**Goal:** Detect the highest-risk regressions quickly before broader execution.

**Expected duration:** 15-30 minutes

**Execution order**

| Order | Case ID | Priority | Why It Is In Smoke | Required Evidence |
| --- | --- | --- | --- | --- |
| 1 | `TC-FUNC-002` | P0 | Confirms explicit override still changes runtime/provider-owned state coherently. | Backend/provider evidence plus persisted provider. |
| 2 | `TC-FUNC-003` | P0 | Confirms invalid provider still fails before side effects. | Error payload/logs plus absence checks. |
| 3 | `TC-INT-004` | P0 | Confirms persisted provider still wins on resume. | `session.json`, global DB row, resumed payload/output. |
| 4 | `TC-INT-008` | P0 | Confirms explicit surface parity before deeper execution. | HTTP, UDS, CLI, and Host API samples. |
| 5 | `TC-UI-010` | P0 | Confirms the web create path still routes through the dialog. | Screenshot plus create payload and provider badge. |
| 6 | `TC-UI-011` | P0 | Confirms removed-provider resume failure is actionable in the UI. | Screenshot plus error payload naming session id/provider. |

**Smoke exit rule**

- If any smoke case fails, stop execution, file or update the bug, fix the root cause, rerun the failing smoke case, then rerun the full smoke lane before proceeding.

## Targeted Lane

**Goal:** Validate every seam touched by the provider-override feature set after smoke passes.

**Expected duration:** 30-60 minutes

**Execution order**

| Order | Case ID | Priority | Focus Area | Evidence Emphasis |
| --- | --- | --- | --- | --- |
| 1 | `TC-FUNC-001` | P1 | No-override baseline behavior | Default provider selection and persistence. |
| 2 | `TC-FUNC-002` | P0 | Override runtime semantics | Provider-owned command/model/MCP replacement. |
| 3 | `TC-FUNC-003` | P0 | Validation-before-persistence | Absence of side effects and explicit logs. |
| 4 | `TC-INT-004` | P0 | Resume determinism | Persisted provider over new agent default. |
| 5 | `TC-INT-005` | P0 | Removed-provider explicit failure | Session id + missing provider in response/logs. |
| 6 | `TC-INT-007` | P0 | Legacy blank-provider repair | One-time repair and deterministic follow-up behavior. |
| 7 | `TC-INT-008` | P0 | HTTP/UDS/CLI/Host API parity | Cross-surface agreement on `provider`. |
| 8 | `TC-INT-009` | P1 | Workspace provider catalog | Sorted provider options from workspace detail. |
| 9 | `TC-UI-010` | P0 | Dialog-driven create flow | Prefill, picker contents, submit payload, provider badge. |
| 10 | `TC-UI-011` | P0 | Inline resume-failure UX | Inline panel, actionable message, no toast-only outcome. |

**Targeted exit rule**

- All P0 targeted cases must pass before proceeding to full regression.
- If a P1 targeted case fails, create a bug and decide whether the failure should be escalated to P0 before moving on.

## Full Lane

**Goal:** Combine the feature matrix with storage migration and the repo verification contract for release-quality confidence.

**Expected duration:** 2-4 hours

**Execution order**

### Phase 1: Repo baseline

1. Run the task 08 baseline verification steps.
2. Record the pre-execution health state in `qa/verification-report.md`.

### Phase 2: Feature matrix

| Order | Case ID | Priority | Focus Area |
| --- | --- | --- | --- |
| 1 | `TC-FUNC-001` | P1 | No-override baseline |
| 2 | `TC-FUNC-002` | P0 | Explicit override runtime semantics |
| 3 | `TC-FUNC-003` | P0 | Invalid-provider create failure |
| 4 | `TC-INT-004` | P0 | Persisted resume after default drift |
| 5 | `TC-INT-005` | P0 | Removed-provider resume failure |
| 6 | `TC-INT-006` | P1 | Global DB migration |
| 7 | `TC-INT-007` | P0 | Legacy repair and reconcile |
| 8 | `TC-INT-008` | P0 | Explicit surface parity |
| 9 | `TC-INT-009` | P1 | Workspace provider catalog |
| 10 | `TC-UI-010` | P0 | Web dialog flow |
| 11 | `TC-UI-011` | P0 | Web resume-failure UX |

### Phase 3: Post-fix reruns

- If a bug is fixed, rerun:
  - the originating `TC-*` case
  - the containing lane
  - the repo coverage that exercises automatic internal creators staying on empty provider defaults when provider plumbing was touched
  - `make verify`
  - `make codegen-check`
  - any targeted web/API/CLI test lane changed by the fix

## Suite Membership Matrix

| Case ID | Smoke | Targeted | Full |
| --- | --- | --- | --- |
| `TC-FUNC-001` | No | Yes | Yes |
| `TC-FUNC-002` | Yes | Yes | Yes |
| `TC-FUNC-003` | Yes | Yes | Yes |
| `TC-INT-004` | Yes | Yes | Yes |
| `TC-INT-005` | No | Yes | Yes |
| `TC-INT-006` | No | No | Yes |
| `TC-INT-007` | No | Yes | Yes |
| `TC-INT-008` | Yes | Yes | Yes |
| `TC-INT-009` | No | Yes | Yes |
| `TC-UI-010` | Yes | Yes | Yes |
| `TC-UI-011` | Yes | Yes | Yes |

## Evidence Publication Rules

- Store screenshots in `qa/screenshots/`.
- Store discovered bugs in `qa/issues/BUG-*.md`.
- Publish the final execution summary in `qa/verification-report.md`.
- Cross-link every failed case to the related bug ID and the later rerun result.
- When the same defect appears across backend and web surfaces, keep one bug file with links to each evidence artifact.

## Handoff Notes for Task 08

- Keep the case order above unless a blocking environment issue requires an explicit deviation in the verification report.
- Do not downgrade a case priority on the fly. If the severity changed, document why in the report and keep the original plan intact.
- Use this suite to decide reruns after fixes instead of rebuilding the matrix ad hoc.
