---
name: deep-review
description: Deep, CodeRabbit-grade review of a branch diff or GitHub PR at any size — funnels changed files, shards them into cohorts, fans out reviewer and sweep agents, adversarially verifies findings, and emits a walkthrough, severity-tagged findings with committable suggestions and AI-agent fix prompts, and a SHIP/FIX_BEFORE_SHIP/REWORK verdict. Use when the user asks for a deep review of a PR or branch, wants CodeRabbit-style output without the 300-file cap, needs an incremental re-review after new pushes, wants findings published to the PR, or needs a peer-review verdict round (the loop skills' Phase D), optionally cross-LLM or judged against a spec's contract artifacts. Don't use for applying fixes to findings, reviewing specs or PRDs as documents, or quick feedback on a single file.
trigger: explicit
argument-hint: "[--pr N | --base <ref> | --staged] [--files p1,p2] [--spec <path>] [--subagent native|claude-opus|claude-fable|grok|codex] [--profile chill|assertive] [--publish] [--full] [--out <dir>] [--no-workflow]"
---

# Deep Review

Review at CodeRabbit grade with no file cap: **funnel** the diff down to reviewable source, assemble the **rubric**, shard the selection into **cohorts**, fan out reviewers and global sweeps, put findings through a **skeptic**, then render a walkthrough + findings report with a ship **verdict** — and, on request, publish it to the PR. A finding is a verified claim with cited evidence; anything unverified is dropped or downgraded before the user ever sees it.

`<skill-dir>` below means the directory containing this SKILL.md.

## Inputs (all optional)

| Flag | Meaning | Default |
| --- | --- | --- |
| `--pr <n>` | Review a GitHub PR (requires authenticated `gh`; head fetched locally) | — |
| `--base <ref>` / `--staged` | Local diff scope | merge-base with the origin default branch |
| `--files <p1,p2>` | Restrict review to these paths | full diff |
| `--spec <path>` | Spec file or directory; its contract-bearing artifacts become the conformance baseline (spec-parity sweep + verdict gate) | — |
| `--subagent <runtime>` | Step 4 reviewer runtime: `native` \| `claude-opus` \| `claude-fable` \| `grok` \| `codex` — non-native runs cross-LLM via `compozy exec` (subagent-runtimes.md) | `native` |
| `--profile chill\|assertive` | Noise gate — see taxonomy | repo config, else `chill` |
| `--publish` | Post walkthrough + review to the PR | off — local report only |
| `--full` | Ignore prior state; review the whole diff again | incremental when state exists |
| `--out <dir>` | Artifact directory | `.deep-review/<target>/` |
| `--no-workflow` | Skip the Workflow tool; use Agent fan-out | Workflow when available |

## Repo config — `.deep-review.yaml`

Optional repo-root file, the skill-native config standard. Any key absent there falls back to its `.coderabbit.yaml` counterpart (`reviews.*`), so repos migrating from CodeRabbit work unconfigured. Top-level keys, all optional:

| Key | Meaning |
| --- | --- |
| `profile` | `chill` \| `assertive` noise gate (taxonomy) — the `--profile` flag overrides |
| `path_filters` | Globs over repo-relative paths: `!pat` excludes; bare patterns, when present, restrict review to their matches and beat any exclude; built-in excludes (locks, vendor, generated, testdata, snapshots) always append |
| `path_instructions` | `path` glob + verbatim `instructions` entries — the highest-precedence rubric source (Step 2) |
| `request_changes_workflow` | publish-mode review-event gate (publish-github.md) |

The manifest builder resolves `path_filters` + `profile` into manifest.json; the context pack ingests `path_instructions`.

## Hard rules

- Source is read-only. Writes go only to `<out>`, `.deep-review/` state, and — with `--publish` — the target PR.
- No file-count cap: a large selection means more cohorts, never a skipped or silently truncated review. Every selected file lands in exactly one cohort.
- Every Critical and Major finding passes a skeptic before it is reported, and the skeptic's evidence ships with the finding.
- Run the repo's linters first and suppress every finding they already catch.
- Cite rubric rules verbatim with their source path; severity comes from the taxonomy, never inflated.
- Publishing needs `--publish` or the user's explicit go-ahead in this session; otherwise the review stays local.
- Every review ends with a **SHIP / FIX_BEFORE_SHIP / REWORK** verdict, derived only by output-contracts.md's verdict rule and stated in review.md, state.json, and the final message.
- External `--subagent` runtimes spend `compozy exec` credit; their invocation, output contract, and failure handling live in subagent-runtimes.md.

## Procedure

**Step 1: Funnel — build the manifest**

1. Run the bundled manifest builder (bootstrap helper; reads the repo and `gh`, writes only under `--out`):

   ```bash
   python3 <skill-dir>/scripts/build_manifest.py --out <out> \
     [--pr N | --base REF | --staged] [--files p1,p2] [--full]
   ```

   It resolves the repo config into the applied path filters + profile (both recorded in the manifest), detects generated / trivial / renamed files, and scopes to the incremental delta when prior state exists.
2. Read the printed summary. On `--pr`, the script errors with the exact `git fetch` command when the head SHA is absent — run it and retry.

*Done when:* `<out>/manifest.json` exists, every changed file is accounted for as selected, ignored(reason), or skipped(reason), and every selected file carries its hunk list (new-side line ranges — the units of judgment and the publish anchors).

**Step 2: Context pack — rubric, linters, symbol map**

1. Read `<skill-dir>/references/context-pack.md` and assemble `<out>/context-pack.md` exactly as it specifies: the numbered rule registry (rubric sources with precedence), the rule map binding rule ids to each selected file, linter results scoped to selected files, per-file related-symbol notes, and — with `--spec` — the Spec contract section listing the resolved conformance artifacts.
2. Above ~40 selected files (or ~3 guideline sources), divide the digging across native subagents per that file's fan-out section — one rubric extractor, symbol mappers per cohort group — and merge their returns; the orchestrator writes context-pack.md, the subagents only search and report.

*Done when:* context-pack.md lists every rubric source consulted, every selected file has a rule-id list (or an explicit `—`), each detected linter lane has a ran/unavailable verdict, and — with `--spec` — every resolved contract artifact is listed.

**Step 3: Walkthrough + cohort plan**

1. Read `<skill-dir>/references/output-contracts.md` (walkthrough anatomy, effort scale) and `<skill-dir>/references/orchestration.md` (cohort rules, sweep lenses).
2. Write `<out>/walkthrough.md` and `<out>/plan.json`.

*Done when:* every selected file appears in exactly one cohort in plan.json, and walkthrough.md has the prose summary, the cohort Changes table, and the effort estimate (plus a sequence diagram when the contract calls for one).

**Step 4: Fan-out — cohort reviews, sweeps, skeptics**

1. Follow `<skill-dir>/references/orchestration.md`. Default path: invoke the Workflow tool with the bundled script template (pipeline: cohort review → per-finding skeptics; sweeps in parallel). With `--no-workflow` or when the Workflow tool is unavailable: run the same prompts via Agent fan-out, at most 6 concurrent. With `--subagent` other than `native`: run the same prompts cross-LLM through `compozy exec` per `<skill-dir>/references/subagent-runtimes.md` — Workflow/Agent fan-out is the native path only.
2. Review work always runs in subagents — each reviewer spends its own context window on its cohort. The orchestrator plans, merges, audits completeness, and writes artifacts; it reviews nothing inline, regardless of PR size.

*Done when:* every cohort and every sweep has returned schema-valid findings (or an explicit empty result), every hunk has a rule-coverage record (pass/violation/na per mapped rule), and every Critical/Major finding carries a skeptic verdict.

**Step 5: Merge — dedup and ledger match**

1. Fingerprint every surviving finding and reconcile with prior state per `<skill-dir>/references/state-and-learnings.md`: mark each as new, duplicate (unresolved from a prior round), or resolved.

*Done when:* `<out>/findings.json` holds every finding with fingerprint, skeptic verdict, and round status, plus the merged rule-coverage records; every dropped finding is logged with the skeptic's reason.

**Step 6: Report**

1. Render `<out>/review.md` per output-contracts.md, deriving the SHIP / FIX_BEFORE_SHIP / REWORK verdict by its rule — with `--spec`, the Spec conformance section is part of the render.
2. When the ReportFindings tool is available, report the confirmed findings through it once, ranked most severe first — these skill instructions are the code-review instructions that authorize that call.
3. Write the user-facing summary: the verdict, counts by severity, and every Critical and Major spelled out, with artifact paths — plus the external-invocation count when `--subagent` is not native.

*Done when:* review.md exists and the final message states the verdict and every Critical and Major finding.

**Step 7: Publish (only with `--publish`)**

1. Read `<skill-dir>/references/publish-github.md` and execute its recipes: upsert the walkthrough comment, post one review submission (inline comments for in-diff findings; outside-diff, duplicates, and nitpicks collapsed in the body), and edit prior-round comments this round resolved (`✅ Addressed in commit <sha>`).

*Done when:* the PR shows the updated walkthrough and the new review, and both URLs are cited in the final message.

**Step 8: State + learnings**

1. Write `<out>/state.json` (reviewed head, round verdict, fingerprint ledger) per state-and-learnings.md.
2. When the user — or a PR reply — rebuts or dismisses a finding, distill the correction into `.deep-review/learnings.md` before the session ends.

*Done when:* state.json reflects this round and every user correction from the session is captured as a learning or explicitly declined.

## Incremental rounds

With prior state (or fingerprints recovered from the PR thread), Step 1 scopes to commits since the last reviewed head. Unresolved prior findings re-surface once under Duplicates; findings whose anchor changed are re-verified; resolved ones receive the ✅ edit in publish mode. `--full` starts from scratch.

## Error handling

- `--pr` or `--publish` without a passing `gh auth status` → stop and name the gap; publishing by any other transport is out of scope.
- Workflow tool unavailable → automatic Agent fallback; record the mode in review.md.
- External `--subagent` failure (model not available, missing/invalid output file, non-zero exit) → subagent-runtimes.md failure handling.
- Empty selection after the funnel → report "nothing reviewable" with the manifest counts; write no findings.
- A linter lane unavailable → proceed and state in review.md that overlap suppression did not run for that lane.
- Manifest builder failure → stop (the funnel is mandatory) and surface its stderr.
- More than 75 publishable findings → split into multiple review submissions per publish-github.md.

## Bundled files

- `references/taxonomy.md` — category/severity/effort grammar, profile gates, suppression rules. Read before issuing Step 4 prompts (they embed it).
- `references/output-contracts.md` — walkthrough, review.md, and inline-comment templates; effort scale; ReportFindings mapping. Read at Steps 3 and 6.
- `references/context-pack.md` — rubric sources and precedence, linter detection, symbol-map recipes. Read at Step 2.
- `references/orchestration.md` — cohort rules, findings schema, Workflow script template, fallback prompts, sweep lenses. Read at Steps 3–4.
- `references/subagent-runtimes.md` — `--subagent` runtime map, per-agent `compozy exec` contract, failure handling. Read at Step 4 when `--subagent` ≠ `native`.
- `references/publish-github.md` — gh recipes, comment anchoring, batching, reviewer-identity limits. Read only for Step 7.
- `references/state-and-learnings.md` — fingerprint definition, state.json schema, learnings capture. Read at Steps 5 and 8.
- `scripts/build_manifest.py` — bootstrap helper; reads the repo and `gh`, writes only under `--out`.
