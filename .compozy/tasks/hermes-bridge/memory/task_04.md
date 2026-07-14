# Task Memory: task_04

## Objective Snapshot

- Replace curl-first bridge onboarding with agent-manageable CLI/HTTP/UDS flows: Slack manifest generation, WhatsApp/Telegram/Discord setup, provider-owned verify probes, doctor aggregation, Telegram webhook registration, and a real send-test path.

## Preflight Decisions

- The workflow directory contains no PRD or user-story companion; `task_04.md`, `_techspec.md`, `_tests.md`, `_qa.md`, and the numbered analysis corpus are the available authoritative artifacts. The stale boilerplate references do not justify inventing missing requirements.
- Every public action crosses the shared API contract and BaseHandlers so HTTP and UDS stay at parity. Contract source changes co-ship generated OpenAPI, TypeScript, CLI docs, contract mocks, and affected Web adapters/fixtures; regeneration happens once after source freeze.
- Provider identity/webhook probes remain behind the adapter boundary. The daemon coordinates and aggregates typed results but never imports platform REST clients. All eight in-tree adapters answer explicitly; unsupported checks return `skipped` records.
- Verify and doctor are read-only against bridge lifecycle state. Only existing operator/report-state paths may mutate instance state.
- Setup reuses existing create-instance and vault-backed secret-binding APIs. No new storage or `config.toml` keys are introduced. Secret echoes are masked; generated verify/webhook secrets exist only in memory until bound.
- `send-test` must enter the real bridge delivery/provider path. Existing `test-delivery` remains target-resolution dry-run and retains zero platform calls.
- Slack scope/event sets live once as typed code constants consumed by manifest generation and scope-diff verification. Tests validate relations and a vendored, source/version-identified Slack schema instead of freezing a full snapshot.
- User override remains active: only focused/proportional tests during Task 04. Global suites, broad Web/site gates, and the single `make verify` run only after all workflow tasks, QA, and review remediation are complete.

## Architecture Council Decision

- Use one ephemeral, single-instance provider subprocess per control operation. A disabled instance is resolved directly with its current bound secrets without calling the service-runtime resolver or any lifecycle transition.
- The bridge runtime handshake gains a required closed purpose (`service | control`). Control carries exactly one managed instance plus an exact allowed-method list. It receives zero Host API actions/resources/handlers, is not installed into `managedExtension`, is not supervised, and is registered only transiently in the existing process registry.
- In control mode bridgesdk creates the negotiated Session/InstanceCache but does not call provider service `Initialize`, `HealthCheck`, or `Shutdown`. Therefore it cannot reconcile ingress, open listeners, call `report_state`, or mutate daemon resources.
- Use separate typed RPCs: `bridges/check` for all eight adapters and `bridges/webhook/register` for Telegram. Do not introduce a generic action enum or raw JSON control bag. Every adapter returns at least one explicit result; unsupported identity probes return `skipped`.
- A single extension-manager runner owns launch → initialize → call → bounded cooperative shutdown → reap → process-registry completion → redaction cleanup on every success/error/cancel path. Control operations are concurrency-bounded globally and serialized by bridge instance so secret binding and lifecycle changes cannot race the snapshot.
- `provider_config.webhook.public_url` is the canonical full external callback URL. It is never inferred from HTTP headers, UDS, `listen_addr`, or a global base. Common structural validation requires an absolute HTTP(S) URL with host and no userinfo/fragment; provider policy requires HTTPS outside trusted loopback/test fixtures. Reverse-proxy rewrites may make its path differ from internal `webhook.path`.
- Verify is stage-aware: provider identity and webhook configuration run before enable; webhook reachability is `skipped` with remediation while disabled and is tested only when the service listener is active. Control mode never starts an ingress listener merely to manufacture reachability evidence.
- `send-test` is not a control operation. It requires a routable active instance and invokes the existing real `bridges/deliver` provider path synchronously; `test-delivery` remains target-resolution-only with zero provider calls.
- Public check records use the closed shape `{check,status,remediation}` with `status = pass | warn | fail | skipped`. Adapter errors are classified into stable redacted records; raw upstream bodies, URLs containing tokens, headers, stderr, and secret values never cross the boundary.

