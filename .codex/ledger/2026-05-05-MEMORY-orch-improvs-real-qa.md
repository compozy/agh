Goal (incl. success criteria):

- Run a fresh `real-scenario-qa` pass for `.compozy/tasks/orch-improvs`.
- Success requires: fresh QA bootstrap lab, Claude Opus/compozy-generated broad QA matrix, real CLI/API/Web/provider-backed-or-blocked evidence, scenario contract/charter/journey log/provider attempt populated, issues filed and root-caused, final auditor and verification report recorded.

Constraints/Assumptions:

- Conversation in Brazilian Portuguese; persistent artifacts in English.
- No destructive git commands without explicit user permission.
- Do not treat mock/unit-only evidence as final proof.
- Use a fresh lab for this independent QA pass unless continuing with an exact manifest.
- Provider-backed behavior must run with real credentials when reachable; missing credentials/tools are BLOCKED evidence, not PASS.
- Browser validation must use `browser-use:browser` first when available, then `agent-browser` fallback only if setup fails.
- Claude Opus-generated charter is planning evidence; the local auditor requires `behavioral-scenario-charter.yaml` to contain JSON text, so the execution charter may need format normalization without changing scenario intent.
- Existing `.compozy/tasks/orch-improvs/qa` artifacts are prior evidence, not sufficient for this requested fresh real-scenario pass.

Key decisions:

- Scenario slug: `orch-improvs-real-qa`.
- Treat `.compozy/tasks/orch-improvs/state.yaml` as authoritative workflow state: tasks 01-32 completed, QA report/execution marked true, next loop phase is CodeRabbit review.
- Reuse prior QA artifacts as input/context only; bootstrap a new isolated QA lab for fresh evidence.
- Claude Opus via `compozy exec` generated the broad QA scenario/test matrix requested by the user before execution.

State:

- Active goal created for validating `.compozy/tasks/orch-improvs` with real-world QA.
- Skills loaded/used so far: `real-scenario-qa`, `agh-qa-bootstrap`, `qa-report`, `qa-execution`, `systematic-debugging`, `no-workarounds`, `compozy`, `codex-loop`, and browser skill preflight pending full read before browser work.
- Worktree was clean when checked.
- Fresh QA lab bootstrapped with `REUSED_LAB=false`.
- Bootstrap manifest: `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/bootstrap-manifest.json`.
- QA output path: `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts`.
- AGH home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-95f1820f24c1/runtime`.
- AGH API target: `http://127.0.0.1:58676`.
- Browser mode: `browser-use`.
- Baseline `make verify` completed with exit code 0 before scenario mutation.
- Isolated daemon is running on `http://127.0.0.1:58676` with PID 66643 and network listener `127.0.0.1:59599`.
- Runtime health checks pass through `/api/daemon/status`, `/api/network/status`, `/api/observe/health`, and CLI `network status`.
- Claude Opus planning completed and produced `qa/test-plans/orch-improvs-opus-expanded-real-scenario-plan.md`, 20 `qa/test-cases/*.md`, `qa/edge-case-matrix.md`, `qa/execution-matrix.json`, and `qa/provider-probe.md`.
- Live provider-backed AGH session `sess-ef8ddec0727a7d5f` completed through the `qa` agent and wrote `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/artifacts/provider-launch-review.md`.
- QA initially suspected a profile-enforcement bug, but inspection showed `agh task run claim <run-id>` is an operator UDS surface and claimed as `human/local-user`; env `AGH_SESSION_ID`/`AGH_AGENT` is intentionally ignored there.
- Correct agent-facing profile proof now uses `agh task next`: ops returned `claimed=false`, backend claimed `run-e84c1722123fe71a`, completed it, and review `review-ff49a79dd87ab98c` was requested.
- Claude Opus reviewer session discovered the reviewer-native tool projection but its model-callable registry path was policy-denied for the handmade lab reviewer agent; it submitted a real operator-scoped rejection through `agh task review submit`, and the daemon queued continuation run `run-01433b131ede2d67`.
- Lab reviewer was updated with `agh__autonomy`/`agh__tasks`; new reviewer session `sess-0813e6baf4a615c7` became eligible for `agh__task_run_review_submit` and approved `review-d1354eea8ffd171a` as `agent_session`.
- Web validation used Playwright fallback because Browser plugin Node REPL tools were unavailable; Web UI and proxy SSE evidence were captured.
- Final report verdict is BLOCKED for unproven external Slack delivery/cursor advancement due no live Slack credentials, while core runtime/provider/review/Web/SSE local behavior passed.
- Final `make verify` passed and strict QA auditor passed.
- Mandatory continuation hook rejected completion because Slack terminal delivery/cursor advancement was not proven.
- Continuation retried the Slack lane in the same lab:
  - enabled bridge `brg-7b3e5ec6ee5cc52e`;
  - patched non-secret local webhook config (`127.0.0.1:59601`, `/slack/orch-improvs-real-qa`);
  - restarted the bridge;
  - observed status `auth_required` with degradation `slack: bot_token secret binding is required`;
  - confirmed bridge secret bindings and `bridges` vault namespace are empty;
  - confirmed credential-name scans found no usable Slack live credentials in operator/bootstrap env;
  - confirmed task notification cursor for `subscription-bridge-task-terminal-prime-agent` stayed at `last_sequence: 0`.
