---
name: cy-codex-loop
description: Drives end-to-end execution of a Compozy techspec across many agent restarts by detecting the current phase from state.yaml plus .compozy/tasks/<slug>/, then activating the correct cy-* skill chain or the Compozy -> Claude Opus delegation lane for one iteration before stopping. Each Phase B iteration ends with an atomic checkpoint commit so every completed task or slice becomes a restorable git snapshot. Use when running a Compozy techspec under the codex-loop-plugin in goal mode, or when manually iterating through tasks, qa-report, and qa-execution with persistent memory and state. Do not use for one-off tasks without a .compozy/tasks/<slug>/ directory, for techspec authoring (use cy-create-techspec), or for any work that does not require iteration tracking across agent restarts.
---

# Codex Loop Driver

Execute one deterministic iteration of a long-running Compozy techspec.
Each iteration: detect the current phase, run exactly one phase action,
write memory, update `state.yaml`, and stop. The next agent invocation
resumes from wherever the filesystem now indicates.

This skill is designed to live inside the Stop→restart loop of
`~/dev/ai/codex-loop-plugin` in goal mode. It is **not** required: the
same skill can be invoked manually for a single iteration, with the
human re-running it for each subsequent step. The plugin itself is not
modified — `optional_skill_path` is left untouched.

## Required Inputs

- `<slug>` — the directory name under `.compozy/tasks/` (for example, `agent-soul`).
- `<goal_text>` — the verbatim text the user pasted in `[[CODEX_LOOP goal="..."]]`, or the manual reason given for the run. Captured once at bootstrap into `state.yaml.goal_signature`.
- A pre-authored `.compozy/tasks/<slug>/_techspec.md`. Without it, the skill stops at bootstrap with a blocker.

## Helper scripts

All helpers are bundled under `.agents/skills/cy-codex-loop/scripts/`. Reference them by the explicit repo-root path when invoking, never by ambiguous shorthand.

| Script | Role | Used in phase |
|--------|------|---------------|
| `.agents/skills/cy-codex-loop/scripts/init-state.py` | bootstrap (mutating once) | 0 |
| `.agents/skills/cy-codex-loop/scripts/detect-phase.py` | read-only | every iteration |
| `.agents/skills/cy-codex-loop/scripts/update-state.py` | mutating | every iteration |
| `.agents/skills/cy-codex-loop/scripts/commit-checkpoint.py` | mutating (git commit) | B (per task / slice) |

Helpers are stdlib-only Python 3.11+. They never call the model, never
hit the network, never require a virtualenv.

## Frontend/docs delegation lane (opt-in)

Phase B has an **opt-in** secondary lane for work whose primary surface
is frontend or documentation. The lane is **OFF by default**: every
Phase B iteration runs locally in the orchestrator session regardless of
the task `type:` field, unless the user explicitly enabled the lane.

**Activation contract (deterministic):** the lane is active for the
current loop only when `state.goal_signature` contains the literal
token `delegation=frontend-docs` (case-insensitive). The user sets this
once at bootstrap by including the token in the `[[CODEX_LOOP goal=...]]`
header — see `references/goal-header-template.md`. There is no per-task
or per-iteration override; the loop either delegates frontend/docs work
for the whole run or it does not.

When the lane is active, read `references/frontend-docs-delegation.md`
for the dispatch contract: the local Codex loop session stays in
orchestration mode only, prepares the prompt, runs
`compozy exec --ide claude --model opus --prompt-file <tmpfile>`, waits
for explicit verify evidence, and never performs a competing local
implementation. When the lane is inactive, ignore that reference
entirely and execute every Phase B task or slice locally.

## Workflow (one iteration)

**Step 1: Read context and detect phase.**

1. Print `pwd` and confirm the working directory is the project repo root (the directory that contains `.compozy/tasks/`). If not, stop and report the mismatch — the helpers expect repo-root cwd.
2. Activate the `cy-workflow-memory` skill so its protocol is loaded for later use.
3. Run `python3 .agents/skills/cy-codex-loop/scripts/detect-phase.py <slug>`. The single-line output decides the rest of the iteration. Possible outputs are listed in `references/phase-transitions.md`.
4. Read `references/phase-transitions.md` for the action mapped to the printed phase.

**Step 2: Branch on the printed phase.**

Run exactly one of the branches below. Do not start another phase in the same iteration.

### Phase 0 — Bootstrap

