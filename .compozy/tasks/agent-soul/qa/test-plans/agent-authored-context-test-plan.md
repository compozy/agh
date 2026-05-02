# Agent Authored Context QA Test Plan

## Executive Summary

This plan validates the Agent Soul plus Agent Heartbeat MVP as a real operator would use it: managed authoring through public surfaces, deterministic diagnostics, session health visibility, advisory wake behavior, and cross-surface truth. Smoke checks are entry criteria only; completion evidence must come from CLI/API/runtime journeys that create real authored artifacts and inspect persisted state.

## Objectives

- Prove `SOUL.md` and `HEARTBEAT.md` are optional authored-context artifacts managed through AGH public surfaces.
- Prove mutations use the managed services, body-level CAS (`expected_digest` or the documented CLI alias), revision history, and deterministic diagnostics.
- Prove Soul and Heartbeat stay independent: invalid Soul fails closed for persona/session behavior without hiding Heartbeat diagnostics, and invalid Heartbeat disables wake behavior without dropping Soul read models.
- Prove session health is metadata-only, operator-readable, and consumed by Heartbeat wake/status without creating task runs or lease ownership.
- Prove a real session/agent journey either reflects Soul-authored persona in provider-backed output or records the exact isolated-provider boundary and validates every reachable runtime projection.
- Prove rollback, delete, restart recovery, and redaction edge cases preserve operator trust and do not leak absolute paths, claim tokens, or raw secret material.
- Prove MVP does not expose a fake Web editor or web-only implementation path; generated contracts and docs remain truthful.

## Scope

In scope:

- Repository verification and codegen readiness.
- Fresh isolated AGH runtime and scenario workspace.
- CLI and HTTP/API flows for Soul, Heartbeat, session health, and wake status.
- Runtime persistence evidence under isolated `AGH_HOME`.
- Provider-backed agent behavior when local credentials and provider homes are reachable; otherwise the exact provider boundary is recorded and all local runtime surfaces are still validated.
- Regression checks for generated OpenAPI/TypeScript contracts, CLI docs, and web guard tests.

Out of scope:

- Web UI editors for Soul or Heartbeat; MVP explicitly excludes them.
- Network-wide Soul propagation.
- Heartbeat-owned task queues, recurring work execution, or task-run lease replacement.
- Compatibility with old local state.

## Behavioral Scenario Charter

- Startup situation: a small startup uses AGH to operate a launch-review workspace with differentiated reviewer and ops agents.
- Operator intent: author persona and wake-policy files, inspect them through public surfaces, start or inspect session health, and request a manual advisory wake only when the runtime says a session is eligible.
- Expected business outcome: the operator can safely manage authored context without direct file mutation, understand why wake is allowed or skipped, and trust that Soul, Heartbeat, session health, tasks, and Web contracts do not drift.
- AGH surfaces used: CLI, HTTP API, generated contracts/tests, runtime persistence, and docs guard tests. UDS parity is covered by existing integration tests in `make verify` unless a live UDS helper is available during execution.
- Real provider/LLM expectation: run at least one provider-backed session when the isolated provider home has usable credentials and drivers; otherwise document the missing provider/tool boundary and validate every reachable runtime boundary.

## Test Strategy

1. Smoke readiness: repository contract discovery, generated contract drift checks, fresh bootstrap, daemon readiness.
2. P0 behavioral journeys: managed Soul authoring, Heartbeat policy/status/wake, session health, provider-backed or explicitly blocked agent output, and fail-closed invalid authored content.
3. P1 regression: HTTP/CLI CAS parity, rollback/delete/history, restart recovery, unsupported `If-Match` headers, generated TypeScript/Web guard tests, docs command truth, and redaction/security output checks.
4. Disruption probes: stale CAS mutation, invalid forbidden fields, invalid Heartbeat policy, wake against ineligible or absent session, rollback to older revision, delete with stale digest, daemon restart, and browser/Web editor absence.
5. Final gate: `make verify`, then rerun the highest-risk CLI/API journeys and persist verification evidence.

## Environment Requirements

- macOS or Linux shell with Go, Bun, Python 3, Make, and SQLite available.
- Fresh AGH QA bootstrap manifest from `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py`.
- Unique `AGH_HOME`, HTTP port, UDS path, provider home, and Codex home from the bootstrap manifest.
- Optional provider credentials in the isolated provider home for live agent/LLM proof.
- Browser automation is not required for authored-context MVP because there is no Web editor, but generated Web contract guard tests must run.

## Entry Criteria

- Root repository instructions, PRD docs, TechSpecs, ADRs, and workflow memory have been read.
- `qa/test-cases/` and `qa/test-plans/` exist and contain this plan plus executable cases.
- Bootstrap manifest exists for the current QA run.
- Smoke readiness checks either pass or blocking failures are documented with root-cause analysis.

## Exit Criteria

- All P0 cases pass or have fixed bug reports with rerun evidence.
- At least 90% of P1 cases pass; any blocked P1 has an exact prerequisite or environment boundary.
- No Critical or High open bugs remain.
- The final `make verify` run passes with zero warnings or failures.
- `.compozy/tasks/agent-soul/qa/verification-report.md` includes command evidence, behavioral evidence, test-case coverage, issues, and the QA bootstrap block.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Missing provider credentials block live LLM proof | Medium | High | Use isolated provider env; document exact boundary; validate CLI/API/runtime state instead. |
| CLI/API/UDS response shapes drift | Medium | High | Run generated contract tests, codegen check, CLI integration tests, and direct HTTP/CLI parity probes. |
| Heartbeat wake accidentally behaves like a work queue | Low | Critical | Validate wake audit/status and run task boundary tests in `make verify`; inspect no task run is created by wake flows. |
| Direct file mutation bypasses managed authoring in tests | Medium | Medium | Use direct file writes only to create invalid preconditions when testing validation; mutations under test must use CLI/API. |
| Web contracts suggest unsupported UI controls | Low | Medium | Run Web/SDK guard tests and inspect generated DTOs; no Web editor flow is counted as product proof. |

## Deliverables

- Test plan: `qa/test-plans/agent-authored-context-test-plan.md`
- Regression suite: `qa/test-plans/agent-authored-context-regression-suite.md`
- Test cases: `qa/test-cases/SMOKE-001.md`, `qa/test-cases/TC-SCEN-001.md`, `qa/test-cases/TC-SCEN-002.md`, `qa/test-cases/TC-SCEN-003.md`, `qa/test-cases/TC-REG-001.md`, `qa/test-cases/TC-REG-002.md`, `qa/test-cases/TC-REG-003.md`, `qa/test-cases/TC-REG-004.md`
- Execution evidence: `qa/evidence/`
- Issue reports: `qa/issues/BUG-*.md`
- Final report: `qa/verification-report.md`
