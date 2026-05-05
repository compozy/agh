# Unified Capabilities Regression Suite

- Feature: Unified capabilities
- Created: 2026-04-20
- Consumed by: task_10 / `qa-execution`

## Purpose

This suite defines the execution order for unified-capability QA. It assumes the executor has already read the feature test plan and will keep all evidence under `.compozy/tasks/unified-capabilities/qa/`.

## Pre-Flight Gates

Run these before lane execution begins:

1. Confirm the artifact root exists and is writable:
   `.compozy/tasks/unified-capabilities/qa/`
2. Capture the baseline repository state:
   `make verify`
3. Capture the required web baseline:
   `make web-lint`
   `make web-typecheck`

If any pre-flight gate fails, stop and record the failure in `qa/verification-report.md` before attempting manual lanes.

## Case Inventory

| Case ID | Title | Priority | Primary Surface |
| --- | --- | --- | --- |
| `TC-INT-001` | Unified capability schema, digest, and no-catalog behavior | P0 | backend/config/session |
| `TC-INT-002` | Capability envelope validation and recipe replacement | P0 | network/validation |
| `TC-INT-003` | Capability transfer lifecycle preservation | P0 | network/router/lifecycle |
| `TC-INT-004` | Discovery, peer details, and typed API contract alignment | P0 | network/api/uds |
| `TC-UI-001` | Web peer-detail UX and typed-client alignment | P1 | `web/` |
| `TC-REG-001` | Protocol reference and example consistency | P1 | `packages/site` protocol |
| `TC-REG-002` | Runtime docs and repo-guide consistency | P1 | `packages/site` runtime |

## Smoke Lane

- Target duration: 15-30 minutes after pre-flight gates.
- Goal: prove the highest-risk seams changed by the unification are still coherent across backend, API, UI, and public docs.

Execution order:

1. `TC-INT-001`
2. `TC-INT-002`
3. `TC-INT-003`
4. `TC-INT-004`
5. `TC-UI-001`
6. `TC-REG-001`

Stop rules:

- Stop immediately on any P0 failure.
- Stop if a backend/API failure invalidates the UI or docs checks that depend on it.
- Open a `BUG-*.md` file for every confirmed regression before moving on.

Required evidence:

- Pass/fail status for each smoke case in `qa/verification-report.md`
- At least one screenshot for `TC-UI-001` if the browser/manual route is exercised
- Any blocking bug captured under `qa/issues/`

## Targeted Lane

- Target duration: 30-60 minutes.
- Goal: cover the full changed surface introduced by tasks 01-08 without broad unrelated regression work.

Execution order:

1. Re-run any smoke case affected by fixes.
2. Execute all P0 cases again if a backend/network/API fix landed.
3. Execute all P1 cases:
   - `TC-UI-001`
   - `TC-REG-001`
   - `TC-REG-002`

Supporting checks:

- Relevant targeted Go tests for `internal/config`, `internal/session`, `internal/network`, `internal/api/contract`, `internal/api/core`, and `internal/api/udsapi`
- Relevant web regression tests for the network route and peer-detail rendering
- `make site-build`

Stop rules:

- Any newly discovered P0 failure reverts execution back to remediation plus a rerun of the smoke lane.
- P1 failures may continue only if they are documented immediately with clear workaround and fix scope.

## Full Lane

- Target duration: 2-4 hours depending on remediation.
- Goal: prove the repo is ready to move past task_10 with fresh end-to-end evidence and clean verification.

Full lane contents:

1. Re-execute every case in this suite that was touched by fixes.
2. Run the full repository gate again:
   `make verify`
3. Run the full required web gates again:
   `make web-lint`
   `make web-typecheck`
   `make web-test`
4. Build the site again:
   `make site-build`
5. Do a final doc/UI sweep for steady-state regressions:
   - no first-class `recipe` wording in surfaced docs or operator-visible network UI
   - protocol examples and runtime docs agree on `kind:"capability"`, `digest`, and `requirements`

Required outputs:

- Final `qa/verification-report.md`
- All screenshots referenced by the report saved under `qa/screenshots/`
- All confirmed issues written to `qa/issues/BUG-*.md`

## Pass / Fail Criteria

PASS:

- All P0 cases pass.
- At least 90% of P1 cases pass.
- No critical or high-severity bug remains open.
- Final rerun verification commands complete successfully.

FAIL:

- Any P0 case fails.
- A critical or high-severity bug is found without a fix or accepted blocker decision.
- Final rerun verification commands fail.

CONDITIONAL:

- Only P1 cases fail, each failure has a documented workaround, and the verification report clearly marks the remaining risk.

## Handoff Notes for Task_10

- Do not change the output root. All evidence stays under `.compozy/tasks/unified-capabilities/qa/`.
- Use the case IDs in filenames, screenshots, issue references, and the final verification report.
- If a fix touches backend discovery or transfer semantics, rerun all four P0 cases before claiming recovery.
- If a fix touches `packages/site`, rerun both doc cases and `make site-build` before closing the issue.
