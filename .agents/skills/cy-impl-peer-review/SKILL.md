---
name: cy-impl-peer-review
description: Runs an optional cross-LLM peer review of an implemented change via compozy exec --ide claude --model opus --reasoning-effort xhigh and requires the reviewer to write a scoped Markdown findings artifact for user-directed remediation. Use after any implementation pass (feature, bug fix, refactor) when the user explicitly asks for an external Opus review of the diff before commit or PR. Do not use for TechSpec review (use cy-spec-peer-review), automatic remediation, batched provider review fetching (use cy-fix-reviews), manual self-review without an external LLM (use cy-review-round), or auto-looped review cycles.
trigger: explicit
argument-hint: "[--files path1,path2] [--context path1,path2] [--base ref] [--out dir]"
---

# Implementation Peer Review

Claude Opus pressure-tests an implementation diff via `compozy exec`. This skill runs that pressure-test only when the user explicitly asks for a review round after an implementation pass. It is decoupled from any PRD/task tracking system — the scope is the diff itself plus any optional context files the user names. The skill never auto-runs, auto-incorporates findings, auto-commits, or auto-loops additional rounds.

The review result is a direct-written Markdown findings file. `compozy exec` stdout/stderr is operational evidence only; never parse it as the review source of truth.

## User Decisions

When this skill instructs the agent to ask whether to incorporate findings or run another round, it MUST use the runtime's dedicated interactive question tool — the tool or function that presents a question to the user and pauses execution until the user responds.

If the runtime does not provide such a tool, present the question as the complete assistant message and stop generating. Do not answer the question on the user's behalf.

## Bundled Path Rule

Resolve bundled helper paths relative to the directory that contains this `SKILL.md`. When invoking the validator from a repository root, use the full repo-relative path:

```bash
bash .agents/skills/cy-impl-peer-review/scripts/validate-findings.sh --kind implementation --round <N> --path <findings-path>
```

The validator is a read-only helper: it inspects the findings artifact and exits non-zero on structural contract violations.

## Optional Inputs

All inputs are optional. Defaults make the common path `cy-impl-peer-review` with no arguments.

- `--files <path1,path2,...>` — scope the review to explicit paths instead of the full branch diff.
- `--context <path1,path2,...>` — additional context files to feed Opus (e.g., a spec, ADR, design doc, RFC, README). The skill never assumes any of these exist.
- `--base <git-ref>` — base ref for the diff. Defaults to `main`. Use `--base HEAD~N` or `--staged` for narrower scopes.
- `--out <dir>` — output directory for round artifacts. Defaults to `.peer-reviews/<UTC-timestamp>/` at repo root.

## Findings Artifact Contract

Each review round has exactly one authoritative findings file:

```
<out>/impl-review-findings-roundN.md
```

The reviewer may write exactly that file and no other file. If the target path is missing, ambiguous, unwritable, or outside the repository/output directory named by the parent, the reviewer must refuse and stop. It must not print findings to stdout as a fallback.

The findings file MUST use this structure:

```markdown
---
schema_version: 1
review_kind: implementation
round: N
verdict: SHIP|FIX_BEFORE_SHIP|REWORK
reviewer_runtime: claude
reviewer_model: opus
generated_at: <ISO-8601 timestamp>
---

# Summary

# Blockers

# Risks

# Nits

# Evidence

# Deferred Or Follow-Up
```

Every blocker, risk, and nit must include an ID, a real file path and line when applicable, the issue, and a concrete suggested fix. Blockers must also include the rationale for why the issue blocks shipment.

## Procedures

**Step 1: Validate Input and Compute Scope**

1. Confirm the user has just completed (or paused) the implementation pass and explicitly asked to review the current state. Do not run review rounds during active editing.
2. Resolve the diff scope:
   - If `--files` is provided, verify each path exists and limit the diff to those paths.
   - If `--staged` is provided as the base, use `git diff --staged`.
   - Otherwise run `git diff <base>...HEAD --name-only` (default `<base>` is `main`) to compute the changed file set. If the diff is empty, abort and tell the user there is nothing to review.
