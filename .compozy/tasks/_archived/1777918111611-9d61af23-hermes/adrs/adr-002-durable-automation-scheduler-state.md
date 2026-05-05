# ADR-002: Use Durable Scheduler State for Automation At-Most-Once Dispatch

## Status

Accepted

## Date

2026-04-24

## Context

Automation currently relies on in-process scheduling and does not persist a scheduler cursor such as `next_run_at`. Selected Hermes issues require restart-safe recovery, misfire handling, at-most-once dispatch, and separation between execution errors and delivery errors.

Without durable scheduler state, a daemon restart can lose the precise intended next fire, duplicate a dispatch, or make catch-up behavior depend on in-memory scheduler internals.

## Decision

Introduce durable automation scheduler state and use it as the source of truth for restart recovery and at-most-once dispatch.

The design will add persisted state for scheduled jobs, including:

- `next_run_at`, `last_run_at`, and `last_scheduled_at`.
- Catch-up policy and misfire grace data.
- Consecutive resume failure counters.
- Delivery-error storage separate from normal run error storage.
- Idempotency/fire identity for a scheduled dispatch.

The scheduler must advance the durable cursor in the same critical path before dispatching work. On boot, automation recovery must reconcile persisted cursor state against wall-clock time and catch-up policy.

## Alternatives Considered

- Keep `gocron` as the only scheduler source of truth with a thin config overlay. This is smaller but does not fully solve at-most-once restart behavior.
- Add a full durable outbox for automation fires. This is robust but broader than the selected Hermes scope.

## Consequences

- Automation will need schema changes and migration support before implementation.
- Scheduler tests must include restart/reconciliation cases, duplicate-fire prevention, misfire grace, and delivery-error classification.
- `gocron` can still be used as an execution timer, but not as the only source of durable schedule truth.

## Implementation Notes

- Keep the design closer to a scheduler cursor than a general-purpose outbox.
- Make run reservation and cursor advancement transactionally safe.
- Surface delivery error separately in store models, API payloads, and CLI output where automation run details are shown.

## References

- `.compozy/tasks/hermes/analysis/analysis_gateway_cron.md`
- Issues: 20, 21, 22, 25
