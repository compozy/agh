# Context Pack

How to assemble `<out>/context-pack.md` — the shared context every reviewer, sweep, and skeptic receives. Target roughly 1:1 context-to-code: enough to judge, not enough to dilute attention.

## 1. Rubric — the review law

Collect, in precedence order (higher wins on conflict):

1. **Path instructions** — top-level `path_instructions` in `.deep-review.yaml`, else `reviews.path_instructions` in `.coderabbit.yaml` (first file that defines them wins); each entry is a glob + verbatim instructions.
2. **Learnings** — `.deep-review/learnings.md` entries whose scope glob matches selected files.
3. **Guideline files** — discovered at any depth, **directory-scoped** (a guideline governs its directory subtree; nearest file wins): `CLAUDE.md`, `AGENTS.md`, `AGENT.md`, `REVIEW.md`, `DESIGN.md`, `CONTRIBUTING.md`, `.cursorrules`, `.cursor/rules/*`, `.github/copilot-instructions.md`.

For each source, extract only rules that can bind a review verdict (error handling, testing shape, layering, security, naming, tokens/design constraints) — skip build lore and tooling walkthroughs. Register each rule as a numbered entry:

```
R07 · scope: **/*_test.go · source: .deep-review.yaml path_instructions
     "MUST use t.Run(\"Should...\") pattern for ALL test cases"
```

`scope` = the path_instructions glob, the guideline file's directory subtree, or the learning's scope glob. Keep rule text verbatim — reviewers cite it and findings carry the rule id.

## 1b. Rule map — rules bound to files

For every selected file, list the rule ids whose scope matches its path (glob/subtree match — deterministic, no judgment). This map is what turns the rubric from prose into per-hunk checks: reviewers validate each hunk against exactly its file's mapped rules and record a verdict per rule (see orchestration.md). A selected file with no matching rules maps to `—` explicitly.

## 2. Linter lanes — run first, suppress overlaps

Detect what the repo already enforces and run it scoped to selected files; findings a lane reports are suppressed from the review (taxonomy rule 1).

| Signal in repo | Lane command (scope to changed files where supported) |
| --- | --- |
| `Makefile` with `lint`/`check` target | `make lint` (authoritative when present — prefer it over raw tools) |
| `golangci-lint` config / Go modules | `golangci-lint run <changed dirs>` |
| `package.json` scripts `lint`/`typecheck` | the repo's own script via its package manager |
| eslint/biome/oxlint config | corresponding tool on changed files |
| `tsconfig.json` | `tsc --noEmit` (project-wide; cheap signal) |
| `ruff.toml` / pyproject | `ruff check <files>` |
| `Cargo.toml` | `cargo clippy` |

Record per lane: `ran` (attach findings on selected files, trimmed) or `unavailable` (tool missing/failed — overlap suppression is off for that lane and review.md must say so). Never install tools to fill a lane.

## 3. Symbol map — code-graph lite

For each cohort, give reviewers the cross-file context the diff hides:

1. From the diff, list the changed/added/removed **exported or shared symbols** per file (functions, types, endpoints, config keys, DB columns).
2. For each, one `rg -n` pass over the repo for definition and usage sites *outside the cohort*; record `symbol → sites (path:line)` with a one-line role note. When the LSP tool is available, prefer its references/definition lookups.
3. Flag symbols whose **contract changed** (signature, return, schema, wire format) — these seed the consistency and contracts sweeps.

Cap the map at what fits the 1:1 budget; deep dives happen inside reviewer agents, which read files themselves.

## 3b. Fan-out for large selections

Above ~40 selected files or ~3 guideline sources, assembling this pack inline would burn the orchestrator's context on legwork that native subagents (Agent tool) can carry:

- **Rubric extractor** — one subagent: reads every guideline source, returns the numbered rule registry (id · scope · source · verbatim text). The orchestrator computes the file→rule map itself (deterministic glob match, no judgment).
- **Symbol mappers** — one subagent per 2–3 cohorts: given those cohorts' files and the diff, returns `symbol → definition/usage sites → contract-changed flag` per §3.

Subagents search and report; the orchestrator merges returns and writes context-pack.md. Below the threshold, assemble inline.

## 4. PR intent

With `--pr`: title, description, linked issues (`gh pr view N --json title,body,closingIssuesReferences`), and base/head. Locally: `git log --oneline <base>..<head>` plus the user's stated intent. Reviewers judge the diff against *stated intent* — a change that does more than its description says is itself a finding.

## 4b. Spec contract (`--spec`)

Resolve the conformance baseline: a file path is itself the artifact; a directory contributes its contract-bearing documents — `_prd.md`, `_techspec.md`, `_tests.md`, `_examples.md`, `_qa.md`, `_user_stories.md`, parity maps, requirement/UX docs, plus any document the spec's own files name as canonical. List every resolved artifact as `path → one-line role`. These are the baseline the `spec-parity` sweep judges against — do NOT extract rubric rules from them: §1 sources are review law, the spec is the contract under test.

## 5. context-pack.md layout

```markdown
# Context Pack — <target>

## Intent
<title/description/commits digest>

## Rubric
### Rule registry
<R<NN> · scope: <glob> · source: <path> · "<verbatim rule>">

## Rule map
<selected file → R-ids (or —)>

## Linters
<lane → ran(findings digest) | unavailable(reason)>

## Symbol map
<per cohort: symbol → definition/usage sites → contract-changed flag>

## Spec contract
<only with --spec: artifact path → role, one per line>
```
