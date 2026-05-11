---
name: cy-research-competitors
description: >-
  Dispatches scoped-write `cy-researcher` subagents in parallel to study reference
  repos under .resources (claude-code, hermes, openclaw, openfang, multica,
  paperclip, goclaw, codex-cli). Each subagent reads its assigned competitor and
  writes exactly one analysis file at
  .compozy/tasks/SLUG/analysis/analysis_NAME.md. The parent agent prepares the
  analysis directory, dispatches the subagents, and verifies schema compliance.
  Each analysis covers mechanisms, relevant paths, transferable patterns, risks,
  open questions, and evidence. Use when a TechSpec or refactor needs cross-system
  reference grounding before architectural decisions. Do not use for scope-internal
  questions or final implementation planning.
trigger: explicit
argument-hint: "[task-slug] [competitor-list]"
---

# Research Competitors

Pedro routinely studies 3-5 reference repos under `.resources/` before drafting any TechSpec. This skill formalizes that pattern as parallel scoped-write subagent dispatch with a fixed analysis schema. The skill uses the dedicated `cy-researcher` custom agent (`.claude/agents/cy-researcher.md`), which is authorized to perform exactly one `Write` per dispatch — at the analysis path the parent names. Every other action remains read-only. The parent prepares the analysis directory, dispatches the subagents in parallel, and verifies schema compliance after every subagent returns.

## Required Inputs

- **task-slug**: the `.compozy/tasks/<slug>/` to receive analysis output. Must already exist.
- **competitor-list** (optional): comma-separated names. When omitted, infer from the task's `_idea.md` references and from `references/competitor-catalog.md`.

## Procedures

**Step 1: Resolve Inputs**

1. Validate `task-slug` and confirm `.compozy/tasks/<slug>/` exists.
2. Create `.compozy/tasks/<slug>/analysis/` if absent.
3. If `competitor-list` is omitted, read `_idea.md` and any existing techspec/PRD for `.resources/<name>/` references. Augment with `references/competitor-catalog.md` for adjacent systems.
4. For each named competitor, verify `.resources/<name>/` exists. Skip missing competitors with a warning rather than failing.

**Step 2: Compose the Analysis Prompt**

1. Read `assets/analysis-template.md`. This is the canonical schema each subagent fills.
2. For each selected competitor, compose a parallel `cy-researcher` subagent prompt instructing it to:
   - Read `.resources/<name>/` (focus on the directories named in the competitor catalog).
   - Cross-reference the AGH TechSpec or `_idea.md` topic.
   - Draft the seven-section markdown matching the schema in the template.
   - Write the result with exactly one `Write` call to the named target path: `.compozy/tasks/<slug>/analysis/analysis_<name>.md`.
   - Return a confirmation message with the written path, the section count, and any Open Questions to surface.
3. Embed `references/dispatch-rules.md` (the scoped-write contract) verbatim in every subagent prompt. State both names explicitly: the competitor directory under `.resources/<name>/` and the exact target file path. The subagent will refuse to write if either is missing.
4. Omit explicit model selection unless the user explicitly requests the multi-LLM pipeline for this run. When explicit model selection is requested and supported, use `gpt-5.4-mini` with `reasoning_effort=high` for breadth or `gpt-5.4` with `reasoning_effort=xhigh` for architecturally complex competitors.

**Step 3: Dispatch Scoped-Write Subagents in Parallel**

1. Launch one `cy-researcher` subagent per competitor, all in the same dispatch round. Set `subagent_type: cy-researcher` on every Agent call. Read `references/dispatch-rules.md` for the scoped-write contract.
2. Each subagent is authorized to perform exactly one `Write` — at the analysis path the parent named. Every other action is read-only. Subagents MUST NOT call `Edit`, MUST NOT call `Write` against any other path, MUST NOT run state-mutating shell commands.
3. Wait for all subagents to complete before continuing. Do not move to verification until every selected competitor has returned a write confirmation.

**Step 4: Verify Files and Schema Compliance**

1. After every subagent returns, list `.compozy/tasks/<slug>/analysis/` and confirm one file exists per dispatched competitor at the expected path.
2. For each `analysis/analysis_<name>.md`, confirm all seven sections are present (Overview, Mechanisms/Patterns, Relevant Code Paths, Transferable Patterns, Risks/Mismatches, Open Questions, Evidence).
3. Confirm the Evidence section cites real file paths under `.resources/<name>/` (no fabricated paths). Sample-check at least one cited path per file with `Read` to confirm existence.
4. Reject empty sections. Re-dispatch the offending subagent with a follow-up requesting completion. Do not let the parent author the missing content — the subagent owns the write.
5. If a subagent failed to write (returned a clarification request instead of writing), correct the dispatch (clarify both names, fix the analysis directory) and re-dispatch. Do not silently substitute a parent-authored file.

**Step 5: Append References to TechSpec / Tasks**

1. After analyses converge, update the TechSpec (`_techspec.md`) or task files to include explicit `.resources/<name>/<file>` paths in their bodies. The implementation agent reads competitor code through these citations.
2. Do not summarize the analyses inside the TechSpec — keep the analysis files as the authoritative artifact and link them.

## Error Handling

- **`.resources/<name>/` missing:** skip with a logged warning. Do not invent the competitor's structure. If the user wants a placeholder, the parent (not the subagent) authors a one-paragraph stub at `analysis_<name>.md` documenting the absence.
- **Subagent writes outside the named analysis path, calls `Edit`, or runs state-mutating shell commands:** treat as a contract violation. Stop, re-read `references/dispatch-rules.md`, and re-dispatch with the contract restated verbatim in the subagent prompt.
- **Subagent returns a clarification request instead of writing:** the dispatch was ambiguous. Fix the prompt (both names, directory must exist) and re-dispatch.
- **Schema-incomplete analysis:** re-dispatch the offending subagent with the schema embedded and a request to fill the gap. Do not let the parent author the missing content; the subagent owns the write.
- **Network/disk error during dispatch:** fail the round entirely. Do not produce a half-set of analyses.
