# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Implement `doc` subcommand for CLI reference generation using Cobra GenMarkdownTree, with post-processing to Fumadocs MDX.

## Important Decisions

- Post-processing implemented as Go package `internal/cli/docpost/` (not shell script) for testability.
- Used `GenMarkdownTree` (not `GenMarkdownTreeCustom`) since post-processing handles all transformations.
- `doc` command is hidden (`Hidden: true`) — build-time only, not user-facing.

## Learnings

- MDX parses `<` and `{` as JSX — must escape them in non-code text. `\<` and `\{` are valid MDX escapes.
- Cobra completion commands (bash, zsh) emit tab-indented shell examples that MDX treats as regular text, not code blocks. Must convert indented blocks to fenced code blocks.
- `fenceIndentedBlocks` must be aware of existing fenced blocks to avoid creating broken nested fences.
- `cobra/doc` is bundled with cobra but requires `go get github.com/spf13/cobra/doc` to populate go.sum entries for transitive dependencies (`go-md2man`, `blackfriday`).
- gosec requires `0o600` file permissions (not `0o644`) for WriteFile.

## Files / Surfaces

- `internal/cli/doc.go` — doc command
- `internal/cli/doc_test.go` — doc command tests
- `internal/cli/docpost/docpost.go` — post-processing logic
- `internal/cli/docpost/docpost_test.go` — post-processing tests
- `internal/cli/root.go` — registered `newDocCommand()` at line 98
- `Makefile` — added `cli-docs` target
- `packages/site/content/runtime/meta.json` — updated to include `...reference`
- `packages/site/content/runtime/reference/meta.json` — new, includes `...cli`
- `packages/site/content/runtime/reference/cli/` — generated output (108 .mdx + meta.json)

## Errors / Corrections

- First attempt used `0o644` permissions — gosec flagged, fixed to `0o600`.
- First `fenceIndentedBlocks` didn't track existing fenced blocks, broke existing code fences.
- Initially only escaped `<` — also needed `{` escaping for MDX compatibility.

## Ready for Next Run
