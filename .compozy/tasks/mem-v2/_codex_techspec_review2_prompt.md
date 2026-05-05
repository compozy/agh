# TechSpec peer review — Round 2

You are reviewing as a senior peer (`gpt-5.5` with `xhigh` reasoning) — **not implementing**.

## Context

This is **Round 2** of a peer-review cycle. Round 1 returned **NOT-READY** with 6 blockers (B1-B6) + 14 nits (N1-N14) + 8 open concerns (OC1-OC8). The TechSpec author incorporated **every blocker and every nit**, plus an explicit Open Concerns Tracking section, plus updated 6 ADRs with post-approval refinement sections, plus normalized HTTP routes and CLI flags.

Round 1 verdict + raw artifacts available at:
- `.compozy/tasks/mem-v2/_codex_techspec_review_response.md` — Round 1 raw output
- `.compozy/tasks/mem-v2/_techspec.md` — TechSpec **post-incorporation** (1588 lines, was 1356)
- `.compozy/tasks/mem-v2/adrs/adr-001..012.md` — ADRs, 6 of which have new "Post-Approval Refinement" sections

## Your job

**Verify Round 1 fixes.** Specifically:

1. Confirm each of **B1-B6 is resolved**. For each, state `RESOLVED` / `PARTIALLY-RESOLVED` / `NEW-ISSUE-INTRODUCED`. If anything other than RESOLVED, name the specific section + line in the TechSpec or ADR where the issue persists.

2. Confirm each of **N1-N14 is fixed**. State `FIXED` / `NOT-FIXED` per nit. For NOT-FIXED, state where.

3. **Detect new blockers introduced by the fixes.** When integration spans multiple sections, fixes can create cross-section contradictions or new gaps. Specifically check:
   - The lexical-only controller algorithm (B1 fix) vs the entity-slot ambiguity LLM tiebreaker — does the LLM still receive enough signal to make a useful decision without embeddings?
   - The `memory_decisions` schema with `post_content` / `idempotency_key` (B3 fix) — replay determinism, idempotency-key composition collision risks, payload size bloat under bursty writes.
   - The `internal/memory/contract` package (B4 fix) — verify the package depends only on stdlib + `internal/logger` and that all controller/recall/extractor types now live in `contract`. Check for hidden cycles in the import graph.
   - The dotted canonical event enum (B5 fix) — does the `memory_events.op` CHECK constraint cover every event the §Monitoring table promises? Any drift?
   - The "no native_context_management coordination" choice (B6 simplification) — is the resulting double-injection risk explicitly accepted, and is the failure mode (operator confusion when AGH and provider both inject memory) addressed in operator docs / docs?
   - The signals-live-in-Slice-1 path (B2 fix) — `RecordRecall` fire-and-forget could cause invisible drops under load. Is there a safety margin or observability for that?

4. **Detect remaining structural concerns** in the dependency graph (Build Order 1-34), in the safety invariants, or in the boundary discipline.

5. **Final verdict**: one of:
   - **READY** — tasks may generate now.
   - **READY-WITH-NITS** — minor fixes needed but tasks can generate (list the nits).
   - **NOT-READY** — at least one blocker remains or a new blocker was introduced; list with the same B-N taxonomy as Round 1.

## Output structure

Write a tight Markdown response. Headings:

### 1. Round-1 Blocker Verification
- B1 status + evidence
- B2 status + evidence
- B3 status + evidence
- B4 status + evidence
- B5 status + evidence
- B6 status + evidence

### 2. Round-1 Nit Verification (compact)
Just `Nx — FIXED|NOT-FIXED` lines; explain only NOT-FIXED.

### 3. Open Concerns Tracking Verification
For OC1-OC8, confirm the disposition (task subitem / QA telemetry / follow-up TechSpec) is present in the TechSpec's Open Concerns Tracking section.

### 4. New Issues Introduced
Number any new issues `NB1`, `NB2`, ... (for new-blockers) or `NN1`, `NN2`, ... (for new-nits). Each: where, why, fix.

### 5. Cross-cutting concerns
Up to 5 items. Each: title, where, why it matters, suggested resolution. These are not blockers but worth tracking.

### 6. Final verdict
One of READY / READY-WITH-NITS / NOT-READY. If READY-WITH-NITS, list which nits should be fixed before tasks generate vs which can land during task execution. If NOT-READY, list the blocking items.

### 7. Recommendation for `cy-create-tasks`
Two paragraphs maximum:
- Paragraph 1: should tasks generate now? Why?
- Paragraph 2: any specific guidance for the task author (e.g., "task #N should explicitly reference ADR-X line Y", "OCY should be a subitem on task #M").

## Tone & rules

- Be opinionated. Do not hedge.
- Anchor every claim in TechSpec line / section, ADR line / section, or analysis file.
- Brazilian Portuguese acceptable for tone; technical content stays in English.
- Tight density — no padding.
- Do not write or modify any code.

Begin.
