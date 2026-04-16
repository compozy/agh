# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Expose canonical desired-state resource CRUD on the daemon control plane with shared DTOs/handlers, UDS-first route wiring, and HTTP mutation routes disabled unless explicit operator auth middleware is present.
- Finish with unit/integration coverage, clean `make verify`, and task tracking updates.

## Important Decisions

- Added the generic operator-facing CRUD seam in `internal/api/core` as a shared `ResourceService` over the raw resource kernel, with optional codec validation/canonicalization on writes.
- Mounted `/api/resources` list/get/put/delete on UDS unconditionally and kept HTTP resource routes entirely unregistered when resource-operator auth middleware is absent.
- Composed the daemon resource service opportunistically from the registry SQL handle so lightweight daemon registries still boot and hand transports a `nil` resource service instead of failing startup.

## Learnings

- The core resource handlers already support `service unavailable`, so daemon/runtime tests do not need a SQL-backed registry just to boot API servers.
- Adding the new resource operations to the shared API spec requires regenerating `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` before the full gate.
- Runtime family endpoints such as hook runs, automation runs, and bridge delivery stay out of the generic resource CRUD surface; transport tests catch accidental route bleed.

## Files / Surfaces

- `internal/api/contract/{resources.go,responses.go}`
- `internal/api/core/{errors.go,handlers.go,interfaces.go,resources.go,resources_test.go,more_coverage_test.go}`
- `internal/api/httpapi/{handlers.go,helpers_test.go,httpapi_integration_test.go,routes.go,server.go,resources_test.go}`
- `internal/api/udsapi/{handlers_test.go,helpers_test.go,routes.go,server.go,udsapi_integration_test.go,resources_test.go}`
- `internal/api/spec/{spec.go,resources_test.go}`
- `internal/api/testutil/apitest.go`
- `internal/daemon/{boot.go,daemon.go,daemon_test.go}`
- `internal/resources/{codec.go,errors.go}`
- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

## Errors / Corrections

- Initial daemon boot wiring treated the resource service as mandatory and broke tests that use registries without a SQL handle (`daemon: registry does not expose the resource database`).
- Corrected by making resource-service composition optional for registries without `DB() *sql.DB` and by adding a daemon boot regression test that asserts transports still start with `deps.Resources == nil`.

## Ready for Next Run

- Task 06 is complete.
- Local code-only commit: `c12935f` (`feat: expose uds resource crud`).
- Verification evidence: `go test ./internal/daemon` passed during regression triage, and `make verify` passed again after the commit.
- Task tracking and workflow memory updates were kept out of the code commit intentionally, per the workspace rule for tracking-only files.
