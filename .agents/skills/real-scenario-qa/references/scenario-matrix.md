# Scenario Matrix

Select rows based on `scope-or-context`. Always include one behavior-first operator journey, one live agent/LLM track when reachable, one cross-surface truth check, and one realistic disruption probe. Smoke checks are readiness gates only and do not count as scenario completion. When `<QA_OUTPUT_PATH>/qa/scenario-contract.json` exists, its minimums are the release gate; missing minimums mean `FAIL` or `BLOCKED`, never `PASS`.

| Track | Use When | Required Surfaces | Evidence |
|---|---|---|---|
| Operator launch day | Preparing a broad release or validating readiness | verify gate, daemon, CLI, API, Web through browser-use or agent-browser fallback, persisted tasks/channels/artifacts | launch/rollback/QA artifacts, operator-visible status, CLI/Web/API parity, final gate |
| Feature in real use | Branch introduces a complex feature | changed feature plus adjacent features that consume, display, or mutate the same state | before/after behavior, real artifacts, live scenario replay, regression tests |
| Live agent work | Provider-backed agents are reachable | sessions, prompts, task-linked sessions, channel messages, generated artifacts, persisted transcripts/events | agent decisions, non-placeholder artifacts, task/channel compliance, provider boundary notes |
| Agent collaboration | Agents coordinate work, hand off, or negotiate ownership | channels, peers, direct/say/receipt/trace messages, tasks, task runs, CLI/Web channel views | message timelines, ownership/claim evidence, handoff outcome, no wrong-agent or wrong-channel behavior |
| Task orchestration | Agents spawn/manage work or task state changed | root tasks, subtasks, dependencies, claims, starts, completions, failures, retries, task-linked sessions | task tree, lifecycle transitions, blocked/unblocked behavior, operator-readable state |
| Automation in context | Jobs, cron, triggers, hooks, or scheduler behavior changed | jobs, cron/every schedules, webhook triggers, hook runs, run history, artifacts, CLI/Web run views | user-visible side effects, generated artifacts, retry/fire-limit behavior, history clarity |
| Knowledge in context | Knowledge, memory, retrieval, or workspace context changed | write/list/search/read/reindex/consolidate flows, agent prompts, CLI/Web knowledge views | created entries, search/open evidence, later agent/operator use, stale/historical behavior |
| Hooks and extensions in context | Lifecycle hooks or extensions changed | extension install/enable/disable, hook catalog, hook runs, side effects, agent/skill/resource exposure | extension-provided capability use, hook side-effect files/logs, status visibility, failure reporting |
| Operator Web understanding | UI behavior or read model changed | `browser-use:browser` flows, or `agent-browser` only when browser-use is unavailable, navigation, filters/toggles, details, loading/error states | screenshot/DOM evidence that real state is understandable and actionable |
| Recovery and long-running behavior | Restart, interruption, stale data, memory, storage, or concurrency risk exists | daemon restart, health endpoints, DB size, process RSS, repeated runs, concurrent agents, historical views | recovery outcome, bounded growth, no stuck work, stale state handling, operator-facing history |

## Minimum Scenario Composition

For broad release validation, include:

1. Startup workspace bootstrap.
2. A Behavioral Scenario Charter with operator intent, expected outcome, agent roles, live provider plan, and realistic disruption probes.
3. At least eight agents across distinct company functions, with at least one provider-backed live agent workflow when reachable.
4. At least five channels representing company areas.
5. At least one automation job and one trigger that produce user-visible side effects.
6. At least one task tree with dependencies and multiple runs tied to the operator journey.
7. At least one knowledge entry that is written, searched, opened, and used by an agent or operator later in the scenario.
8. At least one coherent artifact produced by an agent and used in a later step, not just created.
9. At least one Web UI pass through `browser-use:browser` when the app has a Web surface, with `agent-browser` allowed only when browser-use is unavailable after setup.
10. At least three realistic disruption probes, such as wrong agent assignment, blocked dependency, failed run, invalid trigger, missed handoff, interrupted session, retry, stale channel, historical data view, or confusing operator state.
11. At least three CLI/Web/API/runtime parity checks that prove the same persisted objects or artifacts are correct across exposed surfaces.
12. A strict audit run using `.agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py --qa-output-path "$QA_OUTPUT_PATH" --strict` with no blockers.

The following do not satisfy broad release validation by themselves:

1. Running `make verify`.
2. Creating one task or channel and listing it.
3. Opening a Web page and confirming it renders.
4. Prompting an agent only to echo a token.
5. Producing placeholder artifacts that are not inspected or used.

For feature-focused validation, include the broad release baseline plus:

1. A scenario explicitly built around how a real operator or agent would use the feature.
2. At least two adjacent features that consume, display, mutate, or depend on the feature's state.
3. One historical or stale-data case when persistence is involved.
4. One real agent decision, handoff, or artifact influenced by the feature when provider-backed agents are reachable.
5. One concurrency, repeated-operation, recovery, or interruption case when orchestration is involved.
6. One final browser-use check, or `agent-browser` fallback check when browser-use is unavailable, that proves the feature's output is understandable and actionable in the operator-facing UI.
