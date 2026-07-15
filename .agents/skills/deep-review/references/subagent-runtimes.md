# Subagent Runtimes (`--subagent`)

How Step 4 review agents (cohort reviewers, sweeps, skeptics, minor batch verifiers) execute. `native` — the default — uses the Workflow/Agent fan-out in orchestration.md; every other value runs the SAME prompts cross-LLM through `compozy exec`, one invocation per agent. Step 2 context-pack helpers stay native in every mode: they gather context, and the paid runtime spends only on judgment.

## Runtime map

| Value | Invocation |
| --- | --- |
| `claude-opus` | `compozy exec --ide claude --model opus --reasoning-effort max` |
| `claude-fable` | `compozy exec --ide claude --model fable-5 --reasoning-effort max` |
| `grok` | `compozy exec --ide cursor-agent --model 'grok-4.5[effort=high,fast=true]'` — effort/fast ride inside the model value (no reasoning flag); requesting `grok-4.5` resolves to the same advertised variant |
| `codex` | `compozy exec --ide codex --model gpt-5.6-sol --reasoning-effort xhigh` |

## Invocation shape (per agent)

1. Write the agent's prompt — the orchestration.md template with placeholders filled — to `<out>/agents/<label>.prompt.md`. Embed the JSON schema (findings or skeptic, from orchestration.md) in the prompt: external runtimes have no schema-enforcement layer, so the file contract below replaces it. Append the output contract below.
2. Run from the repo root:

   ```bash
   compozy exec <runtime flags from the map> --format json --timeout 30m \
     --prompt-file <out>/agents/<label>.prompt.md \
     > <out>/agents/<label>.events.jsonl 2> <out>/agents/<label>.err
   ```

3. Read `<out>/agents/<label>.json` — the agent's only product — and validate it against the schema. The JSONL/stderr streams are operational evidence only; never parse them as review output.

Run at most 4 invocations concurrently (background Bash); each is a full ACP session, not a thread.

## Output contract (append to every prompt)

```
OUTPUT CONTRACT:
- The repo checkout is read-only for you; modify nothing.
- Write your result as JSON matching the schema above to exactly this file:
  <out>/agents/<label>.json — no other file, no sibling artifacts.
- Do not print the result to stdout. If you cannot write that exact path,
  stop and report the failure in one sentence.
- After writing, reply with the single sentence: Wrote <label>.json
```

## Failure handling

- **Missing or schema-invalid output file** — the agent run is void even when `compozy exec` exited 0. Retry once with the same prompt; on a second failure run that one agent on the `native` path and record the substitution in review.md — the no-skip invariant outranks runtime purity.
- **`model "X" is not available`** — the error lists the runtime's advertised options. Surface them and stop; never substitute a model silently (L-010).
- **`did not advertise an ACP model option`**, or `compozy` missing from PATH — stop and name the gap; external review has no alternate transport.
- **Non-zero exit** — read `<label>.err` and fail loudly; retry once only when the cause is transient (timeout, session drop).

## Cost

Every external invocation spends `compozy exec` credit — a large PR fans out dozens of agents. `native` fits exploratory runs; external runtimes earn their spend on gate rounds (e.g. loop Phase D's `codex` lane).