- Added `bridge-attempt.json`, updated `provider-attempt.json` boundary, verification report, behavioral charter, journey log, and audit markdown so structural PASS cannot mask behavioral BLOCKED.
- Strict QA auditor passed structurally after the continuation; markdown summary now states Structural PASS and Behavioral BLOCKED / PAUSED.
- Post-boundary `make verify` passed with 339 Vitest files / 2206 tests, `golangci-lint` 0 issues, Go race gate `DONE 8290 tests in 11.621s`, and boundaries OK.
- Claude Opus review via `compozy exec --prompt-file` succeeded as run `exec-20260506-010505-000000000`; normalized summary says `verdict=BLOCKED` and `can_mark_goal_complete=false`.
- Second continuation hook repeated the same missing Slack live lane. No usable Slack credential env vars were present.
- Addressed reviewer nits that do not require secrets:
  - removed the empty `Behavioral Evidence` heading from `verification-report.md`;
  - reconciled bridge delivery defaults to `thread_id=launch-agent`;
  - captured `cli-bridge-update-delivery-defaults-launch-agent.json`, `cli-bridge-get-after-delivery-defaults-launch-agent.json`, and `cli-notification-list-prime-agent-after-defaults-reconcile.json`;
  - re-ran strict auditor; structural PASS remains, behavioral summary remains BLOCKED / PAUSED.
- Third continuation hook repeated the same missing Slack live lane and requested failure/retry coverage after credentials exist.
- Added `.compozy/tasks/orch-improvs/qa/pending-blockers.md` with sanitized unblock prerequisites and rerun steps.
- Updated `.compozy/tasks/orch-improvs/state.yaml` via `.agents/skills/cy-codex-loop/scripts/update-state.py` (not by hand) with iteration 58, outcome `blocked`, and the Slack bridge blocker.
- Updated QA lab reports to lead with behavioral BLOCKED, document target reconciliation, and explicitly mark the failure/retry invariant UNPROVEN until Slack auth succeeds.
- Post-state-blocker `make verify` passed with 339 Vitest files / 2206 tests, `golangci-lint` 0 issues, Go race gate `DONE 8290 tests in 11.542s`, and boundaries OK.
- External Slack boundary was revalidated across multiple continuation hooks with the same result: no usable Slack env secrets, bridge `brg-7b3e5ec6ee5cc52e` `auth_required`, secret bindings `[]`, `bridges` vault `[]`, and Slack cursor `last_sequence: 0`.
- Operator re-scoped the QA with: `não temos slack para testar, pule isso`. Slack is now `skipped_by_operator`, not passed.
- While executing the replacement bridge lane, discovered and fixed a real daemon bug: terminal task events were not wired to `TerminalTaskNotifier.DeliverDue`.
- Re-scoped bridge primitive was proven with the Telegram bridge adapter in the same lab:
  - bridge `brg-068c83126dbe2010` ready;
  - task `task-telegram-delivery-qa-r2`, run `run-74ea41b1513bcc00`;
  - subscription `subscription-telegram-terminal-delivery-qa-r2`;
  - provider `sendMessage` accepted by local Telegram API;
  - durable cursor advanced from `last_sequence=0` to `last_sequence=68`.
- Failure probe with Telegram forced failure mode kept cursor at `last_sequence=0` and recorded `last_error`; same-subscription automatic retry was not promoted to a completion claim.
- `state.yaml` was updated via the official codex-loop updater to iteration 59 with `progress.deliverables_complete=true` for the re-scoped run.
- Final re-scoped `make verify` passed: Bun lint/format 0 warnings/0 errors, Vitest 339 files / 2206 tests, `golangci-lint` 0 issues, Go race gate `DONE 8295 tests in 96.258s`, boundaries OK.
- Final strict QA auditor passed after the report update; behavioral `PASS (RE-SCOPED)` was reattached to `qa-audit-report.md/json` because the auditor emits a structural summary by default.

Done:

