# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 08 so extensions can contribute validated model catalog source rows through manifest capability `model.source` and AGH-to-extension `models/list`, while Host API `models/list|refresh|status` exposes daemon-owned projections/status.
- Success requires capability-gated extension service/Host API methods, extension row/source ID validation before persistence, fail-closed source status errors, marketplace/source-tier grant ceilings, unit/integration tests, `go test ./internal/extension/...`, >=80% relevant coverage, full `make verify`, task tracking updates, and one local commit.

## Important Decisions
- Reuse Task 07's native model catalog payload/service path for Host API results; do not introduce extension-owned projection DTOs or let extension payloads bypass `internal/modelcatalog`.
- Extension model source rows are input only. `internal/modelcatalog` remains the validation, persistence, merge, and public projection authority.
- Extension source IDs are derived through `modelcatalog.SourceKindExtensionID(<extension name>)`, using the TechSpec `<kind>:<slug>` shape (`extension:<slug>`) and rejecting names that cannot normalize to a valid slug.
- The AGH-to-extension `models/list` payload types are part of the extension SDK root type set (`ModelSourceListParams`, `ModelSourceListResponse`, `ModelSourceRow`) so Task 10 docs/SDK work inherits the actual backend contract.
- `make codegen-check` required regenerating `sdk/typescript/src/generated/contracts.ts`; generated docs/prose remain Task 10 scope.

## Learnings
- Shared memory confirms Task 05 injected `core.ModelCatalogService` is the path later API/CLI/extension tasks should consume.
- Shared memory confirms Task 07 completed native HTTP/UDS/CLI/OpenAI model catalog surfaces and generated contract payloads.
- The AGH test-convention helper lives at `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`, not root `scripts/`.
- Existing legacy extension test files still trigger broad heuristic findings; Task 08's new test files and the touched protocol test were shaped to pass the helper.

## Files / Surfaces
- Protocol/contract: `internal/extension/protocol/host_api.go`, `internal/extension/contract/host_api.go`, `internal/extension/contract/sdk.go`.
- Host/runtime: `internal/extension/host_api.go`, `internal/extension/host_api_models.go`, `internal/extension/tool_runtime.go`, `internal/extension/model_source.go`, `internal/daemon/{boot.go,daemon.go,model_catalog.go}`.
- Validation/capabilities: `internal/extension/manifest.go`, `internal/extension/capability.go`, `internal/modelcatalog/source_id.go`.
- Tests: `internal/extension/{capability_models_test.go,host_api_models_test.go,manager_model_source_test.go,manifest_model_source_test.go,model_source_test.go}`, plus protocol/helper updates in existing extension tests.
- Generated: `sdk/typescript/src/generated/contracts.ts`.

## Errors / Corrections
- Corrected the missing root test-convention script path by running the helper from the skill directory.
- Refactored new tests into `t.Run("Should ...")` subtests after the convention helper flagged inline cases.
- Fixed lint findings by splitting protocol/model-source functions and passing daemon extension deps by pointer into helper builders.
- Added SDK root registration for `ModelSource*` types after review showed the Go service payloads were not yet emitted into generated contracts.
- `make verify` regenerates site API reference during typecheck; final scoped Task 08 generated diff is TypeScript contracts only, not OpenAPI.

## Ready for Next Run
- Task 08 is complete in local commit `fef35196 feat: add extension model source contract`.
- Post-commit `make verify` passed with the same existing tool/runtime warnings (`NO_COLOR`/`FORCE_COLOR`, Vite chunk-size advisory, macOS linker warning) and no gate errors.
