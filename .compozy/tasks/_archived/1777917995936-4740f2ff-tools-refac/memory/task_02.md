# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the Task 02 shipped guidance layer: bundled `agh-tools-guide`, startup `tools` prompt section, tool-first catalog/setup/network guidance, aligned site docs, deterministic unit/integration coverage, and a scoped local commit after verification.

## Important Decisions
- Reused the existing bundled-skill loading path for the startup tools section by adding a normal bundled skill and a normal `PromptSectionDescriptor`.
- Gated the new startup `tools` section through `HarnessRuntimeSignals.ToolsPromptSectionEnabled`, sourced from `[tools].enabled`, so disabling the tool system suppresses tool guidance as expected.
- Ordered the tools section after skills and before network (`startupToolsSectionOrder = 150`) to teach discovery after the skill catalog and before channel-specific coordination.
- Kept CLI examples only as operator/fallback paths when a dedicated AGH tool is not exposed or policy denies the tool.

## Learnings
- `internal/skills/bundled/content.go` embeds `skills/**` through the existing bundle, so the new `agh-tools-guide/SKILL.md` needed no manual content index edit.
- `scripts/check-test-conventions.py` referenced by the installed test-conventions skill is absent in this checkout; `rg` found no replacement script.
- Package-wide coverage for `internal/daemon` remains below 80% because the package is broad, but the changed prompt-section functions are covered at or above the target and `internal/skills` / `internal/skills/bundled` report 82.1% / 85.7%.
- A concurrent unrelated task lease/store interface edit appeared after the clean `make verify` run and currently breaks ad-hoc `go test ./internal/daemon`; verify the Task 02 commit in a clean worktree before final handoff.

## Files / Surfaces
- `internal/skills/bundled/skills/agh-tools-guide/SKILL.md`
- `internal/daemon/boot.go`
- `internal/daemon/harness_context.go`
- `internal/daemon/prompt_sections.go`
- `internal/daemon/composed_assembler_test.go`
- `internal/daemon/harness_context_test.go`
- `internal/daemon/harness_context_integration_test.go`
- `internal/daemon/daemon_test.go`
- `internal/skills/catalog.go`
- `internal/skills/catalog_test.go`
- `internal/skills/bundled/bundled_test.go`
- `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`
- `internal/skills/bundled/skills/agh-network/SKILL.md`
- `packages/site/content/runtime/core/configuration/agent-md.mdx`
- `packages/site/content/runtime/core/agents/definitions.mdx`
- `packages/site/content/runtime/core/network/index.mdx`

## Errors / Corrections
- First `make verify` failed in `golangci-lint` with gocritic `appendCombine` on the consecutive tools/network descriptor appends in `internal/daemon/prompt_sections.go`; fixed by combining them into one append.
- `go test -tags integration ./internal/daemon -cover -count=1` exposed pre-existing/unrelated integration failures in agent tool preservation and acpmock blocked cancel behavior before the final full gate; focused Task 02 integration tests passed.
- After the full `make verify` pass, unrelated concurrent edits under `internal/task` and `internal/store/globaldb` introduced a missing `LookupActiveRunForSession` compile error for daemon/api test packages. Do not edit or revert those files for Task 02.

## Ready for Next Run
- Task 02 implementation and self-review are complete.
- Verification evidence before concurrent build break: `make verify` passed with format/oxlint clean, TypeScript typecheck clean, Vitest 257 files / 1838 tests, web build completed, Go lint `0 issues`, Go tests `DONE 7021 tests`, and package boundaries respected.
- Commit: `6640f66a feat: add tools guidance startup section`.
- Post-commit verification: `make verify` passed in clean worktree `/tmp/agh-task02-verify.x0oEQU` after `bun install --frozen-lockfile`, with format/oxlint clean, TypeScript typecheck clean, Vitest 257 files / 1838 tests, web build completed, Go lint `0 issues`, Go tests `DONE 7019 tests`, and package boundaries respected.
- Commit only the Task 02 file set; leave workflow memory/tracking and unrelated dirty files unstaged.
- The shared worktree still contains unrelated uncommitted edits outside Task 02; do not revert them.
