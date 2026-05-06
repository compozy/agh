# Memory v2 QA Execution

## Scope

- Task: `task_26` Real-Scenario QA Execution.
- QA lab: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab`.
- QA output path: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab/qa-artifacts`.
- Manifest: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab/qa-artifacts/qa/bootstrap-manifest.json`.
- Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-2c1957d9ca82/runtime`.
- Base URL: `http://127.0.0.1:50979`.

## Result

- Verdict: PASS.
- Final report: `.compozy/tasks/mem-v2/qa/verification-report.md`.
- Mirrored report: `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab/qa-artifacts/qa/verification-report.md`.
- Strict QA audit: PASS with 0 blockers and 0 warnings.
- Audit reports:
  - `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab/qa-artifacts/qa/qa-audit-report.json`
  - `/Users/pedronauck/dev/qa-labs/agh-mem-v2-final-postfix-20260505-225249-638020-lab/qa-artifacts/qa/qa-audit-report.md`

## Gates

- `make test-e2e-runtime > .compozy/tasks/mem-v2/qa/logs/final-postfix/final-make-test-e2e-runtime-rerun5.log 2>&1` passed.
- `make test-e2e-web > .compozy/tasks/mem-v2/qa/logs/final-postfix/final-make-test-e2e-web-rerun1.log 2>&1` passed with 19 Playwright tests.
- `make lint > .compozy/tasks/mem-v2/qa/logs/final-postfix/final-make-lint-after-verify-fail-rerun1.log 2>&1` passed after fixing lint findings from the first full verify attempt.
- `make verify > .compozy/tasks/mem-v2/qa/logs/final-postfix/final-make-verify-rerun1.log 2>&1` passed with 334 Bun test files, 2150 Bun tests, 8381 Go tests, Go lint `0 issues`, and package boundaries OK.
- `make verify > .compozy/tasks/mem-v2/qa/logs/final-postfix/final-make-verify-post-state.log 2>&1` passed after `task_26`, workflow memory, report, and loop state updates.

## Issues Fixed

- `BUG-001`: Public Memory transports could not resolve stable workspace identity.
- `BUG-002`: Concurrent native Memory writes could expose atomic temp files to controller targeting.
- `BUG-003`: Memory search API hid curated filenames and skipped realistic two-token recall queries.
- `BUG-004`: Recall observability events had empty summaries.
- `BUG-005`: Memory extractor failed queued post-stop turns for provider-backed sessions.
- `BUG-006`: ACP mock fixtures did not match augmented Memory v2 prompts.
- `BUG-007`: Permission fixtures used deny-all while expecting operator approval.
- `BUG-008`: Transcript query could reuse a recorder while its database was closing.
- `BUG-009`: Bridge ingress returned before the routed prompt became observable.
- `BUG-010`: UDS observe parity expected the pre-Memory-v2 augmenter contract.
- `BUG-011`: Resource kernel write transactions surfaced SQLITE_BUSY during browser runtime seeding.
- `BUG-012`: Bridge E2E used an ambiguous global Close button selector.

## Handoff

- `task_26` is complete after report, issue artifacts, strict audit, runtime E2E, web E2E, and full verify evidence.
- `state.yaml` should be advanced with `--task-completed task_26 --qa-execution-done --verify-pass`.
- The next loop phase after state update should be Phase D CodeRabbit review.
