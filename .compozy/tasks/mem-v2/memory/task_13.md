# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement the Memory v2 Slice 1 backend config/settings truth: structs, defaults, validation, overlay merge, settings DTO parse/render, tool/CLI mutability policy, and generated consumer refresh.
- Task references checked: `_techspec.md` `Config Lifecycle`, `Assumptions / Defaults`, `Development Sequencing` step 23, and ADR-001 through ADR-012.

## Important Decisions

- Memory v2 uses the existing `[memory]` config tree with expanded nested sections; no `[memory.v2] enabled` flag or compatibility bridge was added.
- `memory.dream.agent` now defaults to the dedicated `dreaming-curator`; the old inheritance from `[defaults].agent` was removed so dreaming has a stable runtime identity.
- `memory.workspace.toml_path` is read-facing and validation-locked to `<workspace>/.agh/workspace.toml`; tool/settings writers cannot repoint it.
- Agent-facing config writes allow safe scalar/list memory knobs and deny trust-root paths such as global memory roots, extractor inbox/DLQ roots, session ledger root, daily archive path, and workspace TOML path.
- Contract settings payload changes were co-shipped with generated OpenAPI, generated web OpenAPI types, TypeScript SDK contracts, and web test fixtures so later web tasks consume the backend truth.

## Learnings

- Expanding `SettingsMemoryConfigPayload` requires web fixtures/tests to provide the full generated shape even before the dedicated web settings surface task renders every field.
- `internal/config` coverage is sensitive after large struct/default additions; behavior tests for defaults, overlay, validation branches, and tool-surface policy keep it at the required floor.
- `DefaultWithHome`, settings conversion, settings parsing, and settings diff/apply logic need section-level helpers to stay under `funlen`, `gocyclo`, and `hugeParam` lint gates.

## Files / Surfaces

- Production: `internal/config/config.go`, `internal/config/merge.go`, `internal/config/bootstrap.go`, `internal/config/tool_surface.go`, `internal/cli/config.go`.
- Settings contract/core: `internal/api/contract/settings.go`, `internal/api/core/conversions.go`, `internal/api/core/settings.go`, `internal/settings/sections.go`.
- Generated/consumer refresh: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`, web settings fixtures/tests.
- Tests: `internal/config/memory_v2_config_test.go`, config/tool-surface/bootstrap tests, settings/core tests, and web settings mutation/page route tests.

## Errors / Corrections

- `make lint` initially exposed function length, cyclomatic complexity, huge parameter, constant reuse, long-line, and unused-code problems; fixed by splitting memory config/settings helpers and removing the obsolete dream-agent inheritance hook.
- `make web-typecheck` initially failed because settings fixtures still used the old dream-only memory payload; fixed by adding a full Memory v2 settings config fixture and updating route/mutation tests.
- Site OpenAPI generation ran during validation but produced no tracked site diffs.

## Ready for Next Run

- Task 13 focused validation passed: config/API/settings/CLI tests, race tests, codegen-check, web lint/typecheck/test, site typecheck/test/build, `git diff --check`, and pre-tracking `make verify`.
- Final post-state `make verify` passed after task tracking and `state.yaml` update.
- Next task after state update should be `task_14` (Public Memory Contract Surface).
