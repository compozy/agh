# Hermes Hardening QA Test Plan

**qa-output-path:** `.compozy/tasks/hermes`
**Artifact root:** `.compozy/tasks/hermes/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

## Executive Summary

Hermes hardening spans durable persistence, observability, ACP lifecycle diagnostics, automation scheduling, MCP auth, process ownership, memory visibility, setup, environment handling, release packaging, web contracts, and site documentation. This plan turns tasks 01-09 into an execution-ready QA matrix for task_11.

The primary objective is to prove the hardening invariants selected in the Hermes TechSpec without relying on generic smoke checks. Every P0/P1 case must produce evidence through at least one real seam: SQLite state, daemon runtime, CLI output, HTTP/API/SSE payloads, web rendering or typed contracts, or public documentation.

Key risks:

- Restart-sensitive behavior may pass unit tests but fail when durable state and daemon boot recovery interact.
- Secret-bearing MCP, config, and environment surfaces may leak through secondary outputs such as logs, fixtures, settings payloads, CLI JSON, or docs examples.
- Web and site surfaces can drift from generated contracts after backend tasks change DTOs.
- Release config validation can be weaker locally because this repository uses GoReleaser Pro features; task_11 must combine local integrity tests with CI dry-run expectations.

## Objectives

- Prove persistence migrations are ordered, idempotent, and rollback-safe for global and session databases.
- Prove observability retention and health payloads expose persistence, retention, lifecycle, agent probe, automation, and memory signals without deleting required debugging state.
- Prove ACP/session lifecycle failures persist typed `failure.kind`, redacted summaries, crash bundle paths, and API/SSE/CLI-visible diagnostics.
- Prove durable automation scheduler state advances before dispatch, survives daemon restart without duplicate fires, applies `skip_missed`, and separates delivery errors from run errors.
- Prove MCP OAuth 2.1 PKCE token lifecycle, refresh, logout, and redaction boundaries across config, API, CLI, settings, and logs.
- Prove skill and managed extension path loading rejects symlink escapes.
- Prove shared process registry checkpointing, PID/start-time reconciliation, stale cleanup, and scoped interrupts across ACP terminals, environment terminals, hooks, extensions, and shared subprocess helpers.
- Prove memory health/history CLI and API visibility while runtime prompt assembly remains unchanged by future context-ref/provider-hook interfaces.
- Prove CLI setup/config lifecycle, `.env` repair, extension `requires_env`, release packaging trust artifacts, web generated types/settings pages, and site docs all match the final behavior.

## Scope

In scope:

- Backend Go packages changed by tasks 01-09: `internal/store`, `internal/retry`, `internal/observe`, `internal/session`, `internal/acp`, `internal/automation`, `internal/mcp/auth`, `internal/toolruntime`, `internal/memory`, `internal/config`, `internal/extension`, `internal/api/contract`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/cli`, `internal/daemon`, and settings/resource helpers.
- CLI commands: `agh observe health`, `agh session status/list/events`, `agh automation jobs/runs`, `agh mcp auth login/status/logout`, `agh memory health/history`, `agh config show/list/get/set/path/validate/check/edit`, `agh install/update/uninstall/completion`, and extension list/status/install flows.
- HTTP/API/SSE seams: observe health, session DTOs and SSE terminal events, automation job/run payloads, settings/MCP/extension payloads, memory health/history endpoints, and generated OpenAPI/TypeScript contracts.
- Web surfaces: daemon health adapters/fixtures, session failure rendering, automation scheduler and run history panels, settings MCP/auth and hooks/extensions pages, generated OpenAPI types, settings API adapters, and focused route/hook/component tests.
- Site docs: runtime API reference, observe health CLI reference, automation jobs/runs docs, session lifecycle diagnostics, MCP JSON/OAuth docs, memory CLI/API docs, config/env/install docs, extension install/status docs, operations daemon docs, release install docs, and related navigation/source tests.

Out of scope for task_10:

- Executing live backend, CLI, API, browser, release, or docs flows.
- Fixing production bugs discovered only by live execution. Those belong to task_11 with `systematic-debugging` and `no-workarounds`.
- Runtime prompt integration for memory context references and provider hooks. ADR-005 explicitly defers this behavior.
- Compatibility with old alpha state beyond the migrations and greenfield behavior defined by tasks 01-09.

## Environment Matrix

