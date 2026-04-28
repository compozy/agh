# Hermes Hardening TechSpec

## Executive Summary

This TechSpec fixes the selected Hermes release-readiness issues as a single hardening program with domain tracks and shared foundations. Scope includes issues 10, 11, 14, 15, 16, 17, 20, 21, 22, 25, 27, 28, 29, 30, 33, 34, 35, 36, 37, 39, 40, 41, 42, 43, 57, 59, and 60 from `.compozy/tasks/hermes/analysis/analysis.md`. Issues 6, 8, and 9 are explicitly excluded.

The implementation prioritizes durable state, restart-safe runtime behavior, operator-visible health, MCP/tool security, memory CLI visibility, and setup/release ergonomics. Runtime memory context references and memory provider hook execution are prepared through interfaces only; CLI memory health and memory history are implemented now.

## System Architecture

The selected work is organized as one Hermes hardening program with shared foundations first, then domain tracks:

1. State, migrations, retention, and retry foundations.
2. ACP/session lifecycle hardening.
3. Durable automation scheduling.
4. MCP auth, tool security, process registry, and interrupt runtime.
5. Memory CLI health/history and future-facing interfaces.
6. CLI, setup, and release hardening.

The package direction must preserve AGH architecture rules:

- `daemon/` remains the only composition root.
- Lower-level packages must not import `daemon`, `api`, or `cli`.
- Interfaces are defined where consumed.
- Shared runtime packages must remain deterministic, context-aware, and testable.
- No event bus, no reflection-based routing, and no compatibility shims for old alpha state.

## Implementation Design

### Track 1: State, Retention, and Retry Foundations

Issues: 10, 11, 17.

Add a deterministic migration runner in `internal/store` that records applied migrations in `schema_migrations`. Both global and session databases must use the same migration primitive while keeping package-specific migration lists close to their schema owners.

Wire `observability.retention_days` into a daily retention sweep in `internal/observe`. Retention must clean global observability rows that have aged out while preserving active session state and respecting configured observability disablement.

Add a shared jittered exponential backoff primitive, likely under `internal/retry`, for retryable runtime paths. The primitive must support base delay, cap, jitter ratio, deterministic tests through injectable randomness or clocks, and context cancellation.

### Track 2: ACP and Session Lifecycle Hardening

Issues: 14, 15, 16.

Add a typed failure classifier for ACP/session failures. `FailureKind` must sit beside existing `StopReason` rather than replacing it. It should distinguish startup, load-session, handshake, prompt, transport, context cancellation, permission, and process-exit failure classes where the code has enough evidence.

Expose failure kind through session metadata, global summaries, health payloads, and SSE/API conversions. Existing stop-reason behavior must keep working.

Add downstream agent probes for configured agents/providers. Probes should verify command availability with `execabs.LookPath` and optionally perform a lightweight ACP initialize/handshake when configured. Probe results become part of observe health.

Add crash bundle capture for panic/error paths in runtime goroutines owned by session and ACP components. Crash bundles should be written under AGH home with timestamp and PID in the filename, capture stack/error/session context where available, and avoid leaking secrets.

### Track 3: Durable Automation Scheduler

Issues: 20, 21, 22, 25.

Automation must use durable scheduler state as the source of truth for restart recovery and at-most-once dispatch. The design adds persisted scheduler cursor/state with `next_run_at`, `last_run_at`, `last_scheduled_at`, catch-up policy, misfire grace, consecutive resume failure counters, and delivery-error separation.

The scheduler must advance the durable cursor before dispatching a fire. This advancement must be transactional with fire identity/run reservation so that daemon restart cannot duplicate the same scheduled fire.

`gocron` may remain as the local timer engine, but it cannot be the only source of scheduling truth. On boot, automation recovery reconciles persisted scheduler state against current time and catch-up policy.

### Track 4: MCP Auth, Tools, Process Registry, and Interrupts

Issues: 27, 28, 29, 30.

Add a first-class MCP auth subsystem, likely `internal/mcp/auth`, for OAuth 2.1 + PKCE. It must handle authorization metadata, authorization URL generation, callback verification, token exchange, refresh, durable token storage, and redaction.

Extend MCP config/resource modeling to distinguish local subprocess MCP servers from authenticated remote MCP servers. Plain settings responses, config CLI output, logs, and resource snapshots must never expose access tokens or refresh tokens.

Add symlink escape protection for skills. Skill loading, sidecar loading, and provenance hashing must reject or safely ignore symlinks that escape the declared skill root.

