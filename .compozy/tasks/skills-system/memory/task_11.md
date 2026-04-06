# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement local-only `agh skill list`, `view`, `info`, and `create` in `internal/cli/skill.go`, register the parent command in `internal/cli/root.go`, and add unit tests that cover the task-11 acceptance criteria.
- Reuse the current `skills.Registry` load path so CLI visibility matches daemon prompt-catalog visibility, including disabled-skill state and critical-content blocking.

## Important Decisions
- Use an ephemeral registry per command invocation rather than direct filesystem scanning from the CLI layer.
- Keep `agh skill view` human/toon output XML-like for agent consumption; use structured JSON only for `-o json`.
- Scaffold new skills under workspace `.agh/skills/<name>/SKILL.md`.
- Treat the task-owned coverage gate as `internal/cli/skill.go` coverage rather than the entire long-lived `internal/cli` package, whose broader baseline includes unrelated pre-existing surfaces.

## Learnings
- The pre-change baseline confirmed task 11 was missing entirely: there was no `internal/cli/skill.go`, and `newSkillCommand` was absent from `internal/cli/root.go`.
- The current `skills.SkillMeta` only includes `name`, `description`, `version`, and free-form `metadata`, so `info` must reflect the current alpha schema rather than the older project’s broader frontmatter set.
- Bundled skills are exposed through `internal/skills/bundled.FS()` and use relative embedded paths, so resource listing/file reads need separate handling from filesystem-backed skills.
- `make verify` passed after the CLI skill changes, and targeted coverage for `internal/cli/skill.go` measured 81.8% (274/335 statements).

## Files / Surfaces
- `internal/cli/root.go`
- `internal/cli/skill.go`
- `internal/cli/skill_test.go`
- `internal/skills/registry.go`
- `internal/skills/catalog.go`
- `internal/skills/bundled/embed.go`

## Errors / Corrections
- Initial skill-path reads for `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify` used the wrong base path (`/Users/pedronauck/.agents/...`). Correct skill files live under `/Users/pedronauck/Dev/projects/agh/.agents/skills/...`.
- Initial bundled-resource handling used `path.Rel`, which is unavailable in the current target toolchain here; replaced it with slash-prefix trimming.

## Ready for Next Run
- Update `task_11.md` and `_tasks.md`, self-review the task-owned diff, and create the local commit once the tracking files are in sync.