- Read related ledgers: current `orch-improvs` implementation ledger and prior TechSpec ledger.
- Read `internal/CLAUDE.md` and `web/CLAUDE.md` for runtime/Web rules.
- Read relevant QA and debugging skill instructions.
- Confirmed `.compozy/tasks/orch-improvs/state.yaml` reports 32/32 tasks completed and `make verify` last PASS.
- Confirmed prior QA artifacts exist under `.compozy/tasks/orch-improvs/qa`.
- Ran `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario "orch-improvs-real-qa" --repo-root .`.
- Ran baseline `make verify`; key evidence: Bun/Vitest 339 files / 2206 tests, `golangci-lint` 0 issues, Go race gate `DONE 8290 tests`, build and boundaries OK.
- Wrote baseline section to `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/verification-report.md`.
- Ran isolated `agh install --provider codex --model gpt-5.4`; created `general` agent under the QA `AGH_HOME`.
- Started isolated AGH daemon and recorded CLI/API readiness in the journey log.
- Generated broad QA planning artifacts with Claude Opus via `compozy exec`.
- Completed live provider-backed session `sess-ef8ddec0727a7d5f`; evidence is in `qa/provider-session-prompt.jsonl` and `artifacts/provider-launch-review.md`.
- Created the real task tree/profile and installed the packaged Slack bridge provider in the isolated lab.
- Attempted negative explicit run claim with agent env; reclassified as invalid test design because the command uses operator identity. Need use `agh task next` or native agent tools for profile eligibility.
- Proved profile eligibility through `task next`: ops was not assigned; backend claimed and completed the run.
- Recorded a rejected review on `run-e84c1722123fe71a`; continuation run `run-01433b131ede2d67` was created with parent metadata.
- Completed continuation run evidence and native reviewer approval for round 2.
- Captured Web orchestration surface screenshot/JSON and SSE replay after sequence 34.
- Added extra channel run coverage for `marketing`, `ops`, and `leadership`.
- Wrote provider attempt, JSON-compatible charter, journey-log events, and final verification report.
- Ran strict QA auditor successfully.
- Ran final `make verify`; exit code 0 with 339 Vitest files / 2206 tests, `golangci-lint` 0 issues, Go race gate `DONE 8290 tests`, and boundaries OK.
- Continued after hook rejection and proved the exact live Slack boundary through public CLI/API surfaces.
- Captured new bridge evidence:
  - `bridge-attempt.json`
  - `cli-bridge-enable-slack-boundary.json`
  - `api-bridge-update-slack-local-webhook-boundary.json`
  - `cli-bridge-restart-slack-local-webhook-boundary.json`
  - `cli-bridge-get-after-local-webhook-settled.json`
  - `api-bridge-get-after-local-webhook-settled.json`
  - `cli-bridge-secret-bindings-after-local-webhook-settled.json`
  - `cli-vault-list-bridges-after-enable-boundary.json`
  - `cli-notification-list-prime-agent-after-local-webhook-settled.json`
  - `env-slack-credential-name-scan.txt`
  - `bootstrap-env-slack-credential-name-scan.txt`
  - `lab-slack-credential-path-scan.txt`
- Re-ran strict QA auditor after artifact updates; exit code 0.
- Re-ran `make verify` after the continuation; exit code 0.
- Ran Claude Opus boundary review with a single prompt source and captured normalized summary.
- Reconciled Slack target defaults to avoid a spurious `launch` vs `launch-agent` mismatch; status remains `auth_required` and cursor remains `last_sequence: 0`.
- Re-ran strict QA auditor after target reconciliation; exit code 0.
- Created in-tree blocker marker and recorded blocked iteration in codex-loop state using the official updater.
- Re-ran `make verify` after the in-tree blocker/state update; exit code 0.
- Revalidated the Slack credential boundary after the next continuation hook; no new credentials, bindings, or vault entries are available.
- Revalidated the Slack credential boundary again after the seventh continuation hook; no new credentials, bindings, vault entries, or cursor advancement are available.
- Latest verify after re-scope: `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/post-rescope-make-verify-r3.log`, exit code 0.

Now:

- Final response must report PASS (RE-SCOPED): Slack skipped by operator and remains unvalidated; the bridge terminal-delivery/cursor-advance primitive passed through Telegram.

Next:

- Stop the auxiliary Telegram mock server session before final response.
- Report QA bootstrap block, code fix, evidence paths, and final `make verify` result.

Open questions (UNCONFIRMED if needed):

- None for the re-scoped QA. Future Slack-specific validation remains out of scope until Slack credentials and a real target exist.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-05-MEMORY-orch-improvs-real-qa.md`
- `.compozy/tasks/orch-improvs/`
- `.compozy/tasks/orch-improvs/state.yaml`
- `.compozy/tasks/orch-improvs/qa/`
- `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py`
- `.agents/skills/real-scenario-qa/scripts/audit-qa-evidence.py`
- `.agents/skills/real-scenario-qa/scripts/record-scenario-action.py`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/bootstrap-manifest.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/bootstrap.env`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/scenario-contract.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/behavioral-scenario-charter.yaml`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/journey-log.jsonl`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/provider-attempt.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/verification-report.md`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/cli-install.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/daemon-start.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/api-daemon-status.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/api-network-status.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/api-observe-health.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/cli-network-status.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/native-review-submit-prime-agent-r2.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/web-orchestration-surface.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/web-orchestration-surface.png`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/web-sse-replay-after-34.txt`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/final-make-verify.log`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/post-boundary-make-verify.log`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/post-state-blocker-make-verify.log`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/qa-audit-report.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/qa-audit-report.md`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/bridge-attempt.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/cli-bridge-update-delivery-defaults-launch-agent.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/cli-bridge-get-after-delivery-defaults-launch-agent.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/cli-notification-list-prime-agent-after-defaults-reconcile.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/claude-opus-slack-boundary-review.json`
- `/Users/pedronauck/dev/qa-labs/agh-orch-improvs-real-qa-20260505-235414-523045-lab/qa-artifacts/qa/claude-opus-slack-boundary-review-summary.json`
- `.compozy/tasks/orch-improvs/qa/pending-blockers.md`
