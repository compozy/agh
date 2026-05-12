---
name: qa-execution
description: Executes behavior-first project QA by consuming a bootstrap-manifest when present, discovering the verification contract, running gates, exercising real operator journeys through CLI/API/Web, validating live provider-backed agent behavior when reachable, fixing root-cause regressions, and rerunning the full gate. Uses browser-use:browser first and agent-browser as fallback for Web UI validation. Use when validating a branch, release candidate, migration, refactor, or risky commit. Do not use for static review, smoke-only checks, one-off unit edits, or planning without execution.
argument-hint: "[qa-output-path]"
---

# Systematic Project QA

Execute QA as a real operator using the product. Smoke checks, unit tests, integration tests, CRUD-only checks, route-render checks, and mock-backed sessions are readiness or regression evidence only. They do not satisfy behavior-first QA without live or reachable end-to-end user/agent behavior.

## Required Inputs

- **qa-output-path** (optional): Directory where QA artifacts (issues, screenshots, verification reports) are stored. When provided, create the directory if it does not exist and use it for all QA outputs. When omitted, fall back to repository conventions or `/tmp/codex-qa-<slug>`.

## Procedures

**Step 1: Discover the Repository QA Contract**

1. Read root instructions, repository docs, and CI/build files before running commands.
2. Resolve the QA artifact directory. If the user provided a `qa-output-path` argument, use that path. Otherwise, use repository conventions. If neither exists, fall back to `/tmp/codex-qa-<slug>`. Create the `qa/` subdirectory under the resolved path if it does not exist. Store all issues, screenshots, and verification reports under `<qa-output-path>/qa/`.
3. Check for `<qa-output-path>/qa/bootstrap-manifest.json`. When it exists, read it first and reuse its isolated runtime paths, provider env, Web proxy target, browser policy, scenario contract, behavioral charter, journey log, provider attempt file, audit command, and any embedded `project_contract`.
4. If the manifest is absent or does not include a usable `project_contract`, execute the repo-root helper:
   `python3 .agents/skills/qa-execution/scripts/discover-project-contract.py --root .`
5. Prefer repository-defined umbrella commands such as `make verify`, `just verify`, or CI entrypoints over language-default commands.
6. Read `references/project-signals.md` when command ownership is ambiguous or when multiple ecosystems are present.
7. Identify the changed surface and the regression-critical surface before choosing scenarios.
8. Determine whether the project has a Web UI surface. When a bootstrap manifest exists, prefer its `AGH_WEB_API_PROXY_TARGET` and browser policy over hardcoded defaults. Otherwise infer the dev server URL from the discovered contract (default `http://localhost:3000` unless the project specifies otherwise).

**Step 2: Define the QA Scope**

1. Check whether `<qa-output-path>/qa/test-cases/` and `<qa-output-path>/qa/test-plans/` contain artifacts from a prior `qa-report` run. If they exist, read the test plans and test case IDs to seed the execution matrix and prioritize P0/P1 test cases.
2. If `<qa-output-path>/qa/scenario-contract.json` exists, read it before building the matrix. Treat its minimums as blocking evidence requirements, not guidance.
3. Build a short execution matrix covering baseline verification, 2-4 high-risk operator/agent journeys, changed workflows, and unchanged business-critical workflows. For release-grade AGH scenarios, the matrix must include enough journeys to satisfy the scenario contract's agent, channel, task, provider, cross-surface, artifact, and disruption minimums.
4. For each high-risk journey, define actor intent, expected business outcome, required AGH surfaces, expected agent behavior, expected artifacts, cross-surface state assertions, and one realistic disruption probe.
5. Read `references/checklist.md` and ensure every required behavioral and technical category has a planned validation.
6. Prefer public entry points such as CLI commands, HTTP endpoints, browser flows, worker jobs, provider-backed agent sessions, and documented setup commands over internal test helpers.
7. When a Web UI surface exists, read `references/web-ui-qa.md` and select 3-5 critical user flows to exercise through the browser. Prioritize flows that cover the changed surface, business-critical paths, and the same persisted objects used by CLI/API/runtime flows.
8. If the bootstrap manifest defines `BROWSER_MODE=browser-use`, keep browser-use as the default path. Use `agent-browser` only after the browser-use setup procedure fails.
9. Create the smallest realistic scenario fixture or disposable project needed to exercise the workflow when the repository does not already include one, but do not use fake provider or mock agent replies as final proof.
10. Treat mocks as a local unit-test boundary only. Do not use mocks or stubs as final proof that a user or agent flow works.

