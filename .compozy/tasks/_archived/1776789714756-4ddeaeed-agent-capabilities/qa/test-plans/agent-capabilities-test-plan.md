# Agent Capabilities QA Test Plan

- Feature: Agent Capabilities
- Task: `task_06`
- Generated: `2026-04-19`
- `qa-output-path`: `.compozy/tasks/agent-capabilities`
- Planned executor: `task_07` via `qa-execution`

## Executive Summary

This plan defines the execution-ready QA scope for the agent-capabilities feature without running the flows in `task_06`.

Objectives:

- Prove explicit local capability catalogs are the only source of truth for discovery.
- Prove loader behavior across every supported authoring layout and every required rejection path.
- Prove the loaded catalog survives runtime/session join plumbing and powers both brief and rich network discovery.
- Prove API-visible payloads and operator-facing documentation remain aligned with the implemented runtime and RFC contracts.
- Give `task_07` a stable artifact set under `.compozy/tasks/agent-capabilities/qa/` so execution can start without re-deciding scope, priorities, or file locations.

Key risks:

- Loader mode drift could silently merge unsupported file and directory layouts.
- Runtime join plumbing could regress to `nil` or stale capability payloads during create/resume.
- Brief discovery could drift between `peer_card.capabilities`, `agh.capabilities_brief`, and API payloads.
- Rich `whois` discovery could leak full catalogs into ordinary responses or emit invalid oversized envelopes.
- Runtime docs and RFC wording could diverge from the shipped wire keys and local layout rules.

## Scope Definition

### In Scope

- Loader correctness for `capabilities.toml`, `capabilities.json`, `capabilities/*.toml`, and `capabilities/*.json`.
- Hard validation failures for mixed layout, mixed format, basename mismatch, duplicate IDs, and missing required fields where those checks are user-visible.
- Session-to-network join plumbing for capability-aware local peer registration.
- Brief discovery through `greet`, peer registry/listing, and API payload conversion.
- Explicit rich `whois` discovery for full-catalog, filtered-catalog, no-catalog, unknown-ID, and oversized-response scenarios.
- Documentation consistency across `docs/rfcs/005_capability-catalogs-agent-directories.md`, RFC 003, and the shipped runtime behavior from tasks 01-05.

### Out of Scope

- Implementing new runtime behavior or fixing regressions found during execution.
- Running live validation flows in `task_06`.
- Browser or Figma validation; this feature is backend/runtime-first and only requires API-visible evidence.
- Non-capability network features outside the loader, join, discovery, and API surfaces named in the TechSpec.

## Test Strategy And Approach

- Treat this artifact set as planning only. `task_07` owns live execution, bug filing, screenshots, and the final `verification-report.md`.
- Keep `qa-output-path=.compozy/tasks/agent-capabilities` unchanged across planning and execution.
- Use real runtime seams during execution: filesystem-backed agent directories, session activation, manager/router flows, and API payload conversion.
- Use existing package-level regression anchors to accelerate diagnosis, but do not accept isolated unit output as the only proof for runtime or protocol claims.
- Execute in this order during `task_07`:
  1. Baseline repository health.
  2. Smoke lane.
  3. Targeted lane.
  4. Full lane.
  5. Final repository verification after the last change.

### Evidence Expectations

| Surface | Required evidence in `task_07` |
| --- | --- |
| Daemon/runtime loader | Real temp agent directories showing success or hard validation failure through `LoadAgentDefFile` / `LoadWorkspaceAgentDefs` or equivalent runtime path. |
| Session/runtime join | Join payload or manager evidence proving capability-aware local peer registration on create/resume, including deterministic empty slices for no-catalog peers. |
| Router-level envelopes | Captured `greet` and `whois` request/response behavior showing brief metadata, explicit rich discovery, filtering, no-catalog/unknown-ID empty catalogs, and oversized-response rejection. |
| API payload visibility | `internal/api/core` payloads or equivalent handler evidence showing the same brief metadata visible after runtime/router flows. |
| Documentation consistency | Fresh comparison against `docs/rfcs/005_capability-catalogs-agent-directories.md` and RFC 003 so user-visible layout rules and wire keys match implementation exactly. |

## Environment Requirements

| Area | Requirement |
| --- | --- |
| OS | Local development host plus CI-parity Go environment. Record the actual OS in the execution report. |
| Browser/API client | Browser not required. CLI output, `go test`, or HTTP/API payload capture is sufficient. |
| Devices | Not applicable for this backend capability feature. |
| Repository gate | `make verify` before completion. |
| Focused packages | `./internal/config`, `./internal/session`, `./internal/network`, `./internal/api/core`. |
| Fixtures | Temporary agent directories containing `AGENT.md` and capability sidecars/catalogs. |
| Transport/runtime harness | Existing session, manager, router, and API test harnesses already used by tasks 01-04. |

## Traceability Matrix

