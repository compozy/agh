# ADR-003: Identity-Bound Task Execution Uses Dedicated Agent Tools

## Status

Accepted

## Date

2026-04-29

## Context

The original `tools-registry` foundation deliberately excluded claim, release,
complete, fail, and related task-run execution primitives from the first built-in
tool surface. Follow-up review showed that this is not the right final-state
boundary. AGH already exposes these operations through identity-bound CLI
commands such as `agh task next`, `agh task heartbeat`, `agh task complete`,
`agh task fail`, and `agh task release`. Those commands reuse the existing
authoritative task writers and already require agent identity.

This branch still exposes raw `claim_token` in AGH-owned CLI/API contracts, so
the follow-up is not just additive. It is a hard cut from token-visible public
contracts to session-bound lookup plus tool/CLI/API convergence.

Repo memory establishes two constraints that must be reconciled:

- manual and autonomous paths should converge on the same primitives;
- authoritative state transitions must not be duplicated across peer packages.

The user explicitly chose to make these identity-bound execution operations
dedicated agent-callable tools, while keeping spawn and cross-session lifecycle
control outside the agent tool surface.

## Decision

AGH exposes identity-bound autonomy execution tools for task runs:

1. Add dedicated tools for `run_claim_next`, `run_heartbeat`, `run_complete`,
   `run_fail`, and `run_release`.
2. These tools must call the same authoritative task writers already used by the
   CLI and daemon paths; they must not create parallel ownership logic.
3. The tools are bound to the calling session identity and to claim-token
   fencing rules.
4. Raw `claim_token` handling remains subject to existing redaction rules.
5. Spawn, cross-session terminal-state mutation, and daemon/session lifecycle
   control remain operator-only surfaces.

## Alternatives Considered

### Alternative 1: Keep identity-bound execution CLI-only

- **Description**: Leave `task next|heartbeat|complete|fail|release` on the CLI
  and let agents shell out when needed.
- **Pros**: Lower implementation cost; reuses an existing interface.
- **Cons**: Preserves the exact CLI-vs-tool ambiguity the redesign is supposed
  to remove; weakens agent-manageability for the core autonomy path.
- **Why rejected**: The final design should not require shelling out for AGH's
  own task-execution primitives.

### Alternative 2: Expose all lifecycle and spawn operations as tools

- **Description**: Put spawn, session-stop, and other lifecycle controls in the
  same agent-callable tool family.
- **Pros**: Maximum surface symmetry.
- **Cons**: Violates the authority boundary for cross-session lifecycle control
  and increases the blast radius of agent mistakes.
- **Why rejected**: Spawn and cross-session lifecycle remain operator concerns.

## Consequences

### Positive

- Agents can execute their work queue through structured tools instead of shell.
- CLI and tool surfaces converge on the same authoritative writers.
- The final registry better matches AGH's autonomy architecture.

### Negative

- The tool surface must deal with identity-bound inputs and claim-token fencing.
- Tests must cover lease, stale token, and redaction behavior end to end.

### Risks

- Incorrect token or session binding could weaken ownership guarantees.
  Mitigation: reuse the existing task writer APIs and preserve current redaction
  and stale-token validation rules.

## Implementation Notes

- Keep spawn and session lifecycle out of this tool family.
- Reuse the existing operator/session identity helpers and task DTOs where
  possible, but hard-cut raw-token DTO fields where the public contract changes.
- Expose deterministic error codes for stale token, missing claim, expired
  lease, and permission mismatch.

## References

- `.compozy/tasks/tools-registry/adrs/adr-004-mvp-native-tool-scope.md`
- `docs/_memory/lessons/L-004-manual-equals-peer.md`
- `docs/_memory/lessons/L-005-authoritative-primitive-exclusivity.md`
- `internal/cli/task.go`
- `internal/task/interfaces.go`
- `internal/task/lease_manager.go`
