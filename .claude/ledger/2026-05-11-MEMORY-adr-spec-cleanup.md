## Goal

Remove all citations to temporary spec/ADR docs (`.compozy/tasks/redesign[-v2]/`, `ADR-001..016`, "redesign-v2 PR-N", "task_NN", `_techspec.md` references in non-pipeline contexts) from permanent project files. Rules/content stays; pointers to temp docs go.

## Constraints/Assumptions

- Scope: ~130 files with `ADR-[0-9]`, ~12 with `redesign-v2`, ~7 with `.compozy/tasks/redesign`.
- Out of scope: `.compozy/`, `.skeeper/`, `node_modules`, `ai-docs`, `.tmp`, `.claude/plans`, `.claude/ledger`, `web/src/generated`, `docs/_memory/_synthesis.md`, `docs/_memory/analysis/`.
- User directives:
  - Tests that ONLY assert ADR citations → delete those `it(...)` blocks; for mixed, drop ADR portion.
  - L-022 (and other lessons) → keep rule + rationale, drop project/ADR names.
  - JSDoc comments whose only content is ADR cite → delete the comment.
- `make verify` must pass at the end.
- Plan file: `~/.claude/plans/n-s-finalizamos-a-spec-parsed-lightning.md`.

## Key decisions

- Regex pass for parenthetical `(ADR-NNN §K)` / `(ADR-NNN §K + ADR-MMM §L)` removals.
- Snapshots will be regenerated via vitest `-u`.
- `docs/_memory/_synthesis.md` and `analysis/` stay intact — they're forensic evidence catalogs.

## State

- Plan approved at 2026-05-11.

## Done

- Plan written + approved.

## Now

- Starting Category 1 (root CLAUDE.md / AGENTS.md / web/{CLAUDE,AGENTS}.md / packages/site/CLAUDE.md).

## Next

- Cat 2 skills + cy-researcher agent
- Cat 3 DESIGN.md (the big one)
- Cat 4 lessons
- Cat 5 lint-plugins
- Cat 6 tokens.css
- Cat 7 source comments
- Cat 8 stories
- Cat 9 tests + snapshot regen
- Cat 10 packages/ui/README.md
- Cat 11 site
- Cat 12 e2e
- Verify

## Open questions

None.

## Working set

- /Users/pedronauck/Dev/compozy/agh2 (repo root)
- Plan: ~/.claude/plans/n-s-finalizamos-a-spec-parsed-lightning.md
