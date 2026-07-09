---
name: real-scenario-qa
description: Runs release-grade AGH QA by selecting a startup playbook, materializing it into an isolated lab, posting one in-persona operator kickoff, observing the AGH runtime under autonomous agent collaboration, and auditing the produced deliverables (TSX pages, scripts, services, runbooks) plus collaboration loops. The QA observer never instructs agents about QA. Delegates planning/execution mechanics to the qa-report + qa-execution pair; findings land in the repo's living docs/qa tree (BUG-NNNN registry, dated reports). Use when validating AGH releases or complex integration features. Do not use for smoke-only checks, static planning, mock-only tests, simple unit-test edits, or architecture brainstorming without execution.
trigger: explicit
argument-hint: "[playbook-ref]"
---

# Real Scenario QA

Execute release-grade QA by running an entire fictional startup project on the AGH runtime and observing the result. The runtime drives the work; the observer never tells agents they are being evaluated. The auditor enforces real deliverables (compiled/parsed/runnable artifacts) and real collaboration (peer messages, review cycles, disagreement resolution).

The skill rejects any prompt that frames the work as QA. See `references/forbidden-prompt-phrases.md`.

## Required Inputs

- **playbook-ref** (optional): Slug of the playbook to run (e.g., `northstar-pay`, `devtool-oss-launch`, `consumer-saas-growth`). When omitted, rotate from the previous run's `PLAYBOOK_REF` recorded in `bootstrap-manifest.json`.

## Procedures

**Step 1: Select the Playbook**

1. Read `references/playbooks/README.md` and the `references/scenario-matrix.md` selection table.
2. Resolve the playbook ref:
   - If the user supplied a slug, validate it exists at `references/playbooks/<slug>.md`.
   - Otherwise, list `references/playbooks/*.md` (excluding `README`) and rotate from the previous `PLAYBOOK_REF`.
3. Record `PLAYBOOK_REF`.

**Step 2: Bootstrap the Lab With the Playbook**

1. Run the bootstrap helper (bootstrap, mutating):
   `python3 .agents/skills/agh/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<playbook-ref>" --playbook "$PLAYBOOK_REF" --repo-root .`
2. Only when continuing the same active QA session/loop and a manifest is already known, reuse:
   `python3 .agents/skills/agh/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "<playbook-ref>" --playbook "$PLAYBOOK_REF" --repo-root . --reuse-manifest "<manifest-path>"`
3. Read the helper output and record the env block from the bootstrap contract: `SCENARIO_SLUG`, `WORKSPACE_PATH`, `QA_OUTPUT_PATH`, `BOOTSTRAP_MANIFEST`, `BOOTSTRAP_ENV`, `AGH_HOME`, `AGH_HTTP_PORT`, `AGH_UDS_PATH`, `TMUX_BRIDGE_SOCKET`, `AGH_WEB_API_PROXY_TARGET`, `PROVIDER_HOME`, `PROVIDER_CODEX_HOME`, `BROWSER_MODE`, `BROWSER_BLOCKER`, `SCENARIO_CONTRACT`, `BEHAVIORAL_CHARTER`, `JOURNEY_LOG`, `PROVIDER_ATTEMPT`, `AUDIT_COMMAND`, `REUSED_LAB`, `PLAYBOOK_REF`, `KICKOFF_POSTED`.
4. Confirm the bootstrap created `<WORKSPACE_PATH>/.agh/playbook.json`, `<WORKSPACE_PATH>/.agh/agents/*.json`, `<WORKSPACE_PATH>/.agh/tasks/open-tasks.json`, and the knowledge files under `<WORKSPACE_PATH>/knowledge/`.
5. Confirm `<QA_OUTPUT_PATH>/qa/behavioral-scenario-charter.yaml` is materialized from the playbook (no UNFILLED placeholders) and includes `playbook_ref`, `required_deliverables`, `required_collaboration`.

**Step 3: Activate Companion Skills**

