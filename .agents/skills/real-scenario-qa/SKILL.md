---
name: real-scenario-qa
description: Runs behavior-first release and feature QA by bootstrapping an AGH startup lab, proving realistic operator journeys, live provider-backed agent/LLM behavior when reachable, persisted artifacts, CLI/API/Web parity, and root-cause fixes. Use when validating AGH releases or complex integration features through real scenarios. Do not use for smoke-only checks, static planning, mock-only tests, simple unit-test edits, or architecture brainstorming without execution.
trigger: explicit
argument-hint: "[scope-or-context]"
---

# Real Scenario QA

Execute release-grade QA by simulating a real startup operating on AGH. Validate user/operator outcomes, live agent behavior, real persisted artifacts, public CLI/API/Web surfaces, automations, tasks, networks, knowledge, hooks, extensions, and final verification gates.

Smoke checks, CRUD-only checks, page-render checks, unit tests, integration tests, and mock-backed sessions are entry criteria only. They never satisfy this skill by themselves.

## Required Inputs

- **scope-or-context** (optional): Short description of the release, branch, feature, or focus area under test. Examples: `release-candidate`, `autonomy-feature`, `network-tasks`, `cron-triggers-knowledge`. Use it to name the scenario, prioritize flows, and explain why specific surfaces were stressed.

## Procedures

**Step 1: Resolve Scope and Bootstrap the Lab**

1. Parse the optional `scope-or-context`. When omitted, use `release-candidate`.
2. Activate `agh-qa-bootstrap` first and execute the repo-root helper for a fresh lab:
   `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<scope-or-context>" --repo-root .`
3. Only when the current active QA session or loop continuation already provides an exact manifest path for this same run, reuse that lab instead:
   `python3 .agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<scope-or-context>" --repo-root . --reuse-manifest "<manifest-path>"`
4. Read the helper output and record:
   - `SCENARIO_SLUG`
   - `WORKSPACE_PATH`
   - `QA_OUTPUT_PATH`
   - `BOOTSTRAP_MANIFEST`
   - `BOOTSTRAP_ENV`
   - `AGH_HOME`
   - `AGH_HTTP_PORT`
   - `AGH_WEB_API_PROXY_TARGET`
   - `PROVIDER_HOME`
   - `PROVIDER_CODEX_HOME`
   - `BROWSER_MODE`
   - `BROWSER_BLOCKER`
   - `REUSED_LAB`
5. Open `<QA_OUTPUT_PATH>/qa/bootstrap-manifest.json` and treat it as the canonical handoff for every downstream command in this scenario.
6. For a brand-new QA invocation, prefer a fresh lab even if an older lab exists for the same feature. Reuse is only for the same active QA session or loop continuation when `--reuse-manifest` was passed intentionally.
7. Store all QA artifacts under `<QA_OUTPUT_PATH>/qa/`.
8. If the repository has a stricter artifact convention, keep the generated workspace but mirror the final report into the repository convention.

**Step 2: Activate Companion QA and Debugging Skills**

1. Use `qa-report` when planning test cases or documenting issues.
2. Use `qa-execution` when running gates, starting services, exercising CLI/API/Web flows, and writing the verification report.
3. Use `systematic-debugging` and `no-workarounds` for every unexpected behavior, failed test, flaky runtime, memory spike, bad UX, or integration issue.
4. When starting provider-backed commands, follow the provider home policy:
   - Bound-secret, brokered, or explicitly isolated-home lanes:
     `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME" <provider-command>`
   - `native_cli` providers with `home_policy=operator`: preserve the operator `HOME` / native login state unless the scenario explicitly validates isolated provider-home behavior.
5. When starting Web flows against an isolated daemon, export:
   `AGH_WEB_API_PROXY_TARGET="$AGH_WEB_API_PROXY_TARGET"`
6. Use `browser-use:browser` as the primary Web validation path when the Browser plugin is available. Read and follow that skill before any browser interaction. Use the in-app browser against the local Web app and capture DOM snapshot, screenshot, URL, and visible-state evidence for the tested flows.
7. Treat CLI validation and Web validation as separate mandatory release gates. A CLI-only, API-only, or unit-only pass is not enough to claim readiness when the Web app exists.
8. If `browser-use:browser` is unavailable after following its setup procedure, read and follow `agent-browser`, then use it as the approved fallback. Record the failed browser-use prerequisite, the agent-browser commands used, and the resulting URL/snapshot/screenshot evidence.
9. If both `browser-use:browser` and `agent-browser` are unavailable or blocked, record the browser blocker explicitly in the final report. Do not silently replace browser automation with shell-only, API-only, or fake Web checks.
10. Do not treat mocks, stubs, fake agent replies, or unit-only tests as final proof. Final proof must include real integration or end-to-end behavior whenever the surface is reachable.

