# Agent Soul QA Verification Report

## Claim

The current branch was tested against `.compozy/tasks/agent-soul/_techspec.md` with behavior-first QA in an isolated lab. The pass used real operator surfaces first: CLI, HTTP, UDS, SQLite state inspection, daemon restart/rebuild, task claim flows, session spawn flows, config mutation, and live Codex-backed sessions. Focused race/store/security tests were used only for deterministic edge cases that are unsafe or timing-sensitive to force through the live daemon.

## Verdict

PASS for the runtime implementation after three root-cause fixes discovered by this QA work:

- `BUG-004`: HTTP agent-facing context routes rejected valid agent-session origins.
- `BUG-005`: spawned child sessions lost `parent_soul_digest` in durable session rows.
- `BUG-006`: `agh config set` rejected valid `[agents.soul]` and `[agents.heartbeat]` overlay paths.

Provider output observation: live Codex provider execution was reachable and the reviewer output followed the authored evidence-first/conservative Soul shape, but the model twice mangled exact digest text when asked to quote it. Runtime surfaces returned the correct digest; the report does not count LLM digest transcription as deterministic product proof.

## Environment

- Bootstrap manifest: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/qa-artifacts/qa/bootstrap-manifest.json`
- Lab root: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab`
- Runtime home: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.agh/runtime`
- Provider home: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.provider-home`
- Provider Codex home: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.provider-home/.codex`
- Base URL: `http://127.0.0.1:50140`
- UDS path: `/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.agh/runtime/aghd.sock`
- Workspace: `agent-soul-lab` (`ws_abaaa84e9c734bbb`)
- Provider used for live sessions: `codex`, model `gpt-5.4`

```qa-bootstrap
manifest_path=/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/qa-artifacts/qa/bootstrap-manifest.json
lab_root=/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab
runtime_home=/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.agh/runtime
base_url=http://127.0.0.1:50140
provider_home=/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.provider-home
provider_codex_home=/Users/pedronauck/dev/qa-labs/agh-agent-soul-20260502-185208-519542-lab/.provider-home/.codex
evidence_dir=.compozy/tasks/agent-soul/qa/evidence
browser_mode=browser-use
daemon_status=running
```

## Behavioral Coverage

