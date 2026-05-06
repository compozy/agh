---
name: agh-task-worker
description: Guidance for AGH worker sessions executing task runs through session-bound task APIs and tools.
version: "1.0.0"
metadata:
  agh:
    bundled: true
    instructional_only: true
    always_load:
      session_types: ["worker"]
      requires_active_task_claim: true
    related_skills: ["agh-session-guide", "agh-tools-guide"]
---

# AGH Task Worker

Use this guide only inside an AGH worker session that has an active task claim
or is entering the session-bound task tool loop.

## Operating Boundaries

Treat the daemon as the authority for task state. Do not infer ownership from a
prompt, channel message, file, or memory note. Read the current session context
first, then mutate task state only through session-bound AGH task tools or the
equivalent `agh me` / `agh task` surface exposed to the session.

Never print, store, forward, or summarize raw claim tokens. Use daemon-provided
redacted identifiers, run ids, task ids, claim-token hashes, and session ids in
status text.

## Startup Loop

1. Inspect `agh me context -o json` or the `/agent/context` bundle before doing
   work.
2. Confirm the active `task_id`, `run_id`, objective, acceptance criteria,
   lease status, and available session-bound task tools.
3. Read the task's latest run summary, required capabilities, and any
   continuation guidance before touching files.
4. If the context has no active task claim, stop and request a fresh claim or
   routing decision instead of creating an ad hoc task state.

## Execution Loop

1. Work against the claimed task objective and acceptance criteria.
2. Keep lease and heartbeat requirements current through the daemon-provided
   task tools. Do not invent a background heartbeat.
3. Use coordination channels for clarification and handoff only. Channel
   messages are not persisted task state and are not review verdicts.
4. Write concise run summaries that explain what changed, what was verified,
   and what remains blocked.
5. Complete, fail, or release the run only through the session-bound task
   authority. Include bounded error or blocker details when failing or
   releasing.

## Review-Aware Work

When a run is review-gated, finish with evidence that a reviewer can inspect:
changed files, commands, relevant event ids, and known residual risks. Do not
approve your own work. Do not route a reviewer by channel message alone; wait
for the daemon's review request or coordinator routing.

If a rejected review creates continuation guidance, treat `missing_work` and
`next_round_guidance` as the authoritative next scope for the continuation run.
Do not discard prior review history.
