# Orchestration Improvements QA Test Plan

## Executive Summary

This plan defines the mandatory behavior-first QA coverage for the orchestration-improvements
program. The implementation spans task execution profiles, review gates, reviewer-bound native
tools, task context bundles, notification cursors, bridge subscriptions, SSE resume behavior, web
orchestration UI, site documentation, and institutional memory. The release decision must be based
on real runtime evidence, not isolated unit or mock-only confidence.

The highest-risk release path is the full autonomous task loop:

1. An operator configures task orchestration and review defaults.
2. A task execution profile selects the worker, model/provider, sandbox, and claim eligibility.
3. A worker claims and completes a run.
4. A post-terminal review request routes to an eligible reviewer.
5. The reviewer rejects incomplete work through the reviewer-bound native tool.
6. A continuation run receives missing-work guidance without raw claim-token exposure.
7. The continuation is completed and approved.
8. A bridge subscription receives exactly the accepted final terminal notification after durable
   replay.
9. CLI, HTTP, UDS, native tools, web UI, OpenAPI, generated TypeScript, and docs all report the
   same durable state.

## Scope

### In Scope

- Runtime config lifecycle for `[task.orchestration]`,
  `[task.orchestration.profile]`, and `[task.orchestration.review]`.
- GlobalDB migrations and fresh-schema parity for task orchestration profile tables,
  review-gate tables, current-run projections, notification cursors, and bridge subscriptions.
- Task execution profile CRUD, validation, active-run mutation rejection, selector precedence,
  worker selection, provider/model gates, participant policy limits, and sandbox
  `inherit|none|ref` behavior.
- Review request, reviewer routing, reviewer session binding, verdict idempotency,
  rejected-review continuation runs, approved finalization, blocked/error/timeout handling,
  no-route diagnostics, and retry/circuit behavior.
- Native tools for execution profiles and run reviews, including reviewer-bound
  `submit_run_review`.
- HTTP, UDS, CLI, OpenAPI, generated TypeScript, and generated CLI-reference parity.
- Task context bundle redaction, recent event sequence projection, continuation guidance, and
  session overlay composition.
- Notification cursor monotonic advancement, idempotent replay, bridge subscription diagnostics,
  accepted-final delivery semantics, and delete lifecycle.
- Task SSE replay and browser EventSource resume behavior, including named task event listeners.
- Web orchestration tab and run review surfaces.
- Site docs for execution profiles, review gates, notification cursors, bundled skills, config,
  and generated CLI references.
- Workflow memory and `docs/_memory` lessons/glossary alignment.

### Out of Scope

- Generic notification fan-out beyond bridge task terminal notifications.
- Bridge-owned review verdicts, multi-reviewer quorum, and channel messages as verdict authority.
- Compatibility with pre-alpha local state or old route aliases.
- Private frontend plugin SDKs.
- Manual edits to generated OpenAPI, generated TypeScript, or generated CLI-reference output.

## Behavioral Scenario Charter

### Operator Intent

The operator wants AGH to run task work with a configured worker profile, route completion to a
reviewer, continue after rejection with explicit missing work, approve final work, and notify a
bridge exactly once. The operator also needs to inspect and manage this state through CLI, HTTP,
UDS, native tools, and the web UI.

### Startup Situation

- Fresh isolated QA lab with unique `AGH_HOME`, daemon port, web proxy target, provider home, and
  tmux-bridge socket path.
- Workspace config contains explicit task orchestration defaults and a review profile.
- At least one worker-capable agent and one reviewer-capable agent are available.
- Provider credentials are available for a live provider or native CLI provider. Deterministic
  driver runs may be used as smoke/preflight, but release-grade P0 evidence must include a live or
  native-provider-backed path unless the final report explicitly marks that path unavailable.

### Agent Roles

- Operator: creates task, profile, review policy, and bridge subscription; inspects state.
- Coordinator: starts or manages the task run and receives orchestration guidance.
- Worker: claims the task run and completes incomplete and corrected work.
- Reviewer: receives a bound review request and submits verdicts through the native tool.
- Bridge consumer: receives accepted-final terminal notifications and exposes cursor diagnostics.

### Expected Artifacts

- `bootstrap-manifest.json` with lab root, runtime home, provider home, base URL, web proxy target,
  ports, process ids, and cleanup instructions.
- Command transcript or structured evidence for CLI, HTTP, UDS, and native-tool calls.
- Persisted task, run, review, notification cursor, and bridge subscription records.
- Browser screenshots or trace for the web orchestration tab.
- Site/docs build and docs-content checks.
- `verification-report.md` from `qa-execution`.
- Bug reports under `qa/issues/` for any reproduced failure.

### Disruption Probes

- Restart daemon between review rejection and continuation claim.
- Drop or fail a bridge delivery once, then prove cursor does not advance until success.
- Replay review verdict with the same delivery id and then with a conflicting payload.
- Attempt review submission from an unbound session and from the original worker when prohibited.
- Reconnect the task stream with `Last-Event-ID: 0`, a non-zero header, a query seed, and malformed
  seed input.
- Attempt profile mutation while `tasks.current_run_id` is populated.
- Compare docs and generated references after codegen to detect stale claims.

## Test Strategy And Approach

Smoke readiness checks verify that the lab can start, schemas migrate, generated artifacts are in
sync, and the UI can load. Smoke checks are entry criteria only.

Release-grade evidence must execute P0 and P1 cases against real persisted state:

- P0 cases cover the end-to-end orchestration and review loop, migration/profile parity,
  review-tool authority, notification cursor delivery, and security/redaction boundaries.
