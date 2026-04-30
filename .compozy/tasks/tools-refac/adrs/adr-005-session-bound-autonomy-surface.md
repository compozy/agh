# ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens

## Status

Accepted

## Date

2026-04-29

## Context

AGH's lease and ownership model uses raw `claim_token` values internally as the
authoritative fence for heartbeat, release, complete, and fail operations.
Those tokens are intentionally sensitive. Repo-wide security rules and autonomy
lessons are explicit: raw `claim_token` must never appear in logs, SSE, memory,
web UI, error payloads, channel metadata, or AGH-owned transport contracts.

The follow-up `tools-refac` design introduces dedicated agent-callable autonomy
tools for `run_claim_next`, `run_heartbeat`, `run_complete`, `run_fail`, and
`run_release`. A naive translation of the current CLI contract would leak the
raw token into tool inputs/outputs and reproduce the same problem across hosted
MCP, HTTP, UDS, and CLI.

The autonomy memory also preserves another simplifying invariant from the task
run model: one active lease per session in the current design. That allows the
runtime to keep raw token handling server-side while still routing every command
through the same authoritative writers.

## Decision

AGH-owned autonomy surfaces become session-bound and stop exposing raw
`claim_token` values:

1. Dedicated autonomy tools never return raw `claim_token` to the agent.
2. Dedicated autonomy tools never accept raw `claim_token` from the agent.
3. Tool, CLI, HTTP, and UDS autonomy contracts identify the leased work by
   session identity plus `run_id`; the daemon resolves the authoritative raw
   token internally before calling the existing task lease writers.
4. `claim_token_hash` remains observability-only metadata. It is never accepted
   as a write credential.
5. If AGH later allows more than one active lease per session, the follow-up
   design must introduce a separate typed lease handle. It must not reuse or
   expose raw `claim_token`.

## Alternatives Considered

### Alternative 1: Expose raw claim tokens in the new tool family

- **Description**: Reuse the current CLI contract shape directly in tools and
  return/accept raw `claim_token`.
- **Pros**: Lowest implementation cost; mirrors current service inputs.
- **Cons**: Violates existing security invariants; leaks sensitive fence tokens
  into the model-visible and transport-visible surface.
- **Why rejected**: It is explicitly forbidden by AGH security posture.

### Alternative 2: Replace raw claim tokens with `claim_token_hash`

- **Description**: Return `claim_token_hash` and accept it back on later calls.
- **Pros**: Avoids raw token exposure.
- **Cons**: The hash is not the authoritative credential and would require a
  second ownership path or reversible lookup semantics that AGH does not have.
- **Why rejected**: It creates a fake public credential and weakens the
  authority model.

## Consequences

### Positive

- The final autonomy tool surface becomes compatible with AGH's standing
  redaction and transport rules.
- Tools, CLI, HTTP, and UDS can still converge on the same authoritative task
  writers without exposing secret fence tokens.
- Hosted MCP can safely expose the autonomy tool family without inventing a
  second token lifecycle.

### Negative

- Existing CLI contracts for claim/heartbeat/complete/fail/release need a hard
  cut away from raw token arguments and output.
- The daemon must maintain a session-bound lookup from the visible contract to
  the internal lease token.

### Risks

- The session-bound lookup could become ambiguous if AGH later allows multiple
  concurrent leases per session. Mitigation: preserve the current one-active-
  lease-per-session rule in this design and require a future typed lease-handle
  ADR before relaxing it.

## Implementation Notes

- Reuse `task.Service.ClaimNextRun`, `HeartbeatRunLease`, `CompleteRunLease`,
  `FailRunLease`, and `ReleaseRunLease` as the only writers.
- Carry `run_id` explicitly in heartbeat/complete/fail/release contracts so the
  daemon can validate the caller's active lease before resolving the raw token.
- Keep `claim_token_hash` in events and diagnostics for correlation only.

## References

- `internal/CLAUDE.md`
- `docs/_memory/standing_directives.md`
- `docs/_memory/lessons/L-003-task-runs-single-queue.md`
- `docs/_memory/lessons/L-005-authoritative-primitive-exclusivity.md`
- `.compozy/tasks/autonomous/adrs/adr-003.md`