**Step 3: Discover the Product Contract**

1. Read root agent instructions, build files, web instructions, and relevant package docs before running scenarios.
2. Use the bootstrap manifest as the first source of truth for daemon/API URLs, isolated runtime paths, provider env, and browser policy before rediscovering anything manually.
3. Identify the canonical verification gate, startup commands, daemon/API URLs, CLI commands, Web UI entry points, and persistence locations.
4. Run the broadest repository-defined baseline gate before scenario testing.
5. Record baseline command, timestamp, exit code, and output summary in `<QA_OUTPUT_PATH>/qa/verification-report.md`.
6. If baseline fails, root-cause and fix only when the failure is relevant or blocks realistic scenario execution. Otherwise document it as a pre-existing blocker with evidence.

**Step 4: Write the Behavioral Scenario Charter**

1. Before creating scenario data, write `<QA_OUTPUT_PATH>/qa/behavioral-scenario-charter.md`.
2. Define the real-world startup situation being simulated, the operator intent, and the business outcome that must be true when the scenario succeeds.
3. Define the human/operator journey in concrete terms: what the operator is trying to accomplish, which AGH surfaces they use, what they need to understand, and which persisted objects prove progress.
4. Define the agent cast and responsibilities. Include at least four differentiated agents for broad release validation, and include the changed-feature agent roles for feature-focused validation.
5. Define the expected agent behavior: decisions to make, artifacts to create or revise, messages to exchange, task/channel constraints to respect, and handoffs to complete.
6. Define the live LLM/provider plan. Execute at least one provider-backed agent session unless credentials or local prerequisites are unavailable. If unavailable, record the exact boundary and validate every reachable runtime surface instead.
7. Define realistic disruption probes that matter to a user, such as wrong agent ownership, missed handoff, incoherent artifact, stale operator view, failed automation side effect, interrupted session, restart recovery, or confusing history.
8. Mark smoke checks separately as readiness checks. Do not count them as behavioral evidence.

**Step 5: Build a Realistic Startup Scenario**

1. Read `references/scenario-matrix.md` and select the scenario tracks that match `scope-or-context`.
2. Create a startup-like workspace under `WORKSPACE_PATH` with real directories for company, product, marketing, finance, operations, reviews, and QA artifacts.
3. Configure realistic agents from the charter, such as founder, CTO, backend, frontend, marketing, finance, review, QA, ops, and operator agents. Use real project configuration or real AGH provider-backed sessions when reachable.
4. Create channels that represent company areas, such as leadership, development, marketing, finance, operations, review, and launch coordination.
5. Add realistic custom skills, hooks, extensions, automations, cron jobs, webhook triggers, knowledge/memory entries, and tasks/subtasks when those surfaces exist.
6. Make the scenario produce coherent artifacts that a startup would actually use, such as strategy notes, launch plans, rollback plans, campaign copy, frontend pages, backend service stubs that run, review notes, task evidence, automation outputs, and QA reports.
7. Verify artifacts are not placeholders: inspect their content, connect them to the task/channel/session that produced them, and use at least one artifact in a later scenario step.

**Step 6: Execute Real CLI, API, Web, and Agent Flows**

