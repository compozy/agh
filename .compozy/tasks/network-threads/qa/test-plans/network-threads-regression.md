# Network Threads Regression Suite

## Purpose

This suite orders the Network Threads validation work for `qa-execution`. It distinguishes readiness checks from behavior-first proof and keeps the mandatory public-thread, direct-room, summarize-back, and direct-resolve-race journeys at the front of the execution queue.

## Suite Tiers

| Tier | Duration | Frequency | Coverage |
| --- | --- | --- | --- |
| Smoke | 15-30 min | Before QA execution and before every fix loop | Command availability, docs/contract guardrails, dev-server readiness, bootstrap manifest sanity. |
| Targeted | 30-60 min | Every network-thread change | P0 journeys, changed CLI/API/Web/runtime surfaces, direct resolve race, legacy rejection. |
| Full | 2-4 hours | Release / final loop QA | All P0/P1 cases, provider-backed evidence when reachable, browser flows, `make test-e2e-runtime`, `make test-e2e-web`, `make verify`. |
| Sanity | 10-15 min | After hotfix | Re-run the failed case, its paired cross-surface assertion, and the narrow command that reproduced the bug. |

## Execution Order

1. Run SMOKE-001. If readiness fails, stop and file/fix the readiness issue first.
2. Run TC-SCEN-001 public thread coordination.
3. Run TC-SCEN-002 restricted direct-room handoff.
4. Run TC-SCEN-003 summarize-back-to-thread.
5. Run TC-INT-001 direct-room resolve race.
6. Run TC-UI-001 Web thread/direct navigation and browser artifacts.
7. Run TC-REG-001 hard-cut legacy rejection and docs/contract guardrails.
8. Run `make test-e2e-runtime`.
9. Run `make test-e2e-web`.
10. Run full `make verify` after the last code or fixture change.
11. Re-run the highest-risk P0 journey after `make verify` passes.

## Smoke Suite

| ID | Purpose | Expected result |
| --- | --- | --- |
| SMOKE-001 | Confirm QA can start against a current build and current artifact set. | Broad gates and harness commands are discoverable; active docs/contracts do not teach old protocol fields. |

Smoke output is readiness-only. It must not be counted as release-grade behavior-first evidence.

## Targeted Suite

| ID | Priority | Coverage | Required evidence |
| --- | --- | --- | --- |
| TC-SCEN-001 | P0 | Public thread coordination | CLI/API/Web agree on `thread_id`, messages, work state, and participant/summary metadata. |
| TC-SCEN-002 | P0 | Restricted direct-room handoff | Direct room is deterministic, restricted to two peers, and isolated from public thread queries. |
| TC-SCEN-003 | P0 | Summarize-back workflow | Direct-room conclusion appears as public `say` without reusing direct-room `work_id`. |
| TC-INT-001 | P0 | Concurrent direct resolve | Concurrent resolves return one `direct_id`, one persisted room, and no fragmented messages. |
| TC-UI-001 | P1 | Web operator flow | Browser evidence proves thread/direct routes and artifact keys. |
| TC-REG-001 | P1 | Hard-cut guardrails | Legacy fields/flags/kinds are rejected or absent from active docs/contracts. |

## Full Suite

Full suite includes all targeted cases plus:

- `make test-e2e-runtime`
- `make test-e2e-web`
- `make verify`
- Provider-backed AGH session evidence when reachable.
- Browser screenshots for thread list/detail, direct list/detail, missing route/error state, and active composer state.
- Verification report with QA bootstrap block when a healthy lab remains.

## Pass / Fail Criteria

PASS:

- All P0 behavioral journeys pass.
- All live provider boundaries are either exercised or explicitly documented.
- Cross-surface state agrees for at least one persisted thread, direct room, and work item.
- 90% or more P1 cases pass.
- No critical or high bug remains open.
- Final `make verify` exits 0 after the last code or fixture change.

FAIL:

- Any P0 journey fails without a fixed root cause.
- Required live provider behavior is skipped without an exact blocker.
- Public thread and direct-room visibility disagree across surfaces.
- Legacy `interaction_id`, `kind:"direct"`, or old CLI flags are accepted on active surfaces.
- Security or data loss issue is found.
- `make verify` fails after the final fix.

CONDITIONAL:

- Only P2/P3 issues remain, with bug reports and explicit operator impact.
- A live provider boundary is unavailable, but all local product surfaces and E2E harnesses pass.

## Maintenance Notes

- Keep this suite aligned with `.compozy/tasks/network-threads/qa/test-cases/*.md`.
- When a bug is found, add the bug ID to the originating test case execution history.
- When a scenario becomes automated in a narrower test, keep the behavior-first case; automation is supporting evidence, not a replacement for the operator journey.

