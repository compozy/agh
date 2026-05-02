# TC-REG-005: Wake Coalescing, Rate Limits, Retention, And Config Validation

**Priority:** P1

## Objective

Validate Heartbeat wake policy edge cases that affect operators: scheduler wake coalescing, manual wake rate limiting, wake audit retention cleanup, valid config overlay behavior, and invalid config rejection for `[agents.soul]` and `[agents.heartbeat]`.

## Preconditions

- Reused QA lab daemon is running.
- `ops` has a valid managed `HEARTBEAT.md`.
- SQLite is available for wake state/event inspection.
- Config writes against the isolated runtime home are sequential.

## Test Steps

1. Trigger a scheduler wake where cooldown state already exists.
   **Expected:** Decision result is `coalesced` with reason `wake_coalesced`, and `coalesced_count` increments.
2. Trigger a manual wake during the cooldown window.
   **Expected:** Decision result is `rate_limited` with a closed reason.
3. Trigger a batch or max-wakes-per-cycle path.
   **Expected:** The first wake is allowed and the excess wake is `rate_limited` with `heartbeat_rate_limited`.
4. Insert or preserve wake audit rows across the retention boundary, then run the retention cleanup path.
   **Expected:** Expired rows are removed and retained rows remain according to `wake_event_retention`.
5. Apply valid `[agents.soul]` and `[agents.heartbeat]` overlay values in an isolated config.
   **Expected:** Effective config reflects the overlay values.
6. Attempt invalid config values for Soul and Heartbeat bounds.
   **Expected:** Validation fails with actionable field-specific errors and does not silently write an invalid runtime config.
7. Assert `agh session heartbeat` is absent.
   **Expected:** Command lookup/help rejects the session heartbeat namespace.

## Behavioral Evidence

- Operator journey: Heartbeat wake requests return closed, understandable decisions instead of duplicating prompts or silently dropping attempts.
- Artifacts: wake JSON responses, SQLite wake state/event readbacks, config CLI output/errors, focused Go test logs.

## Disruption Probes

- Repeated wake attempts do not interrupt active sessions.
- Invalid config cannot weaken projection bounds or retention invariants.

