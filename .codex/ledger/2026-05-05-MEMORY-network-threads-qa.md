Goal (incl. success criteria):

- Run a fresh real-scenario QA pass for `.compozy/tasks/network-threads` on the current branch.
- Success means current-state evidence proves the network thread/direct-room feature works through realistic operator journeys, CLI/API/Web/runtime state, persisted artifacts, disruption probes, and fresh verification gates, with exact blockers documented.
- New user challenge (2026-05-05): prove the feature with real LLM-backed ACP Claude Code sessions, multiple agents, a startup-like end-user scenario, and concrete 100% passing evidence. Previous deterministic ACP mock evidence is not sufficient for this stricter criterion.

Constraints/Assumptions:

- Current date: 2026-05-05.
- Conversation with the user is in Brazilian Portuguese; QA artifacts are in English.
- Required skills: `agh-qa-bootstrap`, `real-scenario-qa`, `qa-report`, `qa-execution`; use `systematic-debugging` + `no-workarounds` for any unexpected behavior.
- Fresh independent QA pass: create a fresh lab, do not reuse older `.compozy/tasks/network-threads/qa/bootstrap-manifest.json` unless explicitly continuing that exact run.
- No destructive git commands (`git restore`, `git checkout`, `git reset`, `git clean`, `git rm`) without explicit user permission.
- `make verify` is the blocking full-repo gate before claiming completion.

Key decisions:

- Session ledger path: `.codex/ledger/2026-05-05-MEMORY-network-threads-qa.md`.
- Existing `.codex/ledger/2026-05-05-MEMORY-network-threads.md` is read-only cross-session context showing earlier implementation QA completed; this pass still needs fresh evidence for the current objective.

State:

- Live-provider QA is in final gate stage. Real Claude Code ACP sessions completed a startup launch-readiness thread/direct-room journey after a native tool schema fix. CLI/API/SQLite/Web evidence is captured; `make verify` still pending before completion.

Done:

