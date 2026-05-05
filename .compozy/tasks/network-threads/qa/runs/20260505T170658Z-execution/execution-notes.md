# Network Threads QA Execution Notes

## Baseline Correction

- Initial command: `zsh -lc 'set -o pipefail; source .compozy/tasks/network-threads/qa/bootstrap.env; make verify ...'`
- Result: invalid FAIL.
- Root cause: the QA shell exported `AGH_HOME` from the runtime bootstrap before running the monorepo verification gate. Two repository tests intentionally validate environment handling:
  - `internal/acp TestNetworkTurnTerminalOwnershipGuards`
  - `internal/config TestLoadUsesDotEnvForAGHHome`
- Correction: rerun repository gates without sourcing `bootstrap.env`. Use the bootstrap environment only for isolated daemon, CLI/API, and Web scenario commands.

## CLI/API Scenario Harness Correction

- Initial command: `zsh .compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/cli-api-scenario.zsh`
- Result: invalid FAIL before product behavior was exercised.
- Root cause: the QA zsh harness used `local path=...` inside HTTP helper functions. In zsh, `path` is tied to `PATH`; assigning it removed access to utilities such as `sleep` during HTTP readiness polling.
- Additional harness correction: renamed local `status` variables to `exit_code` because `status` is also a zsh special parameter.
- Additional harness correction: changed session ID extraction from `session.id` to top-level `id`, matching the current `agh session new -o json` contract.
- Additional harness correction: made `run_capture` abort explicitly from its `else` branch because zsh did not stop the script on a non-zero function return in this context.
- Additional harness correction: made daemon startup idempotent when the same-session isolated daemon is already running.
- Additional scenario correction: changed receipt and trace bodies to the normative wire shapes (`for_id`/`status`, `state:"working"`) and reran the flow with fresh `qa2` message/work/thread IDs.
- Correction: renamed those local variables, aligned the session ID parser, fixed error propagation, and reran the scenario.

## Runtime E2E Defect

- Initial command: `make test-e2e-runtime`
- Result: confirmed FAIL in `TestDaemonE2EACPmockPermissionDisconnectProjectsRuntimeFailure`.
- Root cause: session event queries could obtain an active recorder while that session was already finalizing and closing the SQLite recorder.
- Fix and evidence: `.compozy/tasks/network-threads/qa/bug-reports/BUG-001-session-event-query-finalization-race.md`.
- Rerun: `test-e2e-runtime-after-fix.log` passed.

## Web Missing Conversation Defect

- Initial browser probes:
  - `http://localhost:3001/network/builders/threads/thread_missing_qa`
  - `http://localhost:3001/network/builders/directs/direct_missing_qa`
- Result: confirmed FAIL for `TC-UI-001`.
- Root cause: detail routes rendered normal empty/composer states while the detail query was still unresolved, and TanStack Query delayed final 4xx detail errors with the default retry policy.
- Fix and evidence: `.compozy/tasks/network-threads/qa/bug-reports/BUG-002-web-network-missing-conversation-state.md`.
- Rerun: invalid thread/direct snapshots now show unavailable states without reply/direct composer controls.
