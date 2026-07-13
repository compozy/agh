# Task Memory: task_10

## Objective Snapshot

- Execute the complete Task 09 bridge QA plan through public CLI/HTTP/UDS/Web/provider-fake surfaces, record evidence-backed verdicts and TTFM, run the required runtime/Web E2E lanes, and tear down every lab process cleanly.

## Execution Contract

- Living QA tree: `docs/qa`; execution report is created before the first persona session and updated after every charter.
- Required charters: the nine content-addressed Hermes charter files from Task 09; required scenarios: `NB-024`, `NB-025`, `NB-026`, `NB-028`, `NB-029`, `NB-031`, `NB-036..NB-039`, plus the seven content-addressed Hermes scenarios.
- Agent-manageability lane: `CH-structured-telegram-setup` uses only structured CLI/HTTP/UDS output. Browser evidence is mandatory for `CH-web-bridge-setup`.
- A provider-fake/sandbox verdict qualifies the result and does not claim a real vendor account. Missing real credentials are recorded, never faked.
- No production code changes are allowed directly in Task 10. A discovered product failure enters the governed fix loop with canonical regression placement.
- Global suites and the single `make verify` remain deferred until the complete tasks/QA/review tail. Task 10 still runs its required serialized `make test-e2e-runtime` and `make test-e2e-web` lanes.

## Fresh Lab

- Playbook: `northstar-pay` (selected because the change touches network channels and dense peer messaging).
- Scenario: `hermes-bridge-task-10-20260713-022226-583543`.
- Workspace: `/home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab`.
- QA output: `/home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts`.
- Manifest: `/home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/bootstrap-manifest.json`.
- Runtime: `AGH_HOME=/tmp/aghqa-e192b01b8545/runtime`, HTTP `40645`, UDS `/tmp/aghqa-e192b01b8545/runtime/aghd.sock`, tmux socket `/tmp/aghqa-e192b01b8545/runtime/tmux-bridge.sock`.
- Web proxy: `http://127.0.0.1:40645`; browser policy: `browser-use`, no blocker.
- Provider home: `/tmp/aghqa-e192b01b8545/provider`; Codex home: `/tmp/aghqa-e192b01b8545/provider/.codex`.
- Bootstrap contract validated: fresh manifest, 11 agent specs, 12 open tasks, seven knowledge files, populated behavioral charter, no `UNFILLED` placeholders.
- Teardown command: `python3 /home/pedronauck/Projects/agh/.agents/skills/agh/agh-qa-bootstrap/scripts/teardown-qa-env.py --manifest /home/pedronauck/dev/qa-labs/agh-hermes-bridge-task-10-20260713-022226-583543-lab/qa-artifacts/qa/bootstrap-manifest.json`.

## Evidence Rules

- Public surface first; independent public read-back; refresh/deep-link persistence; checkpoints and failures captured.
- One in-persona Northstar operator kickoff, then no evaluator prompts to the agents under observation.
- Long-lived processes are registered immediately under `<QA_OUTPUT_PATH>/qa/pids/`.
- Every failure is deduplicated before creating `BUG-<YYYYMMDD>-<slug>.md`; scenario and dated-report observations update after each session. Charters remain immutable.
- Final behavior-first claim requires the strict lab auditor and `teardown.json` with `"clean": true`.

## Automated Precondition Findings

