# ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections

## Status

Accepted

## Date

2026-04-29

## Context

The `tools-registry` foundation already shipped the registry projection/call
path, operator-visible diagnostics, and session-visible callable projections.
It also already evaluates policy during list/search/get/call. The remaining
problem is that the daemon currently feeds that evaluator mostly boot-time
policy inputs from config. Agent definitions, session lineage, discovery
defaults, source policy, health, availability, and hook outputs are all runtime
variables and should participate in the same shipped evaluation path.

Competitor evidence reinforces the same split. Claude Code filters tools before
the model sees them, but still recomputes permission decisions during execution.
Hermes filters model-visible tool definitions by availability and toolsets.
OpenClaw exposes a policy-filtered tool inventory in the startup prompt. In all
cases, discovery improves UX, but runtime dispatch remains authoritative.

The user explicitly chose per-call policy recomputation for `list/search/get/call`.

## Decision

AGH recomputes effective tool policy per call:

1. `list`, `search`, `get`, and `call` must keep using the existing registry
   evaluation path, but the policy inputs for that path must be resolved
   against current runtime state.
2. Effective policy inputs include the current agent definition, session
   lineage, session scope, source policy, availability state, and hook results.
3. Operator projections and session/model projections remain distinct:
   operator views include unavailable or unauthorized tools with reason codes,
   while session/model views expose only callable tools.
4. Discovery-time filtering is a UX optimization, not a security boundary.
   `Registry.Call` must revalidate all execution preconditions.
5. Cached projections are allowed only as invalidatable accelerators keyed by
   the inputs above; cache entries must never become the source of truth.

## Alternatives Considered

### Alternative 1: Dynamic checks only at call time

- **Description**: Cache `list/search/get` results statically per session and
  apply fresh policy only for `call`.
- **Pros**: Simpler read path; lower projection cost.
- **Cons**: Agent-visible catalogs drift from actual runtime state; operators
  and sessions see stale policy decisions until a call fails.
- **Why rejected**: The final design needs runtime-consistent discovery and
  diagnostics, not just runtime-consistent execution failures.

### Alternative 2: Boot-time or startup-static policy

- **Description**: Compute effective policy once at daemon boot or session
  startup and reuse it broadly.
- **Pros**: Lowest implementation complexity.
- **Cons**: Ignores mutable runtime facts such as session lineage, source
  health, approvals, auth status, and hooks.
- **Why rejected**: It contradicts both the user direction and the registry's
  safety model.

## Consequences

### Positive

- Policy decisions stay aligned with actual runtime state.
- Operators can debug the same reasons the runtime uses to allow or deny calls.
- Session/model surfaces stay smaller and more trustworthy.

### Negative

- Projection paths need careful invalidation and caching strategy.
- The policy resolver becomes a central runtime dependency across tool surfaces.

### Risks

- Repeated policy evaluation can add latency. Mitigation: cache normalized
  intermediate inputs and projection results, but only behind versioned,
  invalidatable keys.

## Implementation Notes

- Introduce a single policy resolver that both projection and dispatch call.
- Preserve the existing `tools.Scope`, `Registry`, and `PolicyEvaluator`
  contracts; this follow-up extends the input-resolution layer rather than
  inventing a second policy engine.
- Keep operator and session projections as explicit types with separate
  rendering behavior.
- Ensure hook-denied and source-health-denied states surface deterministic
  reason codes.

## References

- `.compozy/tasks/tools-registry/adrs/adr-006-tool-visibility-by-surface.md`
- `.resources/claude-code/tools.ts`
- `.resources/claude-code/services/api/claude.ts`
- `.resources/claude-code/services/tools/toolExecution.ts`
- `.resources/claude-code/utils/permissions/permissions.ts`
- `.resources/hermes/tools/registry.py`
- `.resources/openclaw/src/agents/tool-policy-pipeline.ts`