- `SMOKE-001`: PASS. Current branch baseline and isolated daemon/bootstrap readiness were proven. Evidence: `project-contract.json`, `smoke-codegen-check.log`, `baseline-current-make-verify.log`, `smoke-daemon-status.log`, `smoke-workspace-add.json`.
- `TC-SCEN-001`: PASS. Managed Soul validate/write/inspect/API/history and stale CAS behavior worked. Evidence: `TC-SCEN-001-cli.log`, `TC-SCEN-001-soul-inspect-cli.json`, `TC-SCEN-001-api.json`, `TC-SCEN-001-soul-history.json`.
- `TC-SCEN-002`: PASS. Managed Heartbeat write/status/API/session health, dry-run wake, missing-session decision, and no side-effect task/session event behavior worked. Evidence: `TC-SCEN-002-cli.log`, `TC-SCEN-002-heartbeat-status-cli.json`, `TC-SCEN-002-session-new.json`, `TC-SCEN-002-heartbeat-wake-dry-run.json`, `TC-SCEN-002-session-events-after-dry-run.json`, `TC-SCEN-002-task-list-after-dry-run.json`.
- `TC-SCEN-003`: PASS WITH OBSERVATION. Live reviewer sessions were created through Codex. `agh me context`, UDS, and HTTP `/api/agent/context` exposed the correct Soul projection after `BUG-004` was fixed. Provider output reflected the Soul persona and did not claim hidden ownership, but digest transcription by the model was not reliable. Evidence: `TC-SCEN-003-session-new-after-http-fix.json`, `TC-SCEN-003-me-context-after-http-fix.json`, `TC-SCEN-003-agent-context.json`, `TC-SCEN-003-provider-boundary.log`, `TC-SCEN-003-agent-output.txt`, `TC-SCEN-003-provider-followup.log`, `TC-SCEN-003-agent-followup-output.txt`.
- `TC-SCEN-004`: PASS AFTER FIX. Idle session Soul refresh succeeded; active task-run refresh rejected with CLI non-zero and HTTP `409`; `task_runs.metadata_json` stored Soul provenance during `ClaimNextRun`; child spawn persisted `sessions.parent_soul_digest` after `BUG-005`. Evidence: `TC-SCEN-004-session-soul-refresh-idle.json`, `TC-SCEN-004-session-soul-refresh-active-http-409.log`, `TC-SCEN-004-task-run-metadata.json`, `BUG-005-parent-child-sessions-sqlite-after-fix.json`.
- `TC-SCEN-005`: PASS. Live UDS read/write parity was proven for Soul and Heartbeat, and `/api/agent/context` projection truncation obeyed `context_projection_bytes`. Evidence: `TC-SCEN-005-uds-http-parity-check.json`, `TC-SCEN-005-uds-soul-put-valid5.json`, `TC-SCEN-005-uds-heartbeat-put.json`, `TC-SCEN-005-context-truncation-summary.json`, `TC-SCEN-005-http-agent-context-truncated.json`, `TC-SCEN-005-uds-agent-context-truncated.json`.
- `TC-REG-001`: PASS. Invalid Soul and invalid Heartbeat failed closed with deterministic diagnostics and did not mutate the valid counterpart feature. Evidence: `TC-REG-001-invalid-soul.err`, `TC-REG-001-invalid-heartbeat.err`, `TC-REG-001-cross-feature.log`.
- `TC-REG-002`: PASS. CLI `--expected-digest`/`--if-match` mapped to body-level CAS. HTTP `If-Match` alone was rejected. HTTP body `expected_digest` succeeded. Evidence: `TC-REG-002-cli.log`, `TC-REG-002-http-if-match.json`, `TC-REG-002-http-body-cas.json`.
- `TC-REG-003`: PASS. `make codegen-check` and `make bun-test` passed. Search evidence shows generated types, tests, and docs references only; no unsupported Web Soul/Heartbeat editor was counted. Evidence: `TC-REG-003-codegen-check.log`, `TC-REG-003-bun-test.log`, `TC-REG-003-web-boundary.log`, `TC-REG-003-docs-cli-boundary.log`.
- `TC-REG-004`: PASS. Rollback/delete/restart recovery worked. Soul deletion persisted as `present=false`; Heartbeat remained active/valid after restart; Heartbeat rollback created no tasks; authored `source_path` values remained relative. Evidence: `TC-REG-004-soul-rollback.json`, `TC-REG-004-soul-delete.json`, `TC-REG-004-heartbeat-rollback.json`, `TC-REG-004-restart-recovery.log`, `TC-REG-004-redaction-scan.log`.
- `TC-REG-005`: PASS AFTER FIX. Wake coalescing, manual cooldown rate limiting, max-wakes-per-cycle limiting, wake retention, config overlay, invalid config rejection, and absent `agh session heartbeat` were covered. Evidence: `TC-REG-005-heartbeat-wake-policy-go-test.log`, `TC-REG-005-heartbeat-wake-retention-go-test.log`, `BUG-006-config-cli-focused-go-test.log`, `TC-REG-005-config-invalid-rejection-summary.json`, `TC-REG-005-session-heartbeat-absent-check.json`.
- `TC-SEC-001`: PASS. Host API writes without grants failed before managed services ran, hook patches remained observation-only, and HTTP missing/stale `X-AGH-Session-ID` returned `401`. Evidence: `TC-SEC-001-focused-go-test.log`, `TC-SEC-001-http-agent-context-missing-session-id.log`, `TC-SEC-001-http-agent-context-invalid-session-id.log`.

The reviewer gap matrix is machine-readable in `confirmation-gap-coverage.json`.

## Issues

