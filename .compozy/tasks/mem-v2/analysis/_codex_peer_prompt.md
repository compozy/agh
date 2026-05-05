# Peer-idea pass — AGH memory v2

You are reviewing as a senior peer (gpt-5.5 / xhigh reasoning) — not implementing.

## Situation

AGH is a Go single-binary daemon (Compozy's open agent runtime) that hosts AI agent
sessions over ACP. Project rules live in `CLAUDE.md` (greenfield alpha; zero legacy
tolerance; local-first; SQLite-first; numbered migrations; every feature must be
agent-manageable AND extensible; two-touch rule).

We are designing **memory v2**. Ten analyst subagents already produced a corpus of
research. **Read these files in this order:**

1. `.compozy/tasks/mem-v2/analysis/analysis_integration-map.md` — the proposed v2
   shape (12 design pillars, 60-row decision matrix, layer-by-layer sketch, delete
   list, open questions, risk register).
2. `.compozy/tasks/mem-v2/analysis/analysis_agh-current.md` — honest audit of what
   AGH has today (`internal/memory/`, schema, hooks, CLI/HTTP/UDS surfaces,
   extensibility, gaps).
3. `.compozy/tasks/mem-v2/analysis/analysis_ai-memory.md` — academic / framework
   survey (Letta, Mem0, Zep, A-MEM, Cognee, etc.).
4. `.compozy/tasks/mem-v2/analysis/analysis_ai-harness.md` — cross-harness
   memory patterns.
5. `.compozy/tasks/mem-v2/analysis/analysis_codex.md` — Codex CLI memory pipeline
   (two-phase consolidation, Compacted.replacement_history, AGENTS.md).
6. `.compozy/tasks/mem-v2/analysis/analysis_claude-code.md` — Claude Code memdir
   (closed taxonomy, WHAT_NOT_TO_SAVE, Sonnet ranker, forked extractor).
7. `.compozy/tasks/mem-v2/analysis/analysis_hermes.md` — Hermes memory
   architecture (4 layers, MemoryProvider ABC, ContextCompressor).
8. `.compozy/tasks/mem-v2/analysis/analysis_openclaw.md` — OpenClaw memory plugin
   (pre-compaction flush, dreaming, bundled stores).
9. `.compozy/tasks/mem-v2/analysis/analysis_openfang.md` — OpenFang memory
   (single SQLite + 6 stores, source vs wiki drift).
10. `.compozy/tasks/mem-v2/analysis/analysis_goclaw-paperclip-multica.md` —
    goclaw 3-tier consolidation, paperclip nativeContextManagement, multica
    minimal-context.

## Your job — peer review, not implementation

Read all 10 files. Then write a peer-idea response that does the following, in
this order, in a single Markdown response (no sections you cannot defend with
evidence; cite source files like `analysis_<name>.md §<section>`):

1. **Pressure-test the integration map** — for each of the 12 design pillars,
   say whether you agree, partially agree, or disagree. If you disagree, name
   the load-bearing failure and propose the corrected pillar.
2. **Decision matrix dispute list** — pick the 8 verdicts you would flip
   (ADOPT↔ADAPT↔REJECT↔DEFER) and justify each in 1-3 sentences anchored in
   AGH's constraints.
3. **Alternative architectures you would seriously consider** — name 2-3
   structural alternatives the integration map under-explored (e.g. event-
   sourced memory log, content-addressed memory, embedding-free retrieval,
   MemGPT-style OS pages, Aura-style single-table). For each: what it buys,
   what it costs, when AGH should reconsider it.
4. **Blind spots in the corpus** — what did the 10 analyses systematically
   miss? (Examples to consider: memory observability/debugging, memory
   versioning/rollback, multi-agent memory federation across the AGH Network,
   memory as a security surface, ACP-level memory primitives.)
5. **Top 5 structural risks** in the proposed v2 — beyond what the risk
   register already names. Each risk: what it is, when it bites, what
   mitigation you would adopt.
6. **What you would build first** — given the AGH greenfield constraint and
   the two-touch rule, what is the minimum vertical slice of v2 that proves
   the architecture? (You are designing the slice that goes first into a
   TechSpec.)
7. **Open questions you would force the spec author to answer before
   shipping** — 6-10 of them, ranked by criticality.

## Tone & rules

- Be opinionated. Hedging is not useful.
- Anchor every claim in the source analyses or in the AGH project rules.
- Do not write any code. Do not modify any files (this is read-only peer
  review).
- Brazilian Portuguese is acceptable for tone; technical content stays in
  English.
- Aim for a tight, dense response — not exhaustive prose. The integration
  map already exists; you are pressure-testing it.

Begin.
