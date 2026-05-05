# Claude Research Prompt — Network Threads Round 1

You are an architecture researcher working inside the AGH monorepo.

Your job is not to review a finished TechSpec. Your job is to help design the best greenfield-hardcut architecture for first-class thread support in AGH Network and its user-facing surfaces.

The product goal is Slack-like thread ergonomics inside AGH Network channels:

- the same channel must support multiple parallel conversations without topic mixing
- both humans and agents must be able to operate this seamlessly
- the protocol, runtime, CLI, HTTP/UDS, web UI, docs, prompts, orchestration, and tests must align
- greenfield alpha rules apply: prefer hard cuts, deletions, and clean primitives over compatibility layers

## Required Reading

Read these files fully before reasoning:

- `/Users/pedronauck/Dev/compozy/agh/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/_memory/standing_directives.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/_memory/glossary.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/rfcs/003_agh-network-v0.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/rfcs/004_agh-network-v1.md`
- `/Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/web/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/packages/site/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/lifecycle.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/router.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/contract/contract.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/core/network_details.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/store/types.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db_network_messages.go`
- `/Users/pedronauck/Dev/compozy/agh/web/src/routes/_app/network.tsx`
- `/Users/pedronauck/Dev/compozy/agh/web/src/hooks/routes/use-network-page.ts`
- `/Users/pedronauck/Dev/compozy/agh/web/src/systems/network/components/network-workspace-shell.tsx`
- `/Users/pedronauck/Dev/compozy/agh/web/src/systems/network/lib/network-formatters.ts`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/complex-scenarios/network.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/network-drafts/agora-spec-v0.2.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/from-goclaw/analysis/analysis_providers_gateway.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/_archived/20260412-040024-network/_techspec.md`

Also read these external knowledge references:

- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/Multi-Agent Orchestration Patterns.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/Agent Handoff and Context Transfer.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/The A2A Protocol.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/Agent Observability and Distributed Tracing.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/raw/articles/a2a-agent-card-specification.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/raw/articles/azure-ai-agent-orchestration-patterns.md`
- `/Users/pedronauck/dev/knowledge/claude-code/wiki/concepts/Agent Swarm and Subagents.md`

## Current Known State

Today AGH Network has:

- `channel` as the main shared lane
- `interaction_id` as a correlation/lifecycle container
- `reply_to`, `trace_id`, `causation_id` for causality and tracing
- no first-class `thread_id` in the RFC, envelope, store, API contract, or web model
- a web UX that splits public channel timelines from directed peer rooms, but still renders flat timelines

This creates a real product gap:

- multiple public conversations in the same channel get mixed together
- concurrent direct interactions with the same peer also get mixed
- the operator has no Slack-like thread lane to enter, inspect, or reply inside

## The Design Question

Decide what the best greenfield architecture is for AGH:

1. Make `thread` a first-class network primitive distinct from `interaction`
2. Reinterpret `interaction` as the thread primitive and redesign everything around that
3. Another design you believe is stronger

You must choose one recommendation and defend it.

## Non-Negotiable Constraints

- Greenfield alpha: hard cuts are allowed and preferred
- No compatibility bridges, aliases, or dual fields unless you can prove they are strictly necessary
- The result must be agent-manageable, not just web-manageable
- Manual operator paths and autonomous agent paths must converge on the same primitives
- Avoid superficial UI-only threading that leaves protocol/runtime semantics ambiguous
- Avoid overdesign: keep the protocol minimal but complete
- Be explicit about delete targets if a concept becomes obsolete
- Distinguish conversation grouping from causality/tracing; do not conflate them without argument
- Think about how prompts and agent instructions change when threads exist
- Think about how a user and multiple agents coexist in the same channel without chaos

## What To Produce

Return strict JSON with this shape:

```json
{
  "recommended_model": {
    "name": "thread-primitive|interaction-becomes-thread|other",
    "summary": "short summary",
    "why_this_wins": ["...", "...", "..."]
  },
  "alternatives_considered": [
    {
      "name": "alternative name",
      "pros": ["..."],
      "cons": ["..."],
      "why_rejected": "..."
    }
  ],
  "core_invariants": [
    "..."
  ],
  "protocol_design": {
    "envelope_changes": ["..."],
    "lifecycle_changes": ["..."],
    "routing_changes": ["..."],
    "querying_changes": ["..."],
    "visibility_rules": ["..."]
  },
  "runtime_design": {
    "router_and_store": ["..."],
    "delivery_and_prompting": ["..."],
    "orchestration_rules": ["..."],
    "observability": ["..."]
  },
  "surface_impact": {
    "cli": ["..."],
    "http_uds_api": ["..."],
    "web": ["..."],
    "docs": ["..."],
    "agent_prompts_and_skills": ["..."],
    "tests": ["..."]
  },
  "ux_model": {
    "operator_flows": ["..."],
    "agent_flows": ["..."],
    "failure_modes_to_design_for": ["..."]
  },
  "delete_targets": [
    "..."
  ],
  "open_questions_for_humans": [
    {
      "id": "Q-001",
      "question": "...",
      "why_it_matters": "..."
    }
  ],
  "proposed_adrs": [
    {
      "title": "...",
      "decision": "..."
    }
  ],
  "summary": "two or three sentences"
}
```

## Quality Bar

- Be opinionated
- Prefer fewer primitives with sharper semantics
- Call out if the web UX should expose channel timeline + thread rail + thread lane + unread model
- Call out if directed peer rooms should remain, shrink, or be redefined once threads exist
- Call out whether `reply_to` and `interaction_id` survive, change semantics, or get deleted
- Call out how a thread starts, how replies join it, and how agents discover the active thread they should answer in
- Call out whether a thread is channel-scoped, peer-scoped, or both
- Include the minimum viable but complete test matrix shape

Do not output anything outside the JSON object.