1. Drive all setup and operations through public CLI, HTTP/API, Web UI, or documented daemon interfaces.
2. Exercise the changed feature inside the behavioral charter first, then exercise adjacent integrations that consume, display, or depend on the same state.
3. Execute at least one complete operator journey from setup through outcome. A journey must include actor intent, command/browser/API actions, agent work, persisted state, and final operator-facing understanding.
4. Execute at least one complete CLI workflow for each selected scenario track, using real commands and persisted state.
5. Execute at least one matching Web workflow through `browser-use:browser` for each operator-facing selected scenario track, using the same or directly related persisted state created by the CLI/runtime flow. If browser-use is unavailable, execute the same Web workflow through `agent-browser`.
6. Compare CLI/API/runtime state against Web rendering for at least one core workflow. The same task, automation run, channel, message, knowledge entry, hook result, extension state, or generated artifact must be visible and correct across surfaces when the product exposes it.
7. For live agent scenarios, prompt provider-backed agents to perform real work, then verify their messages, decisions, artifacts, and task/channel behavior through AGH state. Do not accept token echo, canned text, or fake provider output as final proof.
8. For network scenarios, require multiple agents to join channels, exchange messages, reply, hand off work, coordinate around tasks, and avoid cross-channel or wrong-agent ownership leaks.
9. For task scenarios, create root tasks, subtasks, dependencies, runs, claims, starts, completions, failures, retries, and task-linked sessions that support the charter outcome.
10. For automation scenarios, validate manual runs, scheduled cron/every jobs, webhook triggers, run history, retry/fire-limit behavior, and resulting artifacts as part of a user-visible workflow.
11. For knowledge scenarios, write, search, list, open, and use knowledge entries from real workspace state in a later agent or operator decision.
12. For Web scenarios, validate whether the operator can understand and act on real state: default states, navigation, detail pages, filters/toggles, real data rendering, error states, stale or historical data, and generated artifacts in the browser, not only through API responses.
13. Execute the realistic disruption probes from the charter and record whether the product behavior remains correct or fails with actionable evidence.
14. Read `references/evidence-checklist.md` before marking a scenario complete.

**Step 7: Diagnose, File, Fix, and Re-Verify Issues**

1. Reproduce every issue with the narrowest real command or Web flow before editing code.
2. Write an issue under `<QA_OUTPUT_PATH>/qa/issues/BUG-<num>.md` using `assets/scenario-issue-template.md`.
3. Fix the production code, configuration, or runtime contract at the root cause. Do not patch symptoms or weaken tests.
4. Add focused regression coverage for the bug at the correct layer.
5. Re-run the narrow reproduction, impacted scenario, and relevant package tests.
6. Continue the realistic scenario after the fix; do not stop at the first green unit test.

**Step 8: Validate Final Release Readiness**

1. Re-run the full canonical verification gate from scratch after the last code change.
2. Re-run the highest-risk behavioral journey against the current build, including live agent/provider behavior when it was reachable before the fix.
3. Re-run both the CLI and Web browser versions of the highest-risk workflow after the last code change.
4. Confirm no active sessions, stuck runs, unhealthy memory state, scheduler failures, or runaway persisted data remain unless the scenario intentionally leaves them running.
5. Write the final report using `assets/final-report-template.md`.
6. Include pass/fail status for every selected scenario track from `references/scenario-matrix.md`.
7. Include all blocked validations with exact environment or tool failure details, including browser-use setup failures and any `agent-browser` fallback usage or failure.
8. Append the machine-readable QA bootstrap block from `.agents/skills/agh-qa-bootstrap/references/bootstrap-contract.md` so timed-loop continuations can reuse the healthy lab.
9. Do not claim release readiness unless the full gate, the CLI evidence, the browser-based Web evidence, and the behavioral journey evidence are fresh for the current state.

## Error Handling

- If `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py` fails, inspect stderr, fix the missing prerequisite, and rerun it. Do not fall back to ad-hoc manual lab creation.
- If a required CLI/API/Web surface does not exist, record that as out-of-scope only after proving the repository has no supported public entry point for it.
- If `browser-use:browser` cannot access the app, diagnose setup, local URL, auth, and service health first. If browser-use is unavailable after that procedure, use `agent-browser` with `open`, `snapshot -i`, interaction commands, `get url`, and screenshots as the Web fallback. If `agent-browser` also fails, keep testing via CLI/API/runtime surfaces and document both browser limitations in the final report.
- If a live integration lacks credentials, validate every local boundary up to the credential boundary and record the exact missing prerequisite.
- If live provider-backed agents are unavailable, do not replace them with mocks as final proof. Record the provider boundary, then prove the same behavioral journey through every reachable AGH runtime surface.
- If scenario data grows unexpectedly in memory or disk, stop new load generation, inspect persistence growth, root-cause the largest writer, and fix before continuing.
- If the system produces excessive noisy operational history, distinguish protocol/audit data from operator-facing history and validate both read-side cleanup and write-side prevention.
