# Task Memory: task_31

## Status

Completed 2026-05-05 through local `qa-report` planning work.

## Objective Snapshot

- Produce the mandatory QA report for the full orchestration-improvements program under
  `.compozy/tasks/orch-improvs/qa/`.
- Planning scope had to cover every public surface touched by tasks 01-30 and include
  real-scenario, e2e, negative, concurrency, migration, security/redaction, performance, and
  contract drift cases.
- This task did not execute the QA lab. Real execution belongs to `task_32`.

## Important Decisions

- Created a dedicated QA artifact tree under `qa/test-plans/`, `qa/test-cases/`, `qa/issues/`,
  and `qa/screenshots/` so task 32 can consume the plan without mixing it with historical spec
  peer-review artifacts already present in `qa/`.
- Required release-grade P0 evidence to include a live or native-provider-backed path where
  credentials are available. Deterministic driver runs are allowed as smoke/preflight evidence but
  are not enough for final confidence by themselves.
- Separated smoke readiness from release evidence. Smoke gates prove the lab and generated
  artifacts are runnable; P0/P1 cases prove behavior across real persisted state.
- Kept bug reports and browser screenshots empty during planning, with README handoffs explaining
  that task 32 owns reproduced bugs and visual execution artifacts.

## QA Artifacts

- `qa/test-plans/orch-improvs-test-plan.md`
- `qa/test-plans/orch-improvs-regression-suite.md`
- `qa/test-cases/TC-SCEN-001-full-orchestration-review-loop.md`
- `qa/test-cases/TC-INT-001-config-schema-migration-and-profile-parity.md`
- `qa/test-cases/TC-INT-002-review-gate-contract-and-continuation.md`
- `qa/test-cases/TC-INT-003-notification-cursor-and-bridge-delivery.md`
- `qa/test-cases/TC-UI-001-web-orchestration-tab-operator-truth.md`
- `qa/test-cases/TC-REG-001-generated-contracts-cli-site-docs-drift.md`
- `qa/test-cases/TC-SEC-001-claim-token-redaction-and-reviewer-boundary.md`
- `qa/test-cases/TC-PERF-001-sse-query-churn-and-cursor-replay.md`
- `qa/issues/README.md`
- `qa/screenshots/README.md`

## Coverage Map

| QA case | Primary coverage |
| --- | --- |
| TC-SCEN-001 | End-to-end profile -> worker -> review rejection -> continuation -> approval -> bridge delivery |
| TC-INT-001 | Config defaults/validation, fresh and migrated schema, execution profile parity |
| TC-INT-002 | Review request/routing/binding/verdict/continuation authority and idempotency |
| TC-INT-003 | Notification cursor, bridge subscription diagnostics, accepted-final delivery and replay |
| TC-UI-001 | Web Orchestration tab and run review surfaces against daemon truth |
| TC-REG-001 | OpenAPI, generated TypeScript, generated CLI reference, site docs, lessons, glossary drift |
| TC-SEC-001 | Claim-token redaction and reviewer-bound native tool authority |
| TC-PERF-001 | SSE seed precedence, named event listeners, cursor replay, and UI refetch churn |

## Verification Evidence

- `compozy tasks validate --name orch-improvs --format json` PASS, `scanned: 32`.
- `git diff --check` clean.
- Structural QA checks PASS: every `TC-*.md` case contains priority, objective, preconditions,
  test steps with `Expected`, behavioral evidence, and disruption probes.
- Marker scan PASS: no unfinished-work markers in `qa/`.
- `make verify` PASS: Bun/Vitest monorepo `339 passed (339)` files / `2206 passed (2206)` tests,
  web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 11.125s`, package
  boundaries `OK`.

## Ready for Next Run

- Task 31 complete. Next loop step is `task_32` (Real-Scenario QA Execution).
- `task_32` must activate `agh-qa-bootstrap`, `real-scenario-qa`, `qa-execution`, and
  `agh-worktree-isolation`, create or reuse only a same-run bootstrap manifest, execute the P0/P1
  cases, write bug reports for reproduced failures, write `qa/verification-report.md`, and then
  run the final verification gates.