**Step 3: Establish the Baseline**

1. Install dependencies with the repository-preferred command before testing runtime flows.
2. Run the canonical verification gate once before scenario testing to establish baseline health. Execute in fastest-first order: lint and type-check, then build, then unit tests, then integration tests.
3. If the baseline fails, read the first failing output carefully and determine whether it is pre-existing or introduced by current work before moving on.
4. When the project has a Web UI surface, start the dev server in the background using the discovered start command. If the bootstrap manifest exists, export `AGH_WEB_API_PROXY_TARGET` from the manifest before launching the Web server. Confirm readiness by waiting for the server to respond (e.g., `curl -sf -o /dev/null http://localhost:<port>` returns 0, or startup logs emit a ready signal).
5. Start provider-backed services according to the provider home policy:
   - Bound-secret, brokered, or explicitly isolated-home lanes:
     `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME" <command>`
   - `native_cli` providers with `home_policy=operator`: keep the operator `HOME` / native login state unless the scenario explicitly validates isolated provider-home behavior.
6. Start services in the closest supported production-like mode and confirm readiness through observable signals such as health checks, startup logs, or successful handshakes.

**Step 4: Execute CLI and API Flows**

1. Drive CLI and API workflows through the same interfaces a real operator or user would use.
2. Capture the exact command, input, operator intent, and observable result for each scenario. Append each meaningful action to `<qa-output-path>/qa/journey-log.jsonl`; use `.agents/skills/real-scenario-qa/scripts/record-scenario-action.py` when practical.
3. Validate changed features inside the real operator/agent journey first, then validate at least one regression-critical flow outside the changed surface.
4. Exercise live integrations and provider-backed agent sessions when credentials and local prerequisites exist. Record the attempt in `<qa-output-path>/qa/provider-attempt.json`. When they do not, validate every reachable local boundary and record the blocked live step explicitly; this may support a `BLOCKED` verdict but not a live-provider `PASS`.
5. Verify agent behavior through AGH state: messages, task claims, channel participation, generated artifacts, persisted events, and any operator-visible state.
6. Keep config writes against the same isolated home strictly sequential. Never parallelize `agh config set` or other config mutations against one `PROVIDER_CODEX_HOME`.
7. Re-run the scenario from a clean state when the first attempt leaves the environment ambiguous.

**Step 5: Execute Web UI Flows**

Skip this step if the project has no Web UI surface.

1. Read `references/web-ui-qa.md` for the full browser testing procedure and checklist.
2. When a real-scenario QA playbook is active (`PLAYBOOK_REF` set in the bootstrap manifest), the **primary Web target is the playbook product** — the TSX pages and components produced by the agents under test (e.g., the Northstar Pay hero page, the Lumen Notes variant landings). Render and validate those artifacts. The AGH web UI itself is exercised separately as a cross-surface truth check, not as the user-facing target.
3. Use `browser-use:browser` first when the Browser plugin is available or when the bootstrap manifest says `BROWSER_MODE=browser-use`. Read and follow that skill before the first browser action.
4. If browser-use is unavailable after setup, use the `agent-browser` CLI as the approved fallback. The fallback core loop is: **open, snapshot, interact, re-snapshot, verify**. Valid commands are: `open`, `back`, `forward`, `reload`, `snapshot -i`, `click @ref`, `fill @ref "text"`, `select @ref "value"`, `press Key`, `check @ref`, `uncheck @ref`, `wait`, `get text @ref`, `get url`, `get title`, `screenshot`, `state save`, `state load`, `close`. Do not invent commands outside this set.
5. For each critical user flow identified in Step 2, execute the flow with browser-use first or the approved agent-browser fallback, capture URL/snapshot/screenshot evidence, and record which browser tool was used.
6. Test critical form flows: fill valid data and verify success, fill invalid data and verify error messages appear.
7. When the changed surface includes responsive behavior, test at multiple viewports. Read the viewport testing section of `references/web-ui-qa.md` for session setup.
8. Verify navigation flows: page transitions, back/forward, deep links, and 404 handling.
9. Check error and loading states: trigger error conditions and verify the UI handles them gracefully.
10. Verify the operator can understand and act on the real scenario state in the Web UI. A route rendering, list count, or empty page check is not enough.
11. Close the browser session after all fallback flows complete: `agent-browser close`.