1. Use `qa-report` with `qa-docs-path=docs/qa` to plan the playbook-product validation as session charters (persona + journey + tour + time-box on the playbook deliverables — never on QA itself). `QA_OUTPUT_PATH` remains the lab-side scratch root (journey log, observation, kickoff evidence); the living QA state lives in the repo's `docs/qa/`.
2. Use `qa-execution` with `qa-docs-path=docs/qa` to run those sessions against the lab (does the TSX page render? do scripts run? does the canary control respond?), driving the lab's base URL/daemon from the bootstrap env block.
3. Use `agh-worktree-isolation` only when concurrency was explicitly signaled by the user.
4. Use `systematic-debugging` and `no-workarounds` for any unexpected runtime behavior the observer captures.
5. Provider home policy stays the same as the bootstrap contract: bound-secret/brokered lanes use `HOME="$PROVIDER_HOME" CODEX_HOME="$PROVIDER_CODEX_HOME"`; native_cli with `home_policy=operator` preserves the operator HOME.
6. Web flows must export `AGH_WEB_API_PROXY_TARGET="$AGH_WEB_API_PROXY_TARGET"` before launching the dev server.

**Step 4: Post the Operator Kickoff**

1. Render and validate the kickoff with the helper (mutating):
   `python3 .agents/skills/real-scenario-qa/scripts/post-operator-kickoff.py --workspace "$WORKSPACE_PATH" --playbook "$PLAYBOOK_REF" --qa-output-path "$QA_OUTPUT_PATH" --manifest "$BOOTSTRAP_MANIFEST"`
2. The helper aborts with exit code 2 if the rendered kickoff contains any phrase from `references/forbidden-prompt-phrases.md`. Do not edit the helper to suppress the check; rewrite the playbook's `kickoff_brief` instead.
3. Read `<WORKSPACE_PATH>/.agh/operator-kickoff.txt` for inspection. Use the same text verbatim when the AGH CLI is invoked to deliver the kickoff to the operator session:
   `agh session prompt <operator-session-id> "$(cat $WORKSPACE_PATH/.agh/operator-kickoff.txt)" -o jsonl > $QA_OUTPUT_PATH/qa/operator-kickoff.jsonl`
4. Confirm the manifest now reports `KICKOFF_POSTED=true` and `KICKOFF_TIMESTAMP` is set.
5. From this point on, the QA observer must not send any further prompt to any agent under test. If an agent stalls, file a bug — do not patch over the stall with a prompt.

**Step 5: Observe the Runtime**

1. Run the observer (read-only) for the configured window:
   `python3 .agents/skills/real-scenario-qa/scripts/observe-runtime.py --workspace "$WORKSPACE_PATH" --qa-output-path "$QA_OUTPUT_PATH" --duration-sec 1800 --stall-threshold-sec 300`
2. While the observer is tailing the journey log, capture cross-surface evidence WITHOUT directing agents:
   - CLI: `agh task list`, `agh agent list`, `agh channel list`, `agh session list` against the isolated daemon.
   - API: read endpoints that intersect the playbook's primary domain.
   - Web: open the AGH web app via `browser-use:browser` (or the `agent-browser` fallback) against `$AGH_WEB_API_PROXY_TARGET`. Capture DOM snapshot, URL, screenshot.
   - Runtime: confirm the journey-log keeps growing.
3. When the observer reports stall (exit code 1), open `<QA_OUTPUT_PATH>/qa/observation-summary.json`, identify the silent agent / unstarted task, and proceed to Step 6 with that diagnosis. Do not attempt to "wake" the agent with a prompt.
4. When the observer completes cleanly (exit code 0), proceed to Step 6.

**Step 6: Audit, Diagnose, Fix, Re-Verify**

1. Run the strict auditor against the lab:
   `python3 .agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py --qa-output-path "$QA_OUTPUT_PATH" --strict`
