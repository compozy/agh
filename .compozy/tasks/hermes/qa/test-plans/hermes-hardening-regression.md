# Hermes Hardening Regression Suite

**qa-output-path:** `.compozy/tasks/hermes`
**Artifact root:** `.compozy/tasks/hermes/qa/`
**Status:** Planning complete, not executed
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

## Execution Rules

- Task_11 must activate `qa-execution` with `qa-output-path=.compozy/tasks/hermes`.
- Execute smoke first. If any smoke P0 fails, stop the run, file `BUG-*.md`, fix root cause, rerun the failing case, then restart smoke.
- Execute all P0 before P1. Any P0 failure is release-blocking.
- Do not weaken tests to pass. A failed invariant requires a production/config/docs fix plus narrow regression coverage.
- Capture command output in `.compozy/tasks/hermes/qa/logs/<TC-ID>/`.
- Capture browser or docs screenshots in `.compozy/tasks/hermes/qa/screenshots/<TC-ID>/`.
- Record all final evidence and residual risk in `.compozy/tasks/hermes/qa/verification-report.md`.

## Smoke Lane

Estimated duration: 15-30 minutes.

| Order | Case | Priority | Stop Condition | Minimum Evidence |
|-------|------|----------|----------------|------------------|
| 1 | TC-INT-001 | P0 | Migration table absent, duplicate migration, failed rollback, or retry ignores cancellation | Focused Go test log and DB migration row evidence |
| 2 | TC-INT-003 | P0 | Missing `failure.kind`, unredacted crash/probe data, or API/SSE/CLI mismatch | Mock ACP failure log, session status JSON, observe health JSON, SSE terminal event sample |
| 3 | TC-INT-004 | P0 | Duplicate scheduled fire after restart or delivery error corrupts cursor state | Automation DB/run evidence and API/CLI payloads before and after restart |
| 4 | TC-SEC-001 | P0 | Access/refresh token, authorization code, PKCE verifier, or client secret appears outside auth store | OAuth mock logs, CLI/API/settings output, grep/redaction evidence |
| 5 | TC-INT-005 | P0 | Scoped interrupt signals unrelated owner or stale PID | Registry records, interrupt report, PID/start-time mismatch evidence |
| 6 | TC-FUNC-003 | P0 | `.env` repair rewrites unsafe file or extension env diagnostics leak values | CLI JSON/human output, repaired temp `.env`, settings/API payload |

## Targeted Lanes

Run targeted lanes after smoke passes or after a fix touches the relevant domain.

### Persistence And Observability

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-INT-001 | P0 | Migration runner, global/session DB boot, retry backoff |
| 2 | TC-INT-002 | P1 | Retention cutoff, disabled retention, observe health persistence/retention |

Recommended commands for task_11 evidence:

- `go test ./internal/store ./internal/store/globaldb ./internal/store/sessiondb ./internal/retry`
- `go test ./internal/observe ./internal/api/core ./internal/api/contract`
- CLI/API checks around `agh observe health -o json`

### ACP Lifecycle

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-INT-003 | P0 | Failure kinds, crash bundles, agent probes, session/SSE/CLI parity |

Recommended commands:

- `go test ./internal/acp ./internal/session ./internal/observe ./internal/api/core ./internal/cli`
- Mock provider flow through session start/resume/prompt failure and session event streaming.

### Automation Scheduler

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-INT-004 | P0 | Durable cursor, restart, `skip_missed`, delivery errors |
| 2 | TC-UI-001 | P1 | Web automation scheduler and run history rendering |

Recommended commands:

- `go test ./internal/automation ./internal/store/globaldb ./internal/api/core ./internal/cli`
- Focused web tests for automation detail and run history.
- Site docs review for automation jobs/runs and observe health.

### MCP Auth And Filesystem Security

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-SEC-001 | P0 | OAuth PKCE, refresh/logout, durable token storage, redaction |
| 2 | TC-SEC-002 | P0 | Skill and managed extension symlink escape rejection |
| 3 | TC-UI-002 | P1 | Web settings auth/env diagnostics redaction |

Recommended commands:

