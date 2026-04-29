# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Task 06 scope: extend extension manifest tool declarations, schema digest fixtures, cold non-executable resource publication, deterministic reconciliation/availability reason codes, and lifecycle tests.
- Manifest-side only: do not invoke extension handlers; executable reconciliation protocol/SDK work remains for Tasks 07-08.

## Important Decisions
- Source of truth read before code edits: `_techspec.md`, `_tasks.md`, `task_06.md`, ADR-001, ADR-008, ADR-009, shared workflow memory, and prior Task 01-05 ledgers.
- Preserve canonical extension tool identity as `ext__<extension_name>__<tool_key>` unless an explicit manifest `id` is valid and still inside the extension namespace.

## Learnings
- Shared memory says Tasks 01-05 already committed the core tool contract, policy/projection, dispatch path, and native providers. Extension adapters must enter through registry handles later and not bypass `RuntimeRegistry.Call`.
- Existing `_prd.md` is absent in this PRD directory; `_techspec.md`, `_tasks.md`, ADRs, and task files are the available approved source artifacts.
- Manifest tool declarations now fail validation unless they include backend metadata. `extension_host` tools require a handler binding; explicit IDs must stay under `ext__<extension_name>__*` and cannot claim `agh__*`.
- Cold extension tool resources are still published as desired-state descriptors for installed extensions, including disabled/unregistered lifecycle states. MCP server resources keep the existing stricter enabled+registered publication gate.
- RFC 8785/JCS schema digest fixtures are byte-identical across daemon, TypeScript SDK, and Go SDK fixture directories for downstream parity tests.

## Files / Surfaces
- Edited surfaces: `internal/extension/manifest.go`, `internal/extension/resource_publication.go`, `internal/extension/tool_reconciliation.go`, `internal/tools/schema_digest.go`, `internal/tools/tool.go`, `internal/tools/reason.go`, `internal/daemon/tool_mcp_resources.go`, digest fixtures under `internal/extension/testdata/digest`, `sdk/typescript/test-fixtures/digest`, and `sdk/go/test-fixtures/digest`, plus extension/tools/daemon tests.

## Errors / Corrections
- Early compile pass found old manifests/fixtures without backend binding; tests were updated to assert the new required backend metadata instead of weakening validation.
- `internal/tools` package coverage was initially below the task target after adding digest code; direct schema digest/runtime descriptor tests raised it above 80%.
- Full `make verify` initially failed on lint only (`rangeValCopy`, `hugeParam`, `lll`, and one unused parameter); code was adjusted and the full pipeline passed cleanly afterward.

## Ready for Next Run
- Focused tests currently passing: `go test ./internal/extension -coverprofile=/tmp/agh-task06-extension.cover -count=1` (80.6%), `go test ./internal/tools -coverprofile=/tmp/agh-task06-tools.cover -count=1` (82.1%), `go test ./internal/daemon -count=1`, and targeted integration `go test -tags integration ./internal/daemon -run 'TestToolMCPStaticPublicationAndBootRebuild|TestToolMCPStaticPublicationExtensionLifecycle' -count=1`.
- Task 06 implementation, self-review, task tracking updates, code-only local commit, and post-commit verification are complete.
- Local commit: `132648c6 feat: add extension tool manifest reconciliation`.
- Verification evidence: pre-commit and post-commit `make verify` exited 0. Post-commit output included `Found 0 warnings and 0 errors` from oxlint, `0 issues.` from golangci-lint, `DONE 6729 tests`, and `OK: all package boundaries respected`.