3. Resolve the artifact directory:
   - Use `--out` if provided.
   - Otherwise default to `.peer-reviews/<UTC-timestamp-YYYYMMDDTHHMMSSZ>/` at the repository root.
   - Create the directory if it does not exist.
4. Read `.agents/skills/cy-impl-peer-review/references/readiness-checks.md` and verify every readiness marker passes (build/tests green, no committed `.tmp/` or `ai-docs/`, diff is non-empty, no obvious WIP markers in changed files, codegen co-ship if contracts touched, migration co-ship if schema touched, reviewable size). If any marker fails, report the failed markers and abort — Opus review on a broken or incomplete change wastes credit and produces noise.
5. Determine the next review round number by listing existing `impl-review-findings-round*.md`, `impl-review-summary-round*.md`, and legacy `impl-review-result-round*.json*` files (prior local output only — not a compatibility path) in the artifact directory. Start at `round1` when none exist.

**Step 2: Compose the Review Prompt**

1. Read `.agents/skills/cy-impl-peer-review/references/impl-review-prompt.md` for the canonical executable Opus prompt template. The assembled prompt must start with the reviewer instructions, not with a Markdown wrapper describing the template.
2. Capture the diff payload:
   - Run `git diff <base>...HEAD -- <changed-files>` (or `git diff --staged -- <changed-files>` when the user named `--staged`) and write the raw patch to `<out>/impl-review-diff-roundN.patch`.
   - Run `git log --oneline <base>...HEAD -- <changed-files>` and capture the commit list (empty string if `--staged`).
3. Define the round artifact paths:
   - Findings target: `<out>/impl-review-findings-roundN.md`.
   - Operational event log: `<out>/impl-review-events-roundN.jsonl`.
   - Operational stderr log: `<out>/impl-review-result-roundN.err`.
   - Pre-run status snapshot: `<out>/impl-review-status-before-roundN.txt`.
   - Post-run status snapshot: `<out>/impl-review-status-after-roundN.txt`.
   - Validation error, only when needed: `<out>/impl-review-validation-error-roundN.md`.
4. Substitute the placeholders in the prompt template:
   - `{scope_summary}` — one-paragraph description of what was implemented. Derive from the user's brief, the commit messages, or — if the user passed `--context` — the linked spec/PRD summary.
   - `{context_paths}` — newline-separated repo-root paths from `--context`, or the literal string `none` when not provided.
   - `{changed_files}` — newline-separated repo-root paths.
   - `{diff_path}` — repo-root path to the patch file from step 2.
   - `{findings_path}` — exact absolute path to `<out>/impl-review-findings-roundN.md`.
   - `{round}` — numeric review round `N`.
   - `{commit_list}` — captured `git log --oneline` output, or `none` if `--staged`.
5. Write the assembled prompt to `<out>/impl-review-prompt-roundN.md`.

**Step 3: Execute the Cross-LLM Review**

1. Capture the pre-run status snapshot:

   ```bash
   git status --short > <out>/impl-review-status-before-roundN.txt
   ```

2. Run:

   ```bash
   compozy exec --ide claude --model opus --reasoning-effort xhigh --format json --prompt-file <out>/impl-review-prompt-roundN.md > <out>/impl-review-events-roundN.jsonl 2> <out>/impl-review-result-roundN.err
   ```

3. Capture the post-run status snapshot:

   ```bash
   git status --short > <out>/impl-review-status-after-roundN.txt
   ```

4. If the command returns a non-zero exit code, fail loudly. Do not retry silently. Inspect stderr for model misconfiguration (see Error Handling).
5. Treat `impl-review-events-roundN.jsonl` as operational evidence only. Do not parse it for the review verdict or findings.
6. Require the findings target file to exist after the command exits. If missing, the round is invalid even when `compozy exec` exited 0.

**Step 4: Validate and Summarize Findings**

1. Run the bundled read-only validator:

   ```bash
   bash .agents/skills/cy-impl-peer-review/scripts/validate-findings.sh --kind implementation --round N --path <out>/impl-review-findings-roundN.md
   ```