- `go test ./internal/mcp/auth ./internal/config ./internal/cli ./internal/api/core ./internal/settings`
- `go test ./internal/skills ./internal/extension`
- Focused web settings MCP/hooks/extensions tests.

### Process Registry And Interrupts

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-INT-005 | P0 | Registry checkpointing, PID/start-time validation, scoped interrupt |

Recommended commands:

- `go test ./internal/toolruntime ./internal/acp ./internal/hooks ./internal/extension ./internal/environment ./internal/subprocess`
- Manual restart/interruption flow with local and remote-terminal record variants.

### Memory Visibility

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-FUNC-001 | P1 | Memory health/history CLI/API, filters, redaction, prompt isolation |
| 2 | TC-UI-003 | P1 | Generated memory endpoint types and web contract compatibility |

Recommended commands:

- `go test ./internal/memory ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli`
- `make codegen-check`
- Web typecheck focused on generated OpenAPI consumers.

### CLI Setup, Environment, And Release

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-FUNC-003 | P0 | `.env` repair and extension `requires_env`/`missing_env` |
| 2 | TC-FUNC-002 | P1 | Config setup lifecycle, redaction, managed install/update/uninstall/completion |
| 3 | TC-REG-001 | P1 | Release package targets and trust artifacts |
| 4 | TC-REG-002 | P1 | Site docs and CLI/API reference consistency |

Recommended commands:

- `go test ./internal/config ./internal/extension ./internal/cli ./internal/api/contract ./internal/api/core ./internal/daemon ./internal/settings`
- `go test ./internal/config -run TestGoReleaserConfigPreservesTrustArtifactsAndPackageTargets -count=1`
- Relevant `packages/site` typecheck/build/source tests.

### Web And Documentation

| Order | Case | Priority | Scope |
|-------|------|----------|-------|
| 1 | TC-UI-003 | P1 | Generated DTOs, web API adapters, daemon/session/settings fixtures |
| 2 | TC-UI-001 | P1 | Automation UI contract rendering |
| 3 | TC-UI-002 | P1 | Settings MCP/auth/extension env rendering |
| 4 | TC-REG-002 | P1 | Site docs, navigation, CLI reference, source tests |

Recommended commands:

- `make web-lint`
- `make web-typecheck`
- Focused Vitest lanes for automation, daemon, session, settings, and generated contract tests.
- `bun run --cwd packages/site typecheck`
- `bun run --cwd packages/site build`

## Full Regression Lane

Estimated duration: 2-4 hours.

Execute after all smoke and targeted lanes pass or after the final task_11 fix set.

1. Run all P0 cases in smoke order.
2. Run all P1 cases in this order: TC-INT-002, TC-FUNC-001, TC-FUNC-002, TC-REG-001, TC-UI-001, TC-UI-002, TC-UI-003, TC-REG-002.
3. Run repository gate: `make verify`.
4. Run any extra web/site commands required by files changed during task_11.
5. Populate `.compozy/tasks/hermes/qa/verification-report.md` with command output summaries, evidence paths, unresolved issues, and final verdict.

## Pass, Fail, And Conditional Criteria

PASS:

- All P0 pass.
- At least 90% of P1 pass.
- `make verify` passes after the last change.
- No critical bug, secret leak, data loss, duplicate scheduled fire, unrelated process kill, or docs/web contract blocker remains open.

FAIL:

- Any P0 fails.
- Any secret-bearing value leaks outside approved storage.
- Automation dispatch duplicates a claimed fire after restart.
- Process registry signals a stale or unrelated PID.
- `make verify` fails after the final fix set.

CONDITIONAL:

- A P1 docs or UI issue remains with a documented workaround, `BUG-*.md`, and explicit owner, while all P0 and final repository gates pass.

## Regression Maintenance

After task_11:

- Promote any bug reproduction into the narrowest durable Go, web, or site regression test.
- Add a new `TC-*` case only if the discovered gap represents a reusable Hermes invariant not already covered here.
- Keep evidence paths stable under `.compozy/tasks/hermes/qa/` so future hardening runs can diff reports across releases.
