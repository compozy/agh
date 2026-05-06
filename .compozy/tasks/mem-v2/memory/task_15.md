# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Regenerate and validate the generated OpenAPI/TypeScript consumer artifacts after the Memory v2 public contract surface.
- Task references checked: `_techspec.md` `Public Interfaces / Types`, `Web/Docs Impact`, `Development Sequencing` step 28, and ADR-009/ADR-011 codegen implications.

## Important Decisions

- `make codegen` is the authority for `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and `sdk/typescript/src/generated/contracts.ts`; no generated TypeScript was hand-edited.
- The existing `web/src/lib/api-contract.ts` wrapper already supports Memory v2 operation request/query/path/response extraction, so the task added a focused type-level consumer test rather than changing wrapper behavior.
- `web/src/lib/memory-api-contract.test.ts` now guards the generated Memory v2 operation IDs, selector fields, mutation payloads, recall-trace path params, redaction-safe decisions, and deterministic error envelope shape.
- Public site API reference generation remains filtered to registered HTTP/UDS routes until task 16 lands route parity; OpenAPI/codegen still includes the full task 14 Memory v2 contract.

## Learnings

- Do not run multiple `make bun-*` targets in parallel in this repo; concurrent Mage invocations can race on `mage_output_file.go` and produce false failures. Sequential reruns passed.
- The web generated contract test count increased from 329 files / 2087 tests to 330 files / 2090 tests after adding the Memory v2 type-level assertions.
- Generated Memory v2 route typing is already consumable through `OperationResponse`, `OperationRequestBody`, `OperationQuery`, and `OperationPath` without a transport-specific helper.

## Files / Surfaces

- Generated artifacts refreshed/checked: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`.
- Thin consumer guard: `web/src/lib/memory-api-contract.test.ts`.
- Existing wrappers confirmed: `web/src/lib/api-contract.ts`, `web/src/systems/knowledge/types.ts`, `web/src/systems/knowledge/types.test.ts`.
- Site generator/build guard: `packages/site/scripts/generate-openapi.ts` and generated `packages/site/content/runtime/api-reference/*.mdx` outputs.

## Errors / Corrections

- Parallel `make bun-typecheck` / `make bun-test` attempts failed with Mage temp-file errors (`mage_output_file.go` missing). Rerunning each target sequentially passed without code changes.
- No generated drift remained after `make codegen`; standalone `make codegen-check` passed before and after the consumer test addition.

## Ready for Next Run

- Task 15 focused validation passed: `make codegen`, `make codegen-check`, focused web contract test, `make bun-lint`, `make bun-typecheck`, `make bun-test`, `packages/site` build, `git diff --check`, and pre-tracking `make verify`.
- Pre-tracking full `make verify` passed with Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8354 tests`, and package boundaries `OK`.
- Next task after state update should be `task_16` (HTTP and UDS Route Parity).
