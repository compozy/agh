Goal (incl. success criteria):

- Run a release-grade real-scenario QA pass for `.compozy/tasks/mem-v2`.
- Use `real-scenario-qa` and the QA bootstrap flow, with real CLI/API/Web/runtime evidence.
- Per latest user instruction, cover both `consumer-saas-growth` and `northstar-pay` playbooks for a faithful QA spread.
- Latest user requirement: live LLM validation must use Claude Code as ACP because agent memory behavior is critical.
- Delegate the QA report authoring portion to Claude Code Opus through `compozy`, per user request.
- Success means the auditor verdict, runtime observations, provider boundary, and bootstrap continuation block are recorded; real AGH defects are fixed with verification if found.
- New user correction: do not stop at reporting QA errors; fix the `$qa-execution` errors/root-cause runtime defects and re-verify.

Constraints/Assumptions:

- Conversation in Brazilian Portuguese; artifacts in English.
- Always use `/Users/pedronauck/.codex/RTK.md`; shell commands must be prefixed with `rtk`.
- Never run destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) without explicit user permission.
- After operator kickoff, do not send further prompts to agents under test; stalls are bugs.
- Claude Code ACP must be attempted as a real provider-backed path. If unavailable/uncredentialed, record exact boundary in `provider-attempt.json` and verdict cannot be live-provider PASS.
- `make verify` is the required final gate after code changes; if no code changes, record applicable QA verification instead.
- New independent QA pass should use a fresh lab unless continuing a known active manifest.

Key decisions:

- Treat this as a new independent QA pass for `mem-v2` unless a same-loop manifest is discovered.
- Use `compozy exec --ide claude --model opus --reasoning-effort xhigh` for the report portion, not for directing runtime agents under test.
- Selected playbook `consumer-saas-growth` because the selection matrix maps persistence/web read-model changes to that playbook.
- Added `northstar-pay` after user explicitly requested using both; real-scenario-qa runs exactly one playbook per lab, so this session will maintain two isolated labs and consolidate evidence.
- Claude Code ACP is now a blocking evidence lane for this QA run, not optional smoke coverage.
- Root-cause fix must keep `ClaimNextRun(criteria)` as the only autonomous claim primitive; scheduler/daemon may start or wake role sessions but must not claim runs directly.
- Pool-owned task runs must be claimable only by the matching agent name (`owner.kind=pool`, `owner.ref=<agent>`), and scheduler wake targeting must use the same owner match.
- Enqueued pool-owned runs need daemon role-session activation so queued work can progress even when only the lead/operator session exists.

State:

- Goal created for Codex Loop: `memv2-real-qa`.
- Both selected labs have been bootstrapped and launched against the real `claude` ACP provider.
- Consumer/Lumen completed a long multi-agent Claude ACP run with deliverables written to the lab workspace.
- Northstar completed only the lead Claude ACP prompt and explicitly reported that the other agents had not started; treat as a real QA failure/stall, not an operator prompt gap.
- QA artifacts, bug reports, strict audit reports, Claude Opus report, and verification reports have been written for both labs.
- Repository `make verify` passed with exit code 0 after the QA run.
- Follow-up fix work is active. Need investigate and fix runtime defects before claiming completion.
- Root-cause investigation confirmed two production gaps: `internal/store/globaldb.ClaimNextRun` ignores `tasks.owner_kind/owner_ref`, and `internal/scheduler.isEligibleSession` ignores task ownership.
- Root-cause investigation also confirmed that task-run enqueue has coordinator recovery but no role-agent activation path for `owner.kind=pool` tasks, so Northstar queued work never materializes matching sessions.
- Production fix and focused regressions are implemented locally.
- Full `rtk make verify` passed after the patch.
- Focused post-fix QA evidence is recorded in both QA labs. Northstar post-fix daemon recovery activated 9 live Claude ACP sessions, recovered queued pool-owned work into matching role sessions, and proved manual pool-run claim rejection. Lumen evidence was updated to show the pre-fix human/local-user pool-claim defect and cross-link the post-fix guard.
- Both isolated QA daemons were stopped after evidence capture.
- Strict scenario audits were refreshed and still FAIL for full playbook completion (`northstar-pay`: C10/C11/C16/C17; `consumer-saas-growth`: C17 review-cycle shortfall). These are residual full-scenario completion blockers, not failures of the focused owner/session regression fix.

Done:

- Read RTK instructions.
- Created this ledger.
- Loaded initial `real-scenario-qa`, `compozy`, `codex-loop`, and `agh-qa-bootstrap` skill instructions from visible prompt / disk.
- Loaded `qa-execution` and `qa-report` skill instructions.
- Read relevant prior ledgers: `mem-v2-tasks`, `qa-workflow-debug`, and `qa-skill-hardening`.
- Confirmed `.compozy/tasks/mem-v2` has 26 task files and existing QA plan/cases.
- Confirmed current `compozy` version is `0.2.1`; `compozy tasks validate --name mem-v2 --format json` passes with `scanned: 26`.
- Noted `_tasks.md` still marks tasks as `pending`, but current code/search surfaces show Memory v2 implementation symbols and OpenAPI routes exist.
- Validated `consumer-saas-growth` playbook successfully: 7 agents, 4 channels, 11 open tasks.
- Bootstrapped fresh lab with `REUSED_LAB=false`.
- Verified bootstrap has no `UNFILLED` charter placeholders, 11 open tasks, 7 agent files, playbook JSON, knowledge files, and scenario contract minimums.
- Confirmed local Claude Code is installed and authenticated; do not record private account details in artifacts.
- Read provider docs and L-016: Claude Code is built-in provider `claude`, harness `acp`, command `npx -y @agentclientprotocol/claude-agent-acp@latest`, auth mode `native_cli`, default model `claude-sonnet-4-6`, and `home_policy=operator` should preserve the operator HOME.
- Validated `northstar-pay` playbook successfully: 11 agents, 10 channels, 12 open tasks.
- Bootstrapped fresh `northstar-pay` lab with `REUSED_LAB=false`, no `UNFILLED` charter placeholders, 12 open tasks, 11 agent files, playbook JSON, knowledge files, and scenario contract minimums.
- Materialized executable AGH `AGENT.md` definitions for both labs under `.agh/agents/`, using provider `claude`, model `claude-sonnet-4-6`, and `approve-all` permissions.
- Registered both workspaces in their isolated AGH homes and verified agent lists.
- First Claude ACP session attempt exposed a real environment boundary: the daemon could not resolve `npx` because its inherited `PATH` lacked the mise Node install path.
- Restarted both daemons with the Node/mise bin directory in `PATH`; subsequent `agh session new` calls succeeded with provider `claude` and ACP capabilities.
- Consumer kickoff posted to session `sess-941ffe8afb6ed7b3` / ACP session `ffa05fc9-f126-4a1c-99a2-3e85c900e2d7`; output captured in `operator-kickoff.jsonl` with empty stderr.
- Consumer seeded 11 task runs and observed many live Claude ACP sessions, including role sessions plus memory extractor sessions; daemon status showed active sessions in the 7-10 range and total sessions growing to roughly 80.
- Northstar kickoff posted to session `sess-1f67dfd3489e27cf` / ACP session `7647f3c3-492f-4787-a3eb-d2452e10462d`; output captured in `operator-kickoff.jsonl` with empty stderr.
- Northstar seeded 12 task runs but observed only the lead session completing; the lead wrote one decision file and reported all other deliverables unstarted because other agents were not online.
- Recorded structured CLI/API/Web/runtime/provider evidence in both `journey-log.jsonl` files.
- Filled both `provider-attempt.json` files with live Claude ACP proof and the initial daemon PATH boundary.
- Ran `observe-runtime.py` in a short post-run window for both labs; both reported stalls.
- Filed Lumen `BUG-001-task-state-not-closed-after-agent-output.md`.
- Filed Northstar `BUG-001-multi-agent-startup-stalls-after-lead.md`.
- Ran `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json` for the report-only analysis; saved output to `opus-report.md`.
- Ran `make verify`; exit code 0, log copied to both QA roots.
- Wrote `verification-report.md` for both labs.
- Ran strict `audit-qa-evidence.py` for both labs after report update.
- Read `qa-execution`, `systematic-debugging`, `no-workarounds`, Go/test skills, `internal/CLAUDE.md`, RTK, current ledger, and relevant scheduler/claim/execution-boundary ledgers.
- Inspected scheduler selection, global DB claim SQL, task manager claim/lease behavior, daemon task session bridge, hooks observer path, coordinator runtime observer pattern, and CLI/native claim surfaces.
- Added owner-aware `ClaimNextRun` SQL filtering, scheduler owner/session matching, daemon task role session activation/recovery, explicit pool-claim rejection, and task session bridge owner/channel binding.
- Added regressions in `internal/store/globaldb`, `internal/scheduler`, `internal/daemon`, and `internal/task`.
- Verification passed: focused tests for owner claim, owner wake, explicit pool claim rejection, task role activation, task session bridge; broader `go test ./internal/task ./internal/store/globaldb ./internal/scheduler ./internal/daemon -count=1`; scheduler integration `go test -tags integration ./internal/scheduler -count=1`; `git diff --check`.
- First `rtk make verify` found a lint `unparam` in `internal/daemon/task_role_runtime.go`; fixed by removing the unnecessary return value.
- `rtk make lint` passed after the lint fix.
- Second `rtk make verify` passed with exit code 0.
- Reused the Northstar lab to validate boot recovery from the exact queued-work failure state.
- Saved post-fix Northstar evidence: `post-fix-daemon-status.json`, `post-fix-session-list.json`, `post-fix-role-session-summary.json`, `post-fix-task-list.json`, `post-fix-task-summary.json`, `post-fix-run-owner-query.txt`, `post-fix-explicit-claim-guard.txt`, `post-fix-explicit-claim-state.txt`, and `post-fix-daemon-stop.json`.
- Confirmed `agh task run claim run-2a648c626fa633aa -o json` exits 1 with permission denied for a pool-owned frontend task and leaves the run queued.
- Saved Lumen pre-fix ownership evidence in `pre-fix-human-pool-claims.tsv`, showing five pool-owned runs claimed by `human/local-user` without `session_id`.
- Updated both bug reports and verification reports with confirmed root causes, fix details, focused post-fix evidence, refreshed strict audit outcomes, and stopped-daemon bootstrap status.
- Refreshed strict QA audits after report updates. Northstar still fails full scenario audit with 6 blockers; Lumen still fails full scenario audit with 1 C17 blocker.
- Stopped the Northstar post-fix daemon and the original Consumer/Lumen daemon; subsequent status checks reported `stopped`.