- `BUG-004`: Fixed. HTTP agent-facing routes rejected valid agent-session origins because task actor validation did not allow `OriginKindHTTP`. Fixed in `internal/task/actors.go`, covered by `internal/task/actors_test.go` and `internal/api/httpapi/agent_context_test.go`.
- `BUG-005`: Fixed. Hook lifecycle forwarding lost full session lineage/provenance before the global session observer persisted spawned child rows. Fixed in `internal/daemon/hooks_bridge.go`, covered by `internal/daemon/notifier_test.go`.
- `BUG-006`: Fixed. Config CLI mutation kind allowlist omitted Agent Soul and Heartbeat keys. Fixed in `internal/cli/config.go`, covered by `internal/cli/config_test.go`.

Previously fixed issues from earlier Task 17 evidence remain covered by the current pass where relevant:

- `BUG-001`: CLI managed authoring create required CAS.
- `BUG-002`: Heartbeat dry-run exposed non-persisted identifiers.
- `BUG-003`: Heartbeat missing-session wake returned raw session errors.

## Verification

VERIFICATION REPORT
-------------------
Claim: Agent Soul second-round gap coverage is ready for the final monorepo gate.
Command: Focused verification suite for the newly covered gaps:

- `go test -v -race ./internal/heartbeat -run '^TestManagedWakeServiceDecision$' -count=1`
- `go test -v -race ./internal/store/globaldb -run 'TestGlobalDBHeartbeatMigration|TestGlobalDBHeartbeatSnapshotAndRevisionStore|TestGlobalDBClaimNextRunPersistsSoulProvenanceMetadata|TestGlobalDBSessionHealthStaleDetection|TestGlobalDBSoulSessionProvenance' -count=1`
- `go test -v -race ./internal/store/globaldb -run '^TestGlobalDBHeartbeatWakeAuditStore$' -count=1`
- `go test -v -race ./internal/session -run 'TestSessionHealth|TestPromptSyntheticQueuesBehindActiveTurnAndPreservesStoredOrder|TestPromptSyntheticHeartbeatWakeOptions' -count=1`
- `go test -v -race ./internal/cli -run '^TestConfigSetSupportsAgentAuthoredContextPaths$' -count=1`
- `go test -v -race ./internal/api/httpapi ./internal/extension ./internal/hooks -run 'TestAgentContextHTTPIdentity|TestHostAPIAuthoredContextWriteBypassRejections|TestAuthoredContextHooksRemainObservationOnly' -count=1`
- `go test -v -race ./internal/daemon -run '^TestHooksNotifierLifecycleForwarding$' -count=1`

Executed: 2026-05-02, before the final monorepo gate.
Exit code: `0` for every focused command.
Output summary: all focused packages returned `PASS`/`ok`; verbose logs include the named subtests for coalescing, rate limiting, prompt race, migration v13, wake retention, ClaimNextRun Soul metadata, config overlay validation, Host API grant denial, hook observation-only behavior, and HTTP identity rejection.
Warnings: Go test convention helper logs for some pre-existing files still report unrelated file-level legacy shape violations; new or touched subtests use `Should ...` naming.
Errors: none in the focused commands.
Verdict: PASS for focused confirmation-gap coverage.

VERIFICATION REPORT
-------------------
Claim: Agent Soul QA pass is complete against the full monorepo gate.
Command: `TURBO_ENV_MODE=loose MAGE='go run github.com/magefile/mage@v1.15.0' make verify`
Executed: 2026-05-02 after the second-round QA report, issue files, focused tests, and root-cause fixes.
Exit code: `0`
Output summary: Bun lint passed with `Found 0 warnings and 0 errors`; Turbo typecheck passed for 5 workspaces; Vitest passed with `266` files and `1886` tests; Web build completed; Go lint reported `0 issues`; Go tests completed with `DONE 7732 tests in 52.213s`; package boundaries ended with `OK: all package boundaries respected`.
Warnings: raw `make verify`, `make MAGE= verify`, and `MAGE= make verify` were blocked before code checks by the local `mise` `mage` shim inside Turbo-filtered workspace subprocesses. The successful full gate used `TURBO_ENV_MODE=loose` and an explicit `MAGE` command so nested `make codegen-check` calls used the repository Mage fallback. Vite retained the existing chunk-size warning, and macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building golangci-lint.
Errors: none in the successful full gate.
Verdict: PASS.