**Step 6: Diagnose and Fix Regressions**

1. Reproduce each failure consistently before proposing a fix.
2. Activate companion debugging and test-hygiene skills when available, especially root-cause debugging and anti-workaround guidance.
3. Add or update the narrowest regression test that proves the bug when the repository supports automated coverage for that surface, after naming the invariant, owning layer, and canonical suite.
4. Fix production code or real configuration at the source of the failure. Do not weaken tests to match broken behavior.
5. Re-run the narrow reproduction, the impacted behavioral journey, and the baseline gate after each fix.
6. For Web UI regressions, reproduce the visual failure with browser-use first or the approved agent-browser fallback, capture before/after screenshots under `<qa-output-path>/qa/screenshots/`, and verify the fix through the same browser flow.
7. Use `assets/issue-template.md` to write issue files under `<qa-output-path>/qa/issues/`. Create the subdirectory if it does not exist. Name each file using the `BUG-<num>.md` convention (e.g., `BUG-001.md`). Assign Severity (Critical/High/Medium/Low) and Priority (P0-P3) to every issue. When an issue was discovered while executing a test case from `qa-report`, include the TC-ID in the Related section.

**Step 7: Verify the Final State**

1. Re-run the full repository verification gate from scratch after the last code change.
2. Re-run the most important behavioral journey after the full gate passes, including CLI/API/runtime state and live provider-backed agent behavior when it was reachable before.
3. Re-run the most important CLI and API scenarios after the full gate passes.
4. When Web UI flows were tested, re-run the critical browser flows and capture final screenshot evidence.
5. Summarize the evidence using `assets/verification-report-template.md` and write the report to `<qa-output-path>/qa/verification-report.md`. The report must include these mandatory fields: Claim, Command, Executed timestamp, Exit code, Output summary, Warnings, Errors, Verdict (PASS or FAIL). When behavior-first QA was executed, append Behavioral Evidence with: operator journey, live agent/LLM evidence or blocked provider boundary, artifacts produced and used, disruption probes, cross-surface state checks, and smoke checks separated as readiness-only evidence. When Web UI flows were tested, append a Browser Evidence section with: Browser tool used, Dev server URL, Flows tested count, per-flow entry (name, entry URL, final URL, verdict, screenshot path), Viewports tested, Authentication method, and Blocked flows.
6. Run the configured auditor when `<qa-output-path>/qa/scenario-contract.json` exists:
   `python3 .agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py --qa-output-path "<qa-output-path>" --strict`
7. Add an `Audit Result` section to `<qa-output-path>/qa/verification-report.md` with the command, exit code, `qa-audit-report.json`, blockers, warnings, and final verdict.
8. Treat auditor exit code `2` as blocking. Do not claim behavior-first QA completion when the auditor reports missing release-grade minimums.
9. Report blocked scenarios, missing credentials, or environment gaps with the exact command or prerequisite that stopped execution.
10. Append the machine-readable QA bootstrap block from `.agents/skills/agh-qa-bootstrap/references/bootstrap-contract.md` when a healthy reusable lab remains after the run.
11. Do not claim completion without fresh verification evidence from the current state of the repository.
12. Do not claim behavior-first QA completion when the final evidence is only smoke, unit/integration, CRUD, mock, fake provider, or page-render evidence.

## Error Handling

- If command discovery returns multiple plausible gates, prefer the broadest repository-defined command and explain the tie-breaker.
- If no canonical verify command exists, read `references/project-signals.md`, choose the broadest safe install, lint, test, and build commands for the detected ecosystem, and state that assumption explicitly.
- If a required live dependency is unavailable, validate every local boundary that does not require the missing dependency and report the blocked live validation separately.
- If a workflow requires data or services absent from the repository, create the smallest realistic fixture outside the main source tree unless the repository has its own fixture convention.
- If a failure appears unrelated to the requested change, prove that with a clean reproduction before excluding it from the QA scope.
- If live provider-backed agents are unavailable, document the exact provider, credential, or tool boundary. Continue with reachable runtime surfaces, but do not describe that as live agent/LLM proof.
- If browser-use fails, record the failed prerequisite and retry the flow once with `agent-browser`. If `agent-browser` is also unavailable or the dev server fails to start, document the blocker in the verification report and continue with CLI and API validation only.
- If a browser fallback flow hangs or times out, close the session with `agent-browser close`, record the failure, and attempt the flow once more from a clean session before marking it as blocked.