1. Confirm `.compozy/tasks/<slug>/_techspec.md` exists. If missing → write a `## Open Risks` entry to `.compozy/tasks/<slug>/memory/MEMORY.md` (creating the file with the canonical sections from `references/memory-protocol.md`), call `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --blocker "techspec_missing" --action "bootstrap halted" --outcome blocked`, and stop.
2. Run `python3 .agents/skills/cy-codex-loop/scripts/init-state.py <slug> --goal "<goal_text>"`. The script auto-detects `mode=tasks` if `_tasks.md` plus at least one `task_*.md` exist; otherwise `mode=free`.
3. Activate `cy-spec-preflight` for whichever phase the next iteration will enter (`tasks` if mode=tasks, `task-body` if a single concrete file is about to be implemented).
4. Scaffold `.compozy/tasks/<slug>/memory/MEMORY.md` with the section schema from `references/memory-protocol.md`.
5. `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase B --action "bootstrap (mode=<mode>)" --outcome completed --memory-written "memory/MEMORY.md"`.

### Phase B mode=tasks — Execute one task

1. Pick the head of `state.tasks.pending`. Read the corresponding `.compozy/tasks/<slug>/<task_NN>.md` to confirm its frontmatter `status:` is `pending` or `in_progress`. Frontmatter wins — if it disagrees with state.yaml, reconcile state with `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --task-completed <stem>` for any already-finished task, or `--reconcile-tasks` when the task graph was authored after free-mode bootstrap, then re-run `.agents/skills/cy-codex-loop/scripts/detect-phase.py`.
2. Activate `cy-spec-preflight` in `task-body` mode for the picked file.
3. Check whether the frontend/docs delegation lane is active for this loop: it is active **only** when `state.goal_signature` contains the literal token `delegation=frontend-docs` (case-insensitive). When the lane is active, also read `references/frontend-docs-delegation.md` and apply its classification rules to decide whether THIS specific task qualifies (task frontmatter `type:` is `frontend`/`docs`, or owned paths are exclusively frontend/docs surfaces). When the lane is inactive, skip this delegation decision entirely and proceed to step 6.
4. Pass these paths to `cy-workflow-memory` in the lane that will execute the work (per `references/memory-protocol.md`): workflow memory directory `.compozy/tasks/<slug>/memory/`, shared `.compozy/tasks/<slug>/memory/MEMORY.md`, current `.compozy/tasks/<slug>/memory/<task_NN>.md`.
5. If the delegation lane applies (active AND task qualifies): prepare the temp prompt described in `references/frontend-docs-delegation.md` and run `compozy exec --ide claude --model opus --prompt-file /tmp/cy-codex-loop-<slug>-<task_NN>.md`. The delegated Claude run must own `cy-execute-task`, memory updates, validation, and `cy-final-verify`, but MUST NOT commit — the orchestrator owns the checkpoint commit (step 9).
6. Else: activate `cy-execute-task` against the picked task file with **auto-commit disabled** so the orchestrator owns the commit. Let it own implementation, validation, and self-review, then activate `cy-final-verify`. Capture its evidence text — it goes into the iteration summary.
7. Update memory before any state flip (sequence per `cy-execute-task` step 5). In the delegation lane, do not mark the task complete unless the delegated run exited successfully, updated the required memory files, and reported explicit PASS evidence from `cy-final-verify`.
8. `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase B --task-completed <stem> --action "executed <stem>" --outcome completed --memory-written "memory/<task_NN>.md,memory/MEMORY.md" --verify-pass`.
9. Run `python3 .agents/skills/cy-codex-loop/scripts/commit-checkpoint.py <slug> --task <stem>` to create the per-task checkpoint commit. The script's stdout is either a commit SHA or the literal `SKIP: no changes`; copy that value into the iteration summary's `commit_sha_or_skip_or_none` field. If the script exits 1 (pre-commit hook failure or git error), do NOT retry with `--no-verify`: run `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase B --verify-fail --action "checkpoint commit failed for <stem>" --outcome blocked --blocker "checkpoint-commit-failed: <stderr summary>"` and stop the iteration.

### Phase B mode=free — Execute one slice