## Test Placement

- Manifest relations/schema, wizard validators/JSON mode, CLI verify/send-test behavior: existing `internal/cli/bridge_test.go` and `cli_integration_test.go`.
- Doctor category filtering/aggregation: existing `internal/doctor/doctor_test.go`.
- Public route status/body and OpenAPI shape: existing API core/HTTP/UDS/spec suites; no duplicate standalone contract suite.
- Provider probe payloads and classification: each provider's canonical provider/runtime suite; Slack owns scope-diff and bot-token diagnostics.
- Daemon→provider aggregation and all-eight explicit responses: existing extension integration/conformance lanes.
- Runtime request/response mocks: existing `TestDaemonE2E...` bridge contract lane.

## AGH Impact Audit (Preflight)

- Native tools: no planned `agh__*` ID/schema/digest change; inspect CLI/API fallback documentation before completion.
- Extensibility and hooks: bridgesdk/extension provider contract gains typed check/action requests implemented by all eight adapters; no new hook event or bundle/resource surface.
- Workspace data isolation: manifest/verify/webhook/send-test resolve one bridge instance under its existing scope/workspace ownership; no global cache or new persisted datum. Prove cross-workspace instance access remains rejected through shared handlers.
- Official AGH skill: public verbs/routes are task_08's full parity owner, but Task 04 must leave accurate hand-off shapes and update any immediately false runtime-operations guidance.

## Completed Implementation

- The bridge handshake now requires closed `service | control` purpose. Control sessions expose exactly one typed provider method, one managed instance, zero Host API grants/handlers, and no service lifecycle callbacks.
- Extension Manager admission, lifecycle cancellation, global concurrency bound, per-instance serialization, process-registry ownership, redacted cleanup, cooperative shutdown, and reap are enforced for every transient control call. Manager shutdown cancels and joins admitted controls.
- Daemon control locking is context-aware and writer-preferring: lifecycle mutations remain extension-exclusive, controls for different instances can share the extension gate, the instance gate stays exclusive/cancelable, and cleanup remains inside the lock boundary.
- All eight in-tree adapters implement `bridges/check`; Telegram also implements typed `bridges/webhook/register`. Identity error classes map to stable redacted pass/warn/fail/skipped records instead of raw provider text.
- Credential-bearing upstream destinations are a greenfield hard cut from mutable `provider_config`. The eight adapters use operator-owned defaults/environment seams. Webhook probes accept only public HTTPS, reject redirects/internal literals/mixed DNS/rebinding, pin a validated IP while preserving TLS SNI, bypass proxies, and classify response status truthfully.
- Slack manifest generation derives one typed scope/event source, validates against the pinned official v0.5.0 schema fixture with provenance/SHA, and is available through CLI, HTTP, and UDS.
- WhatsApp, Telegram, and Discord setup support interactive or strict one-object JSON input, disabled-instance creation, write-only bindings, safe reruns, provider validation, and exact next steps. WhatsApp verify tokens and Telegram `--print-only` webhook secrets cannot become irretrievable: the operator supplies them or explicitly requests one-run JSON disclosure before any write. Normal Telegram registration may safely consume a hidden generated secret inside the daemon.
- Verify, doctor `CategoryBridge`, real `send-test`, dry-run `test-delivery`, and Telegram registration share typed CLI/HTTP/UDS contracts. Verify never changes lifecycle state; send-test traverses the existing real provider delivery path.
- OpenAPI, generated Web/SDK TypeScript contracts, CLI reference pages, provider READMEs, `setup.mdx`, the official AGH runtime skill, and QA scenario `NB-bridge-provider-setup` co-ship the public behavior.

