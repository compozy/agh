# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed task 01 by adding automation config parsing plus foundational automation domain types and validation without implementing runtime behavior.

## Important Decisions
- Follow the hooks pattern: TOML-facing structs live in `internal/config`; shared transport/runtime/store-facing contracts live in `internal/automation`.
- Treat strict trigger template validation as parse plus activation-envelope shape validation, not parse-only.
- Keep config-defined workspace bindings as `Workspace string` in `internal/config` and reserve canonical `WorkspaceID` fields for `internal/automation`.
- Apply automation defaults during overlay normalization so config-defined jobs and triggers always validate with explicit `source`, retry, enablement, and fire-limit values.

## Learnings
- `text/template` with `missingkey=error` does not fail for `index .Data "key"` on a missing map key, so explicit validation is required for strict prompt semantics.
- Repo-wide Go package testing depends on `web/dist` because `web/embed.go` is imported by several packages; final verification must go through the normal Mage pipeline.
- The template parse tree may contain typed-nil `ElseList` values; traversal must guard them explicitly to avoid panics during validation.

## Files / Surfaces
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/automation.go`
- `internal/config/automation_test.go`
- `internal/config/automation_integration_test.go`
- `internal/config/hooks.go` as the conversion/validation precedent
- `internal/daemon/boot.go`
- `internal/automation/` (new)

## Errors / Corrections
- Removed `t.Parallel()` from config tests that rely on `t.Setenv()` in the shared environment helper.
- Hardened template AST traversal against nil parse-node branches after a panic surfaced in tests.
- Switched template validation to use `subtemplate.Root` directly to satisfy lint expectations.

## Ready for Next Run
- Task complete. Next tasks can persist and consume the `internal/automation` types directly, resolving config-level workspace bindings into canonical workspace IDs as part of that flow.