| Environment | Purpose | Required Evidence In Task 11 |
|-------------|---------|------------------------------|
| macOS local dev, `AGH_HOME` in `t.TempDir()` or temp shell home | Primary CLI, daemon, SQLite, process, and web verification | Command logs under `qa/logs/`, isolated state paths recorded in `qa/verification-report.md` |
| Linux CI-compatible shell | Release config and package target sanity where available | `make verify`, release config integrity test, and CI GoReleaser Pro dry-run reference |
| SQLite temp global/session DBs | Migration, retention, automation cursor, MCP token, process registry, and memory operation persistence | DB inspection logs or test output proving durable rows and restart behavior |
| Mock ACP/OAuth/process providers | Lifecycle, MCP auth, and process registry edge cases without real provider secrets | Mock server logs, redacted payload samples, crash bundle evidence |
| Browser desktop 1280px, tablet 768px, mobile 375px | Web settings/session/automation contract rendering and docs page review | Screenshots under `qa/screenshots/`, route/test logs under `qa/logs/` |
| Site docs build | Public documentation consistency and broken-link/source validation | `packages/site` typecheck/build/source test output |

## Artifact Layout

Task_11 must keep the same `qa-output-path=.compozy/tasks/hermes` and write only under this root:

| Path | Owner | Purpose |
|------|-------|---------|
| `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-test-plan.md` | task_10 | Feature QA plan |
| `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-regression.md` | task_10 | Smoke, targeted, and full regression lanes |
| `.compozy/tasks/hermes/qa/test-cases/TC-*.md` | task_10 | Manual execution cases seeded by this plan |
| `.compozy/tasks/hermes/qa/issues/BUG-*.md` | task_11 if needed | Structured bug reports tied to a TC ID |
| `.compozy/tasks/hermes/qa/screenshots/<TC-ID>/...` | task_11 | Browser or docs screenshots for web/site evidence |
| `.compozy/tasks/hermes/qa/logs/<TC-ID>/...` | task_11 | Command, daemon, mock server, test, and build logs |
| `.compozy/tasks/hermes/qa/verification-report.md` | task_11 | Final execution report from `qa-execution` |

## Test Strategy

1. Smoke first. Run the P0 cases that establish state boot, health, session diagnostics, automation restart safety, MCP auth redaction, process registry safety, and environment diagnostics. Any P0 failure blocks deeper execution.
2. Targeted lanes next. Execute each domain lane with the relevant focused Go tests, CLI/API checks, web tests, and docs checks listed in the regression suite.
3. Full regression last. Run `make verify`, required web/site gates, and the full manual case matrix after the final fix set.
4. Real seams over parser-only checks. Parser/config tests are acceptable only when paired with surfaced CLI/API/web/docs evidence for the same invariant.
5. Redaction checks are mandatory wherever tokens, client secrets, authorization codes, PKCE verifiers, `.env` values, paths, or crash summaries can surface.

## Entry Criteria

- Tasks 01-09 are completed in tracking and their implementation commits are present.
- The working tree state is captured before task_11 execution.
- `qa-output-path=.compozy/tasks/hermes` is passed unchanged to `qa-execution`.
- Test fixtures use isolated temp homes/workspaces and never rely on private local credentials.
- Web and site dependencies are installed and the repository verification contract is known from the Makefile.
- Any task_11 bug fix starts from a failing reproduction and adds durable regression coverage before the final gate.

## Exit Criteria

- All P0 cases pass.
- At least 90% of P1 cases pass; any P1 exception must have a `BUG-*.md` issue with severity, impact, workaround, and fix owner.
- No critical security, data loss, duplicate automation dispatch, unrelated process kill, secret leak, or docs contract mismatch remains open.
- `make verify` passes after the last task_11 change.
- Required web/site gates pass for touched surfaces: `make web-lint`, `make web-typecheck`, `make web-test` or focused Vitest lanes, plus `packages/site` typecheck/build/source validation where docs are touched.
- `.compozy/tasks/hermes/qa/verification-report.md` cites executed commands, evidence paths, pass/fail status, open bugs, and final verdict.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Scheduler cursor advances but duplicate fire still occurs after restart | Medium | Critical | P0 automation restart case with persisted `last_fire_id`, run rows, and post-restart dispatch count evidence |
| MCP or config output leaks token material in secondary surfaces | Medium | Critical | P0 redaction case checks CLI human/JSON, API/settings payloads, logs, fixtures, and docs examples |
| Process registry kills unrelated PID after restart | Medium | Critical | P0 registry case requires PID/start-time mismatch evidence and zero signal to stale/reused PID |
| Crash bundles contain secrets or unbounded stderr | Medium | High | P0 lifecycle case validates redaction and bounded bundle contents |
| Memory history is mistaken for runtime prompt context | Low | High | P1 memory case proves CLI/API visibility and separately verifies no prompt assembly integration |
| Web generated types drift from API payloads | Medium | High | P1 web contract case includes codegen check, typecheck, adapter tests, and fixture review |
| Site docs explain old navigation or old fields | Medium | Medium | P1 docs case checks changed runtime docs, CLI references, generated source tests, and docs build |
| Local GoReleaser OSS cannot validate Pro config | High | Medium | P1 release case uses Go YAML integrity test locally and records CI Pro dry-run as required release evidence |

## Traceability Matrix