## Security and Breaking-Change Audit

- Fixed the two pre-freeze P1 findings: transient control no longer bypasses Manager lifecycle/Stop, and bridge-owned endpoints can no longer redirect bound credentials or turn reachability into blind SSRF.
- Fixed the remaining P2 findings: canonical purpose validation precedes grants, lock waits honor cancellation without over-serializing different instances, provider error taxonomy is stable, reachability rejects meaningless redirects/not-found responses, and operator-needed generated secrets are never persisted before disclosure.
- Delete targets completed with no compatibility aliases: removed `api_base_url`, `oauth_token_url`, `service_url`, `openid_metadata_url`, and `token_url` from every in-tree adapter config/fixture/doc; removed implicit service-purpose initialization; replaced curl-first setup guidance. Repository scans find no blocked endpoint JSON tags.
- Cross-review found no remaining P1/P2 after remediation. Its only lint finding was formatted and the final scoped lint is clean.

## Focused Verification Evidence

- Fresh `-race` provider packages pass with coverage: Slack 81.3%, Telegram 80.5%, Discord 80.4%, Teams 80.0%, Google Chat 80.5%, WhatsApp 80.4%, GitHub 80.1%, and Linear 80.6%.
- Focused `-race` suites pass for control handshake/runtime, Manager lifecycle/cleanup, daemon locking, bridge/bridgesdk security and check classification, API contract/core/HTTP/UDS/spec, doctor, SDK generation, Slack control, CLI manifest/setup/verify/send-test, and the reference adapter.
- Exact CLI integrations pass for Slack manifest and setup → bindings → verify → real send-test; the final setup integration rerun passed in 12.7s after the generated-secret remediation.
- Canonical codegen and codegen-check pass. The TypeScript extension SDK has 49/49 tests plus typecheck green; Web generated-contract typecheck is green; site typecheck and the exact manual-route integrity test are green.
- `golangci-lint --new-from-rev=HEAD` reports no issues across the touched internal packages, Slack adapter, and reference example. Three changed Go test files pass the AGH convention checker.
- `git diff --check`, focused Oxfmt, the 433-row/16-column/unique-id QA CSV check, blocked-endpoint scan, and the new production-file cap audit all pass. Largest new production files are 456 lines (`bridge_lifecycle_locks.go`) and 441 lines (`bridge_setup.go`).
- Global suites and the single `make verify` remain intentionally deferred until all ten tasks, QA, and review remediation are complete per user direction.

## AGH Impact Audit (Completion)

- Native tools: no `agh__*` IDs, toolsets, descriptors, capability gates, or schemas changed; generated native catalog/codegen-check remains consistent. Public management is through CLI/HTTP/UDS bridge surfaces.
- Extensibility and hooks: bridge adapter handshake/SDK gains closed purpose plus typed check/register methods implemented by all eight adapters; no hook, bundle, resource, MCP sidecar, or bridge SDK fallback alias was added. Operator-owned endpoint environment seams replace mutable provider config; no `config.toml` key changes.
- Workspace data isolation: control requests resolve exactly one existing global/workspace bridge instance through the shared runtime/store ownership path; no new persisted datum or global cache exists. Instance ID, scope, workspace ID, secret snapshot, HTTP/UDS request, and provider response remain within that instance boundary.
- Official AGH skill: `skills/agh/references/runtime-operations.md` documents setup, one-run disclosure, Telegram registration, verify, and send-test behavior.

## Open Risks

- Unrelated Daytona sidecar assets remain modified and must never be staged.
- Live-provider accounts are unavailable; faithful evidence uses fake upstreams, exact subprocess/daemon integrations, public routes, and later Task 09/10 scenario QA.
- Global suites and the single `make verify` remain deferred to the workflow tail.

## Next

- Safe scoped checkpoint `d362403` contains Task 04 and excludes both Daytona sidecars. Detect the next phase and continue directly into Task 05.
