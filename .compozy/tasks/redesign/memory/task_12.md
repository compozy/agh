# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Delivered `packages/ui/README.md` (235 lines, well under 500-line budget) as the canonical `@agh/ui` contributor guide: canonical references, ADR map, primitive inventory (Foundations / Structural / Form / Feedback / Chat) with per-primitive story links, UIProvider wiring, motion vs CSS decision rules, story contribution rules, Playwright snapshot workflow, anti-patterns, quick-reference matrix.
- Replaced the spec's "unit tests" requirement with a Vitest suite (`packages/ui/tests/readme.test.ts`) that acts as the automated link-check + inventory-parity + header snapshot per the task's docs-substitute language.

## Important Decisions

- **Linked `docs/design/design-system/README.md` + `.agents/skills/agh-design/SKILL.md`** instead of the techspec-referenced `docs/design/design-system/SKILL.md`. That SKILL.md does not exist in the repo; the agh-design skill file IS the Claude-invocable skill and lives under `.agents/skills/agh-design/SKILL.md`. Link-check test enforces this, so future drift will fail the test.
- **Local link-check over `markdown-link-check`.** Used a vitest-level filesystem check instead of pulling in `markdown-link-check` as a devDep. Reasons: (1) no network dependency for CI determinism, (2) avoids adding a new npm dep for a docs gate, (3) the spec said "e.g., `markdown-link-check`" — i.e., the tool is illustrative, not required.
- **Inventory-parity uses identifier-name grep.** Parses `export { … } from "…"` blocks in `src/index.ts`, extracts each identifier (handling `type` modifier + `as` aliases), and asserts every name appears as a whole word in README.md. The README lists each exported name verbatim in tables grouped by primitive family.
- **Heading-snapshot uses inline vitest snapshot.** Code-fence aware (skips content inside ```…``` blocks) so `# Local (…)` comments in bash examples don't pollute the heading list.
- **Kept the inventory in five tables that match the task's mandated grouping** (foundations / structural / form / feedback / chat). `CodeBlock` sits in Chat (not Feedback) because its primary consumers are the session message thread and tool-call output.

## Learnings

- `oxlint` flags `/^.../` regexes as `unicorn/prefer-string-starts-ends-with` warnings. Use `.startsWith("…")` in test utilities — the monorepo treats warnings as blocking per CLAUDE.md.
- `oxlint` also flags `(?<!\!)` with `\!` as `no-useless-escape`. Inside a lookbehind, `!` doesn't need escaping — write `(?<!!)`.
- `oxfmt` rewrites markdown tables to align pipe columns. Run `bunx oxfmt <file>` after drafting; the check step runs in CI.
- `packages/ui/vitest.config.ts` includes `src/**/*.{test,spec}.{ts,tsx}` and `tests/**/*.test.{ts,tsx}`. Non-component tests (like this README suite) belong under `packages/ui/tests/`.

## Files / Surfaces

- `packages/ui/README.md` — new contributor guide.
- `packages/ui/tests/readme.test.ts` — new Vitest suite (link-check + inventory-parity + header snapshot + line-budget).
- `.compozy/tasks/redesign/task_12.md` — status → completed, subtasks ticked.
- `.compozy/tasks/redesign/_tasks.md` — task_12 row → completed.

## Errors / Corrections

- First header-snapshot run leaked a `# Local (macOS / darwin baselines)` line from a bash code fence. Fixed by skipping code-fence content in both the heading collector and the link collector.

## Ready for Next Run

- No follow-up required. Task 13 (app-sidebar rewrite) consumes the same `UIProvider` + Sidebar contracts already documented here; no action needed in this file before task 13 starts.
