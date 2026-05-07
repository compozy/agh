# TC-FUNC-009: Live Discovery Never Touches ACP Sessions

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog` live sources + `internal/acp`
**Requirement:** TechSpec Safety Invariants SI-2, ADR-001.
**Status:** Not Run

## Objective

Verify live provider discovery never calls `session/new`, `session/load`, `session/set_model`, or `session/set_config_option`, and that unavailable side-effect-free discovery paths surface as source-status failures, not session blockers.

## Preconditions

- [ ] ACP fake driver instrumented to assert it is not invoked from discovery code paths.
- [ ] Discovery sources registered for built-in providers and adapter-config providers (OpenClaw, Hermes, Pi).

## Test Steps

1. **Run a refresh storm against every provider.**
   - Command: `for p in codex anthropic openrouter ollama opencode openclaw hermes pi; do agh provider models refresh $p; done`.
   - **Expected:** ACP fake driver records zero invocations.
2. **Provider without `discovery.command`/`discovery.endpoint`.**
   - Configure OpenClaw with `discovery.enabled=true` but no command/endpoint.
   - **Expected:** Source status `refresh_state="failed"`, `last_error` references missing discovery contract; session creation for that provider remains usable; manual model entry still valid.
3. **Provider discovery enabled with invalid HTTP endpoint.**
   - Set `endpoint = "http://127.0.0.1:0"`.
   - **Expected:** Source status `failed` with redacted error; ACP driver still untouched.
4. **Concurrent session creation while discovery refresh runs.**
   - Trigger refresh and a session create simultaneously for the same provider.
   - **Expected:** Session creation completes without waiting on discovery; ACP fake records only `session/new` from the session caller, not from discovery.

## Audit Coverage

- C6 task tree (Task 04, Task 06).
- SI-1, SI-2, SI-3.

## Pass Criteria

- Zero ACP `session/*` calls originate from discovery code.
- Missing discovery configuration produces source status, never blocks sessions.

## Failure Criteria

- Discovery code path invokes any ACP session method.
- Failure to discover blocks session creation.
