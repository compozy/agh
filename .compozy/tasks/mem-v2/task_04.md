---
status: pending
title: Scan Policy and Memory Prompt Assets
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 04: Scan Policy and Memory Prompt Assets

## Overview

Prepare the policy and prompt assets that the controller, extractor, and dreaming runtime will consume. This task isolates the pre-write content-scan rules and the versioned prompt templates so later behavior-heavy tasks can rely on stable assets instead of embedding prose directly in runtime logic.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Write controller`, `Extractor`, `Dreaming v2`, and `Safety Invariants`.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go or runtime prompt-loading code.
- MINIMIZE CODE churn outside scan/prompt asset seams.
- TESTS REQUIRED: template loading, version selection, deterministic content-scan rules, and redaction-safe failure behavior must ship here.
- NO WORKAROUNDS: do not hardcode policy prompts inline in controller, extractor, or dreaming logic.
</critical>

<requirements>
- MUST add versioned prompt/template assets for controller tiebreak, extractor, dreaming, and `WHAT_NOT_TO_SAVE` policy.
- MUST implement deterministic pre-write scanning that can reject or annotate unsafe content before persistence.
- MUST keep prompt/template loading explicit and testable rather than hidden in daemon wiring.
- MUST preserve Slice 1’s lexical-only write/recall assumptions and avoid embedding/vector prompt drift.
- MUST expose stable helpers so later controller/extractor/dream tasks can consume assets without duplicating policy text.
</requirements>

## Subtasks
- [ ] 4.1 Add versioned prompt assets for controller, extractor, dreaming, and write-policy scans.
- [ ] 4.2 Implement deterministic content-scan helpers for `WHAT_NOT_TO_SAVE` and related safety rules.
- [ ] 4.3 Add loader helpers that keep asset lookup/versioning explicit and testable.
- [ ] 4.4 Add focused tests for asset loading, invalid assets, and content-scan decisions.

## Implementation Details

See TechSpec `Write controller`, `Extractor`, `MemoryProvider ABC`, and `Development Sequencing` step 10. This task should produce reusable assets and helpers only; it should not yet wire full controller/extractor/dream behavior.

### Relevant Files
- `internal/memory/prompt.go` — current prompt-related helpers to extend or replace.
- `internal/memory/document.go` — current memory parsing helpers that may share policy-scanning inputs.
- `internal/memory/*` — location for new asset loader helpers or prompt registries.
- `.compozy/tasks/mem-v2/analysis/analysis_write-controller.md` — policy rationale and competitor evidence.
- `.compozy/tasks/mem-v2/analysis/analysis_extraction-location.md` — extractor prompt and staging evidence.

### Dependent Files
- `internal/memory/controller/*` — controller will consume the tiebreak and scan assets.
- `internal/memory/extractor/*` — extractor runtime will consume extraction prompts.
- `internal/memory/dream.go` — dreaming runtime will consume promotion prompts.
- `.compozy/tasks/mem-v2/task_05.md` — controller task depends on the assets from this task.
- `.compozy/tasks/mem-v2/task_10.md` — extractor runtime task depends on these assets.

### Related ADRs
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — defines prompt/tiebreak responsibilities.
- [ADR-010: Fact Extraction Location — Hybrid Per-Turn Hook + Optional Compaction Flush](adrs/adr-010.md) — defines extractor prompt use.
- [ADR-007: Daily-Log Retention Policy](adrs/adr-007.md) — influences dreaming and retention prompts.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: no new extension surface yet, but the assets defined here must remain consumable by the bundled provider and future external providers.
- Agent manageability: none — no public CLI/HTTP/UDS/native-tool change lands here.
- Config lifecycle: none — checked surfaces are memory controller/extractor/dream config keys and settings payloads; public config work is deferred.

### Web/Docs Impact

- `web/`: none — checked surfaces are generated types and memory/settings/session UI; no public contract changes happen in this task.
- `packages/site`: none — checked surfaces are runtime docs and references; docs update after behavior is wired.

## Deliverables

- Versioned prompt assets for controller, extractor, dreaming, and write policy.
- Deterministic content-scan helpers for unsafe persistence cases.
- Asset loader/test helpers with focused coverage.

## Tests

- Unit tests:
  - [ ] Prompt assets load by explicit version and fail clearly on invalid or missing templates.
  - [ ] Content-scan helpers flag or reject unsafe persistence inputs according to Slice 1 policy.
  - [ ] Asset helpers remain deterministic and do not depend on daemon wiring or global mutable state.
- Integration tests:
  - [ ] Controller/extractor/dream packages can import and consume the shared assets without duplication.
  - [ ] `go test` for memory packages passes with asset-loading coverage and no hidden filesystem assumptions.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/codex/codex-rs/memories/write/src/prompts.rs`
- `.resources/codex/codex-rs/memories/write/src/phase1.rs`
- `.resources/codex/codex-rs/memories/write/src/phase2.rs`
- `.resources/claude-code/memdir/memoryTypes.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Scan policy and memory prompt assets are centralized, versioned, and ready for controller/extractor/dream consumption.
- No behavior-heavy task needs to embed policy prose inline.