1. Re-read `_techspec.md` deliverables and acceptance section in full. Compare against `state.progress.checklist[]`.
2. Identify the smallest coherent slice (≤ ~4 hours) that advances at least one acceptance criterion. Capture the slice text exactly.
3. `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --add-progress "<slice text>" --action "slice picked" --outcome completed`. The script appends the slice with `status: in_progress`.
4. Re-read `state.yaml` after step 3. Pass these paths to `cy-workflow-memory`: workflow memory directory `.compozy/tasks/<slug>/memory/`, shared `.compozy/tasks/<slug>/memory/MEMORY.md`, current `.compozy/tasks/<slug>/memory/free-iter-<NNN>.md` where `<NNN>` is the `iteration` value on the checklist entry that step 3 just created, zero-padded to three digits.
5. Check whether the frontend/docs delegation lane is active (the same `state.goal_signature` token check from mode=tasks step 3). When inactive, skip this decision and proceed to step 7. When active, read `references/frontend-docs-delegation.md` and apply its classification: this slice qualifies only when its owned paths are exclusively frontend/docs surfaces per that reference.
6. If the delegation lane applies (active AND slice qualifies): prepare the temp prompt described in `references/frontend-docs-delegation.md` and run `compozy exec --ide claude --model opus --prompt-file /tmp/cy-codex-loop-<slug>-free-iter-<NNN>.md`. The delegated Claude run must own implementation, memory updates, validation, and `cy-final-verify`, but MUST NOT commit — the orchestrator owns the checkpoint commit (step 10).
7. Else: implement the slice locally. Keep scope tight. Record decisions and learnings in the current memory file's canonical sections, then activate `cy-final-verify`.
8. Self-check: re-read the techspec acceptance section. If every criterion has a `status: completed` checklist entry, set `--deliverables-complete` in the next call. In the delegation lane, do not complete the slice unless the delegated run exited successfully, updated the required memory files, and reported explicit PASS evidence from `cy-final-verify`.
9. `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase B --complete-progress "<slice text>" [--deliverables-complete] --action "slice <text>" --outcome completed --memory-written "memory/free-iter-<NNN>.md,memory/MEMORY.md" --verify-pass`.
10. Run `python3 .agents/skills/cy-codex-loop/scripts/commit-checkpoint.py <slug> --slice "<slice text>"` (pass the exact text from step 3) to create the per-slice checkpoint commit. Same SKIP / exit-1 semantics as `mode=tasks` step 9: surface the SHA or `SKIP: no changes` in the iteration summary, and on exit 1 mark the iteration blocked via `update-state.py --verify-fail --blocker "checkpoint-commit-failed: ..."` instead of bypassing the hook.

### Phase C — QA

The printed action is either `qa_report` or `qa_execution`. Only run the printed one.

1. If `.compozy/tasks/<slug>/qa/bootstrap-manifest.json` is missing AND a QA bootstrap skill is installed (e.g. `agh-qa-bootstrap` in AGH), activate it first. If no such skill exists in this project, skip and let `qa-report` / `qa-execution` create what they need.
2. For `qa_report`: activate the `qa-report` skill. After it produces its artifacts under `.compozy/tasks/<slug>/qa/`, run `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase C --qa-report-done --action "qa-report produced" --outcome completed --memory-written "memory/qa-report.md,memory/MEMORY.md"`.
3. For `qa_execution`: activate `qa-execution`. After it produces `verification-report.md`, run `python3 .agents/skills/cy-codex-loop/scripts/update-state.py <slug> --phase C --qa-execution-done --action "qa-execution produced" --outcome completed --memory-written "memory/qa-execution.md,memory/MEMORY.md"`. If verification reports FAIL, also pass `--verify-fail`; otherwise `--verify-pass`.

### Phase E — Done

1. Run a final `cy-final-verify` and confirm `state.verify.last_status=PASS`. If verify fails here, treat as a Phase C regression: re-enter C and do not emit the done-signature.
2. Read `references/checklist.md` Phase E section and confirm every item passes.
3. Skip Step 3 below. Instead, print the iteration summary block from `assets/iteration-summary.template.md` once, with `phase_out=E`. The `commit_sha_or_skip_or_none` field for this iteration is `n/a (phase != B)`.
4. On a separate line, print the literal contents of `assets/done-signature.txt`. This is the evidence the codex-loop goal-check confirmation prompt scans for.
5. Stop. Do not start another action.

**Step 3: Self-audit and emit the iteration summary.**

1. Walk `references/checklist.md` for the phase that was just executed. Every box must pass.
2. For phases 0-C, print the iteration summary block from `assets/iteration-summary.template.md`. It MUST be the last block of the assistant message. Phase E already emitted the summary in its branch and adds only the done-signature line after it.

## Memory protocol

Memory is written via the existing `cy-workflow-memory` skill — `cy-codex-loop` does not invent its own format. The exact paths to pass per phase are in `references/memory-protocol.md`. Update memory **before** flipping any tracking field, mirroring `cy-execute-task` step 5.

## Goal-mode integration

When invoked under `~/dev/ai/codex-loop-plugin` in goal mode, the user pastes a prompt header that names this skill. The canonical template is in `references/goal-header-template.md`. Nothing in the plugin's `~/.codex/codex-loop/config.toml` needs to change.

