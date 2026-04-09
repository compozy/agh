# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement declaration normalization, matcher evaluation, and deterministic ordering in `internal/hooks`, with unit coverage above 80% and repository-wide verification via `make verify`.

## Important Decisions
- Added `ValidateHookDecl` / `NormalizeHookDecl` split so later config and agent-definition loaders can reuse declaration validation without requiring concrete executors.
- Added internal-only `HookSkillSource` and `HookDecl.PrioritySet` fields in `internal/hooks/types.go` to preserve skill precedence ordering and explicit zero priorities without importing `internal/skills`.
- Kept executor inference strict: shell fields require `subprocess`, `native` executors are limited to native hook sources, and non-native declarations without command or executor kind fail normalization.

## Learnings
- Permission matcher uses `PermissionToolCall.Kind` as the tool-name surface because the current permission payload does not expose a separate `ToolName` field.
- `internal/hooks` package coverage reached 87.4% after adding family-level matcher tests and declaration-slice normalization tests.
- Full repository verification passed after the implementation (`make verify`).

## Files / Surfaces
- `internal/hooks/types.go`
- `internal/hooks/normalize.go`
- `internal/hooks/matcher.go`
- `internal/hooks/ordering.go`
- `internal/hooks/types_test.go`
- `internal/hooks/normalize_test.go`
- `internal/hooks/matcher_test.go`
- `internal/hooks/ordering_test.go`

## Errors / Corrections
- Initial package coverage was 79.5%; added broader matcher-family coverage and declaration-slice tests to raise it above the 80% task gate.
- Self-review caught that `SkillSource` should be rejected for non-skill declarations; normalization now enforces that invariant.

## Ready for Next Run
- Task 02 is implemented and verified. Next dependent tasks can consume `ValidateHookDecl`, `NormalizeHookDecl(s)`, the family matcher helpers, and `SortResolvedHooks` / `OrderedResolvedHooks`.