- P1 cases cover web UI truthfulness, docs/contract drift, and SSE/concurrency behavior.
- P2 exploratory probes cover rare malformed inputs, browser resize/layout details, and operator
  diagnostics that do not block the core lifecycle.

Each executed case must record:

- The exact command, URL, browser action, or native-tool call.
- The state observed in at least two surfaces when parity is required.
- Persisted identifiers: task id, run id, review id, subscription id, cursor id, and event sequence.
- Failure mode and bug report path when behavior does not match expectations.

## Coverage Matrix

| Area | Tasks | Primary QA Cases | Required Evidence |
| --- | --- | --- | --- |
| Config defaults and validation | 01, 28, 30 | TC-INT-001, TC-REG-001 | Config load, invalid config rejection, docs parity |
| GlobalDB schema and migrations | 02, 03, 04, 06, 07 | TC-INT-001, TC-INT-003 | Fresh DB and migrated DB schema parity |
| Current run projection | 05, 24, 27 | TC-SCEN-001, TC-UI-001 | `current_run_id` set/cleared and web active-run lock |
| Execution profile authority | 08, 11, 12, 13, 16, 18 | TC-INT-001, TC-SCEN-001 | CLI/HTTP/UDS/native parity and session-start effect |
| Review gate authority | 09, 10, 15, 17, 19, 22 | TC-INT-002, TC-SCEN-001, TC-SEC-001 | Bound reviewer verdict and continuation lineage |
| Bundled skills and native tools | 14, 15, 16, 17, 29 | TC-INT-002, TC-SEC-001 | Instructional skills plus service-owned authority |
| Bridge notifications and cursors | 06, 07, 21, 25, 29 | TC-INT-003, TC-SCEN-001 | Durable replay, diagnostics, and accepted-final delivery |
| Context bundle and redaction | 23, 24 | TC-SEC-001, TC-SCEN-001 | No raw claim token and continuation guidance present |
| SSE and event sequence | 24, 26, 27 | TC-PERF-001, TC-UI-001 | `latest_event_seq`, seed precedence, named events |
| Web data and UI | 26, 27 | TC-UI-001, TC-SCEN-001 | Playwright/browser evidence against daemon-served UI |
| Site docs and CLI references | 28, 29, 30 | TC-REG-001 | Site generation/build and docs truth checks |
| QA handoff | 31, 32 | This plan, regression suite | Executable cases and final verification report |

## Environment Requirements

- macOS or Linux development host with Go, Bun, Playwright browser dependencies, SQLite, and the
  project toolchain installed.
- Fresh isolated AGH lab created by `agh-qa-bootstrap`.
- Unique daemon HTTP port and UDS path for this QA run.
- `AGH_WEB_API_PROXY_TARGET` derived from the bootstrap manifest when web QA runs against a
  non-default daemon port.
- Provider environment matching the selected provider contract:
  - Bound-secret or brokered providers use isolated provider homes from the bootstrap manifest.
  - `native_cli` providers with `home_policy = operator` preserve the operator login state.
- Browser validation through `browser-use:browser` when available, with the approved fallback
  recorded in the final report if unavailable.

## Entry Criteria

- `task_31` QA artifacts exist and pass structural checks.
- `compozy tasks validate --name orch-improvs --format json` passes.
- `git diff --check` passes.
- The latest implementation gate before QA has a passing `make verify`.
- The task executor has read `_techspec.md`, child TechSpecs, ADRs, task files, and workflow memory.

## Exit Criteria

- All P0 cases pass or have fixed bugs with rerun evidence.
- At least 90 percent of P1 cases pass, with no unresolved critical or high bug.
- Fresh and migrated schema checks both pass.
- CLI, HTTP, UDS, native-tool, OpenAPI, generated TypeScript, web, and site/docs surfaces match the
  same persisted state.
- `make test-e2e-runtime`, `make test-e2e-web`, and final `make verify` pass after any fixes made
  during task 32.
- `qa/verification-report.md` records the manifest path, lab root, runtime home, base URL, provider
  mode, executed cases, bug reports, and final gate evidence.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| Mock-only QA hides provider/session behavior | Medium | Critical | Require live or native provider evidence for P0 lifecycle |
| Review verdict authority leaks to channels or web | Medium | Critical | TC-INT-002 and TC-SEC-001 negative binding probes |
| Migration and fresh schema drift | Medium | High | TC-INT-001 runs both fresh and migrated homes |
| Cursor advances before confirmed bridge delivery | Medium | High | TC-INT-003 forced delivery failure and replay |
| Web stream misses named SSE events | Medium | High | TC-UI-001 and TC-PERF-001 named-event replay checks |
| Documentation overclaims non-existent controls | Medium | Medium | TC-REG-001 compares docs to generated/runtime surfaces |
| SQLite/race flake masks a real persistence bug | Low | High | Rerun focused package with `-race` and require final `make verify` |
| Provider credentials unavailable | Medium | High | Record availability; deterministic driver may preflight but cannot replace P0 release evidence |

## Timeline And Deliverables

- Task 31 deliverables:
  - `qa/test-plans/orch-improvs-test-plan.md`
  - `qa/test-plans/orch-improvs-regression-suite.md`
  - `qa/test-cases/TC-*.md`
  - `qa/issues/README.md`
  - `qa/screenshots/README.md`
- Task 32 deliverables:
  - Isolated bootstrap manifest.
  - Executed test evidence and screenshots/traces.
  - Bug reports for reproduced failures.
  - `qa/verification-report.md`.
  - Final gate evidence.