## Critical Rules

- One phase action per iteration. Do not chain "execute_task" → "qa_report" in the same response — let the loop drive the next iteration.
- `state.yaml` is mutated **only** through `.agents/skills/cy-codex-loop/scripts/init-state.py` and `.agents/skills/cy-codex-loop/scripts/update-state.py`. Hand-edits void resume guarantees.
- `state.yaml` has no top-level `current_phase`. `detect-phase.py` computes the next phase from durable state and filesystem truth; `update-state.py --phase` only labels the appended `iterations[]` entry.
- Frontmatter `status:` on `task_NN.md` and `issue_NNN.md` is the source of truth. State.yaml mirrors it for fast detection; reconcile state if they disagree.
- Memory updates precede status flips. Always.
- The frontend/docs delegation lane is **opt-in** and **OFF by default**. It activates only when `state.goal_signature` (captured verbatim at bootstrap) contains the literal token `delegation=frontend-docs` (case-insensitive). Without that token, every Phase B iteration — including tasks whose `type:` is `frontend` or `docs` — runs locally in the orchestrator session. When the token IS present and the picked task/slice qualifies per `references/frontend-docs-delegation.md`, the local session becomes an orchestrator only and must dispatch via `compozy exec --ide claude --model opus`.
- Every Phase B iteration ends with `commit-checkpoint.py`. The orchestrator owns the commit; `cy-execute-task` MUST be invoked with auto-commit disabled, and the delegation-lane prompt MUST instruct the delegated Claude run never to commit. The checkpoint captures code, memory, task frontmatter, master tasks file, and the advanced `state.yaml` in one atomic snapshot — this is the only mechanism that makes each iteration a restorable git checkpoint.
- Phase E requires `qa.report_done=true`, `qa.execution_done=true`, and `make verify` PASS (`verify.last_status=PASS`). If verify is not PASS, `detect-phase.py` re-emits `phase=C action=qa_execution` instead of declaring done.
- Do not invoke `cy-create-tasks`, `cy-create-techspec`, `cy-tasks-tail-qa-pair`, or `cy-web-docs-impact` from this skill. Spec and task authoring is a separate workflow; this skill consumes their output.

## Error Handling

- **`_techspec.md` missing at bootstrap**: write blocker to memory, run `.agents/skills/cy-codex-loop/scripts/update-state.py --blocker "techspec_missing"`, stop.
- **Mode disagreement**: `.agents/skills/cy-codex-loop/scripts/init-state.py` exits 4 if `--mode` overrides the filesystem-detected mode. Reconcile by adding/removing `_tasks.md` before bootstrap, or by running `.agents/skills/cy-codex-loop/scripts/update-state.py <slug> --reconcile-tasks` when `_tasks.md` and task files were intentionally authored after a free-mode bootstrap.
- **`state.yaml` parse failure**: `.agents/skills/cy-codex-loop/scripts/detect-phase.py` exits 1 with the parse error on stderr. Inspect the file; the most likely cause is hand-editing. Restore from `git diff` and resume.
- **`cy-final-verify` FAIL**: do not advance the phase. Run `.agents/skills/cy-codex-loop/scripts/update-state.py --verify-fail --action "verify FAIL: <summary>" --outcome blocked` and let the next iteration re-enter the same phase to fix the failure. After two consecutive verify failures in the same phase, declare a blocker and stop (per the two-touch rule).
- **`commit-checkpoint.py` exit 1 (pre-commit hook failure or git error)**: do not retry with `--no-verify`. Record the failure via `update-state.py --verify-fail --action "checkpoint commit failed for <stem|slice>" --outcome blocked --blocker "checkpoint-commit-failed: <stderr summary>"` and stop the iteration. The next iteration re-enters Phase B; the user must resolve the underlying issue (typically a lint/test regression in the just-completed work) before the loop can continue. `SKIP: no changes` on stdout is NOT a failure — it just means the iteration produced no diff and is recorded that way in the iteration summary.
- **Delegated Compozy/Claude run exits non-zero or lacks explicit PASS evidence**: do not advance the phase. Record the failure as blocked (or `--verify-fail` when appropriate), cite the missing verify evidence, and let the next iteration re-enter the same phase.
- **Two-touch rule**: if the same task or issue area receives a third corrective change in this loop, escalate to a blocker rather than landing the third patch. Many projects mandate this in their CLAUDE.md / AGENTS.md; honor that if present.
- **Blocker recorded**: stop without the done-signature even if `phase=E` was printed. The next iteration will re-detect the same blocker until the human resolves it.
