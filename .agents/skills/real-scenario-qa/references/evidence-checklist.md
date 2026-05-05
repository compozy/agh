# Evidence Checklist

Use this checklist before claiming a realistic scenario is complete.

## Behavioral Anti-Smoke Gates

- A Behavioral Scenario Charter exists and names the operator intent, startup situation, expected business outcome, agent roles, live provider plan, and realistic disruption probes.
- `<QA_OUTPUT_PATH>/qa/scenario-contract.json` exists and its minimums are treated as blockers, not suggestions.
- `<QA_OUTPUT_PATH>/qa/behavioral-scenario-charter.yaml` is filled with JSON-compatible YAML, not freeform Markdown.
- `<QA_OUTPUT_PATH>/qa/journey-log.jsonl` contains structured rows for meaningful CLI/API/Web/runtime/provider actions.
- `<QA_OUTPUT_PATH>/qa/provider-attempt.json` records live provider proof or a concrete blocked boundary.
- At least one complete operator journey was executed from setup through outcome, not only isolated commands or endpoint checks.
- Smoke checks, `make verify`, unit/integration tests, CRUD-only checks, and page-render checks are reported as readiness or regression evidence only, not as final behavioral proof.
- The scenario includes at least one realistic disruption that a user would care about, such as wrong agent ownership, missed handoff, incoherent artifact, stale operator view, failed automation side effect, interrupted session, restart recovery, or confusing history.
- The final claim explains why the tested behavior matters to an operator or agent, not only which technical path executed.
- The strict auditor passed. If it reported blockers, the QA result is `FAIL` or `BLOCKED`, not `PASS`.

## Baseline and Environment

- Canonical verification gate was discovered from repository files.
- Baseline gate was run before scenario mutation or runtime stress.
- Scenario workspace path is outside unrelated user work.
- Daemon/API/Web services were started through supported commands.
- Health/readiness checks succeeded before interaction.
- `browser-use:browser` was loaded and used for Web validation when the Browser plugin was available.
- If browser-use was unavailable after setup, `agent-browser` was used as the approved Web fallback.
- Any Web fallback names the failed browser-use prerequisite and includes the exact `agent-browser` commands or captured evidence.

## Realistic Data

- Agents were created or selected from real project configuration.
- Agent roles map to the charter and have differentiated responsibilities.
- Channels represent real company areas or product functions.
- Tasks, subtasks, dependencies, and runs use persisted task APIs or CLI commands.
- Automations, cron jobs, and triggers create real runs and artifacts.
- Knowledge entries are written to the real workspace or configured memory store.
- Hooks and extensions produce real observable side effects when in scope.
- Generated artifacts are coherent, inspected, connected to their producing task/session/channel, and used by a later scenario step.
- The journey log proves the scenario-contract minimums for agents, channels, tasks, runs, artifact reuse, and cross-surface object overlap.

## Live Agent and LLM Behavior

- At least one provider-backed agent session was executed when credentials and local prerequisites were available.
- If provider-backed agents were unavailable, the exact credential/tool/runtime boundary is documented in `provider-attempt.json` and no mock/stub/fake reply is used as final proof. A blocked provider boundary cannot be reported as live-provider `PASS`.
- Agent outputs include meaningful decisions, revisions, handoffs, or artifacts instead of token echoes or canned text.
- Agent messages, task claims, channel participation, and generated artifacts match the intended role and workspace/channel constraints.
- Any wrong-agent, wrong-channel, hallucinated artifact, or incoherent workflow behavior is filed as a product issue, not dismissed as test noise.

## Public Surface Coverage

- CLI commands were exercised for every selected scenario track with a public CLI surface.
- Web UI was exercised through `browser-use:browser` for every selected operator-facing track with a Web surface, or through `agent-browser` only when browser-use was unavailable.
- CLI and Web evidence cover at least one overlapping persisted object, such as the same task, automation run, channel, message, knowledge entry, hook result, extension state, or generated artifact.
- HTTP/API endpoints were exercised for at least one core workflow when the product exposes public API behavior.
- Browser evidence includes the tool used, final URL, and a DOM snapshot or screenshot for each high-risk flow.
- Browser evidence proves the operator can understand and act on real scenario state, not only that a route rendered.
- Persistence was inspected through supported APIs or direct DB reads when needed for debugging.
- Logs, health payloads, and generated artifacts were captured as evidence.

## Failure Handling

- Every discovered issue has a reproducible command or browser flow.
- Issues that affect both CLI and Web include reproduction steps for both surfaces.
- Every bug report includes expected behavior, actual behavior, impact, and evidence.
- Fixes target production behavior, not test expectations.
- Regression coverage proves the bug and protects the intended invariant.
- Full verification gate was rerun after the last code change.
- The affected behavioral journey was replayed after a fix, including live provider-backed agent behavior when it was part of the original failure and remains reachable.

## Release Readiness

- No critical bugs remain open.
- No active sessions, stuck task runs, runaway cron jobs, or unhealthy memory state remain unless intentionally left for a soak.
- Operator-facing history is understandable and not dominated by protocol noise.
- Browser-use limitations, `agent-browser` fallback usage or failure, missing credentials, and external blockers are named explicitly.
- Release readiness is not claimed unless the behavioral journey, live agent/provider evidence when reachable, CLI evidence, and Web browser evidence pass through browser-use or the approved `agent-browser` fallback, or the Web surface is proven out-of-scope.
- Release readiness is not claimed unless `qa-audit-report.json` reports no blockers.
