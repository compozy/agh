# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Task 01 as the provider config hard cut: remove flat provider model fields, add nested provider model config/catalog source/discovery config, update settings/API/web generated consumers, add tests, run required gates, and commit only after `make verify`.

## Important Decisions
- Preserve ACP session `SupportedModels`/`supported_models` where it represents active session capabilities; the hard cut targets provider config/settings/session provider option payload fields, not ACP caps.
- Treat existing modified worktree files as user/branch state. Do not restore or clean; layer Task 01 changes onto the existing diff.

## Learnings
- Pre-change residue exists in `internal/config/provider.go`, `internal/config/merge.go`, settings/API conversions, generated OpenAPI/TS, web settings/session consumers, and site docs. Task 01 owns codegen and minimal web consumers; docs cleanup remains Task 10 unless required by generated drift.
- `config.toml` at repo root was still using `default_model`; the new hard-cut validation rejected it during `go test ./internal/config`, so it was updated to `[providers.<id>.models] default = ...`.
- Backend consumers under settings/API/CLI/workspace/session/situation now compile against `ProviderModelsConfig`; ACP caps still legitimately expose `SupportedModels` as session capability metadata.
- Web settings now edits `settings.models.default` plus `settings.models.curated` IDs; the session create dialog no longer sources pre-session model options from provider `supported_models`.
- Self-review found a nil-versus-empty merge edge case for `ProviderModelsConfig.Curated`; `providerModelsConfigIsZero` now preserves an explicitly empty curated slice so overrides can clear builtin curated models.

## Files / Surfaces
- Expected primary surfaces: `internal/config`, `internal/settings`, `internal/api/contract`, `internal/api/core`, `internal/cli`, `internal/workspace`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, web settings/session consumers and tests.
- Touched so far: `config.toml`, `internal/config/*`, `internal/cli/*`, `internal/settings/*`, `internal/api/contract/*`, `internal/api/core/*`, `internal/api/httpapi/*`, `internal/api/udsapi/*`, `internal/session/*`, `internal/situation/service.go`, `internal/workspace/*`, web settings/session/workspace consumers and fixtures.

## Errors / Corrections
- RTK must prefix shell commands for this workspace. After loading `/Users/pedronauck/.codex/RTK.md`, all shell commands in this run use `rtk`.
- First full `rtk make verify` failed on an old `DefaultModel` fixture in `internal/testutil/e2e/config_seed_test.go`; the fixture now uses `Models.Default`.
- Go lint then found goconst/gocritic/unused issues; fixed constants, single-case switches, and removed unused clone helpers before rerunning the full gate.

## Ready for Next Run
- Task 01 implementation and verification are complete. Fresh evidence after final self-review correction:
  - `rtk go test ./internal/config` -> 622 passed.
  - `rtk make verify` -> exit 0; oxfmt/oxlint reported 0 warnings and 0 errors, Bun typecheck/tests/build, Go fmt/lint/test/build, codegen-check, and boundaries completed.
  - Local commit: `0ff846d4 refactor: hard cut provider model config`.
  - Post-commit `rtk make verify` -> exit 0 with the same non-blocking warning classes.
  - Non-blocking tool warnings observed: Node `NO_COLOR`/`FORCE_COLOR`, Vite chunk-size warning, macOS linker `-bind_at_load` warning.
