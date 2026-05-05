# TechSpec peer review — Round 3 (final verification before task generation)

You are reviewing as a senior peer (`gpt-5.5` `xhigh`) — **not implementing**.

## Context

Round 1: NOT-READY → 6 blockers + 14 nits + 8 open concerns.
Round 2: NOT-READY → 6 blockers RESOLVED, but 2 new blockers (NB1, NB2) + 5 NOT-FIXED nits + 3 new nits + 5 cross-cutting concerns.
Round 3 (this one): verify the Round-2 incorporation.

The TechSpec author has now:
- Broadened the `idempotency_key` composition to include `op|post_content_hash|prompt_version` (NB1) + 3 regression tests
- Added bounded-queue + metrics + 2 canonical events for `RecordRecall` (NB2)
- Fixed all 5 NOT-FIXED nits from Round 2 (N7, N8, N10, N11, N13)
- Addressed all 3 new nits (NN1 ADR superseded markers, NN2 docs subitem, NN3 QA proof rename)
- Addressed all 5 cross-cutting concerns (LLM prompt fields, build-order split, WAL retention, greenfield wording, ADR-004 fallback)

## Inputs

- `.compozy/tasks/mem-v2/_techspec.md` — TechSpec post-Round-2 incorporation (~1627 lines)
- `.compozy/tasks/mem-v2/adrs/adr-001..012.md` — ADRs with Round-1 + Round-2 refinement sections
- `.compozy/tasks/mem-v2/_codex_techspec_review2_response.md` — Round 2 raw findings (your previous output)

## Your job

**Verify Round 2 fixes.** Specifically:

1. Confirm **NB1, NB2 resolved**. State `RESOLVED | PARTIALLY-RESOLVED | NEW-ISSUE`.
2. Confirm Round-2 **NOT-FIXED nits (N7, N8, N10, N11, N13) are now FIXED**. State `FIXED | NOT-FIXED` with line evidence.
3. Confirm Round-2 **new nits (NN1, NN2, NN3) addressed**. State `FIXED | NOT-FIXED`.
4. Confirm **5 cross-cutting concerns addressed**. State which the TechSpec now resolves.
5. **Detect any new blockers introduced** (especially regressions in idempotency_key shape, NB2 metrics integration, ADR superseded markers leaving stale guidance for implementers).
6. **Final verdict**: READY / READY-WITH-NITS / NOT-READY.

## Output structure

Tight Markdown:

### 1. NB1 + NB2 verification
- NB1 status + line evidence
- NB2 status + line evidence

### 2. Previously-NOT-FIXED nits status (compact)
N7, N8, N10, N11, N13 — each `FIXED|NOT-FIXED` + 1-line evidence.

### 3. Previously-introduced nits status
NN1, NN2, NN3 — each `FIXED|NOT-FIXED` + evidence.

### 4. Cross-cutting concerns status
For each of CC#1 (LLM prompt fields), CC#2 (build order split), CC#3 (WAL retention), CC#4 (greenfield wording), CC#5 (ADR-004 fallback): `RESOLVED|PARTIALLY|NOT-RESOLVED` + evidence.

### 5. New issues introduced (if any)
Number them `NB3+`, `NN4+`. Same format as Round 2: where/why/fix.

### 6. Final verdict
**READY** / **READY-WITH-NITS** / **NOT-READY**.

If READY-WITH-NITS: which nits should land before task generation vs which can land during task execution.

If NOT-READY: list the blocking items.

### 7. cy-create-tasks recommendation
Two paragraphs max:
- Paragraph 1: should tasks generate now? Why?
- Paragraph 2: any specific guidance for the task author (task #N should reference ADR-X line Y; OCY should be a subitem on task #M; etc.).

## Tone & rules

- Be opinionated, no hedging.
- Anchor every claim in TechSpec line / ADR line / analysis file.
- Brazilian Portuguese acceptable; technical content stays in English.
- Tight density — no padding.
- Read-only, do not modify any code.

Begin.
