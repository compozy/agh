# Orchestration Improvements Regression Suite

## Purpose

This suite turns the QA plan into an executable release sequence for `qa-execution`. Smoke checks
prove that the lab is ready. Release-grade checks prove that AGH can run the full orchestration,
review, continuation, notification, web, and docs loop with durable state and no authority leaks.

## Suite Tiers

| Tier | Duration Target | Frequency | Cases | Purpose |
| --- | --- | --- | --- | --- |
| Smoke readiness | 15-30 minutes | Before every QA run | SMOKE gates below | Stop early on broken lab or generated artifacts |
| P0 behavioral | 60-120 minutes | Every release candidate | TC-SCEN-001, TC-INT-001, TC-INT-002, TC-INT-003, TC-SEC-001 | Release-blocking behavior |
| P1 cross-surface | 30-60 minutes | Every release candidate | TC-UI-001, TC-REG-001, TC-PERF-001 | Operator trust and drift protection |
| P2 exploratory | 30 minutes | When P0/P1 pass | Extra malformed input and resize probes | Discover lower-probability regressions |

## Execution Order

1. Create a fresh isolated QA lab with `agh-qa-bootstrap`.
2. Run smoke readiness:
   - Build the AGH binary used by the lab.
   - Start daemon with isolated `AGH_HOME`.
   - Confirm HTTP health, UDS access, and web proxy target.
   - Run `compozy tasks validate --name orch-improvs --format json`.
   - Run `make codegen-check`.
   - Run site `source:generate`, `content:generate`, `typecheck`, and focused docs tests.
3. Execute P0 behavioral cases in this order:
   - TC-INT-001
   - TC-INT-002
   - TC-INT-003
   - TC-SEC-001
   - TC-SCEN-001
4. Execute P1 cross-surface cases:
   - TC-UI-001
   - TC-REG-001
   - TC-PERF-001
5. File bug reports for every reproduced failure under `qa/issues/`.
6. Fix production root causes only when task 32 execution reproduces a bug.
7. Rerun affected focused cases and then the full release gates.
8. Write `qa/verification-report.md`.

## Smoke Readiness Gates

Smoke gates are not release proof. They only decide whether the QA run can proceed.

- `compozy tasks validate --name orch-improvs --format json`
- `git diff --check`
- `make codegen-check`
- `make bun-lint`
- `make bun-typecheck`
- `make bun-test`
- `make web-build`
- `make fmt`
- `make lint`
- Focused package tests for the surfaces touched by any bug fix before broad reruns.

## Release Gates

These gates are required after P0/P1 execution and after every production fix:

- `make test-e2e-runtime`
- `make test-e2e-web`
- `make verify`

If either e2e target is impossible because the local environment lacks a required provider,
browser, or credential, the verification report must include the exact missing prerequisite and
the nearest successful substitute. The final release claim remains conditional until the missing
P0 evidence is supplied.

## Pass, Fail, And Conditional Rules

### PASS

- All P0 behavioral cases pass.
- At least 90 percent of P1 assertions pass.
- No unresolved critical or high bug remains.
- Final `make verify` passes after all fixes.

### FAIL

- Any P0 case fails.
- Any data loss, security boundary failure, raw claim-token leak, unbound review verdict, or missed
  accepted-final notification is reproduced.
- Contract/codegen drift remains after regeneration.
- Final `make verify` fails.

### CONDITIONAL

- A P1-only failure has a documented fix plan and does not affect P0 behavior.
- A live provider path is unavailable, but deterministic runtime evidence passes. This cannot be
  used for final release confidence without explicit acknowledgment in the verification report.

## Regression Hot Spots

- Review continuation atomicity: verdict persistence and continuation-run creation must not split.
- Duplicate/replayed delivery ids: idempotent replay must not create extra continuation runs.
- `tasks.current_run_id`: profile mutation locks and projection clearing must track real run state.
- `Last-Event-ID`: header precedence must not regress to query fallback on value `0`.
- Named SSE events: browser listeners must receive AGH task event names, not just `onmessage`.
- Bridge cursor advancement: delivery failure must leave cursor sequence unchanged.
- Generated surfaces: OpenAPI, generated TypeScript, generated CLI reference, site docs, and web
  adapters must agree on DTO names and route shapes.
- Authority boundaries: scheduler, channel routing, web UI, and bridge notifications cannot record
  review verdicts or mutate token-fenced runs.

## Required Evidence Index

The execution report must include these identifiers for traceability:

- QA bootstrap manifest path.
- AGH lab root and runtime home.
- Daemon base URL and UDS path.
- Web base URL and `AGH_WEB_API_PROXY_TARGET`.
- Provider mode and provider home.
- Task id, first run id, review id, continuation run id, final run id.
- Bridge subscription id, cursor id, last delivered event sequence.
- Browser screenshot or trace path for the orchestration tab.
- Commands and exit statuses for release gates.

