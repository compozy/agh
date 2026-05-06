# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Delivered centralized Memory v2 Slice 1 prompt/policy assets and deterministic pre-write scan helpers for later controller, extractor, and dreaming tasks.
- Scope stayed intentionally limited to reusable assets/helpers and focused tests; no daemon/controller/extractor/dream runtime wiring was added in this task.

## Important Decisions

- Versioned prompt assets live in `internal/memory/prompts` and are loaded explicitly by `Name*` plus `VersionV1`.
- `prompts.ParseTemplate` uses `missingkey=error` so later runtime tasks fail fast when prompt data contracts drift.
- Deterministic scan helpers live in `internal/memory/scan`; the public API is `scan.Content` and `scan.Candidate` to avoid Go package stutter.
- Scan results expose redaction-safe rule IDs/reasons via `Result.Reason` and `Result.RuleHits`; matched raw content is never copied into explanations.
- Slice 1 remains lexical-only. The scanner uses deterministic regex/rune rules only; no embeddings, vector scores, or provider calls were introduced.

## Learnings

- `make verify` initially failed in `make lint` because revive rejected exported names `scan.ScanContent` and `scan.ScanCandidate`; renaming them to `Content` and `Candidate` fixed the API shape.
- `scripts/check-test-conventions.py` remains absent in this repository; focused Go tests, race tests, coverage, and `make verify` are the available guardrails.

## Files / Surfaces

- `internal/memory/prompts/`: embedded `decide.v1.tmpl`, `extract.v1.tmpl`, `dream.v1.tmpl`, `what_not_to_save.v1.md`, explicit registry, and asset-loader tests.
- `internal/memory/scan/`: deterministic pre-write scanner, redaction-safe result helpers, and scan decision tests.
- `.compozy/tasks/mem-v2/task_04.md`, `_tasks.md`, `memory/MEMORY.md`, and this file updated for completion tracking.

## Errors / Corrections

- Corrected the exported scanner API after lint evidence rather than suppressing revive or adding a workaround.

## Ready for Next Run

- Task 05 should consume `prompts.Load`, `prompts.LoadLatest`, or `prompts.ParseTemplate` by explicit prompt name/version instead of embedding policy prose inline.
- Task 05 should run `scan.Candidate` before persistence decisions and store only safe scan rule IDs/reasons in rule traces.
