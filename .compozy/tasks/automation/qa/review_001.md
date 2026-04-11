# QA Review 001

Date: 2026-04-11
Scope: `.compozy/tasks/automation`
Method: real daemon + CLI + HTTP usage against the branch implementation

## Issue 1: automation runs complete but leave system sessions active

- Severity: high
- Status: resolved
- Surfaces: dispatcher, session lifecycle, runtime health

### Reproduction

1. Start the daemon with automation enabled and a real `codex` agent.
2. Create a dynamic automation job and trigger it.
3. Wait for the run to reach `status = "completed"`.
4. Inspect `agh session list` or `GET /api/observe/health`.

### Observed

- Completed automation runs keep their created system sessions in `state = "active"`.
- Repeating the same job accumulates leaked active sessions.
- Runtime health showed `active_sessions` climbing from `1` to `2` after two successful runs.

### Expected

- When an automation run reaches a terminal state, its system session must also be driven to a terminal stopped state.

### Root Cause

- `internal/automation/dispatch.go` marks the run as completed after prompt collection finishes, but it never stops the created session.

### Resolution

- The dispatcher now explicitly stops automation-created sessions on terminal run states.
- Successful runs stop with `completed`; failed runs stop with an error-classified cause.
- Regression tests now assert that terminal automation runs also invoke session shutdown.

## Issue 2: config-backed webhook triggers are listed but not actually registered

- Severity: high
- Status: resolved
- Surfaces: config sync, runtime registration, webhook ingress

### Reproduction

1. Define a config-backed webhook trigger under `automation.triggers`.
2. Start the daemon and list triggers through CLI/API.
3. POST to the advertised webhook endpoint.

### Observed

- The trigger is persisted and listed as enabled.
- The webhook endpoint returns `404` with `automation: webhook trigger not registered`.
- Daemon logs show `automation.trigger.skipped_webhook_registration`.

### Expected

- A config-backed webhook trigger must either register successfully with a usable secret source, or be rejected during config load/sync so the system never exposes a dead endpoint.

### Root Cause

- Config triggers currently have no secure secret source, but the runtime registration path requires one.
- The manager silently skips runtime registration while still surfacing the trigger as available.

### Resolution

- Config-backed webhook triggers now require `webhook_secret_env`.
- Config validation fails fast if the env var is missing or empty.
- The automation manager sync persists the resolved secret into write-only trigger secret storage so runtime registration succeeds on startup and sync.
