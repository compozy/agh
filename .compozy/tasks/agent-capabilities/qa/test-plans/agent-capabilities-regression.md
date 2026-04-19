# Agent Capabilities Regression Suite

- Feature: Agent Capabilities
- Generated: `2026-04-19`
- Consumed by: `task_07`
- `qa-output-path`: `.compozy/tasks/agent-capabilities`

## Purpose

This suite defines the execution lanes and stop conditions for capability QA. It assumes the executor will use the shared artifacts under `.compozy/tasks/agent-capabilities/qa/` and will not redefine paths, priorities, or evidence expectations.

## Artifact Inputs

- `qa/test-plans/agent-capabilities-test-plan.md`
- `qa/test-cases/TC-INT-001.md` through `TC-INT-013.md`
- `qa/test-cases/TC-FUNC-014.md`
- `qa/issues/BUG-*.md` only when execution discovers a discrepancy
- `qa/screenshots/` only when extra evidence is useful

## Command Seeds

These are guidance seeds for `task_07`, not a replacement for live execution evidence.

- Baseline and final gate: `make verify`
- Focused loader sweep: `go test ./internal/config`
- Focused join/runtime sweep: `go test ./internal/session`
- Focused discovery sweep: `go test ./internal/network`
- Focused API visibility sweep: `go test ./internal/api/core`

Recommended regression anchors when a case fails:

- `internal/config/capabilities_test.go`
- `internal/config/agent_capabilities_test.go`
- `internal/session/manager_test.go`
- `internal/session/manager_integration_test.go`
- `internal/network/manager_test.go`
- `internal/network/manager_integration_test.go`
- `internal/network/router_test.go`
- `internal/network/router_integration_test.go`
- `internal/network/peer_test.go`
- `internal/api/core/network_test.go`

## Execution Order

1. Run the smoke lane. Stop immediately if any smoke case fails.
2. Run remaining P0 cases in the targeted lane.
3. Run P1 cases in the targeted lane.
4. Run the full lane only after smoke and targeted evidence are captured.
5. After the last execution-time change, rerun `make verify` before closing `task_07`.

## Smoke Lane

Expected duration: 15-30 minutes.

| Order | Case | Priority | Reason | Required evidence |
| --- | --- | --- | --- | --- |
| 1 | `TC-INT-001` | P0 | Confirms the primary single-file authoring path still works. | Runtime loader output with non-nil catalog. |
| 2 | `TC-INT-005` | P0 | Confirms AGH still rejects unsupported mixed layouts and formats. | Hard validation error naming conflicting files. |
| 3 | `TC-INT-008` | P0 | Confirms capability catalogs survive session-to-network join plumbing. | Join payload evidence on create/resume. |
| 4 | `TC-INT-009` | P0 | Confirms brief discovery still appears on the wire and in API payloads. | `greet`, peer listing/detail, and API payload comparison. |
| 5 | `TC-INT-010` | P0 | Confirms explicit rich `whois` full and filtered discovery still works. | One full-catalog response and one filtered response. |
| 6 | `TC-INT-011` | P0 | Confirms empty/no-catalog semantics remain deterministic. | Empty join slice plus explicit empty rich catalog response. |
| 7 | `TC-INT-013` | P0 | Confirms oversized rich responses are rejected before publish. | `ErrEnvelopeTooLarge` or equivalent guard with zero publish evidence. |

Smoke stop condition:

- Stop the lane on any failure above.
- Do not continue to targeted or full execution until the failure is diagnosed and fixed.

## Targeted Lane

Expected duration: 30-60 minutes.

Run P0 items first, then P1 items.

### P0 Order

| Order | Case | Why it stays P0 outside smoke |
| --- | --- | --- |
| 1 | `TC-INT-007` | Duplicate IDs can poison runtime discovery even when layouts are otherwise valid. |
| 2 | `TC-INT-008` | Join plumbing is the runtime bridge to all discovery surfaces. |
| 3 | `TC-INT-009` | Brief discovery is operator-visible on multiple surfaces and must not drift. |
| 4 | `TC-INT-010` | Rich discovery is the explicit user-facing contract for full capability details. |
| 5 | `TC-INT-011` | Empty-catalog behavior must remain deterministic for both runtime and protocol surfaces. |
| 6 | `TC-INT-013` | Envelope-size safety is a release-blocking guardrail. |

### P1 Order

| Order | Case | Why it is P1 |
| --- | --- | --- |
| 1 | `TC-INT-002` | Valid JSON catalog support is required but not the primary smoke path. |
| 2 | `TC-INT-003` | Directory TOML is important secondary coverage. |
| 3 | `TC-INT-004` | Directory JSON is important secondary coverage. |
| 4 | `TC-INT-006` | Basename mismatch is a specific authoring validation seam. |
| 5 | `TC-INT-012` | Unknown-ID filtering must be correct but is not a baseline availability outage. |
| 6 | `TC-FUNC-014` | Docs drift does not crash runtime flows but does affect operator correctness. |

## Full Lane

Expected duration: 2-4 hours, typically tied to release or final signoff.

Run every case in this order:

1. `TC-INT-001`
2. `TC-INT-002`
3. `TC-INT-003`
4. `TC-INT-004`
5. `TC-INT-005`
6. `TC-INT-006`
7. `TC-INT-007`
8. `TC-INT-008`
9. `TC-INT-009`
10. `TC-INT-010`
11. `TC-INT-011`
12. `TC-INT-012`
13. `TC-INT-013`
14. `TC-FUNC-014`

## Pass / Fail Criteria

PASS:

- All P0 cases pass.
- At least 90% of P1 cases pass.
- No critical or high-severity open bug remains.
- Final `make verify` passes after the last fix.

FAIL:

- Any P0 case fails.
- A rich `whois` response is emitted without an explicit include request.
- A `whois` response exceeds the envelope limit or an oversized response is published.
- A no-catalog peer returns `nil`/missing behavior where the TechSpec requires deterministic empties.
- Final `make verify` fails.

CONDITIONAL:

- One or more P1 cases fail but a documented workaround, issue file, and fix plan exist.

## Issue Handling

- Create `qa/issues/BUG-*.md` only for execution-time discrepancies.
- Every bug must link the originating `TC-*` case.
- Keep screenshots optional and only add them when they materially improve diagnosis.

## Handoff Contract

- `task_07` must write `qa/verification-report.md` under the same root.
- Do not relocate any artifact directory.
- If a new regression case is added during execution, append it without renumbering existing case IDs.
