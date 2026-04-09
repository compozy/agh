# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Add Skills HTTP endpoints (GET list, GET detail, POST enable, POST disable) exposing skills.Registry through the API layer.

## Important Decisions

- Added `SkillsRegistry` interface to `core.SkillsRegistry` (defined where consumed, per Go convention)
- Used sentinel errors `ErrSkillNotFound` and `ErrSkillValidation` in `core/errors.go` for skill error mapping
- Exported `SkillSourceName` from skills package (was unexported `skillSourceName`) for use by conversion helper
- Added skills routes to both HTTP and UDS transports for consistency
- Enable/disable handlers are placeholders that verify the skill exists and log the action — the Registry doesn't yet have Enable/Disable mutations, so actual state change is deferred

## Learnings

- `ResolvedWorkspace` embeds `Workspace` — access fields like `ID` directly, not via `resolved.Workspace.ID` (staticcheck QF1008)
- Route count tests in both httpapi and udsapi must be updated when adding new routes

## Files / Surfaces

- `internal/api/contract/contract.go` — added SkillPayload, ProvenancePayload, SkillActionResponse
- `internal/api/core/interfaces.go` — added SkillsRegistry interface
- `internal/api/core/handlers.go` — added SkillsRegistry field to BaseHandlerConfig and BaseHandlers
- `internal/api/core/conversions.go` — added SkillPayloadFromSkill, SkillPayloadsFromSkills
- `internal/api/core/errors.go` — added ErrSkillNotFound, ErrSkillValidation, StatusForSkillError
- `internal/api/core/skills.go` — new file with ListSkills, GetSkill, EnableSkill, DisableSkill handlers
- `internal/api/core/skills_test.go` — new file with comprehensive handler tests
- `internal/api/httpapi/server.go` — added skillsRegistry field, WithSkillsRegistry option, route registration
- `internal/api/httpapi/handlers_test.go` — updated route count test
- `internal/api/udsapi/server.go` — added skillsRegistry field, WithSkillsRegistry option
- `internal/api/udsapi/routes.go` — added skills route group
- `internal/api/udsapi/handlers_test.go` — updated route count test
- `internal/skills/registry.go` — exported SkillSourceName
- `internal/daemon/daemon.go` — added SkillsRegistry to RuntimeDeps
- `internal/daemon/boot.go` — wired skillsRegistry into RuntimeDeps

## Errors / Corrections

- First attempt used `resolved.ID` directly in struct literal — ResolvedWorkspace doesn't have top-level ID field, it's embedded from Workspace
- staticcheck QF1008 caught redundant `resolved.Workspace.ID` when `resolved.ID` works due to embedding

## Ready for Next Run

- Task 05 (skills frontend system) can now consume these endpoints
- Enable/disable endpoints are stubs — Registry needs actual Enable/Disable mutations for full functionality
