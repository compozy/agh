# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extract `internal/frontmatter` as the only shared parser for YAML frontmatter across `config`, `memory`, and `skills` without changing caller-visible behavior.

## Important Decisions
- Introduced `frontmatter.Split` for normalized delimiter/body parsing and `frontmatter.Decode` for shared decode orchestration with caller-owned YAML decoding.
- Preserved legacy `config` missing/unterminated error strings via a local wrapper helper instead of package-local sentinel errors.

## Learnings
- `skills` needed a thin `parseSkillContent` helper after the extraction so unknown-field warnings stay skill-specific while parsing logic remains shared.
- `make verify` passes cleanly when `NO_COLOR` is unset in the shell environment; otherwise frontend tooling emits unrelated `NO_COLOR` noise.

## Files / Surfaces
- `internal/frontmatter/frontmatter.go`
- `internal/frontmatter/frontmatter_test.go`
- `internal/config/agent.go`
- `internal/config/agent_test.go`
- `internal/memory/store.go`
- `internal/skills/loader.go`
- `internal/skills/loader_test.go`
- `internal/skills/registry.go`

## Errors / Corrections
- Initial multi-file patch missed `internal/skills/loader.go`; reapplied the refactor in smaller patches after re-reading the live file.
- `internal/config` coverage stayed at 79.5% after the first regression test addition, so added focused agent-loading and discovery-root coverage to reach the task target.

## Ready for Next Run
- Verification evidence:
  - `go test -cover ./internal/frontmatter ./internal/config ./internal/memory ./internal/skills`
  - `env -u NO_COLOR make verify`
- Tracking files are updated in the working tree, and commit `fac9b20` contains only the task-relevant source/test changes.
