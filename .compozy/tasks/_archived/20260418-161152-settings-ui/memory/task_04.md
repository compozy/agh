# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Define shared settings API DTOs and the authoritative OpenAPI surface for `/api/settings/*`, restart polling, observability log-tail metadata, and HTTP-visible extension operations, then regenerate checked-in artifacts and verify the generated web types against the new contract.

## Important Decisions
- Added dedicated settings/restart DTOs under `internal/api/contract` instead of reusing `internal/settings` or daemon/service structs directly, so the API contract stays transport-neutral and JSON-tagged.
- Kept one spec surface in `internal/api/spec` for both HTTP and UDS and exposed the Hooks & Extensions HTTP parity in the same registry instead of forking transport-specific definitions.
- Modeled `/api/settings/observability/log-tail` as an SSE route with a `200` response that intentionally has no JSON body in the OpenAPI contract.
- Switched the web contract check to field-level type assertions after the first broad structural fixture produced a false-negative type mismatch during `tsgo --noEmit`.

## Learnings
- `kin-openapi` response lookups in this repo use `operation.Responses.Status(code)`, not `Get(code)`.
- `make codegen` updates `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`; the new settings route surface produces large generated diffs but remains stable under `make verify` / `codegen-check`.
- The generated settings section responses require shared metadata fields like `scope` and optional `workspace_id`, so web-facing type assertions need to check those explicitly rather than assuming partial structural matches.

## Files / Surfaces
- `internal/api/contract/settings.go`
- `internal/api/contract/settings_test.go`
- `internal/api/spec/spec.go`
- `internal/api/spec/settings_test.go`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`
- `web/src/lib/settings-api-contract.test.ts`

## Errors / Corrections
- Fixed the first spec test pass by replacing `Responses.Get(...)` with `Responses.Status(...)`.
- Reworked the new web OpenAPI type assertion test after `tsgo --noEmit` failed on an incomplete structural fixture for `getSettingsGeneral`.

## Ready for Next Run
- Task 04 code was committed as `3a164064` (`feat: add settings api contract surface`) and the post-commit `make verify` rerun passed; only the intentionally unstaged workflow/task tracking files remain in the worktree.
