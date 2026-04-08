# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add `agh skill search/install/remove/update` with config-driven marketplace client construction, secure install/update/remove flows, and the task-required unit/integration coverage.

## Important Decisions
- Task 10 must implement the missing `skills.marketplace` config plumbing from task 02 because the CLI cannot construct a registry from config otherwise.
- Use the approved PRD/techspec/task bundle as the design baseline for this run instead of reopening a separate brainstorming approval loop.
- Keep the CLI on the pluggable registry interface: per-command registry construction flows through config plus `marketplace.Registry`, while install/update/remove continue to rely on sidecar provenance instead of duplicating registry state elsewhere.

## Learnings
- The current repository already has marketplace registry/client/provenance primitives, but `internal/config` still lacks `MarketplaceConfig` even though `task_02.md` says completed.
- `internal/cli` package coverage reaches the task threshold with the marketplace command tests plus helper edge cases and dedicated integration coverage for install/remove/hash flows.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/config_test.go`
- `internal/config/merge_test.go`
- `internal/cli/skill.go`
- `internal/cli/skill_test.go`
- `internal/cli/skill_marketplace_integration_test.go`

## Errors / Corrections
- Corrected the initial assumption from task tracking: `_tasks.md` still marks task 02 pending and the code confirms marketplace config is not implemented yet.
- Closed the resulting config gap inside task 10 so the CLI can actually construct a marketplace client from loaded config.

## Ready for Next Run
- Task is implemented and verified. Remaining closeout is limited to task tracking state and the local commit for the verified code changes.
