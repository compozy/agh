# Orchestration Improvements QA Behavioral Scenario Charter

## Scenario

- Slug: `orch-improvs-qa-20260505-223520-643030`
- Lab root: `/Users/pedronauck/Dev/compozy/agh/.tmp/qa-labs/agh-orch-improvs-qa-20260505-223520-643030-lab`
- Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-e81f17828a60/runtime`
- Daemon API target: `http://127.0.0.1:63022`
- QA artifact root: `.compozy/tasks/orch-improvs/qa`

## Startup Situation

The simulated operator is validating the orchestration-improvements release before enabling it for
agent-managed task work. The lab is isolated from the operator's default AGH state, uses a unique
daemon port and UDS path, and starts with a clean runtime home. The existing repository QA plan
from task 31 supplies the execution matrix.

## Operator Intent

The operator needs to prove that AGH can configure task execution profiles, route post-terminal
review requests, bind reviewer authority, continue rejected work with redacted guidance, expose
notification cursor diagnostics, and present the same task truth across CLI/API/UDS/native-tool
surfaces, web UI, generated docs, and final verification gates.

## Expected Business Outcome

The release can only proceed if:

- Public runtime surfaces agree on the same durable task/profile/review/notification state.
- Web UI surfaces are truthful and actionable for operators.
- Review verdict authority stays bound to reviewer sessions and explicit transports.
- Raw claim tokens do not leak through context bundles, task streams, web, or docs examples.
- Bridge notification cursors advance only after confirmed accepted-final delivery.
- Full repository verification and e2e gates pass after the scenario.

## Agent Roles

- Operator: creates the QA lab, runs CLI/API/UDS checks, inspects web state, and reviews evidence.
- Worker agent: claims and completes task runs under execution profile constraints.
- Reviewer agent: receives a bound review request and records verdicts through reviewer authority.
- Coordinator agent: starts or coordinates the task lifecycle and verifies continuation guidance.
- Bridge consumer: receives terminal accepted-final delivery and exposes cursor diagnostics.

## Live Provider Plan

Provider-backed or native CLI agent execution is attempted when local credentials and provider
prerequisites are available in this lab. If no provider-backed lane is reachable, the report must
name the exact boundary and cannot treat deterministic or mock-backed checks as live LLM evidence.
Reachable public runtime boundaries still must be validated with real persisted state.

## High-Risk Journeys

- TC-SCEN-001: full profile -> worker -> review rejection -> continuation -> approval -> bridge
  notification lifecycle.
- TC-INT-001: config, schema migration, and execution profile parity.
- TC-INT-002: review gate contract, reviewer binding, and continuation authority.
- TC-INT-003: notification cursor and bridge delivery semantics.
- TC-UI-001: web Orchestration tab operator truth.
- TC-SEC-001: claim-token redaction and reviewer boundary.
- TC-REG-001: generated contracts, CLI references, site docs, and memory drift.
- TC-PERF-001: SSE resume, named events, query churn, and cursor replay.

## Realistic Disruption Probes

- Restart daemon between rejected review persistence and continuation inspection.
- Attempt unbound or wrong-actor review verdict submission.
- Attempt active-run profile mutation while `current_run_id` is set.
- Force or simulate failed bridge delivery before successful cursor advancement evidence.
- Reconnect task stream with conflicting header/query seeds and malformed seed input.
- Compare docs/generated contract claims against live CLI/API/Web output.

## Smoke Checks

Smoke checks are readiness-only evidence:

- `make verify`
- `make codegen-check`
- site source/content/typecheck/test/build
- daemon health check
- web route render
- unit and integration package tests