Now:

- Final sanity checks: git status/diff summary, test evidence recap, and no running QA daemons.

Next:

- Produce the final response with changed files, verification evidence, residual full-scenario audit blockers, and QA bootstrap paths.

Open questions (UNCONFIRMED if needed):

- None currently blocking.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-06-MEMORY-memv2-real-qa.md`
- `.compozy/tasks/mem-v2/`
- `.compozy/tasks/mem-v2/qa/test-plans/memory-v2-test-plan.md`
- `.compozy/tasks/mem-v2/qa/test-cases/TC-SCEN-001.md`
- `.compozy/tasks/mem-v2/qa/test-cases/TC-SCEN-002.md`
- `.agents/skills/real-scenario-qa/`
- `.agents/skills/agh-qa-bootstrap/`
- `rtk compozy tasks validate --name mem-v2 --format json`
- Lab root: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab`
- Manifest: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts/qa/bootstrap-manifest.json`
- QA output: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts`
- Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-223e473847ee/runtime`
- Base URL: `http://127.0.0.1:50106`
- Playbook: `consumer-saas-growth`
- Consumer lead session: `sess-941ffe8afb6ed7b3`
- Consumer ACP session: `ffa05fc9-f126-4a1c-99a2-3e85c900e2d7`
- Consumer verification report: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts/qa/verification-report.md`
- Consumer audit report: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts/qa/qa-audit-report.json`
- Consumer audit verdict: FAIL, blocker `C17 review_cycles_min 2 < required 3`
- Consumer/Lumen post-fix focused regression verdict: PASS for confirmed owner/claim defect; strict full scenario audit still FAIL with blocker `C17 review_cycles_min 2 < required 3`.
- Northstar lab root: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-northstar-20260506-051548-387796-lab`
- Northstar manifest: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-northstar-20260506-051548-387796-lab/qa-artifacts/qa/bootstrap-manifest.json`
- Northstar QA output: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-northstar-20260506-051548-387796-lab/qa-artifacts`
- Northstar runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-b4ca55d36007/runtime`
- Northstar base URL: `http://127.0.0.1:51661`
- Northstar playbook: `northstar-pay`
- Northstar lead session: `sess-1f67dfd3489e27cf`
- Northstar ACP session: `7647f3c3-492f-4787-a3eb-d2452e10462d`
- Northstar verification report: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-northstar-20260506-051548-387796-lab/qa-artifacts/qa/verification-report.md`
- Northstar audit report: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-northstar-20260506-051548-387796-lab/qa-artifacts/qa/qa-audit-report.json`
- Northstar audit verdict: FAIL, blockers `C10`, `C11`, `C16`, `C17`
- Northstar post-fix focused regression verdict: PASS for BUG-001; strict full scenario audit still FAIL with blockers `C10`, `C11`, `C16`, `C17`.
- Claude Opus report: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts/qa/opus-report.md`
- Make verify log: `/Users/pedronauck/dev/qa-labs/agh-memv2-real-qa-20260506-051216-337090-lab/qa-artifacts/qa/make-verify.log`
- Code under investigation/fix: `internal/store/globaldb/global_db_task_claim.go`, `internal/scheduler/scheduler.go`, `internal/daemon/task_runtime.go`, planned new daemon role-session runtime/tests.
- Added/modified code: `internal/daemon/task_role_runtime.go`, `internal/daemon/task_role_runtime_test.go`, `internal/daemon/boot.go`, `internal/daemon/task_runtime.go`, `internal/daemon/task_runtime_test.go`, `internal/scheduler/scheduler.go`, `internal/scheduler/scheduler_test.go`, `internal/store/globaldb/global_db_task_claim.go`, `internal/store/globaldb/global_db_task_claim_test.go`, `internal/task/manager.go`, `internal/task/manager_test.go`.