| Seam | Priority | Source of truth | Planned cases | Primary evidence surfaces |
| --- | --- | --- | --- | --- |
| Single-file loader success | P0/P1 | Task 01, TechSpec "Testing Approach", ADR-002 | `TC-INT-001`, `TC-INT-002` | Runtime loader, workspace discovery |
| Directory-mode loader success | P1 | Task 01, TechSpec validation rules, ADR-002 | `TC-INT-003`, `TC-INT-004` | Runtime loader, workspace discovery |
| Mixed-layout and mixed-format rejection | P0 | Task 01, ADR-002 | `TC-INT-005` | Runtime validation errors |
| Basename mismatch rejection | P1 | Task 01, TechSpec validation rules | `TC-INT-006` | Runtime validation errors |
| Duplicate ID rejection | P0 | Task 01, ADR-003 normalization semantics | `TC-INT-007` | Runtime validation errors |
| Join plumbing | P0 | Task 02 | `TC-INT-008` | Session lifecycle, network manager |
| Brief peer-card projection | P0 | Task 03, TechSpec projection rules, RFC 003 brief extension | `TC-INT-009` | `greet`, peer list/detail, API payloads |
| Rich `whois` full and filtered discovery | P0 | Task 04, RFC 003 rich discovery extension | `TC-INT-010` | Router request/response envelopes |
| No-catalog behavior | P0 | Tasks 01-04, TechSpec projection rules | `TC-INT-011` | Join payload, brief omission, explicit empty rich catalog |
| Unknown-ID rich discovery | P1 | Task 04, RFC 003 rich discovery rules | `TC-INT-012` | Router response envelope |
| Oversized-response guard | P0 | Task 04, TechSpec envelope-size guard | `TC-INT-013` | Router rejection path, zero publish evidence |
| Documentation consistency | P1 | Task 05, RFC 003, runtime guide | `TC-FUNC-014` | Docs and key-string comparison |

## Entry Criteria

- Tasks 01-05 are already completed and remain the source of truth for execution scope.
- All planning artifacts live under `.compozy/tasks/agent-capabilities/qa/`.
- The executor has read this plan, the regression suite, and all P0/P1 test cases before running validation.
- `task_07` uses the same `qa-output-path` and does not relocate artifacts.
- Any environment gaps are recorded before smoke execution begins.

## Exit Criteria

- Every required artifact exists under `.compozy/tasks/agent-capabilities/qa/`.
- Every required seam from `task_06` has at least one traceable manual test case.
- Smoke, targeted, and full lanes are documented with explicit P0/P1 ordering.
- All P0 cases pass during `task_07`.
- At least 90% of P1 cases pass, or any exceptions have a documented bug, workaround, and fix plan.
- No critical or high-severity unresolved capability regression remains open.
- Final `make verify` passes after the last execution-time fix.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Mixed layout or mixed format accidentally merges instead of failing hard | Medium | High | Keep `TC-INT-005` in smoke; require conflicting-file evidence and stop on failure. |
| No-catalog peers regress from deterministic empty slices to `nil` handling | Medium | High | Keep `TC-INT-011` as P0 and verify both join and explicit rich discovery behavior. |
| Brief projection drifts between runtime/router/API surfaces | Medium | High | Require `TC-INT-009` to compare `peer_card.capabilities`, `agh.capabilities_brief`, and API payloads from the same peer. |
| Rich discovery leaks into ordinary `whois` or overflows the envelope | Low | Critical | Run `TC-INT-010` and `TC-INT-013` in smoke; block the lane immediately on failure. |
| Docs or RFC strings drift from shipped keys | Medium | Medium | Include `TC-FUNC-014` in targeted and full lanes before closing the task. |

## Timeline And Deliverables

### Phase 1: Baseline

- Read the plan, regression suite, and P0/P1 cases.
- Capture baseline repository health and environment details.

### Phase 2: Runtime And Protocol Execution

- Execute loader success and rejection cases.
- Execute join plumbing, brief discovery, and rich discovery cases.
- Capture runtime, router, and API evidence under the shared `qa/` root.

### Phase 3: Closeout

- File `BUG-*` artifacts for any discrepancies found during execution.
- Update or add narrow regression coverage for every discovered bug.
- Produce `.compozy/tasks/agent-capabilities/qa/verification-report.md`.
- Rerun the final repository gate.

Deliverables created by this task:

- `qa/test-plans/agent-capabilities-test-plan.md`
- `qa/test-plans/agent-capabilities-regression.md`
- `qa/test-cases/TC-*.md`
- reserved `qa/issues/` and `qa/screenshots/` paths for `task_07`

## Handoff Notes For Task 07

- Do not change the artifact root or filenames unless a blocking discrepancy is found and documented.
- Use the regression suite ordering as the default execution order.
- Create bug reports only when execution reveals an actual discrepancy; planning found none.
- Keep live evidence under this same `qa/` tree so the task history remains local and compact.