Add a shared process and interrupt runtime, likely `internal/toolruntime` or `internal/runtime/processes`. This package owns checkpoint-on-write process records, PID/start-time validation, boot reconciliation, ownership metadata, and scoped interrupts. ACP terminal manager, local/daytona environment tool hosts, hooks, extension host API execution, and future tools integrate with this package.

### Track 5: Memory Health, History, and Future Interfaces

Issues: 33, 34, 35, 60.

Implement memory CLI surfaces now:

- `agh memory health`
- `agh memory history`

These commands should reuse existing memory health stats and `memory_operation_log`/observe event-summary data where possible.

Prepare interfaces for future context references and memory provider lifecycle hooks, but do not wire runtime prompt integration in this phase. Specifically, define seams for future `@file`, `@folder`, `@git`, `@url`, token budgeting, sensitive-path filtering, and hooks such as `OnTurnStart`, `OnSessionEnd`, and `OnPreCompress`.

### Track 6: CLI, Setup, and Release Hardening

Issues: 36, 37, 39, 40, 41, 42, 43, 57, 59.

Add CLI config management:

- `agh config show`
- `agh config get`
- `agh config set`
- `agh config edit`
- `agh config path`
- `agh config check`

Add uninstall, shell completion, install script, update command, and `AGH_MANAGED` environment convention. Add `.env` sanitization and repair for non-ASCII or malformed multi-key values. Extend extension manifests/install behavior with `requires_env` detection and operator-visible missing-env reporting.

Extend release packaging with GoReleaser package targets such as Homebrew and Linux packages while preserving existing signing and SBOM behavior.

## Core Interfaces

The implementation should converge on interfaces shaped like the following without copying them verbatim into tasks:

```go
type Migration struct {
	Version int
	Name    string
	Up      func(ctx context.Context, tx *sql.Tx) error
}

type ProcessRegistry interface {
	Register(ctx context.Context, rec ProcessRecord) (ProcessHandle, error)
	Checkpoint(ctx context.Context, handle ProcessHandle, state ProcessState) error
	ReconcileBoot(ctx context.Context) (ProcessRecoveryReport, error)
	Interrupt(ctx context.Context, scope InterruptScope) error
}

type AutomationScheduleStore interface {
	LoadDue(ctx context.Context, now time.Time) ([]ScheduledFire, error)
	AdvanceNextRun(ctx context.Context, jobID string, fireID string, next time.Time) error
	RecordDeliveryError(ctx context.Context, runID string, err error) error
}

type MCPTokenStore interface {
	Load(ctx context.Context, serverID string) (TokenSet, error)
	Save(ctx context.Context, serverID string, token TokenSet) error
	Delete(ctx context.Context, serverID string) error
}

type MemoryContextRefResolver interface {
	Resolve(ctx context.Context, refs []ContextRef, budget TokenBudget) (ResolvedContext, error)
}
```

## Data Models

### Store Migrations

Add `schema_migrations` with version, name, applied timestamp, and checksum or equivalent integrity field. Each DB opener must execute pending migrations once, in order, under a transaction.

### Session Failure Classification

Add failure-kind fields to the metadata/global store surfaces that need query and health support. Preserve existing stop-reason fields.

### Automation Scheduler State

Add durable scheduler state with enough data to recover the next intended fire:

- job ID
- next run timestamp
- last run timestamp
- last scheduled timestamp
- last fire ID or idempotency key
- catch-up policy
- misfire grace
- consecutive resume failures

Add automation run delivery-error storage separate from normal execution error storage.

### MCP Auth

Persist token state outside plain config overlays. Token records must include server identity, expiry, refresh state, scopes, and updated timestamp. API/CLI DTOs must expose only redacted status.

### Process Registry

Persist process records with PID, observed start time, process group when local, command metadata, owner metadata, state, last checkpoint timestamp, and terminal/tool identifiers where available.

## API Endpoints and CLI

### API and Payloads

Extend observe health with:

- agent probe statuses
- failure-kind aggregates
- automation scheduler lag and delivery-error counts
- process registry reconciliation health
- retention sweep state where useful

Extend automation run payloads with delivery-error fields. Extend settings/config payloads with redaction for MCP auth state.

### CLI

Add or extend:

- `agh config show|get|set|edit|path|check`
- `agh mcp auth login|status|logout`
- `agh memory health`
- `agh memory history`
- `agh completion`
- `agh uninstall`
- `agh update`
- install script entrypoint
- extension install/status reporting for `requires_env`

