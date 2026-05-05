# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Expose session `provider` on every explicit create/read surface for task_04: shared contracts, HTTP/UDS handlers, CLI, extension Host API, and generated artifacts.
- Finish with transport/codegen coverage plus clean `make codegen-check` and `make verify`.

## Important Decisions
- Reused `session.Info.Provider` from tasks 02/03 as the single source of truth; task_04 only projects it outward.
- Kept scope on explicit session surfaces. No new endpoints or side channels, and no workspace provider-catalog expansion beyond the generated session shapes already consumed by typed clients.
- Treated generated web type fallout as in-scope verification work because `SessionPayload.provider` became required and downstream typed fixtures had to match the contract.

## Learnings
- HTTP, UDS, CLI, Host API, OpenAPI, web generated types, and SDK generated contracts all derive from the shared contract layer, so adding `SessionPayload.provider` propagates widely and quickly exposes stale typed fixtures.
- Transport parity tests cannot hardcode a provider in the integration harness; the test runtime provider should be derived from the registered mock agent/provider pairing.
- CLI human output and TOON output both need provider visibility to keep explicit surfaces aligned.

## Files / Surfaces
- `internal/api/contract/{contract.go,contract_test.go}`
- `internal/api/core/{conversions.go,conversions_parsers_test.go,coverage_helpers_test.go,handlers.go,handlers_test.go}`
- `internal/api/{httpapi/transport_parity_integration_test.go,udsapi/transport_parity_integration_test.go,spec/spec_test.go}`
- `internal/cli/{session.go,session_test.go,cli_integration_test.go}`
- `internal/extension/{contract/host_api.go,host_api.go,host_api_test.go,host_api_integration_test.go}`
- `openapi/agh.json`
- `sdk/typescript/src/generated/contracts.ts`
- `web/src/generated/agh-openapi.d.ts`
- Typed web fixtures/tests updated for required `provider` in:
- `web/src/components/app-sidebar.test.tsx`
- `web/src/hooks/routes/use-home-page.test.tsx`
- `web/src/routes/_app/-network.test.tsx`
- `web/src/systems/network/mocks/fixtures.ts`
- `web/src/systems/session/components/{chat-header.test.tsx,session-sidebar-item.test.tsx}`
- `web/src/systems/session/hooks/{use-session-actions.test.tsx,use-sessions.test.tsx}`
- `web/src/systems/session/mocks/fixtures.ts`
- `web/src/systems/workspace/mocks/fixtures.ts`

## Errors / Corrections
- Initial HTTP/UDS transport parity tests assumed provider `fake`; the harness actually resolved the automation agent through its registered provider. Fixed by deriving the provider via `runtimeHarness.MockAgentRegistration(...)`.
- A fresh `make verify` exposed one CLI `lll` lint violation from the new provider column header; fixed by expanding the slice literal over multiple lines.

## Ready for Next Run
- Verification closed cleanly with `make codegen-check` and `make verify`.
- Remaining closeout after this memory update is task tracking plus the local commit for task_04.