2. Auditor exit code 2 is a blocking failure. Read `qa-audit-report.json` and act per check. All bugs go to the repo's global registry `docs/qa/bugs/BUG-NNNN.md` (dedup against the registry first, per `qa-report`'s bug-registry rules) and are linked into the affected `docs/qa/state.csv` rows:
   - **C15** forbidden phrase in a prompt → rewrite the playbook source (system_prompt or kickoff_brief), not the auditor or the regex list.
   - **C16** deliverable count short → file a runtime bug (which AGH agent failed to produce the artifact, why, what state shows the failure). Do not author the missing artifact yourself — the runtime is what's under test.
   - **C17** collaboration loop short → file a runtime bug describing which channel, agent, or review cycle did not complete. Cite journey-log timestamps.
   - **C18** stall → the registry bug is mandatory and must name the silent agent and stalled task.
3. If the bug is a real AGH runtime defect (channel delivery failed, task scheduler stuck, hook misfired), fix it at the root cause in production code with regression coverage. Do not patch the playbook to dodge the bug.
4. If the bug is a playbook authoring mistake (impossible task, missing knowledge file, ambiguous handoff), fix the playbook .md, regenerate the bootstrap, rerun from Step 2.
5. Re-run the auditor after every fix.
6. Re-run the broadest verification gate (`make verify` or repository equivalent) after the last code change.
7. Write the observer report as a dated run report at `docs/qa/reports/<YYYY-MM-DD>-<playbook-ref>.md` using `docs/qa/templates/report.md`, extended with the scenario sections: playbook_compliance counts, collaboration counts, stall diagnosis, cross-surface evidence (CLI/API/Web), the audit verdict, and lab-side evidence indexed by path (journey-log, observation-summary, kickoff jsonl). Update the affected `docs/qa/state.csv` rows.
8. Append the machine-readable QA bootstrap block from `.agents/skills/agh/agh-qa-bootstrap/references/bootstrap-contract.md` so timed-loop continuations can reuse the lab.

**Step 7: Tear Down the Lab (MANDATORY)**

1. After the report is written and evidence is indexed by path, run the teardown recorded in the bootstrap env block: `eval "$TEARDOWN_COMMAND"`. This applies on every terminal verdict — PASS, FAIL, or BLOCKED.
2. Cite `<QA_OUTPUT_PATH>/qa/teardown.json` (`"clean": true`) in the final summary. Survivors (exit 1) are a blocking failure.
3. Only exception: an explicitly continuing timed loop keeps the lab alive; the continuation that ends the loop inherits the teardown obligation. A stalled or aborted run tears down like any other — the stall evidence lives in files, not in live processes.

## Error Handling

- If `bootstrap-qa-env.py` fails to load the playbook, validate the playbook .md against `references/playbook-schema.json` and re-run. Do not bypass the playbook by falling back to the legacy skeleton charter.
- If the kickoff helper aborts on a forbidden phrase, rewrite the playbook's `kickoff_brief`. Do not edit `references/forbidden-prompt-phrases.md` to remove the rule.
- If `observe-runtime.py` reports a stall, do NOT inject a prompt to wake the agent. The runtime stall IS the bug under test. File it in `docs/qa/bugs/` against the AGH runtime.
- If a required deliverable type cannot be parsed by the auditor (e.g., a TSX file with non-standard exports), fix the artifact in the workspace via the agent that authored it (re-prompting in-persona is fine; new operator prompts are not). If the agent cannot fix it, that is a runtime bug.
- If `browser-use:browser` is unavailable, follow the `agent-browser` fallback per the bootstrap browser policy. Do not silently drop the Web surface.
- If providers are unreachable, record the boundary in `provider-attempt.json`. The run verdict becomes BLOCKED, never PASS.
- If the auditor's `playbook_compliance` block reports zero counts despite agents working, confirm `WORKSPACE_PATH/.agh/playbook.json` exists and `journey-log.jsonl` is being written. Empty counts often mean the runtime is not wired to the journey log — that is a runtime bug.