- `make test-e2e-runtime` found `BUG-20260712-goal-judge-fixture-model` and `BUG-20260712-reasoning-evidence-attribution`. The Goal fixture issue is fixed by advertising the configured judge model in its canonical acpmock fixture; exact Goal 6/6 and Goal+reasoning 14/14 pass under race. Reasoning evidence attribution remains open: background ACP processes share one per-agent diagnostics file and process-local provider-session IDs collide, so the full reasoning lane cannot attribute records structurally. Isolated stress passes 80/80; a third patch is forbidden by the two-touch rule.
- The first `make test-e2e-web` passed 62/70 in 16.8 minutes but is invalid current-product evidence. BUG-0037 proved the lane reused a stale `web/dist` and a reduced per-spec daemon environment discarded `AGH_WEB_DIST_DIR`; the Bridge trace sent obsolete `enabled: true` while current source says `false`.
- Existing `BUG-0037` was re-found and fixed at both owners: Mage always runs codegen-check plus Web build before an E2E lane, and the browser runtime preserves only four machine-controlled E2E variables when a spec supplies a reduced environment. Focused Mage is 6/6; runtime fixture is 17/17 via Turbo; Web typecheck and targeted Oxfmt/Oxlint pass.
- `BUG-20260712-bridge-e2e-retired-route` updated the existing Bridge Playwright owner for the current `/bridges` catalog and `/bridges/:id` detail routes, human-readable status copy, and per-route responsive assertions. No product contract was weakened: exact HTTP/UDS/CLI enums remain asserted. Current-bundle create/edit/enable/ingress passed in 13.7s; secret rotation/auth/restart recovery passed in 15.4s.
- The six non-Bridge failures from the invalid full Web run remain attached to BUG-0037 and will be reclassified only from the single fresh full lane at the workflow tail; no early global rerun is permitted.

## Manual Charter Findings

- The single Northstar kickoff reproduced open BUG-0028: six duplicate unowned tasks were added, all twelve seeded tasks stayed ready with zero runs, nine collaborator sessions remained idle, and the observer stalled. Evidence is indexed in the lab and the canonical bug verification section; do not send another kickoff.
- WhatsApp/Telegram/Discord guided setup, masking, validation, Telegram register/verify/enable/send, and HTTP/UDS parity were exercised against deterministic provider fakes. Slack manifest, verify, inbound progress/redaction, progress-off, and edit reuse were exercised with a dedicated acpmock fixture.
- `BUG-20260713-telegram-route-shapes` is open: Telegram guided setup persists required group+thread routing, which rejects the public-guide peer-only direct-message send and ordinary groups without a topic. Group+topic delivery succeeds, proving provider transport is healthy. The fix requires alternative route shapes in the routing contract.
- Browser QA used `agent-browser` after `browser-use` required an unavailable human Chrome permission. `Slack Web QA` (`brg-f10be295e2e1de65`) was created once, copied one daemon manifest, persisted bindings and enabled state across refresh, surfaced reachability remediation, resolved a dry-run target, and delivered one real fake-provider message (`del-12205c96fa3a6ab4`, remote `U_QA:1710000000.000710`). API, current-source CLI, and provider logs confirm the UI result.
- Browser screenshots: `qa/screenshots/ch055-{bridge-catalog,create-handoff,failed-remediation,refreshed-checklist,dry-run-result,send-result}.png`.

## Remaining

- Charter/TTFM evidence, living-scenario verdicts, report observations, and exact owner lanes are complete. The manifest teardown passed at 2026-07-13T05:29:36Z with `clean:true`, no survivors, and all registered processes stopped. Remaining workflow work is Task 10 bookkeeping/checkpoint followed by Phase C/D and the deferred final global gate.

## Working Set

- `.compozy/tasks/hermes-bridge/{task_10.md,state.yaml,memory/MEMORY.md,memory/task_10.md}`
- `docs/qa/{scenarios/,charters/CH-<slug>.md,reports/,bugs/,README.md}`
- Fresh lab paths listed above; bulk evidence remains lab-side and is indexed by the committed report.
- Precondition fixes: `internal/testutil/acpmock/testdata/goal_command_fixture.json`, `magefiles/{e2e.go,magefile_test.go}`, `web/e2e/{__tests__/bridges.spec.ts,fixtures/runtime.ts,fixtures/__tests__/runtime.test.ts}`.
- Bugs/report: content-addressed Hermes findings, the `BUG-0037` re-found note, and `docs/qa/reports/2026-07-12-hermes-bridge.md`.
