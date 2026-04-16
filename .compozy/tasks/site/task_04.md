---
status: completed
title: "Implement CLI doc generation (Cobra GenMarkdownTree)"
type: backend
complexity: medium
dependencies: [task_03]
---

# Task 04: Implement CLI doc generation (Cobra GenMarkdownTree)

## Overview

Add a `doc` subcommand to the AGH CLI that uses Cobra's `doc.GenMarkdownTree()` to generate markdown reference documentation for all CLI commands. Create a post-processing script that transforms the raw Cobra markdown output into Fumadocs-compatible MDX files with proper frontmatter. Add a `make cli-docs` target that runs the full pipeline: build Go binary, generate markdown, post-process to MDX. See TechSpec "Integration Points > Go Binary" and ADR-004 for the auto-generation strategy.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST add a `doc` subcommand to the CLI command tree in `internal/cli/` that accepts an `--output-dir` flag defaulting to `packages/site/content/runtime/reference/cli/`
- MUST use `github.com/spf13/cobra/doc.GenMarkdownTree(rootCmd, outputDir)` to generate markdown files
- MUST create a post-processing script (Go or shell) that performs the following transformations on each generated `.md` file:
  - Strip Cobra boilerplate (auto-generated header, "SEE ALSO" section links that point to local files)
  - Extract command name and description from the markdown content
  - Add Fumadocs-compatible YAML frontmatter with `title` and `description` fields
  - Rename `.md` to `.mdx` extension
- MUST output processed files to `packages/site/content/runtime/reference/cli/`
- MUST create a `meta.json` in the CLI reference directory for sidebar ordering of command groups
- MUST add `make cli-docs` target to `Makefile` that runs the full pipeline: `go run ./cmd/agh doc --output-dir <tempdir> && <post-process> <tempdir> <final-dir>`
- MUST add the `doc` command to `newRootCommand()` in `internal/cli/root.go`
- MUST NOT expose the `doc` command in production help output — mark it as hidden (`cmd.Hidden = true`)
- MUST handle the case where the output directory does not exist (create it)
- MUST ensure generated docs do not contain absolute file paths or machine-specific information
- MUST pass `make verify` — the new Go code must compile, lint, and test cleanly
- MUST NOT import the `doc` package in production paths — the `doc` subcommand is build-time only
</requirements>

## Subtasks

- [ ] 4.1 Create `internal/cli/doc.go` with `newDocCommand()` function
- [ ] 4.2 Implement `doc` subcommand using `cobra/doc.GenMarkdownTree()`
- [ ] 4.3 Register `doc` command in `newRootCommand()` with `Hidden: true`
- [ ] 4.4 Create post-processing script (`scripts/process-cli-docs.sh` or `internal/cli/docpost/`)
- [ ] 4.5 Implement frontmatter injection: extract title/description, write YAML header
- [ ] 4.6 Implement Cobra boilerplate stripping (auto-generated comments, SEE ALSO cleanup)
- [ ] 4.7 Implement `.md` to `.mdx` rename
- [ ] 4.8 Create `meta.json` for CLI reference sidebar ordering
- [ ] 4.9 Add `make cli-docs` target to Makefile
- [ ] 4.10 Run `make cli-docs` and verify output in `packages/site/content/runtime/reference/cli/`
- [ ] 4.11 Verify `make verify` passes (Go lint, test, build)
- [ ] 4.12 Verify `make site-build` succeeds with generated CLI docs

## Implementation Details

See TechSpec sections: "Integration Points > Go Binary — CLI Reference Generation", "Development Sequencing > Build Order step 3".

Cobra's `doc.GenMarkdownTree()` generates one `.md` file per command, named `<use-path>.md` (e.g., `agh_session_list.md`). The raw output includes an auto-generated header comment, command synopsis, flags table, and a "SEE ALSO" section with links to parent/child commands.

The post-processing script needs to:

1. Parse the command name from the filename (replace `_` with space)
2. Extract the first paragraph after the `##` heading as the description
3. Remove the `###### Auto generated` footer
4. Clean up "SEE ALSO" links to use relative MDX paths instead of `.md` file references
5. Prepend YAML frontmatter: `---\ntitle: "agh session list"\ndescription: "..."\n---`
6. Write the result as `.mdx`

The `doc` command should be hidden because it is a build-time tool, not a user-facing feature. It is registered in the command tree so it has access to the full command hierarchy, but `agh doc` should not appear in `agh --help` output.

### Relevant Files

- `internal/cli/root.go:67-99` — `newRootCommand()` where `doc` command must be registered (line 97 area, after existing `cmd.AddCommand` calls)
- `internal/cli/root.go:63-66` — `NewRootCommand()` public constructor
- `cmd/agh/main.go` — Main entry point (no changes expected — uses `NewRootCommand()`)
- `Makefile:42-61` — Web targets section; add `cli-docs` target after this block
- `packages/site/content/runtime/` — Output directory for generated CLI reference docs (task_03 deliverable)

### Dependent Files

- `packages/site/content/runtime/reference/cli/*.mdx` — Generated output consumed by Fumadocs build
- `packages/site/app/runtime/[[...slug]]/page.tsx` — Renders the generated CLI reference pages (task_03)

### Related ADRs

- [ADR-004: Auto-Generated CLI and API Reference](adrs/adr-004.md) — Cobra GenMarkdownTree chosen over custom JSON export; post-processing adds Fumadocs frontmatter

## Deliverables

- `internal/cli/doc.go` with `newDocCommand()` implementing `doc` subcommand
- Post-processing script (shell or Go) for Cobra markdown to Fumadocs MDX transformation
- `meta.json` for CLI reference sidebar ordering
- `make cli-docs` Makefile target
- Generated `.mdx` files in `packages/site/content/runtime/reference/cli/`
- Unit tests for post-processing logic
- `make verify` passing

## Tests

- Unit tests:
  - [ ] `doc` command creates output directory if it does not exist
  - [ ] `doc` command generates markdown files for all CLI commands
  - [ ] Post-processing: raw Cobra markdown is transformed to valid MDX with frontmatter
  - [ ] Post-processing: `title` in frontmatter matches command name
  - [ ] Post-processing: `description` in frontmatter is extracted from command short description
  - [ ] Post-processing: auto-generated Cobra footer is stripped
  - [ ] Post-processing: `.md` files are renamed to `.mdx`
  - [ ] Post-processing: no absolute file paths or machine-specific paths in output
- Integration tests:
  - [ ] `make cli-docs` runs end-to-end without errors
  - [ ] Generated files are valid MDX (parseable by Fumadocs build)
  - [ ] `make site-build` succeeds with generated CLI reference pages in the content tree
  - [ ] Generated CLI reference pages appear in the runtime sidebar
- Go quality:
  - [ ] `make verify` passes with zero warnings
  - [ ] `doc` command does not appear in `agh --help` output (hidden)
- Test coverage target: >=80% for `internal/cli/doc.go` and post-processing logic

## Success Criteria

- `make verify` passes
- `make cli-docs` generates `.mdx` files in `packages/site/content/runtime/reference/cli/`
- Generated files have valid Fumadocs frontmatter (`title`, `description`)
- No Cobra boilerplate or auto-generated comments in output
- `make site-build` succeeds with generated CLI docs included
- `doc` command is hidden from user-facing help
- Unit test coverage >=80% for doc generation and post-processing code
