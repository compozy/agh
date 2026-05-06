Goal (incl. success criteria):

- Implement project-local codex-loop `pre_loop_continue` review hook for AGH.
- Success: accepted plan persisted; root `codex-loop.toml` config added; hook script supports cy-codex-loop task-aware review and generic implementation review; static/dry-run validation passes.

Constraints/Assumptions:

- Do not run destructive git commands.
- Conversation may be Portuguese; durable artifacts are English.
- Hook is blocker-lite and read-only: no state.yaml/frontmatter/review artifact/git mutation.
- Use project-local config only; do not edit global `~/.codex/codex-loop/config.toml` or `~/.codex/hooks.json`.

Key decisions:

- Use `python3 .codex/scripts/codex-loop-pre-loop-review.py --input $INPUT_FILE` in `codex-loop.toml` to avoid executable-bit dependence.
- Detect cy-codex-loop mode only from concrete prompt/state evidence; otherwise run generic implementation review.
- Claude review is injected into continuation prompt via stdout, not stored as a durable review round.

State:

- Implementation complete; validation complete; live Compozy review smoke completed.

Done:

- Read current ledger list and required codex-loop/compozy/cy-codex-loop skill context.
- Persisted accepted plan under `.codex/plans/2026-05-05-codex-loop-pre-loop-review.md`.
- Added project-local `codex-loop.toml` with `pre_loop_continue` command and continuation guidance.
- Added `.codex/scripts/codex-loop-pre-loop-review.py`.
- Hardened cy-codex-loop detection so a matching loop slug alone is not enough; the prompt must carry `$cy-codex-loop`/skill marker or explicit `.compozy/tasks/<slug>` reference.
- Added `sys.dont_write_bytecode = True` to prevent hook-time `__pycache__` writes.
- Validation passed:
  - `PYTHONDONTWRITEBYTECODE=1 python3 -m py_compile .codex/scripts/codex-loop-pre-loop-review.py`
  - TOML parse via Python `tomllib`
  - `git diff --check` for changed files
  - dry-run cy path selected `Mode: cy-codex-loop-task`, `Target: mem-v2/task_18`
  - dry-run false-positive path with loop slug `mem-v2` but no cy marker selected `Mode: generic`
  - isolated `CODEX_HOME` Stop hook smoke injected `pre_loop_continue output` with `AGH_PRE_LOOP_REVIEW`
  - `make verify` passed: Bun 330 files / 2090 tests; Go lint 0 issues; Go 8356 tests; package boundaries OK.
- Live test without `CODEX_LOOP_REVIEW_DRY_RUN` called Compozy/Claude against latest completed cy-codex-loop task.
  - Target selected: `Mode: cy-codex-loop-task`, `Target: mem-v2/task_18`.
  - Claude returned valid `AGH_PRE_LOOP_REVIEW`.
  - Verdict: `PASS`, confidence `0.78`, with non-blocking risks.
  - Test harness command exited 1 only because zsh treats variable name `status` as read-only; hook output itself was valid.

Now:

- Ready to report live Compozy review evidence.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `codex-loop.toml`
- `.codex/scripts/codex-loop-pre-loop-review.py`
- `.codex/plans/2026-05-05-codex-loop-pre-loop-review.md`
- `.codex/ledger/2026-05-05-MEMORY-codex-loop-review-hook.md`
- Existing unrelated status still present: `.agents/skills/cy-codex-loop/scripts/__pycache__/_state_io.cpython-314.pyc` modified; not restored or touched intentionally.
- Live command: `PYTHONDONTWRITEBYTECODE=1 python3 .codex/scripts/codex-loop-pre-loop-review.py --input /tmp/agh-live-cy-loop-payload.*`
