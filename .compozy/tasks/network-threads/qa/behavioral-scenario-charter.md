# Network Threads QA Behavioral Scenario Charter

## Run

- Run ID: `20260505T170658Z-execution`
- QA output path: `.compozy/tasks/network-threads`
- Bootstrap manifest: `.compozy/tasks/network-threads/qa/bootstrap-manifest.json`
- Lab workspace: `/Users/pedronauck/dev/qa-labs/agh-network-threads-20260505-170603-687358-lab`
- Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-0517b5b397c4/runtime`
- API base URL: `http://127.0.0.1:60149`

## Startup Situation

A fresh isolated AGH QA lab exercises the Network Threads hard cut from flat
channel timelines to explicit public thread and restricted direct-room
containers. The `builders` channel is the scenario coordination space.

## Operator Intent

The operator coordinates launch-review work publicly, moves detailed review work
into a restricted direct room, summarizes the outcome back to the public thread,
and verifies that CLI, API, Web, and runtime evidence describe the same
conversation state.

## Business Outcome

The operator can tell which state is public, which state is restricted to a
direct room, which work item belongs to which conversation container, and whether
legacy `interaction_id` / peer-room behavior is rejected on active surfaces.

## Agent Cast

| Actor | Role | Expected behavior |
| --- | --- | --- |
| Operator | Scenario driver | Creates, queries, and compares thread/direct state across public surfaces. |
| Requester agent | Public initiator | Starts the review request in the public thread. |
| Reviewer agent | Direct-room worker | Performs or records restricted review work in one deterministic direct room. |
| QA observer | Cross-surface verifier | Compares CLI/API/Web/runtime state and records defects with evidence. |

## Provider Plan

Provider-backed AGH session execution will be attempted only when local provider
tools and credentials are reachable from the bootstrap manifest environment. If
the provider boundary is blocked, the report will name the exact missing
credential/tool/account boundary and keep local harness evidence separate from
live LLM proof.

## Execution Matrix

| ID | Priority | Scope | Evidence target |
| --- | --- | --- | --- |
| SMOKE-001 | P0 | Readiness, broad gate, docs/contracts scan | Run log and scan output. |
| TC-SCEN-001 | P0 | Public thread coordination | CLI/API/Web/runtime thread evidence or exact CLI/API blocker. |
| TC-SCEN-002 | P0 | Restricted direct-room handoff | Direct resolve/send/isolation evidence or exact CLI/API blocker. |
| TC-SCEN-003 | P0 | Summarize-back | Public summary and direct-room isolation evidence or exact blocker. |
| TC-INT-001 | P0 | Direct-room resolve race | Runtime E2E harness and direct-room evidence. |
| TC-UI-001 | P1 | Web route and browser artifacts | Browser screenshots using `browser-use` or documented fallback. |
| TC-REG-001 | P1 | Legacy hard-cut guardrails | Rejection output and active-docs scan. |

## Disruption Probes

- Restart or rerun runtime harness and confirm thread/direct state remains coherent.
- Resolve the same direct room concurrently and confirm one deterministic `direct_id`.
- Submit legacy fields or stale flags and confirm deterministic rejection.
- Navigate to missing thread/direct routes and confirm operator-readable states.
