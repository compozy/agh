# Task Memory: task_24.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Co-ship the discoverability surfaces — generated CLI reference, generated API reference, site docs-truth/discovery/manual-cli-examples guards — with the final Slice 1 Memory v2 runtime. Regenerate from source-of-truth, do not hand-patch generated pages, and keep stale verbs/routes from being documented as current behavior.

## Important Decisions

- Did not hand-patch any `packages/site/content/runtime/cli-reference/**` page. The only CLI source change was an indentation fix in `internal/cli/memory.go` `agh memory write` example (replaced leading tabs with two spaces) so the regenerated `memory/write.mdx` shows clean, aligned shell examples.
- Treated `make verify` (specifically the `bun-lint` stage running `bunx oxfmt`) as the canonical post-generator formatter for `.mdx` tables. `make cli-docs` emits unpadded markdown tables; `oxfmt` re-pads them to the committed form. This is why a fresh `make cli-docs` produces a wide diff that disappears after `bun-lint`.
- Added the Slice 1 docs-truth guards in `packages/site/lib/runtime-docs-truth.test.ts` against actual repo files only (no fake regex anchors), and the discoverability assertions in `packages/site/lib/runtime-docs-discovery.test.ts` against the generated `meta.json` files. Both rely on the existing `runtimePageExists`/`readRuntimeJSON` helpers; they do not introduce new repo helpers.
- Added the manual-CLI-example guard in `packages/site/lib/runtime-manual-cli-examples.test.ts` that scans every documented bash block (after collapsing `\\\n` continuations) for the replaced verbs `agh memory read` and `agh memory consolidate`.
- Preserved the task_23 caveat: `runtime-docs-truth.test.ts` deliberately contains forbidden-pattern literals (e.g. `[memory.v2]`, `memory_read`, `memory_history`, `PUT /api/memory`) inside negative assertions; these are test infrastructure, not docs leakage.

## Learnings

- Running `make cli-docs` regenerates every CLI page in the canonical Cobra → Fumadocs form. It will produce a wide diff against the committed pages because the committed form is post-`oxfmt`. Always run `make verify` (or at least `make bun-lint`) after `make cli-docs` so the working tree settles to the canonical, formatted form before assessing drift.
- `packages/site` test/typecheck/build all run `bun run generate:openapi && bun run source:generate && bun run content:generate` in their pre-scripts. Generated `runtime/api-reference/*.mdx`, `.source/`, and `.velite/` outputs flicker during these runs but stay reproducible — never commit `.source/`, `.velite/`, or `.next/`.
- The `runtime-docs-truth` "memory CLI reference" test reads concrete generated MDX files (`memory/index.mdx`, `memory/show.mdx`, `memory/dream/index.mdx`, `memory/dream/trigger.mdx`) and asserts directory listings via `readdirSync` rather than regex on the index — this catches both content drift (renamed verbs) and structural drift (a stale `read.mdx` re-appearing on disk).
- Generated CLI subcommand `meta.json` files are alphabetized by the docpost generator with `index` (when present) listed first. Discoverability tests must use `toContain`/`not.toContain` rather than `toEqual` because new sibling commands can land later without invalidating the Slice 1 guarantees.

## Files / Surfaces

- `internal/cli/memory.go` — adjusted `agh memory write` example indentation so the regenerated `memory/write.mdx` example block aligns cleanly (no remediation of CLI verbs themselves; those were already correct after task 17).
- `packages/site/content/runtime/cli-reference/memory/write.mdx` — regenerated output of the indentation fix above; no other generated CLI page changed once `oxfmt` ran.
- `packages/site/lib/runtime-docs-truth.test.ts` — added five new specs (Slice 1 narrative surfaces, Memory v2 config keys vs `internal/config/config.go`, file-locations forensic ledger paths, generated CLI memory reference, generated API memory reference + orientation page, builtin tool registry IDs).
- `packages/site/lib/runtime-docs-discovery.test.ts` — added three new specs (core memory narrative meta, generated cli-reference memory + dream meta, generated api-reference memory tag).
- `packages/site/lib/runtime-manual-cli-examples.test.ts` — added the `agh memory read|consolidate` forbidden-verb scan.

## Errors / Corrections

- Initial regenerated `make cli-docs` output unpadded markdown tables across every CLI page; the orchestrator constraint says no destructive git commands and no hand-patching of generated content. Resolution: ran `make verify`, which executed `bun run format` (`bunx oxfmt`) as part of `bun-lint`, and the formatter re-padded every table back to the committed canonical form. No manual edits to generated MDX were needed.
- No test/build failure occurred. No workaround, suppression, or test-weakening was used.

## Validation Evidence (2026-05-05 closeout)

- `make cli-docs` — PASS (`CLI docs generated in /Users/pedronauck/Dev/compozy/agh3/packages/site/content/runtime/cli-reference`).
- `cd packages/site && bun run test -- runtime-manual-api-routes runtime-manual-cli-examples runtime-docs-truth runtime-docs-discovery` — PASS (4 files, 27 tests).
- `cd packages/site && bun run typecheck` — PASS (tsgo --noEmit clean after generate:openapi/source:generate/content:generate prebuild steps).
- `cd packages/site && bun run build` — PASS (Next.js 16 production build, full static + SSG path tree generated, no MDX errors).
- `make codegen-check` — PASS (exit 0; no openapi/typescript drift).
- `git diff --check` — clean.
- `make verify` — PASS (`/tmp/mem-v2-verify.log`, exit 0; full monorepo gate; final lines `DONE 8359 tests in 16.332s` and `OK: all package boundaries respected`).

## Ready for Next Run

- Task 24 is complete. Next iteration should pick up `task_25` (QA Plan and Test Coverage) with `task_26` still pending after `task_25`. Do not commit `.source/`, `.velite/`, or `.next/`. Treat any future CLI-reference drift as an `oxfmt`-needed run of `make verify` first; only investigate further if a diff persists post-format.
