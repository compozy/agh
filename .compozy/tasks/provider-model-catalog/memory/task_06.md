# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Upgrade AGH's ACP runtime path from `github.com/coder/acp-go-sdk` v0.6.3 to v0.12.2 and make active ACP session `configOptions` the source of truth for model/reasoning controls.
- Success requires the breaking-change analysis artifact, dependency upgrade, config option capture/update behavior, contract payload exposure, focused tests, full verification, tracking updates, and one local commit.

## Important Decisions
- Treat the pre-session model catalog and active ACP session config as separate surfaces. This task only owns session-scoped ACP `configOptions`.
- Prefer `session/set_config_option` for requested model changes when a model select config option exists; use legacy `session/set_model` only when no config options exist and the legacy model state advertises the requested model.
- Prefer `session/set_config_option` for reasoning effort only when a conservative reasoning select option exists and contains the requested value. Do not invent reasoning levels from catalog metadata.
- Keep task-local analysis in `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md` before migrating production code.

## Learnings
- ACP v0.12.2 adds `configOptions` to `NewSessionResponse`/`LoadSessionResponse`, `config_option_update` notifications, and `session/set_config_option`.
- ACP v0.12.2 keeps the legacy `session/set_model` method constant but renames its request/response types to `UnstableSetSessionModelRequest`/`UnstableSetSessionModelResponse`.
- Existing AGH code currently captures only modes/models in ACP caps and has no config option state, dynamic config update handling, or reasoning effort start option in the ACP driver.
- Additional v0.12.2 compile impacts found during migration: `FileSystemCapability` -> `FileSystemCapabilities`, terminal kill request/response renamed, permission tool calls now use `ToolCallUpdate`, and prompt `_meta` expects `map[string]any`.
- Focused changed-behavior coverage is >=80% for new ACP config option functions and model/reasoning application functions. Whole `internal/acp` package coverage remains below 80% due unrelated pre-existing uncovered runtime surfaces.

## Files / Surfaces
- Touched: `go.mod`, `go.sum`, `internal/acp/*`, `internal/session/*`, `internal/api/contract/*`, `internal/api/core/*`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.
- Deliverable created: `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md`.

## Errors / Corrections
- Initial compile after `go get` surfaced SDK symbols missing from the first audit pass; the analysis artifact was updated before continuing migration.
- Existing preferred-model fixture used values outside helper legacy `SessionModelState`; tests now use advertised values and separate tests cover deterministic unsupported-model errors.
- Self-review found one remaining direct production read of `process.Caps` in `loadSession`; it now uses `CapsSnapshot()` like the rest of the active config path.

## Completion State
- Implemented and pre-commit verification passed after self-review corrections.
- Tracking updated in `task_06.md` and `_tasks.md`.
- Commit: `cc1e31b6` (`feat: upgrade acp session config options`).
- Pending: none for Task 06 implementation.
- Evidence:
  - `rtk go test ./internal/acp ./internal/session ./internal/api/contract ./internal/api/core -count=1` passed 1454 tests.
  - `rtk make codegen-check` passed.
  - `rtk go test ./internal/acp -count=1` passed after lint refactor.
  - `rtk make lint` passed with 0 issues after lint refactor.
  - `rtk make verify` passed pre-commit after the final self-review correction.
  - `rtk make verify` passed post-commit.
