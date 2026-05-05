# Frontend/docs delegation lane

Use this lane when Phase B work is primarily frontend or documentation.
The local Codex loop session remains the orchestrator; Claude Opus does
the implementation and verification work through the Compozy CLI.

## Classification

1. Trust task frontmatter `type:` first.
2. Delegate when `type:` is exactly `frontend` or `docs`.
3. In free mode (no task frontmatter), delegate only when the chosen
   slice is explicitly limited to frontend/docs surfaces such as:
   - `web/**`
   - `packages/ui/**`
   - `packages/site/**`
   - `docs/**`
   - repo-root product docs/copy surfaces (`COPY.md`, `DESIGN.md`,
     `README*.md`, `*.md`, `*.mdx`)
4. Keep mixed backend/runtime slices local unless the slice text narrows
   the iteration to the frontend/docs subset only.

## Dispatch contract

1. Write a temporary prompt file under `/tmp/` named
   `/tmp/cy-codex-loop-<slug>-<action>.md`.
2. Fill the prompt with:
   - repo root
   - workflow slug
   - current action (`task_NN` or `free-iter-NNN`)
   - exact task file path or slice text
   - shared memory path and current memory path
   - verification requirement (`cy-final-verify` PASS is mandatory)
   - instruction to read all scoped `AGENTS.md` / `CLAUDE.md` files
3. Run exactly:

```bash
compozy exec --ide claude --model opus --prompt-file /tmp/cy-codex-loop-<slug>-<action>.md
```

4. Use this Compozy lane instead of a competing local
   `cy-execute-task` run.

## Prompt requirements

Require the delegated Claude Opus run to:

- read the task file / `_techspec.md` / design docs and the scoped
  instructions for the touched surfaces
- activate `cy-spec-preflight` when the task file requires task-body
  discipline
- use `cy-workflow-memory` with the provided shared/current memory paths
- perform the implementation itself
- run the relevant validation commands for the touched surface
- run `cy-final-verify`
- update memory before changing task status or completion markers
- print changed files plus explicit PASS/FAIL verify evidence

## Completion gate

The orchestrating Codex iteration may mark the Phase B action complete
only when all of the following are true:

- the `compozy exec` command exited successfully
- the delegated run updated the required memory files
- the task/status artifacts now reflect completion
- the delegated output contains explicit `cy-final-verify` PASS evidence

If any item is missing, keep the phase open and record a blocker or
verify failure instead of advancing `state.yaml`.
