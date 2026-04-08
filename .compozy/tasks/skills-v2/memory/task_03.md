# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Populate `Skill.MCPServers` and `Skill.Hooks` from `metadata.agh` during both disk and bundled skill loading, with warning-only handling for malformed declarations.

## Important Decisions
- Use nested `map[string]any` assertions off `skill.Meta.Metadata["agh"]`; do not add another YAML parse path.
- Required validation behavior will be skip-with-warning, not hard failure, for malformed `metadata.agh`, invalid MCP entries, and unknown hook events.
- Add fixture `SKILL.md` files under `internal/skills/testdata/` to satisfy the task deliverable instead of relying only on inline temp-file strings.
- Leave hook declarations permissive apart from event validation. Empty or malformed optional fields fall back to zero values with warnings where needed; future hook execution tasks can tighten command/runtime validation if required.

## Learnings
- Task 01 already added `MCPServerDecl`, `HookDecl`, and hook constants; loader and tests are the missing link.
- `parseBundledSkill()` is in `internal/skills/registry.go`, not `loader.go`, so task 03 needs edits in both files despite the task summary focusing on the loader.
- YAML frontmatter decoded into `map[string]any` yields string-based duration values like `5s`, so timeout parsing belongs in the metadata extraction pass rather than in the initial frontmatter decode.

## Files / Surfaces
- `internal/skills/loader.go`
- `internal/skills/registry.go`
- `internal/skills/loader_test.go`
- `internal/skills/testdata/`

## Errors / Corrections
- Added explicit bundled-loader coverage after wiring `parseAGHMetadata()` because task requirements apply to both `ParseSkillFile()` and `parseBundledSkill()`.
- Verified package coverage with `go test ./internal/skills -cover -count=1` at `81.5%`, clearing the task-local coverage gate before repo-wide verification.
- Full repository verification passed via `make verify`, including lint, tests, build, and boundary checks.
- The local code-only commit is `331f30d` (`feat: parse agh skill metadata`). Unrelated `docs/rfcs/skills-system-final.md` and the workflow tracking tree remain outside the commit.

## Ready for Next Run
- Task 03 is complete. Future tasks can assume loader-populated `Skill.MCPServers` and `Skill.Hooks` on both disk-loaded and bundled skills.
