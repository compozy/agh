# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Plan task: produced the QA test plan, four regression suites, traceability matrix, and 97 manual test cases under `.compozy/tasks/tools-registry/qa/`. No execution; task_16 owns execution.

## Important Decisions

- Reused `qa-output-path = .compozy/tasks/tools-registry` for both this task and task_16 so plans and execution evidence share `qa/`.
- Reserved subdirectories `test-plans/`, `test-cases/`, `issues/`, `screenshots/`, `logs/`, `traces/`, `fixtures/` so task_16 does not redefine paths.
- Defined a redaction sentinel set (`mcp:test:bearer:OAUTHTOKEN_v1`, `mcp:test:refresh:REFRESHTOKEN_v1`, `BIND_NONCE_v1_TESTONLY`, `APPROVAL_TOKEN_v1_TESTONLY`, `CLAIM_TOKEN_v1_TESTONLY`, `tools.sensitive.field:LEAK_v1`, etc.) and a sentinel-scan command in `security-redaction-regression.md`. Task 16 must seed these via a fixture harness before security execution.
- Smoke lane = 14 P0 cases enforcing the highest-impact safety invariants (1, 4, 7, 11, 13, 16, 21, 27) plus codegen drift / `make verify`.
- Targeted lane is keyed off touched-surface inclusion map so per-change runs do not rerun the full matrix.
- Full lane covers all P0 + P1 + P2 sampling; required pre-release.
- Security/redaction lane is mandatory pre-release with sentinel scan as the pass gate.

## Learnings

- Tasks 13 and 14 are listed as `pending` in `_tasks.md` but their individual task files are `status: completed`; the QA plan covers their surfaces either way because it works from the task file scope, not from `_tasks.md` status.
- Per-task memory files contain durable context that informs which automation already exists vs which is `Missing` per case (e.g. dispatch ordering tests exist; concurrency stress tests are largely missing).
- Hosted MCP launch records and CLI/HTTP/UDS approval tokens both live only in daemon memory; daemon restart invalidates them — case TC-SEC-011 step 7 verifies this explicitly.
- Hosted MCP request deadline must be ≥ `approval_timeout_seconds + 5s` (Safety Invariant 25) — encoded into TC-FUNC-048.

## Files / Surfaces

Files written under `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/tools-registry/qa/`:

- `test-plans/tool-registry-test-plan.md`
- `test-plans/smoke-regression.md`
- `test-plans/targeted-regression.md`
- `test-plans/full-regression.md`
- `test-plans/security-redaction-regression.md`
- `test-plans/traceability-matrix.md`
- `test-cases/TC-SEC-001..014.md` (14 files)
- `test-cases/TC-FUNC-001..058.md` (58 files)
- `test-cases/TC-INT-001..016.md` (16 files)
- `test-cases/TC-UI-001..006.md` (6 files)
- `test-cases/TC-PERF-001..003.md` (3 files)

Total: 97 manual test cases + 6 plan documents.

Reserved (empty) directories: `qa/issues/`, `qa/screenshots/`, `qa/logs/`, `qa/traces/`, `qa/fixtures/`.

## Errors / Corrections

- None.

## Ready for Next Run

- task_16 can consume `traceability-matrix.md` directly to drive execution priority (P0 first, then P1).
- task_16 must seed `qa/fixtures/redaction-sentinels.json` with the sentinel set defined in `security-redaction-regression.md` before running security cases.
- task_16 should append all command outputs into `qa/logs/<lane>/<TC-ID>.log`, screenshots into `qa/screenshots/<flow>/<viewport>/<state>.png`, and Playwright/browser traces into `qa/traces/<flow>/`.
- task_16 must write `qa/verification-report.md` with manifest path, lab root, runtime home, base URL, provider homes, command evidence, and final `make verify` result.
