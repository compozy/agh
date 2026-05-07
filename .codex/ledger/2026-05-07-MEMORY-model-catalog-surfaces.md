Goal (incl. success criteria):

- Execute provider-model-catalog Task 07 end-to-end: expose daemon-owned model catalog through shared contract payloads, HTTP/UDS native routes, HTTP-only OpenAI `/api/openai/v1/models`, CLI commands, tests, codegen, tracking updates, verification, and one local commit.

Constraints/Assumptions:

- Must use RTK prefix for shell commands.
- Must use workflow memory before editing code and before finishing.
- Must follow `cy-workflow-memory`, `cy-execute-task`, `cy-final-verify`, `agh-contract-codegen-coship`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- Must read AGENTS/CLAUDE guidance, PRD docs, `_techspec.md`, `_tasks.md`, Task 07, and ADRs before implementation.
- No destructive git commands without explicit user permission.
- `make verify` is required before completion and before/after commit.
- Conversation in Brazilian Portuguese; code/docs/artifacts in English.

Key decisions:

- Ledger created for this agent session at `.codex/ledger/2026-05-07-MEMORY-model-catalog-surfaces.md`.

State:

- Completed Task 07 implementation in local commit `742c4b58`; pre-commit and post-commit `make verify` both passed.

Done:

- Read RTK.
- Scanned `.codex/ledger/` and read relevant provider-model-catalog cross-agent ledgers for Tasks 02-05/spec.
- Loaded required `cy-workflow-memory`, `cy-execute-task`, and `cy-final-verify` skill entrypoints.
- Read workflow memory and Task 07 memory.
- Recorded Task 07 objective/pre-change signal in `.compozy/tasks/provider-model-catalog/memory/task_07.md`.
- Added model catalog DTOs and OpenAI error/list DTOs in `internal/api/contract`.
- Added shared core model catalog handlers/converters and error mapping.
- Registered native HTTP/UDS routes via `/api/providers/*catalog_path`; registered `/api/openai/v1/models` on HTTP only.
- Added `agh provider models list|refresh|status` and UDS client methods.
- Added focused core/httpapi/udsapi/spec/CLI tests; focused `go test` over affected packages passes.
- Ran `make codegen` and `make codegen-check`; both passed.
- Fixed the site API-reference generator/navigation and manual-route test matcher so catch-all Gin routes cover exact provider model OpenAPI paths and the `providers`/`openai` tags are mapped.
- Focused site API reference tests passed.
- Fixed Go lint findings in new model catalog code; `make lint` now passes with 0 issues.
- Moved HTTP/UDS model catalog canonical JSON parity coverage into `internal/api/testutil` with a short UDS home path to keep transport packages from importing each other and satisfy macOS socket length limits.
- Full pre-commit `make verify` passed.
- Updated Task 07 tracking files to completed.
- Created local commit `742c4b58 feat: expose provider model catalog surfaces`.
- Full post-commit `make verify` passed.

Now:

- Prepare final report.

Next:

- None.

Open questions (UNCONFIRMED if needed):

- None.

Working set (files/ids/commands):

- `.codex/ledger/2026-05-07-MEMORY-model-catalog-surfaces.md`
- `.compozy/tasks/provider-model-catalog/memory/MEMORY.md`
- `.compozy/tasks/provider-model-catalog/memory/task_07.md`
- `internal/api/contract/contract.go`
- `internal/api/core/model_catalog.go`
- `internal/api/core/model_catalog_conversions.go`
- `internal/api/testutil/model_catalog_parity_test.go`
- `internal/api/httpapi/routes.go`
- `internal/api/udsapi/routes.go`
- `internal/api/spec/model_catalog.go`
- `internal/cli/provider_models.go`
- `internal/cli/client_provider_models.go`
- `packages/site/lib/runtime-navigation.ts`
- `packages/site/scripts/generate-openapi.ts`
- `packages/site/lib/__tests__/runtime-manual-api-routes.test.ts`
- `packages/site/content/runtime/api-reference/meta.json`
- `.compozy/tasks/provider-model-catalog/task_07.md`
- `.compozy/tasks/provider-model-catalog/_tasks.md`
- `.compozy/tasks/provider-model-catalog/_techspec.md`
- `.compozy/tasks/provider-model-catalog/adrs/`
- `.compozy/tasks/provider-model-catalog/memory/task_07.md`