- Loaded required QA skill instructions and `internal/CLAUDE.md`.
- Scanned `.codex/ledger/` and read relevant network-thread ledgers.
- Confirmed `.compozy/tasks/network-threads` has prior QA artifacts and final `make verify` logs from the implementation loop.
- Confirmed `git status --short --branch` is clean on branch `network-threads`.
- Created fresh QA bootstrap for scenario `network-threads-real-scenario-20260505-192724-176663`.
- Bootstrap manifest: `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/bootstrap-manifest.json`.
- Lab workspace: `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab`.
- Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-fbf27d455e9c/runtime`.
- API base URL: `http://127.0.0.1:63404`.
- Browser mode: `browser-use`.
- Wrote fresh QA charter and plan under `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/`.
- Baseline `make verify` passed before scenario mutation; log path `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/baseline-make-verify.log`.
- User requested Claude Code review of QA report/cases via `$compozy`; prompt written to `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/claude-qa-gap-review-prompt.md`.
- `compozy exec --ide claude --model opus --reasoning-effort xhigh --format json ...` completed successfully; stderr empty; run ID `exec-20260505-192937-000000000`.
- Claude verdict: `NEEDS_MORE_CASES`.
- Blocking missing cases from Claude: GAP-001 concurrent direct-room resolve race; GAP-002 terminal/post-terminal `work_id` lifecycle; GAP-003 summary counter agreement; GAP-004 duplicate `message_id` idempotency; GAP-005 native `agh__network_*` tool path.
- Updated fresh charter and plan to include the five P0 gaps as execution requirements.
- Added executable fresh scenario script `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/cli-api-tool-scenario.zsh`; `zsh -n` passed.
- First script run failed before product exercise because `go build ./internal/...` ran from the evidence directory with no `go.mod`; fixed the harness to call `go -C "$REPO_ROOT" build` and to preserve command exit codes through explicit `set +e` capture. `zsh -n` passed after the fix.
- A rerun showed product correctly rejected post-terminal work (`network: work closed`), but the harness still captured zsh status incorrectly because `local exit_code` reset `$?`; fixed by declaring locals before command execution.
- Added dynamic ID suffixes for reruns and updated generated AGENT.md files to declare `tools: ['agh__network_*', 'agh__tool_info']` plus `toolsets: ['agh__coordination']` so session-scoped native network tools are visible.
- Fresh CLI/API/native-tool scenario completed successfully. Scenario IDs: `THREAD_ID=thread_builders_gap_20260505194411`, `DIRECT_ID=direct_5d821603e5a68ba0c2182b5dfc64c906`, `WORK_ID=work_direct_gap_20260505194411`, `OPS_SESSION_ID=sess-57402b99d62414c6`, `PATCH_SESSION_ID=sess-2b3e289649ed958c`.
- Passed Claude P0 probes in the script: GAP-001 concurrent direct resolve + one DB row; GAP-002 terminal work + post-terminal rejection; GAP-003 counter parity CLI/API/SQL; GAP-004 duplicate `message_id` idempotency + one timeline row; GAP-005 session-scoped native `agh__network_*` tools and legacy-field rejection.
- User authorized `$agent-browser`; `browser-use` Node REPL tool was unavailable via tool discovery, so Web validation uses `agent-browser` against Vite on `http://127.0.0.1:3013` with `AGH_WEB_API_PROXY_TARGET=http://127.0.0.1:63404`.
- Agent-browser thread snapshot/screenshot captured: `web-thread-snapshot.txt`, `web-thread-screenshot.png`.
- Agent-browser direct snapshot exposed BUG-003: direct room showed `1 active work in flight` even though `/api/network/channels/builders/directs/direct_5d821603e5a68ba0c2182b5dfc64c906` returned `open_work_count: 0`. Root cause: `useOpenWork` let a non-terminal lifecycle message with the same second overwrite a terminal `completed` state due timestamp precision and message_id ordering.
- Applied focused Web fix in `web/src/systems/network/hooks/use-work.ts` and regression test in `web/src/systems/network/hooks/use-work.test.tsx`.
- Targeted test passed: `cd web && bun run test:raw src/systems/network/hooks/use-work.test.tsx` -> 1 file / 3 tests passed.
- Agent-browser direct reload after fix no longer shows `1 active work in flight` or the Work inspector; snapshot/screenshot stored as `web-direct-after-fix-snapshot.txt` and `web-direct-after-fix-screenshot.png`. Browser console/errors were clear.
- Provider auth probes completed with exit code 0 for Claude and Codex in no-probe and live modes; both providers reported `state: native_cli`, `home_policy: operator`.
- Wrote bug artifact `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/issues/BUG-003-web-direct-open-work-terminal-order.md`.
- Final `make verify` passed after the Web fix; log path `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/final-make-verify.log`.
- Wrote final QA report `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/verification-report.md`.
- Completion audit: `git diff --check` clean; git status only shows expected Web hook/test edits plus this ledger; QA Vite server stopped; isolated daemon on port 63404 stopped; no listener remains on port 63404.
- Started stricter live-provider rerun after user challenge.
- Fresh live-provider QA lab: scenario `network-threads-live-claude-startup-20260505-200035-498426`; lab `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab`; manifest `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/bootstrap-manifest.json`; AGH_HOME `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-7349d607ded1/runtime`; API `http://127.0.0.1:53541`; `REUSED_LAB=false`.
- Live daemon started on `http://127.0.0.1:53541`; workspace `signalforge` registered; provider auth status for Claude reported `native_cli`/`home_policy=operator`.
- Spawned two real Claude-backed ACP sessions: coordinator `sess-8b1bb7e8e0e40f9f` (`acp_session_id=726d478b-e286-4ac9-ba7f-c16521cc7bda`) and reviewer `sess-2525ea1e4d0ebc31` (`acp_session_id=15b12063-41cf-4ef6-b819-3b6edcb54e2f`). Channel peers visible: `launch-coordinator.sess-8b1bb7e8e0e40f9f`, `risk-reviewer.sess-2525ea1e4d0ebc31`.
- Real coordinator prompt failed after reading workspace files and loading AGH Network tools: Claude rejected `agh__network_send` schema with `tools.9.custom.input_schema: input_schema does not support oneOf, allOf, or anyOf at the top level`. Crash bundle `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-7349d607ded1/runtime/logs/crash-bundles/sess-8b1bb7e8e0e40f9f-prompt_failure-1778011422084748000.json`.
- Root cause fixed locally: removed top-level `oneOf` from built-in `agh__network_send` schema in `internal/tools/builtin/network.go`; added built-in descriptor provider-compat test in `internal/tools/builtin/builtin_test.go`; existing native handler validation still rejects invalid conversation payloads.
- Focused tests passed: `go test ./internal/tools/builtin -count=1`; `go test ./internal/tools -run 'TestNativeProviderDispatch/Should enforce enum oneOf and not schema rules before native handler invocation' -count=1`; `go test ./internal/daemon -run 'TestDaemonNativeTools/Should reject native network send with the same conversation validation as HTTP payloads' -count=1`; `make build` passed; isolated daemon restarted with rebuilt `bin/agh`.
- Spawned post-fix real Claude ACP sessions: coordinator `sess-b61795116e1ac6da` (`acp_session_id=3124f707-5132-4950-b169-38eed0bc6050`) and reviewer `sess-814307617c105677` (`acp_session_id=7bb9b9a9-cb1b-473f-b3ac-4dfeed3088c7`).
- Live startup scenario completed through real Claude work: coordinator read workspace files, wrote `launch/coordinator-live-note.md`, opened thread `thread_live_launch_readiness_20260505_postfix` and work `work_live_launch_readiness_20260505_postfix`; reviewer accepted, traced work, issued risk assessment, and completed the work. Both sessions ended idle/healthy with no active prompt.
- Direct-room follow-up completed in `direct_6046567df819a284582947b0ef7b39b0` with work `work_live_direct_launch_followup_20260505`; direct work is completed with `open_work_count=0`.
- Native tool schema gap closed with evidence: `agh tool info agh__network_send` shows no top-level `oneOf`/`anyOf`/`allOf`, and `agh tool invoke agh__network_send` succeeded with message `msg-dbc865aefbf22f19` without waking agents.
- Final CLI/API/SQLite parity after native-tool message: thread has `message_count=17`, `participant_count=2`, `open_work_count=0`, SQLite timeline count `17`, thread work `completed`; direct has `message_count=7`, `open_work_count=0`, SQLite timeline count `7`, direct work `completed`.
- Agent-browser Web validation against Vite `http://127.0.0.1:3014` and isolated API target `http://127.0.0.1:53541` captured thread/direct snapshots and screenshots; browser console/errors were clear.
- Final `make verify` passed for the current code state; log: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/runs/20260505T200035Z-live-claude-startup/final-make-verify.log`.
- Updated BUG-004 to `Fixed and verified`; wrote live verification report: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/verification-report.md`.
- Cleanup completed: agent-browser session closed; isolated daemon stopped with `active_sessions=0`; no listeners remain on ports `3014` or `53541`; cleanup evidence `cleanup-check.log`.
- Completion audit written: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/runs/20260505T200035Z-live-claude-startup/completion-audit.md`.

Now:

- Ready to mark the active goal complete.

Next:

- Final response to user in BR-PT with artifact paths, bugs fixed, gates, and residual observation.

Open questions (UNCONFIRMED if needed):

- None blocking. Observation: live agents generated redundant close messages before self-stopping; no active prompt remains, but report should call this out as real-LLM behavior rather than hide it.

Working set (files/ids/commands):

- `.compozy/tasks/network-threads/`
- `.codex/ledger/2026-05-05-MEMORY-network-threads-qa.md`
- `.agents/skills/agh-qa-bootstrap/scripts/bootstrap-qa-env.py --scenario <scenario> --repo-root .`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/bootstrap-manifest.json`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/baseline-make-verify.log`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/claude-qa-gap-review-prompt.md`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/claude-qa-gap-review-output.md`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/cli-api-tool-scenario.zsh`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/cli-api-tool-command-log.txt`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/counter-parity.json`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/runs/20260505T192724Z-real-scenario/cli-api-tool-scenario-summary.md`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/verification-report.md`
- `/Users/pedronauck/dev/qa-labs/agh-network-threads-real-scenario-20260505-192724-176663-lab/qa-artifacts/qa/issues/BUG-003-web-direct-open-work-terminal-order.md`
- `make verify`
- Live rerun manifest: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/bootstrap-manifest.json`
- Live rerun env: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/bootstrap.env`
- Live rerun report target: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/verification-report.md`
- Live rerun final gate target: `/Users/pedronauck/dev/qa-labs/agh-network-threads-live-claude-startup-20260505-200035-498426-lab/qa-artifacts/qa/runs/20260505T200035Z-live-claude-startup/final-make-verify.log`
