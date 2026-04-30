# ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default

## Status

Accepted

## Date

2026-04-29

## Context

The first `tools-refac` draft corrected the CLI-vs-tool ambiguity, but it did
so with a conservative posture that left too many mutable domains on
operator-only or read-only surfaces. That posture conflicts with AGH's product
premise: runtime features are incomplete if agents cannot manage them through
structured surfaces.

AGH already has validated management paths for several mutable domains:

- automation job and trigger lifecycle;
- extension search/install/update/remove/enable/disable;
- config overlay mutation through the validated writer;
- hook declaration normalization and validation.

At the same time, the current branch still projects only a narrow built-in
subset (`agh__bootstrap`, `agh__catalog`, `agh__coordination`, `agh__tasks`).
The follow-up expands that shipped registry surface; it does not start from an
empty catalog.

The real safety boundaries are not "write vs read". They are:

- trust-root bootstrap configuration;
- raw secret material;
- human-interactive flows such as browser OAuth;
- cross-session lifecycle operations without explicit ownership or lineage.

The user explicitly rejected a design where mutable domains stay artificially
operator-only just because they are writes.

## Decision

Mutable AGH management surfaces are tool-callable by default:

1. If AGH already has an authoritative writer or validated management command
   for a domain operation, the default expectation is to expose it as a tool.
2. Mutation is contained by scope, policy, approval, trust-source checks, and
   deterministic errors rather than by forcing the entire domain onto
   operator-only surfaces.
3. Operator-only remains reserved for:
   - bootstrap-root daemon and transport control;
   - raw secret and auth material;
   - human-interactive browser/OAuth flows;
   - cross-session lifecycle operations without explicit lineage-bound
     authority.
4. Automation management, extension lifecycle, mutable hook declarations, and
   validated config overlay mutation therefore belong on the canonical agent
   tool surface.

## Alternatives Considered

### Alternative 1: Keep mutable management CLI-only

- **Description**: Expose read models and a few narrow toggles as tools, but
  keep create/update/delete/install/remove flows on CLI/HTTP only.
- **Pros**: Smaller tool surface; lower implementation cost.
- **Cons**: Violates AGH's agent-manageability premise; preserves hidden
  capability classes that require shelling out or out-of-band operator action.
- **Why rejected**: It treats policy as a substitute for product design instead
  of exposing the capability properly.

### Alternative 2: Expose every write, including trust-root and human-auth flows

- **Description**: Make all writes agent-callable, including daemon bootstrap,
  sandbox roots, and OAuth login/logout.
- **Pros**: Maximum symmetry.
- **Cons**: Pushes trust-root and human-interactive responsibilities into the
  normal tool loop; weakens governance and complicates approval semantics.
- **Why rejected**: Some boundaries are not ordinary mutable runtime behavior.

## Consequences

### Positive

- AGH becomes much closer to its stated self-management model.
- CLI, HTTP, UDS, and tool surfaces converge on shared writers rather than
  splitting "safe" and "mutable" capability classes.
- Policy, approval, and observability become the real control plane for mutable
  operations.

### Negative

- The tool catalog grows significantly.
- Mutation-heavy tool contracts need stronger tests and clearer docs.

### Risks

- Broader mutation increases the blast radius of agent mistakes.
  Mitigation: reserve operator-only for trust-root boundaries, and use approval,
  policy, trust-source checks, and auditability for everything else.

## Implementation Notes

- Reuse domain writers; do not create tool-only mutation paths.
- Keep source-owned and extension-owned resources structurally immutable when
  the current mutable overlay is not their owner.
- Keep secret-bearing inputs on redacted or operator-managed surfaces.

## References

- `AGENTS.md`
- `docs/_memory/standing_directives.md`
- `internal/cli/automation.go`
- `internal/cli/extension.go`
- `internal/cli/config.go`
- `internal/config/hooks.go`
