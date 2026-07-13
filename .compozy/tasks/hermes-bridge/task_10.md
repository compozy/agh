---
status: completed
title: Execute the bridges QA cycle (qa-execution)
type: test
complexity: medium
---

# Task 10: Execute the bridges QA cycle (qa-execution)

## Overview
Executes the QA cycle in a fresh bootstrap lab against task_09's plan; records verdicts, bugs,
and a dated report. This is the gate that catches what `make verify` cannot (SD-005) —
progress rate limits, markdown escaping, restart recovery, setup friction, and docs
truthfulness under real persona walks. Requires E2E (`requires_e2e=true`: web UI + CLI) and
mandatory teardown (L-029).

<critical>
- ALWAYS READ the PRD, the TechSpec, and their catalogs (`_user_stories.md`, `_tests.md`) before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — implement every test case assigned in ## Tests
</critical>

<requirements>
- MUST activate `agh-qa-bootstrap` + `qa-execution` + `real-scenario-qa` skills; read
  task_09's charters, journeys, and living scenario files before starting.
- MUST bootstrap a fresh lab via `agh-qa-bootstrap` (new manifest; unique `AGH_HOME`, ports,
  tmux-bridge sockets if concurrency is signaled — worktree-isolation rules apply). Web QA
  MUST export `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest — never hardcode `:2123`.
- MUST execute every task_09 charter: setup flows (manifest paste simulation, wizard `--json`
  lane, verify/send-test against fake or sandbox platform endpoints), progress rendering modes,
  chunking/markdown fidelity, restart-mid-turn recovery, inbound edit/reply context, web
  checklist walk (browser evidence).
- MUST validate the agent-manageability lane end-to-end: drive one full bridge setup using ONLY
  structured CLI/HTTP output (no web UI) per SD-011.
- MUST update the owned `docs/qa/scenarios/*.md` files with evidence-backed verdicts and register
  bugs in `docs/qa/bugs/BUG-<YYYYMMDD>-<slug>.md` after deduplicating by symptom (timestamp,
  command, observed output — forensic-first, SD-006). Never edit or commit generated `state.csv`;
  the lab holds run-scratch evidence indexed by path.
- MUST measure time-to-first-message step counts per channel and compare against the Hermes
  baselines recorded in task_09 / `_qa.md` §7.
- MUST run automated E2E lanes as preconditions/evidence: `make test-e2e-runtime` AND
  `make test-e2e-web` (`requires_e2e=true`). Browser evidence via browser-use with
  `agent-browser` fallback for the web checklist journey.
- MUST end with a dated report under `docs/qa/reports/` and the machine-readable QA bootstrap
  block (manifest path, lab root, runtime home, base URL, verification evidence).
- MUST tear down the lab on every terminal path (pass/fail/blocked/abort) via
  `eval "$TEARDOWN_COMMAND"` or `make qa-reap` (L-029); cite `teardown.json` (`"clean": true`)
  as evidence. Register long-lived lab processes at `<QA_OUTPUT_PATH>/qa/pids/<name>.pid` on
  spawn. Completing while lab daemons/tmux/dev servers/browsers remain alive is a blocking
  failure.
- MUST NOT change production code in this task; fixes discovered here go through the
  fix-loop governor (`qa-execution`) with red-before/green-after regression placement per
  `consolidate-test-suites`.
</requirements>

## Subtasks
- [x] 10.1 Fresh lab bootstrap + manifest persisted; unique home/ports; `AGH_WEB_API_PROXY_TARGET`
      derived; long-lived PIDs registered
- [x] 10.2 Charter execution across CLI/HTTP lane, channel-delivery lane, and web lane with
      browser evidence (browser-use → agent-browser fallback); agent-only structured-output
      setup (SD-011)
- [x] 10.3 Content-addressed bug registration with reproduction evidence + living-scenario
      verdict updates; TTFM step counts recorded per channel
- [x] 10.4 Dated report under `docs/qa/reports/` + machine-readable QA bootstrap block
- [x] 10.5 Mandatory teardown (L-029): `eval "$TEARDOWN_COMMAND"` or `make qa-reap`;
      `teardown.json` shows `"clean": true`

## Implementation Details
No production code changes from this task. Skills own the session protocol, evidence
standard, status enum, and fix-loop governor — follow them; `_qa.md` §8 owns lab/evidence
rules. Platform lanes default to fake/sandbox endpoints; REAL platform spot-checks are
optional and recorded as such — never block the cycle on missing credentials
(`blocked-verify` with exact human instructions). Automated E2E lanes
(`make test-e2e-runtime`, `make test-e2e-web`) are preconditions/evidence, not a substitute
for persona walks.

### Relevant Files
- `_qa.md` — QA base (lab/evidence rules §8, TTFM protocol §7, expected-bug taxonomies §6)
- `task_09.md` — the plan this task executes (journeys, charters, living scenarios)
- `docs/qa/{scenarios/,bugs/,reports/,charters/,journeys/}` — living tree write-back
- Lab scratch tree (uncommitted; indexed by path in the report)

### Dependent Files
- None (terminal task of the program graph)
- Fix-loop regressions land in owning suites via `consolidate-test-suites` if bugs are fixed
  in-cycle

### Competitor References
- Not applicable — execution against AGH's own lab.

## Deliverables
- Every task_09 charter executed (or explicitly deferred with reason) with evidence
- Updated living-scenario verdicts; deduplicated content-addressed BUG registry entries for every failure
- Dated report + QA bootstrap block
- `teardown.json` with `"clean": true` (L-029)
- Measured TTFM per channel vs Hermes baseline

## Tests

QA execution task — regression tests for found bugs land via the fix loop in their owning
suites. Gate checklist:

- Unit tests (suite: Not applicable — QA execution; no production code):
- Integration tests (suite: Not applicable — see above):
- E2E tests (lanes REQUIRED — `requires_e2e=true`):
  - [x] `make test-e2e-runtime` green (or failures filed as content-addressed bugs with evidence) — bridge
        contract / restart lane evidence
  - [x] `make test-e2e-web` green (or failures filed as content-addressed bugs with evidence) — Playwright
        bridges setup/verify/send-test flows
  - [x] Web checklist walk (`J-complete-web-bridge-setup` / `CH-web-bridge-setup`) captured with browser screenshots cited in the report
        (browser-use; `agent-browser` fallback)
- Session / charter gate (REQUIRED):
  - [x] Every task_09 charter executed to a recorded verdict, OR explicitly deferred with
        reason (no silent scope caps)
  - [x] Agent-manageability lane: one full bridge setup via structured CLI/HTTP only (SD-011)
  - [x] Restart-mid-turn recovery charter exercised (one explicit fail-open terminal error, no text replay)
  - [x] Inbound edit + reply-to context charter exercised
  - [x] Progress storm / mode-off / markdown-fidelity hunts covered per risk taxonomies
- Tracker / report gate:
  - [x] Every failure has a deduplicated `BUG-<YYYYMMDD>-<slug>.md` record with reproduction evidence (timestamp, command,
        observed output)
  - [x] Owned living scenario files updated with pass/fail/deferred verdicts and evidence
  - [x] Dated report under `docs/qa/reports/` includes per-journey verdicts and measured
        TTFM per channel vs Hermes baseline
  - [x] Machine-readable QA bootstrap block present (manifest path, lab root, runtime home,
        base URL, verification evidence)
- Teardown gate (L-029 — blocking):
  - [x] `eval "$TEARDOWN_COMMAND"` or `make qa-reap` run on the terminal path used
  - [x] `teardown.json` cites `"clean": true`; no lab daemons/tmux/dev servers/browsers left
        alive

## Success Criteria
- 100% of task_09 charters executed or explicitly deferred with reason
- Every failure has a deduplicated content-addressed bug with reproduction evidence
- Report includes measured time-to-first-message per channel vs the Hermes baseline
- `make test-e2e-runtime` and `make test-e2e-web` run as evidence; web walk has browser
  captures
- Lab torn down cleanly (L-029) — `teardown.json` `"clean": true`
