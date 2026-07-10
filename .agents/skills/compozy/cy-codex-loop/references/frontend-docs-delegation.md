# Frontend/docs delegation lane (opt-in)

This lane is **OFF by default**. Without an explicit opt-in, every
Phase B iteration runs locally in the orchestrator session regardless of
task `type:` or owned paths — do not read further into this file unless
the activation gate below is satisfied.

## Activation gate

The lane is active for the current loop only when
`state.goal_signature` contains the literal token
`delegation=frontend-docs` (case-insensitive). The user sets this once
at bootstrap by including the token in their `[[CODEX_LOOP goal="..."]]`
header (see `goal-header-template.md`). The opt-in spans the whole loop:
either every qualifying Phase B iteration delegates, or none do.

When the token is absent: skip this entire reference. The Phase B
branches in `SKILL.md` fall through to the local lane.

## Classification (only when the gate is active)

When the lane is active, apply these rules to decide whether THIS
specific task or slice qualifies. Tasks that do not qualify still run
locally even with the lane active.

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
- **DO NOT commit.** The orchestrating cy-codex-loop session owns the checkpoint commit and runs `commit-checkpoint.py` after receiving PASS evidence. The delegated run must leave the worktree dirty (staged or unstaged) so the orchestrator's `git add -A` captures everything.

## Completion gate

The orchestrating Codex iteration may mark the Phase B action complete
only when all of the following are true:

- the `compozy exec` command exited successfully
- the delegated run updated the required memory files
- the task/status artifacts now reflect completion
- the delegated output contains explicit `cy-final-verify` PASS evidence
- the delegated run did NOT create a commit (verified by comparing `git rev-parse HEAD` before and after the dispatch)

If any item is missing, keep the phase open and record a blocker or
verify failure instead of advancing `state.yaml`.
