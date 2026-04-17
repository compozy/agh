# Task Memory: task_16.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Generate CLI reference docs with `make cli-docs`, add the non-generated editorial CLI overview and sidebar metadata, verify site build/browser routes, update task tracking, and commit locally after clean evidence.
- Success criteria include valid frontmatter on generated pages, complete command tree in overview, documented global flags/env vars/common workflows, no generated-file manual edits, `make cli-docs` idempotence for `index.mdx`, docs build, and browser QA.

## Important Decisions

- Keep hand-authored content in `packages/site/content/runtime/cli-reference/index.mdx` and `meta.json`; generated command pages must remain owned by `make cli-docs`.
- Existing unrelated worktree changes are out of scope and must not be reverted or staged as task 16 work.
- Reconciled stale task path with current implementation: the active generator, root runtime nav, existing generated pages, and cross-links use `packages/site/content/runtime/cli-reference/` and `/runtime/cli-reference`. Task 16 edits should target that current path rather than creating an unreachable duplicate under `runtime/reference/cli`.
- Generator correction implemented: root command reference now generates to `agh.mdx`, while hand-authored root `index.mdx` and `meta.json` survive `make cli-docs`.

## Learnings

- Shared workflow memory says task_04 is complete: Cobra `GenMarkdownTree` plus Go post-processor exists, `make cli-docs` target exists, and 108 MDX files were previously generated.
- Baseline for this run: `packages/site/content/runtime/reference` was absent before generation in the current worktree.
- Actual baseline for the current site route: `packages/site/content/runtime/cli-reference/` already exists with generated command pages and hand-maintained root `meta.json`; the root `index.mdx` is currently generated, not editorial.
- Actual public global CLI flag is only `--output`/`-o` (`human`, `json`, `toon`). `--config`, `--socket`, and `--log-level` are not current CLI flags; socket/log/home behavior is configured through TOML and environment.
- `make cli-docs` completed after implementation and emits 117 CLI MDX files plus 21 `meta.json` files under `packages/site/content/runtime/cli-reference/`.
- Focused verification passed: `go test ./internal/cli/...`; frontmatter scan found no CLI MDX page missing opening frontmatter, `title`, or `description`.
- Site build with stale task selector failed as expected (`No package found with name 'packages/site'`); correct package selector `env -u FORCE_COLOR bunx turbo run build --filter=@agh/site` passed with 164 static pages.
- Browser QA with `agent-browser` covered `/runtime/cli-reference/`, `/runtime/cli-reference/agh/`, `/runtime/cli-reference/session/new/`, `/runtime/cli-reference/workspace/add/`, and `/runtime/cli-reference/completion/bash/`; all rendered nonblank pages and the dev server returned 200s.
- `make cli-docs` idempotence check preserved `packages/site/content/runtime/cli-reference/index.mdx` hash `9b543eab6382e8967af37ce55b1b3b8fc78072b046072b7f63b041ee562ff1ef`.

## Files / Surfaces

- Expected task surfaces: `packages/site/content/runtime/cli-reference/`, `internal/cli/docpost/`, selected `internal/cli/*.go` Cobra examples if needed, `.compozy/tasks/site/task_16.md`, `.compozy/tasks/site/_tasks.md`, `.compozy/tasks/site/memory/task_16.md`.

## Errors / Corrections

- QMD collection searches were run against `agh-site-archived`, `agh-site-ledger`, and `agh-site-plans`; one archived search hit `SQLITE_BUSY_RECOVERY`, later archived search succeeded but returned stale/non-specific historical context. Current code remains authoritative for command behavior.
- Final root `make verify` failed on the pre-existing web token mismatch in `web/src/styles.test.ts`: tests expect `#121212/#1C1C1E/#2C2C2E`, while current stylesheet defines `#141312/#1e1c1b/#2e2c2b`. Task tracking and commit are blocked until that unrelated gate is fixed or explicitly accepted.

## Ready for Next Run

- Implementation is in place, but task is not marked complete and no commit was created because `make verify` failed on the unrelated web token test gate.
