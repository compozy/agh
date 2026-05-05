# TechSpec peer review — AGH memory v2 Slice 1

You are reviewing as a senior peer (`gpt-5.5` with `xhigh` reasoning) — **not implementing**.

## Situation

AGH is a Go single-binary daemon (Compozy's open agent runtime). Project rules
live in `CLAUDE.md` (greenfield alpha; zero legacy tolerance; local-first;
SQLite-first; numbered migrations; every feature must be agent-manageable AND
extensible; two-touch rule).

We are about to generate tasks from a TechSpec for Memory v2 Slice 1 (a fat
slice covering 4 Eixos: spine + dreaming v2 + provider ABC + Hermes engineering
parity, plus extractor Mode A). Before tasks generate, we need a cross-LLM
pressure-test on the TechSpec.

## Your inputs

Read these in order:

1. **The TechSpec under review**: `.compozy/tasks/mem-v2/_techspec.md` (1356
   lines). This is the artifact you are reviewing.
2. **The 12 ADRs that the TechSpec rests on**:
   `.compozy/tasks/mem-v2/adrs/adr-001.md` through `adr-012.md`.
3. **The synthesis analysis the TechSpec extends**:
   `.compozy/tasks/mem-v2/analysis/analysis.md` (overarching) +
   `analysis_agh-current.md` (audit of what AGH has today) +
   `analysis_integration-map.md` (initial decision matrix). Cross-reference
   when something in the TechSpec contradicts the analysis.
4. **The 8 competitor analyses** under `.compozy/tasks/mem-v2/analysis/` —
   pull facts to substantiate or refute TechSpec design decisions when
   relevant. You don't need to read all 8 fully; jump to specific sections
   when the TechSpec cites them.

## Your job — peer review, not implementation

Read the TechSpec carefully. Then write a peer-pressure-test response with
the following structured output, in this exact order. Use Markdown headings.
**Do not modify any files** — this is read-only review.

### 1. Readiness verdict

One of:

- **READY** — TechSpec is approval-grade as-is. Tasks may generate.
- **READY-WITH-NITS** — TechSpec is approval-grade pending minor edits
  (typos, wording, missing cross-references). List the nits.
- **NOT-READY** — TechSpec has structural blockers. List the blockers; tasks
  must NOT generate until resolved.

State the verdict in one sentence at the top.

### 2. Blockers (only if NOT-READY)

For each blocker:

- **Title** (≤80 chars)
- **Where** (TechSpec section + line range; or ADR + section if the blocker
  lives in an ADR rather than the TechSpec itself)
- **Why it blocks** (load-bearing reason — what breaks at implementation,
  test, integration, or deployment if this ships as drafted)
- **Suggested resolution** (concrete, ≤3 sentences)

Number each blocker `B1`, `B2`, etc.

### 3. Nits (regardless of verdict)

For each nit:

- **Title** (≤80 chars)
- **Where**
- **Why it matters** (≤2 sentences)
- **Suggested fix** (concrete)

Number each nit `N1`, `N2`, etc. Cap at 30 nits — pick the highest-signal
ones.

### 4. Strengths

3-7 items the TechSpec gets distinctly right (so the user knows what to
preserve in any rewrite). Each item ≤2 sentences.

### 5. Open architectural concerns (NOT blockers, but worth flagging)

Up to 8 items where you have concerns the TechSpec cannot fully resolve at
the spec level — they'll need decisions during implementation, observation
during slice 1 use, or eval data to close. Each item:

- **Title**
- **Concern** (≤3 sentences)
- **What would close it** (eval data / production observation / specific
  follow-up TechSpec)

### 6. Pressure test the 13 numbered Safety Invariants

For each of the 13 invariants in §"Safety Invariants" of the TechSpec, state
one of: `AGREE` / `AGREE-WITH-CAVEAT` / `DISAGREE` / `INSUFFICIENT-EVIDENCE`.
For caveats and disagreements, write 1-2 sentences naming the failure mode.

### 7. Pressure test the dependency graph in §"Development Sequencing"

The Build Order has 30 numbered steps. Identify any:

- **Cycles** — A depends on B but B (transitively) depends on A
- **Hidden dependencies** — step N implicitly requires something from a later
  step
- **Unrealistic parallelism** — steps marked independent that actually share
  state or surfaces
- **Missing steps** — work the TechSpec implies but didn't sequence

If the graph is sound, say so explicitly.

### 8. Pressure test scope decisions

For each of these top-level decisions (captured as ADRs), say `AGREE` /
`AGREE-WITH-CAVEAT` / `DISAGREE`:

- ADR-001 (hybrid escopado: Markdown authoritative + event log + derived catalog)
- ADR-002 (3 scopes with agent two-tier; shadow-by-id precedence)
- ADR-003 (per-workspace catalog DB)
- ADR-004 (stable workspace_id UUID in workspace.toml)
- ADR-005 (`_system/` invariant namespace)
- ADR-006 (session ledger hybrid: events.db live + ledger.jsonl forensic)
- ADR-007 (daily retention: 1 MiB cap, 7-day dreaming window, 30-day cold-archive, never hard-delete)
- ADR-008 (MemoryProvider ABC: 10 hooks, single active, bundled local reference)
- ADR-009 (write controller: hybrid rule-first + LLM tiebreaker, ambiguity band 0.72-0.88)
- ADR-010 (extraction location: Mode A on_post_response forked extractor in Slice 1; Mode B compaction-flush deferred)
- ADR-011 (recall pipeline: deterministic-only in Slice 1; vector + LLM ranker deferred)
- ADR-012 (Slice 1 fat: 4 Eixos in single TechSpec)

For caveats and disagreements: 1-3 sentences explaining the load-bearing
concern AND naming the alternative you would adopt instead.

### 9. Final recommendation

One paragraph (≤200 words). Either:

- "Generate tasks now from this TechSpec." (only if verdict is READY or
  READY-WITH-NITS-that-can-fix-during-task-execution)
- "Resolve blockers B1, B2, ... before generating tasks." (if NOT-READY)
- "Resolve blocker(s) and re-review." (if heavy)

State which open concerns from §5 should be tracked in the task list as
explicit follow-up subitems vs deferred to slice 1 retrospective vs deferred
to a post-slice-1 TechSpec.

## Tone & rules

- Be opinionated. Hedging is not useful.
- Anchor every claim in the TechSpec, an ADR, an analysis file, an AGH
  project rule (`internal/CLAUDE.md`, `docs/_memory/standing_directives.md`),
  or a Go invariant.
- Brazilian Portuguese is acceptable for tone; technical content stays in
  English.
- Aim for tight density — not exhaustive prose. The user will scan your
  output for blockers and nits before deciding which to apply.
- Do not write or modify any code.

Begin.
