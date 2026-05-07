# TC-FUNC-008: Live Provider Source Timeout + Effective Auth/Home/Env

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog/live_sources.go`
**Requirement:** TechSpec Live Provider Sources, SI-3.
**Status:** Not Run

## Objective

Verify each registered live provider source is timeout-bound, uses the provider's effective auth/home/env policy, never inherits the request context's deadline implicitly, and records source status (not session blockers) on failure.

## Preconditions

- [ ] Stub or fake provider subprocess and HTTP endpoints.
- [ ] Provider config with `home_policy`, `env_policy`, `auth_mode` set per provider.
- [ ] Daemon base env injected for live discovery.

## Test Steps

1. **Timeout enforcement.**
   - Stub server delays 30s; provider discovery timeout 1s.
   - **Expected:** Source status records `failed` with redacted timeout message; no panic; coalescing serializes per provider (TC-PERF-001 covers concurrent storms).
2. **Provider home policy honored.**
   - Set `home_policy=isolated`; spawn live discovery subprocess.
   - **Expected:** Subprocess `HOME` matches provider isolated home; daemon does not leak operator `HOME`.
3. **Auth status command non-zero.**
   - Stub `auth_status_command` returns exit 2.
   - **Expected:** Source status `failed`; daemon does not raise an operator error; manual model entry still works.
4. **Provider secret resolver exposes redacted env.**
   - Resolver injects `OPENAI_API_KEY=secret-xyzzy`.
   - **Expected:** Source error log entries do not contain `secret-xyzzy`; refer to TC-SEC-001 for cross-surface redaction.
5. **Source IDs are `provider_live:<provider_id>` with priority 110.**
   - **Expected:** SQLite rows match the documented IDs and priority (Task 04 invariant).

## Audit Coverage

- C6 task tree (Task 04).
- SI-1 (no session blocker), SI-3, SI-9.

## Pass Criteria

- Timeouts enforced.
- Effective home/env honored.
- Source IDs and priority match Task 04 contract.

## Failure Criteria

- Subprocess inherits operator `HOME`.
- Source error contains raw secret material.
- Timeout exceeds configured timeout.
