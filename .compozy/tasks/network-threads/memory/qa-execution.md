# Task Memory: qa-execution

## Objective Snapshot

- Execute Task 19 QA for `.compozy/tasks/network-threads` using `qa-output-path=.compozy/tasks/network-threads`.
- Consume the Task 18 QA plan and test cases, run realistic CLI/API/Web operator journeys, fix root-cause defects, and persist evidence under `.compozy/tasks/network-threads/qa/`.
- Completion requires final `make verify` PASS and a verification report under `qa/verification-report.md`.

## Important Decisions

- Created a fresh isolated QA bootstrap instead of reusing an older lab.
- Kept repository verification gates clean of the bootstrap `AGH_HOME`; the isolated env is only sourced for daemon, CLI/API, and Web scenario commands.
- Used `agent-browser` as the browser fallback because `browser-use` was selected in the manifest but the required Node REPL browser tool was not available in the active tool list.
- Treated both confirmed failures as production defects and fixed root causes rather than weakening tests or documenting around the behavior.

## Learnings

- `make verify` must not be run from a shell that sourced the isolated `bootstrap.env`; doing so intentionally changes `AGH_HOME` and invalidates env-isolation tests.
- Session event query paths need to coordinate with recorder finalization. Returning an active recorder while finalization is closing the DB can surface `sql: database is closed`.
- Network detail UI must not show thread/direct composers until the corresponding conversation detail exists. Missing-conversation 4xx responses are not transient and should not wait on default query retries.
- `agent-browser screenshot` did not honor the requested target path in this run; screenshots were copied from `~/.agent-browser/tmp/screenshots/` into the QA output directory.

## Files / Surfaces

- QA bootstrap and run evidence:
  - `.compozy/tasks/network-threads/qa/bootstrap-manifest.json`
  - `.compozy/tasks/network-threads/qa/bootstrap.env`
  - `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/`
  - `.compozy/tasks/network-threads/qa/screenshots/`
- Bug reports:
  - `.compozy/tasks/network-threads/qa/bug-reports/BUG-001-session-event-query-finalization-race.md`
  - `.compozy/tasks/network-threads/qa/bug-reports/BUG-002-web-network-missing-conversation-state.md`
- Runtime fixes:
  - `internal/session/query.go`
  - `internal/session/query_test.go`
  - `internal/store/sessiondb/session_db.go`
  - `internal/store/sessiondb/session_db_extra_test.go`
- Web fixes:
  - `web/src/systems/network/lib/query-options.ts`
  - `web/src/systems/network/lib/query-options.test.ts`
  - `web/src/systems/network/components/empty-states/conversation-error.tsx`
  - `web/src/systems/network/components/directs/direct-room.tsx`
  - `web/src/systems/network/components/directs/direct-room.test.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay.tsx`
  - `web/src/systems/network/components/thread-overlay/thread-overlay.test.tsx`
  - `web/src/systems/network/hooks/use-direct-room.ts`
  - `web/src/systems/network/hooks/use-thread-overlay.ts`

## Errors / Corrections

- Invalid baseline gate: first `make verify` was run after sourcing `bootstrap.env`; reran clean and recorded the invalid harness result in `execution-notes.md`.
- CLI/API harness bugs were fixed before product assertions: zsh `path`/`status` special variables, session ID parsing, explicit `run_capture` failure propagation, daemon-start idempotence, and normative `receipt`/`trace` body shapes.
- BUG-001 fixed the session event query/finalization race and was verified with targeted race tests plus `make test-e2e-runtime`.
- BUG-002 fixed missing-conversation UI and 4xx detail retry behavior, verified with targeted Vitest, Web guardrails, `make test-e2e-web`, and browser snapshots.

## Ready for Next Run

- Task 19 QA execution is complete.
- Final report: `.compozy/tasks/network-threads/qa/verification-report.md`.
- Final gate: `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/final-make-verify.log` passed.
- Expected next phase from `detect-phase.py`: CodeRabbit review round 001 (`phase=D action=coderabbit_round`).

## QA Artifacts Produced

- `.compozy/tasks/network-threads/qa/verification-report.md`
- `.compozy/tasks/network-threads/qa/behavioral-scenario-charter.md`
- `.compozy/tasks/network-threads/qa/bootstrap-manifest.json`
- `.compozy/tasks/network-threads/qa/bootstrap.env`
- `.compozy/tasks/network-threads/qa/bug-reports/BUG-001-session-event-query-finalization-race.md`
- `.compozy/tasks/network-threads/qa/bug-reports/BUG-002-web-network-missing-conversation-state.md`
- `.compozy/tasks/network-threads/qa/runs/20260505T170631Z-bootstrap/bootstrap-output.txt`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/baseline-make-verify-clean.log`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/cli-api-scenario.zsh`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/scenario.env`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/test-e2e-runtime-after-fix.log`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/test-e2e-web-after-ui-fix.log`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/legacy-scan-classification.md`
- `.compozy/tasks/network-threads/qa/runs/20260505T170658Z-execution/final-make-verify.log`
- `.compozy/tasks/network-threads/qa/screenshots/post-final-network-thread-detail.png`
- `.compozy/tasks/network-threads/qa/screenshots/post-final-network-direct-detail.png`
- `.compozy/tasks/network-threads/qa/screenshots/post-final-network-missing-thread.png`
- `.compozy/tasks/network-threads/qa/screenshots/post-final-network-missing-direct.png`
