# Task Memory: task_07

## Objective Snapshot

- Hoist the duplicated eight-provider runtime scaffold into small bridgesdk owners, delete provider-local marker/main copies, and add auto-discovered provider plus docs-contract conformance without changing observable provider behavior.

## Preflight Baseline

- Current aggregate `extensions/bridges/*/provider.go` size is 17,688 lines; the required 40% reduction target is at most 10,612 lines.
- Eight `markers.go` files differ only by the provider prefix in side-effect error text; eight `main.go` files differ only by provider name and constructor.
- `internal/bridgesdk.Runtime` already owns JSON-RPC initialize/deliver/check/webhook/health/shutdown dispatch. The missing duplicated layer is provider lifecycle orchestration, ownership/state Host API calls, typed route/config storage, delivery state, marker instrumentation, and command entry wiring.
- The current integration conformance matrix is manually enumerated and behavior-scenario-oriented; it does not discover provider directories or validate every manifest/schema/runtime.
- The docs drift suite does not yet exist. Task 07 owns invariant harnesses; Task 08 owns the complete provider docs that make the setup-section/table assertions green before the workflow leaves Phase B.
- User override: run only focused package/subtest/integration/boundary lanes during Task 07. Do not run global test targets or `make verify`; reserve them for the workflow tail after all tasks and review remediation.

## Design Constraints

- Composition, not a generic god base. Split lifecycle, Host API ownership/state operations, route/config storage, delivery state, markers, and entry command by responsibility; every new production file stays below 500 lines.
- bridgesdk may import `internal/bridges` contract types, extension contract DTOs already used by the SDK, subprocess contracts, and leaf utilities. It must not import provider packages, daemon, store, or extension runtime implementation.
- Providers retain platform-specific config decoding/validation, signature verification, inbound mapping/webhook handling, platform clients, delivery/progress execution, and formatters.
- Existing behavior assertions are regression oracles. Provider test edits may update construction/wiring/imports only; do not weaken expected states, payloads, errors, or side effects.
- A local Go command package still requires `func main`; replace copied main bodies with minimal provider-specific entrypoints calling one shared bridgesdk command runner. Do not retain copied command parsing/error logic.
- Marker support must be a real shared implementation, not provider-local aliases/re-exports.

## Test Placement

- Invariant: lifecycle initialize performs ownership list/get with not-initialized retry, reports the reconciled states, and shutdown joins owned work. Owning layer: bridgesdk provider lifecycle. Canonical suite: new `internal/bridgesdk/provider_lifecycle_test.go`.
- Invariant: add/remove/path-conflict reconciliation atomically swaps the route map and cleans retired values. Owning layer: bridgesdk typed route store/reconciler. Canonical suite: new runtime/reconcile owner under `internal/bridgesdk`; port strongest existing provider cases rather than duplicate them.
- Invariant: every provider directory with `extension.toml` is discovered and its manifest, slots/schema, binary, five runtime methods, initialize, and health contracts are truthful. Owning layer: extension integration conformance. Canonical suite: `internal/extension/provider_conformance_matrix_integration_test.go`.
- Invariant: provider docs and code evolve together. Owning layer: public bridge docs contract. Canonical suite: `internal/extension/bridge_docs_conformance_test.go`; relations only, no snapshots or prose-literal pinning beyond table/heading contract structure.
- Invariant: all eight provider behavior suites retain their existing assertions after wiring migration. Owning layer: existing provider suites; no new duplicated regression files.

## Resolved Architecture

- `ProviderLifecycle`, `ProviderHost`, `ManagedConfigReconciler`, `RouteTable`, `DeliveryStateStore`, `ProviderHTTPServer`, `AdapterMarkers`, and `RunProviderCommand` are separate composition owners; every new production file is below 500 lines.
- All eight providers invoke `ManagedConfigReconciler` for resolve → provider-specific prepare → first atomic route publication → provider-specific final probe → final atomic publication. GitHub/Linear admit routes through `OnPublish` before slow probes; platform hooks retain only conflicts, listener policy, probes, and resource cleanup.
- The service conformance contract derives implemented methods from the canonical initialize response: deliver, target snapshot, health, and shutdown. Initialize is the handshake, while check/webhook registration are control-runtime methods rather than unconditional service methods.
- Shared markers preserve the existing environment names and JSON shapes. File open/write/close failures remain observable through `AdapterMarkers.Error` and provider stderr reporting.
- Runtime service calls are rejected after shutdown admission. This fixed the conformance deadlock where health checks remained accepted while shutdown waited for a blocked Host call.

## Completion Evidence

- Atomic Task 07 checkpoint: `749498228ed54e9861e24f8bcdbc7ea7cdecb283`.
- Aggregate provider size: 17,688 → 10,521 lines (40.5% reduction); `find extensions/bridges -name markers.go` returns zero; eight providers reference `ManagedConfigReconciler`.
- Fresh focused race coverage: bridgesdk 80.0%; Slack 80.1%; Telegram 80.6%; Discord 82.2%; WhatsApp 81.1%; Teams 80.1%; Google Chat 80.1%; GitHub 80.0%; Linear 80.0%.
- Auto-discovered provider conformance passes 11/11 under `-race`; focused daemon bridge E2E passes 4/4; scoped golangci-lint reports zero issues; `make boundaries` passes.
- Docs invariant harness is intentionally red-first as permitted by Task 07: Slack scopes pass, while Task 08 must add slots rows for gchat/github/linear/teams/whatsapp and setup coverage for gchat/github/linear/teams.
- QA impact reset NB-029 and NB-031 to `untested` after initialization error-ordering and authoritative HTTP shutdown fixes; historical evidence remains intact.

## Working Set

- `internal/bridgesdk/`
- `extensions/bridges/{slack,telegram,discord,whatsapp,teams,gchat,github,linear}/`
- `internal/extension/provider_conformance_matrix_integration_test.go`
- `internal/extension/bridge_docs_conformance_test.go`
- `packages/site/content/runtime/core/bridges/`
- `.compozy/tasks/hermes-bridge/{task_07.md,_techspec.md,_tests.md,memory/MEMORY.md,state.yaml}`
