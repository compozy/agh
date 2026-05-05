# Claude Research Prompt — Network Threads Round 2 (Concise)

You are doing a second-pass architecture recommendation for AGH Network threads.

The first research round already established the current state:

- AGH today has `channel`, `interaction_id`, `reply_to`, `trace_id`, and `causation_id`
- there is no first-class `thread_id` in RFC, runtime, store, contract, or web UI
- public channel conversations mix together in one flat timeline
- peer rooms are a UI split, not a protocol primitive

Your task now is narrower:

- choose the best architecture fork
- keep the answer concise enough to fit in one complete JSON object
- bias toward greenfield hard cuts and minimal but complete semantics

## Read Before Answering

Read these files:

- `/Users/pedronauck/Dev/compozy/agh/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/rfcs/003_agh-network-v0.md`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/lifecycle.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/router.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/contract/contract.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/api/core/network_details.go`
- `/Users/pedronauck/Dev/compozy/agh/internal/store/types.go`
- `/Users/pedronauck/Dev/compozy/agh/web/src/hooks/routes/use-network-page.ts`
- `/Users/pedronauck/Dev/compozy/agh/web/src/systems/network/components/network-workspace-shell.tsx`
- `/Users/pedronauck/Dev/compozy/agh/internal/skills/bundled/skills/agh-network/SKILL.md`
- `/Users/pedronauck/Dev/compozy/agh/internal/network/delivery.go`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/Multi-Agent Orchestration Patterns.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/Agent Handoff and Context Transfer.md`
- `/Users/pedronauck/dev/knowledge/agent-networks/wiki/concepts/The A2A Protocol.md`
- `/Users/pedronauck/dev/knowledge/claude-code/wiki/concepts/Agent Swarm and Subagents.md`

## Question To Resolve

Which model should AGH adopt?

1. `thread` becomes a first-class protocol field distinct from `interaction`
2. `interaction` is redefined to serve as the thread primitive
3. another option

You must pick one.

## Output Rules

- Return strict JSON only
- No prose outside JSON
- Keep every string under 220 characters
- Prefer 4-8 items per list, not huge exhaustive dumps
- Do not repeat the same point across sections

## Output Shape

```json
{
  "recommended_model": {
    "name": "thread-primitive|interaction-becomes-thread|other",
    "summary": "..."
  },
  "why_not_the_others": [
    {
      "name": "...",
      "reason": "..."
    }
  ],
  "core_invariants": ["..."],
  "must_change_now": {
    "protocol": ["..."],
    "runtime": ["..."],
    "cli_http_web": ["..."],
    "prompts_and_skills": ["..."],
    "tests_and_docs": ["..."]
  },
  "delete_targets": ["..."],
  "human_decisions_needed": [
    {
      "id": "Q-001",
      "question": "...",
      "why": "..."
    }
  ],
  "proposed_adrs": [
    {
      "title": "...",
      "decision": "..."
    }
  ],
  "summary": "..."
}
```

Important:

- If you pick `thread` as first-class, be explicit about the surviving role of `interaction_id`
- If you pick `interaction` as thread, be explicit about what lifecycle semantics get deleted or moved
- Call out whether peer rooms survive as first-class UX or become derived views