## Integration Points

- `internal/store/schema.go`, `internal/store/globaldb`, `internal/store/sessiondb`
- `internal/observe`, `internal/api/contract`, `internal/api/core`
- `internal/acp`, `internal/session`, `internal/subprocess`, `internal/procutil`
- `internal/automation`, `internal/config/automation.go`
- `internal/config/provider.go`, `internal/config/mcp_resource.go`, `internal/skills`
- `internal/tools`, `internal/hooks`, `internal/sandbox`, `internal/extension`
- `internal/memory`, `internal/cli/memory.go`, `internal/cli/observe.go`
- `internal/cli/root.go`, `internal/cli/install.go`, `internal/cli/extension.go`
- `.goreleaser.yml`

## Impact Analysis

The migration runner changes database bootstrap behavior and must be introduced before schema-dependent work. Automation durable scheduler state changes runtime semantics around scheduled fire dispatch and restart recovery. MCP auth adds secret-bearing state and requires strict redaction. Process registry centralizes process ownership and will touch multiple runtime packages.

Because AGH is in greenfield alpha, implementation should avoid backward-compatibility shims for old state. If old alpha state conflicts with the new shape, prefer deleting or rebuilding it through migrations rather than adding compatibility branches.

## Testing Approach

Each implementation task must include tests and target at least 80% coverage for changed packages.

Required coverage areas:

- migration ordering, idempotence, and failed migration rollback
- observability retention sweep by timestamp and disabled configuration
- retry backoff bounds, jitter, cap, and context cancellation
- failure-kind classification for ACP startup/load/prompt/process errors
- agent probe command-missing and handshake-failure statuses
- crash bundle creation on recovered panic paths
- automation cursor advancement before dispatch
- automation restart reconciliation and duplicate-fire prevention
- delivery-error persistence and API/CLI surfacing
- MCP OAuth PKCE generation, callback verification, refresh, redaction, and logout
- skill symlink escape rejection
- process registry checkpointing, PID reuse detection, boot reconciliation, and interrupt scoping
- memory health/history CLI output
- config, uninstall, completion, update, install script, `.env` repair, and `requires_env` behavior

## Development Sequencing

1. Build the migration runner and schema foundation.
2. Add shared retry/backoff and failure-kind models. Depends on: step 1.
3. Add observability retention and health payload foundations. Depends on: steps 1-2.
4. Implement ACP/session failure classification, probes, and crash bundles. Depends on: steps 2-3.
5. Implement durable automation scheduler state. Depends on: steps 1-3.
6. Implement MCP OAuth and skill symlink guard. Depends on: step 1.
7. Implement shared process registry and interrupts. Depends on: steps 1-2.
8. Implement memory health/history CLI and future interfaces. Depends on: steps 1 and 3.
9. Implement CLI/setup/release hardening. Depends on: prior config, payload, and release changes.
10. Run full verification and regression QA. Depends on: steps 1-9.

## Monitoring and Observability

Observe health must make hardening outcomes visible:

- migration state where useful for diagnostics
- retention sweep last run and error state
- agent probe status per provider/agent
- failure-kind totals
- automation scheduler lag and delivery-error counts
- process registry reconciliation results
- memory health and operation history

Logs must use structured `slog` and redact secrets.

## Technical Considerations

- No destructive git operations are part of this work.
- Do not hand-edit dependencies in `go.mod`; use `go get` if needed.
- Prefer existing package patterns and helpers before adding abstractions.
- Keep new interfaces small and close to consumers.
- Every goroutine must be owned by context cancellation and waited on or otherwise explicitly tracked.
- No `time.Sleep()` in orchestration tests; use synchronization primitives.
- Secrets must be redacted from CLI, logs, config views, and API responses.
- `make verify` is the final implementation gate for tasks.

## Architecture Decision Records

- `adrs/adr-001-hermes-hardening-tracks.md`: Organize Hermes Fixes as a Single Hardening TechSpec.
- `adrs/adr-002-durable-automation-scheduler-state.md`: Use Durable Scheduler State for Automation At-Most-Once Dispatch.
- `adrs/adr-003-mcp-oauth-auth-subsystem.md`: Implement MCP OAuth as a First-Class Auth Subsystem.
- `adrs/adr-004-shared-process-registry-and-interrupt-runtime.md`: Own Tool Processes and Interrupts in a Shared Runtime Package.
- `adrs/adr-005-memory-health-history-before-runtime-contextrefs.md`: Prioritize Memory Health and History Before Runtime Context References.
