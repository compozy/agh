# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 07 executable extension-host tools: protocol constants/structs, required method validation, manager-side `provide_tools` reconciliation, `tools/call` invocation through existing subprocess runtime and `Registry.Call`, TypeScript SDK `extension.tool(...)`, create-extension TypeScript tool-provider template, and real subprocess integration tests.

## Important Decisions
- Treat the accepted Task 07 spec, `_techspec.md`, ADR-001, ADR-008, and ADR-009 as design authority; do not reopen the generic brainstorming flow.
- Reuse existing extension subprocess JSON-RPC and TypeScript `Extension.handle(...)` transport pattern; no in-process third-party handlers and no parallel TypeScript runtime.
- Manifest tool descriptors remain authoritative; runtime descriptors are reconciliation proof only and must fail closed on handler/schema/risk drift.
- Wire extension-host tools into the existing central tool registry as a provider over installed extension manifests plus a live runtime resolver; the provider lists cold manifest descriptors and calls subprocess `provide_tools` only for availability/call reconciliation.
- TypeScript `extension.tool(...)` auto-advertises `tool.provider`, derives default IDs as `ext__<extension>__<handler>`, exports runtime descriptors with canonical schema digests, and owns `provide_tools`/`tools/call` handlers.

## Learnings
- Shared workflow memory confirms Task 06 already added cold manifest-authoritative extension tool metadata, runtime digest proof metadata, deterministic lifecycle/mismatch reason codes, and shared digest fixtures.
- Existing dirty state before Task 07 edits includes tools-registry tracking files for prior tasks and untracked workflow memory; do not revert or stage unrelated artifacts.
- Baseline signal before Task 07 code edits: protocol constants for `provide_tools`/`tools/call`, manager `ExtensionToolInvoker` methods, and TypeScript `extension.tool(...)` are absent; focused subprocess/extension/TypeScript tests passed before edits.
- TOML manifests cannot currently decode inline schema objects directly into `json.RawMessage`; Task 07 subprocess integration fixtures use JSON manifests for real schema object coverage.
- Focused validation passed after implementation: Go provider/subprocess/daemon/tools tests, TypeScript SDK/create-extension unit tests, TypeScript SDK stdio integration, codegen-check, typechecks, and `git diff --check`.

## Files / Surfaces
- Expected Go surfaces: `internal/extension/protocol`, `internal/subprocess`, `internal/extension`, `internal/tools`, and daemon composition if provider wiring needs it.
- Expected TypeScript surfaces: `sdk/typescript/src`, `sdk/typescript/test-fixtures/digest`, and `sdk/create-extension/src`.
- Added/modified Go surfaces: `internal/extension/tool_provider.go`, `internal/extension/tool_runtime.go`, `internal/extension/tool_provider_test.go`, `internal/extension/manager_test.go`, `internal/daemon/native_tools.go`, `internal/tools/tool.go`, `internal/subprocess/process_test.go`, and SDK contract generation roots.
- Added/modified TypeScript surfaces: `sdk/typescript/src/extension.ts`, `sdk/typescript/src/schema-digest.ts`, `sdk/typescript/src/errors.ts`, SDK tests/integration, generated contracts, and `sdk/create-extension/templates/tool-provider`.

## Errors / Corrections
- Initial SDK implementation bound `provide_tools`/`tools/call` but did not include them in `implemented_methods` after `bindTransport`; fixed `getImplementedMethods` and transport rebinding coverage.
- Strict TypeScript build rejected unsafe sensitive-field path indexing; fixed by guarding empty path segments before indexing.
- Initial Go test TOML fixtures attempted inline schema maps into `json.RawMessage`; changed fixtures to JSON manifests instead of weakening schema expectations.
- Full `make verify` first failed Go lint on `cloneManifestToolDescriptor` passing a large `ManifestToolDescriptor` by value; fixed the helper to accept a pointer while preserving defensive clone semantics.
- Self-review found the new create-extension tool-provider template used a TOML inline `input_schema`, which is not a loadable `json.RawMessage` manifest shape; changed that template to `extension.json` and updated scaffold tests to parse the generated JSON manifest.
- Final pre-commit `make verify` passed after the template correction: frontend format/lint/typecheck/tests/build, Go lint `0 issues`, Go tests `DONE 6746 tests`, and package boundaries respected.
- Created local code-only commit `f88f47b9 feat: add extension tool runtime sdk`.
- Post-commit `make verify` passed: frontend format/lint/typecheck/tests/build, Go lint `0 issues`, Go tests `DONE 6746 tests`, and package boundaries respected.

## Ready for Next Run
- Task implementation is complete and post-commit verified. Tracking/memory files remain unstaged by policy.
