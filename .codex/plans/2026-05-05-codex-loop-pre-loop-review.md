# Conditional AGH `pre_loop_continue` Review Hook

## Summary

Add a project-local `codex-loop.toml` and a small AGH-owned hook script that asks Claude Opus, via `compozy exec --ide claude --model opus`, to review the latest implementation before each automatic codex-loop continuation.

The hook has two modes:

- **`cy-codex-loop` mode:** when the active loop is clearly running `$cy-codex-loop`, review against the specific Compozy task or free-mode slice that was just completed.
- **Generic loop mode:** when the active loop is not using `$cy-codex-loop`, review the implementation work done so far using the loop prompt, latest assistant message, and current repo changes/status.

The hook remains blocker-lite: it injects review findings into the next continuation prompt, but does not mutate `state.yaml`, task frontmatter, review artifacts, or git state.

## Key Changes

- Add root `codex-loop.toml` with project-local `[pre_loop_continue]`:
  - `cwd = "workspace_root"`.
  - `command = "python3 .codex/scripts/codex-loop-pre-loop-review.py --input $INPUT_FILE"`.
  - `timeout_seconds = 720`.
  - `max_output_bytes = 12000`.
  - `extra_continuation_guidance` tells Codex to treat injected `Verdict: FIX` findings as mandatory work before stopping again.

- Add `.codex/scripts/codex-loop-pre-loop-review.py`:
  - Reads the JSON payload from `--input $INPUT_FILE` or stdin.
  - Resolves the repo root from `payload.workspace_root` / `$WORKSPACE_ROOT`.
  - Runs from the workspace root and never writes files.
  - Creates a temporary prompt file under the system temp dir.
  - Calls `compozy exec --ide claude --model opus --access-mode default --timeout 8m --prompt-file <temp-prompt>`.
  - Prints only compact Markdown review output to stdout so codex-loop injects it into the next continuation prompt.
  - Exits `0` for skips, missing context, Claude failures, or timeouts, printing a concise advisory instead of breaking the loop.

- Detect `cy-codex-loop` mode only when there is concrete evidence:
  - `payload.loop.task_prompt` or `payload.loop.activation_prompt` contains `$cy-codex-loop`, `/cy-codex-loop`, or `.agents/skills/cy-codex-loop`.
  - Or the prompt explicitly references `.compozy/tasks/<slug>` and that directory contains `state.yaml`.
  - The loop slug matching an existing `.compozy/tasks/<slug>/state.yaml` is not sufficient by itself; ordinary codex-loop runs can reuse similar names and must stay in generic review mode.

- In `cy-codex-loop` task mode:
  - Load `.compozy/tasks/<slug>/state.yaml` using `.agents/skills/cy-codex-loop/scripts/_state_io.py`.
  - If the latest completed iteration is `executed task_NN`, review against the task file, task memory, shared memory, state file, latest assistant message, and continuation reason.
  - Confirm `task_NN.md` frontmatter has `status: completed`; if not, fall back to generic review with a warning.
  - If the latest `cy-codex-loop` iteration is a free slice, QA phase, or review-fix phase, review against the relevant state/memory artifacts rather than pretending there is a `task_NN`.
  - If no completed `cy-codex-loop` action is identifiable, skip task-specific review and use generic mode.

- In generic loop mode:
  - Do not require `.compozy/tasks`.
  - Build the Claude prompt from the loop prompt, latest assistant message, continuation reason, `git status --short`, `git diff --stat`, `git diff --name-only`, and bounded changed-file diffs.
  - Ask Claude to review the implementation so far relative to the original loop prompt and latest assistant claims.
  - Require Claude to focus on real bugs, regressions, missing verification, incomplete requirements, and risky shortcuts.

## Test Plan

- Static checks:
  - `python3 -m py_compile .codex/scripts/codex-loop-pre-loop-review.py`
  - `git diff --check`
  - Parse `codex-loop.toml`.

- Script dry-run checks:
  - `CODEX_LOOP_REVIEW_DRY_RUN=1` with fake `pre_loop_continue` JSON payloads for both `$cy-codex-loop` and generic loop modes.
  - Confirm selected mode, target, prompt intent, and command intent without calling Claude.

- Optional live check:
  - Run the script once with real `compozy exec --ide claude --model opus` against the current AGH workspace.
  - Confirm output obeys the contract and remains below `max_output_bytes`.

## Assumptions

- The same project hook should support both structured `$cy-codex-loop` workflows and ordinary codex-loop work.
- `cy-codex-loop` task-aware review must happen only when the active loop provides concrete evidence that it is using that skill.
- Generic review should be based on implementation evidence available from the workspace, not on a Compozy task file.
- `Verdict: FIX` is advisory-but-mandatory for the next continuation prompt; the hook does not enforce it by mutating loop state.
- `features.codex_hooks = true` is already enabled globally.