| Case | Priority | Surface | Proves | Source |
|------|----------|---------|--------|--------|
| TC-INT-001 | P0 | Backend persistence/retry | Migration ordering/idempotence/rollback and shared retry cancellation/jitter | Task 01, TechSpec issues 10/11/17, ADR-001 |
| TC-INT-002 | P1 | Observe/API/CLI | Retention sweep, disabled retention, and typed health payloads | Task 02, TechSpec issue 17, ADR-001 |
| TC-INT-003 | P0 | ACP/session/API/SSE/CLI | Failure kinds, crash bundles, probes, and redaction | Task 03, TechSpec issues 14/15/16, ADR-001 |
| TC-INT-004 | P0 | Automation/store/API/CLI/web/docs | Cursor-before-dispatch, restart duplicate prevention, `skip_missed`, delivery errors | Task 04, TechSpec issues 20/21/22/25, ADR-002 |
| TC-SEC-001 | P0 | MCP auth/API/CLI/settings | OAuth PKCE lifecycle, durable token storage, refresh/logout, redaction | Task 05, TechSpec issue 27, ADR-003 |
| TC-SEC-002 | P0 | Skills/extensions/filesystem | Symlink escape rejection for skills and managed extensions | Task 05, TechSpec issue 28, ADR-003 |
| TC-INT-005 | P0 | Process registry/ACP/hooks/extensions | Checkpointing, PID validation, stale cleanup, scoped interrupts | Task 06, TechSpec issues 29/30, ADR-004 |
| TC-FUNC-001 | P1 | Memory CLI/API | Health/history visibility, filters, redaction, prompt isolation | Task 07, TechSpec issues 33/34/35/60, ADR-005 |
| TC-FUNC-002 | P1 | CLI setup/config | Config commands, redaction, install/update/uninstall/completion, `AGH_MANAGED` | Task 08, TechSpec issues 36/37/39/40/41/42/43 |
| TC-FUNC-003 | P0 | Environment/extensions/API/web | `.env` repair, `requires_env`, `missing_env`, no value leaks | Task 09, TechSpec issues 57/59 |
| TC-REG-001 | P1 | Release infra/site docs | Homebrew cask, nFPM `deb`/`rpm`, checksums, signing, SBOMs | Task 09, TechSpec issue 59 |
| TC-UI-001 | P1 | Web automation | Scheduler and run delivery diagnostics render from updated contracts | Task 04, task_10 extra coverage notes |
| TC-UI-002 | P1 | Web settings | MCP auth and extension env diagnostics are redacted and actionable | Tasks 05/09, ADR-003 |
| TC-UI-003 | P1 | Web generated contracts | Memory, health, session failure, automation, and extension DTOs stay type-safe | Tasks 02/03/04/07/09 |
| TC-REG-002 | P1 | packages/site docs | Docs/navigation cover all operator-visible hardening behavior | Tasks 02-09, ADR-001 through ADR-005 |

## Web And Site Verification Requirements

| Task | Required web verification | Required site verification |
|------|---------------------------|----------------------------|
| 01 | Confirm no web typed-client/settings/story changes were needed for internal migration/retry foundations. | Confirm no public docs required beyond future QA traceability. |
| 02 | Daemon health adapters/fixtures tolerate `health.persistence` and `health.retention`. | Observe health docs describe persistence and retention fields. |
| 03 | Session failure DTOs and daemon health fixtures render failure kind, summary, crash bundle path, and agent probes without secret leakage. | Session lifecycle and observe health docs describe failure diagnostics and crash bundles. |
| 04 | Automation detail and run history render scheduler cursor and delivery-error fields. | Automation jobs/runs and observe health docs describe durable scheduler state and delivery errors. |
| 05 | Settings MCP server fixtures/types expose redacted `auth_status` only; remote auth is not token-editable. | MCP JSON and CLI auth docs explain OAuth PKCE and token-free config. |
| 06 | Confirm no new web UI contract is required for process registry unless interrupt payloads surface; session interruption must not regress. | Operations daemon docs explain tool process recovery, stale PID safety, and scoped interrupts. |
| 07 | Generated OpenAPI types expose `getMemoryHealth` and `listMemoryHistory`; no settings UI behavior is required. | Memory CLI/API docs describe health/history and clarify history is not prompt context. |
| 08 | Settings/config clients remain compatible with CLI lifecycle and redaction semantics. | Config, install, update, uninstall, and completion references match command behavior. |
| 09 | Settings hooks/extensions page and generated contracts show `requires_env` and `missing_env` names only. | Config/env, extension install/status, and installation docs cover `.env` repair, environment requirements, and package trust artifacts. |

## Deliverables

- This feature QA plan.
- `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-regression.md`
- Manual test cases under `.compozy/tasks/hermes/qa/test-cases/`
- Reserved `issues/`, `screenshots/`, and `logs/` evidence locations for task_11.
- A task_11-ready traceability matrix and P0/P1 execution order.
