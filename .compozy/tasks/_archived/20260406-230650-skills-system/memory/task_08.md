# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `SkillsConfig` to `internal/config`, add workspace/global merge overlay support for `[skills]`, add `SkillsDir` to `HomePaths`, and cover the new behavior with unit tests before marking task 08 complete.

## Important Decisions
- Follow the existing `MemoryConfig` / `memoryOverlay` / `DreamConfig` patterns exactly so later daemon and CLI tasks can consume the new config surface consistently.
- Use `time.Duration` for `skills.poll_interval` and default it to `3 * time.Second`, matching the tech spec and task requirements.
- Keep the scope inside `internal/config`; later skills tasks will consume `cfg.Skills` and `HomePaths.SkillsDir`.
- Add a minimal `SkillsConfig.Validate()` and wire it through `Config.Validate()` so `Load()` rejects non-positive `skills.poll_interval` values when skills are enabled.

## Learnings
- `internal/config` currently has strict TOML overlay decoding in `merge.go`, so `[skills]` must be added to both `configOverlay` and a dedicated overlay struct or loading will reject the section as unknown.
- `internal/config` currently has no `merge_test.go`, so overlay-specific assertions will need a new focused test file or expanded coverage elsewhere.
- Adding `SkillsConfig` to `Config` makes whole-struct equality less convenient because of the slice field; tests should compare the skills section explicitly instead of relying on struct comparability.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/home.go`
- `internal/config/config_test.go`
- `internal/config/home_test.go`
- `internal/config/merge_test.go` (new)

## Errors / Corrections
- None.

## Ready for Next Run
- Implementation complete and verified locally.
- Verification evidence:
  - `go test -race -cover ./internal/config` passed with `81.2%` coverage.
  - `go vet ./...` passed.
  - `make verify` passed after the committed state as well.
- Tracking files updated in the worktree: `task_08.md` and `_tasks.md`.
- Local commit created: `35f85ee` (`feat: add skills config support`).
- Tracking and memory files were intentionally left unstaged because the skills-system task directory already has unrelated tracking edits in the dirty worktree.
