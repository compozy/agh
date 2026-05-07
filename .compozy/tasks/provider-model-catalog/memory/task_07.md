# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose the daemon-owned provider model catalog through shared contract payloads, native HTTP/UDS routes, an HTTP-only OpenAI-compatible `/api/openai/v1/models` route, and `agh provider models` CLI commands.
- Success requires contract/codegen updates, route/CLI/parity tests, deterministic validation/service-unavailable behavior, task tracking updates, `make verify`, and one local commit after clean verification.

## Important Decisions
- Use the `core.ModelCatalogService` injected by Task 05; do not compose or locate model catalog services in API or CLI layers.
- Keep Task 07 focused on native surfaces and generated API contracts. Site CLI/docs regeneration is called out for Task 10 unless verification forces a same-task update.
- Native HTTP/UDS model catalog routes are registered with a `/api/providers/*catalog_path` Gin dispatch route to avoid static/wildcard sibling conflicts while preserving the exact public paths in OpenAPI and request handling.
- OpenAI-compatible middleware errors are shaped through `core.RespondOpenAIError` for the existing HTTP API loopback/CORS denial paths.
- Because the native provider routes use a Gin catch-all dispatch route, the site OpenAPI reference generator and manual-route coverage test must treat `*` path segments as wildcards when matching generated/documented exact paths against registered routes.

## Learnings
- Pre-change scan found model catalog service wiring and dependency injection, but no public `/api/providers/models`, `/api/openai/v1/models`, or `agh provider models` surface.
- Shared workflow memory confirms refresh lifetime is daemon-owned and callers should trigger refresh through the injected service wrapper.
- Focused `go test ./internal/modelcatalog ./internal/api/contract ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/cli` passed after adding route coverage updates and model catalog tests.
- `make verify` initially failed in the site `bun-test` lane because new OpenAPI tags `openai`/`providers` were not fully mapped into API reference navigation; it then exposed the same catch-all matching gap in the manual API route test. Focused site tests pass after updating navigation and both catch-all matchers.
- `make lint` initially flagged only local new-code issues: a long OpenAPI operation builder, repeated route/output strings, redundant lambdas, and one long line. The fixes are structural/local and `make lint` passes with 0 issues.
- `make verify` later exposed one boundary violation from putting HTTP/UDS parity coverage inside `internal/api/httpapi` with a `udsapi` import. The parity test now lives under existing `internal/api/testutil`, uses a short `/tmp` AGH home for UDS socket length, and can import both transports without adding a production dependency between them.
- Full pre-commit `make verify` passed after site navigation, lint, and boundary corrections.
- Task tracking updated: `task_07.md` status/checklists and `_tasks.md` row are marked completed.
- Local implementation commit created: `742c4b58 feat: expose provider model catalog surfaces`.
- Post-commit `make verify` passed with exit code 0.

## Files / Surfaces
- Planned backend surfaces: `internal/api/contract`, `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, `internal/api/spec`, `internal/cli`, `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`.
- Touched implementation surfaces: `internal/modelcatalog/errors.go`, `internal/modelcatalog/service.go`, model catalog API/CLI files, route files, spec registry, `internal/api/testutil` parity tests, and focused tests.
- Gate-forced generated docs support: `packages/site/lib/runtime-navigation.ts`, `packages/site/scripts/generate-openapi.ts`, `packages/site/lib/__tests__/runtime-manual-api-routes.test.ts`, and `packages/site/content/runtime/api-reference/meta.json`.

## Errors / Corrections
- Initial grounding commands were run without the required `rtk` prefix before this memory update. All subsequent shell commands must use `rtk`.

## Ready for Next Run
- Task 07 implementation is complete in local commit `742c4b58`; pre-commit and post-commit `make verify` both passed.