2. Manually inspect the findings file and verify the semantic contract:
   - every finding has a real file path/line or an explicit reason line is not applicable;
   - blockers include a rationale tied to project rules, lessons, or architecture constraints;
   - no `TBD`, placeholder text, invented paths, or stdout-only findings;
   - comparing the pre/post status snapshots shows no changes outside the expected review artifact/log paths.
3. If validation fails, write `<out>/impl-review-validation-error-roundN.md` with the failed checks, command, exit status, and artifact paths. Do not summarize the round as `SHIP`.
4. Write `<out>/impl-review-summary-roundN.md` from the validated findings file with:
   - verdict (`SHIP` / `FIX_BEFORE_SHIP` / `REWORK`);
   - one-line rationale per blocker;
   - risks list;
   - nits list;
   - files most likely affected by remediation;
   - operational artifact paths.
5. Present a concise user-facing summary of the review. Include the verdict, blocker/risk/nit counts, the main themes, and the artifact paths written for the round.
6. Do NOT modify any source code, tests, configs, docs, or commits yet.

**Step 5: User-Directed Remediation**

1. Ask the user which findings to incorporate:
   - A) all blockers
   - B) selected blockers/risks/nits
   - C) nothing — keep the review as a record only
   - D) manual edits before any remediation
2. Apply only the findings the user selected. Do not silently apply all blockers, all risks, or all nits.
3. Re-run the project's verification gate (`make verify` in this repo, or whatever command the user names) after applying any code change. Do not declare remediation done if verification fails — fix the new failure or surface it back to the user.
4. Record the remediation decision in `<out>/impl-review-remediation-roundN.md`, listing:
   - incorporated items with the new commit/diff range;
   - deferred items;
   - files changed;
   - verification command and outcome with timestamp.
5. Show the user what changed and what remains deferred. Do not commit or push without an explicit user instruction.

**Step 6: Optional Additional Rounds**

1. Ask whether the user wants another peer-review round against the updated code or wants to stop with the current state.
2. If the user requests another round, re-run from Step 2 against the new diff and create a fresh `roundN+1` artifact set in the same `<out>` directory.
3. Do not auto-loop. The user explicitly requests further rounds.

## Critical Rules

- This skill never commits, pushes, opens PRs, or invokes provider review fetchers. Remediation is local-only; commit/PR steps belong to the user or `cy-fix-reviews`.
- The skill is not bound to any task-tracking directory layout. Every artifact lives under the resolved `<out>` directory and is versioned with `-roundN`. Never overwrite a prior round.
- The `compozy exec` call is the only place this skill spends external review credit. Do not invoke it more than once per round unless the round is explicitly invalid and the user requests a rerun.
- The bundled helper paths used by this skill (`references/impl-review-prompt.md`, `references/readiness-checks.md`, `scripts/validate-findings.sh`) are read-only templates/helpers — the skill reads or runs them, never edits them during a review round.

## Error Handling

- **Model misconfiguration (`The model 'X' does not exist`):** stop and surface the configured model. The IDE may be set to a stale name like `gpt-5.5`. Do not mutate the call to substitute a model — verify with the user. (See `docs/_memory/lessons/L-010-model-name-validation.md`.)
- **`compozy exec` not found:** the skill assumes Compozy CLI is on `PATH`. If absent, fail with the install hint rather than swallowing.
- **Readiness markers missing:** if Step 1 readiness checks fail, do not run Opus. Print the failed markers and exit so the user can fix the underlying problem first.
- **Empty diff:** if `git diff` yields no changes, abort. There is nothing to review.
- **Oversized diff (`> 5000` changed lines or `> 80` files):** warn the user, ask whether to scope down with `--files`, and proceed only on explicit confirmation. Opus review on a sprawling diff produces shallow findings.
- **Missing findings file:** treat this as an invalid round, not a clean review. Write a validation-error artifact and ask whether to rerun.
- **Malformed findings frontmatter or missing required sections:** treat this as an invalid round. Do not infer the verdict from stdout.
- **Existing peer-review files for the round:** never overwrite. Increment to the next `roundN` instead.
- **Verification failing during remediation:** stop and surface the new failure. Do not commit broken code to preserve the review trail.
