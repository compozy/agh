# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Define the canonical public Memory v2 contract surface before transport-specific route implementation.
- Task references checked: `_techspec.md` `Public Interfaces / Types`, `API Endpoints`, `Agent Manageability Plan`, `Config Lifecycle`, and ADR-001 through ADR-012.

## Important Decisions

- Memory v2 public DTOs live in `internal/api/contract/memory.go` and cover CRUD, scope/tier selectors, decisions/revert, recall trace, dreaming, daily logs, extractor, providers, ad-hoc notes, config metadata, and session ledger/replay/prune/repair shapes.
- Public Memory v2 payloads use `workspace_id` and `agent_name`; old path-style `workspace` selectors are not part of the public v2 DTOs.
- Public decision and LLM trace payloads are redaction-safe by construction: replay bytes (`post_content`, `prior_content`) and raw LLM responses are intentionally absent from contract DTOs and OpenAPI schemas.
- The OpenAPI source now exposes the v2 contract route surface while `packages/site/scripts/generate-openapi.ts` filters public reference docs to only routes implemented by HTTP/UDS routers until task 16 wires transport parity.
- Web Knowledge consumers were refreshed against the generated v2 contract now because the task 14 OpenAPI/codegen change otherwise leaves TypeScript consumers stale.

## Learnings

- Contract-only route expansion can still force web and site generated consumers to move in the same task; leaving them stale fails `make bun-typecheck` and `make bun-test`.
- Generated site API reference must stay truthful to registered runtime routes even when OpenAPI is ahead of implementation for the next transport task.
- Spec tests should assert both required v2 fields and hard-cut legacy operations so old verbs cannot quietly re-enter the public contract.

## Files / Surfaces

- Contract/spec: `internal/api/contract/memory.go`, `internal/api/contract/contract_test.go`, `internal/api/spec/spec.go`, `internal/api/spec/spec_test.go`.
- Generated consumers: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`.
- Web generated consumers/mocks: `web/src/systems/knowledge/**`, `web/src/hooks/routes/use-knowledge-page.*`, `web/src/hooks/routes/use-settings-memory-page.*`.
- Site generated reference filtering: `packages/site/scripts/generate-openapi.ts` and regenerated `packages/site/content/runtime/api-reference/*.mdx` as needed by the generator.

## Errors / Corrections

- Initial `make bun-typecheck` exposed stale web Knowledge types and mocks after OpenAPI route/shape changes; fixed by moving Knowledge adapters/hooks/tests to the v2 response/request shapes.
- `make bun-test` exposed generated site docs for unimplemented task 16 routes; fixed by filtering API reference docs to registered HTTP/UDS routes while keeping OpenAPI/codegen complete.
- `make lint` exposed one long spec line; formatting/refactor brought `internal/api/spec/spec.go` back under the zero-warning lint gate.
- A standalone `make codegen-check` rerun was needed after an earlier parallel make invocation hit the known Mage temp-file conflict.

## Ready for Next Run

- Task 14 focused validation passed: contract/spec tests, race tests, coverage for `internal/api/spec` and `internal/api/contract`, `make codegen`, `make codegen-check`, `make bun-lint`, `make bun-typecheck`, `make bun-test`, `make lint`, and `git diff --check`.
- Full pre-tracking `make verify` passed after implementation with Bun tests 329 files / 2087 tests, Go lint `0 issues`, Go tests `DONE 8354 tests`, and package boundaries `OK`.
- Next task after state update should be `task_15` (Codegen and Generated Consumer Refresh).
