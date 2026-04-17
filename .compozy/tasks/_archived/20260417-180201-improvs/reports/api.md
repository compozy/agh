# Improvements Report — internal/api

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/api/core/perf_bench_test.go`, baseline and post-fix numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -ignore '.*_test\\.go' -top 10 -over 0 internal/api | sort -rn`:

| Complexity | Function | File |
| --- | --- | --- |
| 16 | `(*BaseHandlers).resolveMemoryLocation` | `internal/api/core/memory.go:235` |
| 16 | `validateBridgeDeliveryDefaultsPayload` | `internal/api/contract/bridges.go:429` |
| 15 | `writeSSERaw` | `internal/api/core/sse.go:68` |
| 15 | `ParseResourceFilter` | `internal/api/core/resources.go:262` |
| 14 | `applyTriggerPatch` | `internal/api/core/automation.go:892` |
| 14 | `StatusForBridgeError` | `internal/api/core/errors.go:144` |
| 14 | `(*BaseHandlers).StreamBridgeHealth` | `internal/api/core/bridges.go:171` |
| 14 | `(*BaseHandlers).CreateNetworkChannel` | `internal/api/core/network_details.go:40` |
| 13 | `(*Server).Start` | `internal/api/udsapi/server.go:430` |
| 13 | `(*Server).Shutdown` | `internal/api/udsapi/server.go:504` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/api/spec/spec.go` | 2573 | OpenAPI schema/runtime glue is monolithic and difficult to review piecemeal. |
| `internal/api/testutil/apitest.go` | 1210 | Test transport setup, fixture wiring, and stub types are co-located in one oversized helper unit. |
| `internal/api/core/tasks.go` | 1196 | Task CRUD, dependency management, run lifecycle, parsing, and validation all share one handler file. |
| `internal/api/core/automation.go` | 965 | Job CRUD, trigger CRUD, run queries, webhook ingestion, and parser helpers are tightly packed. |
| `internal/api/core/handlers.go` | 860 | Session CRUD, streaming, hooks, observe, health, and daemon metadata are mixed in one transport facade. |
| `internal/api/core/network_details.go` | 767 | Network channel create/detail/message/peer logic shares one file with rollback flow and payload shaping. |
| `internal/api/udsapi/server.go` | 646 | Socket startup, shutdown, router install, and lifecycle signaling are bundled together. |
| `internal/api/core/conversions.go` | 572 | Many unrelated payload conversions live in one central file, making hot-path changes noisy. |
| `internal/api/contract/contract.go` | 570 | Large shared contract set with many unrelated request/response payloads. |
| `internal/api/httpapi/server.go` | 536 | HTTP server lifecycle, listener binding, route wiring, and graceful shutdown share one unit. |
| `internal/api/core/bridges.go` | 513 | Bridge CRUD, streaming health, secret bindings, and delivery tests are co-located. |
| `internal/api/contract/bridges.go` | 499 | Bridge-specific request parsing and validation remains dense. |
| `internal/api/httpapi/prompt.go` | 467 | Prompt request shaping and SSE emission state machine live together. |
| `internal/api/core/resources.go` | 463 | Resource CRUD, query parsing, and resource-service adapter code are tightly coupled. |
| `internal/api/core/memory.go` | 432 | Memory CRUD, scope resolution, consolidation, and workspace derivation are bundled. |
| `internal/api/core/network.go` | 417 | Network status, peer/channel reads, send, inbox, and payload conversion sit in one file. |
| `internal/api/core/bundles.go` | 348 | Bundle catalog, activation lifecycle, preview, and network settings are mixed together. |

### Refactoring — Duplication

`dupl -plumbing -t 120 internal/api` notable findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/api/core/automation.go:187-212` | `internal/api/core/automation.go:387-412` | Production duplicate between job and trigger delete handlers. |
| `internal/api/core/tasks.go:220-261` | `internal/api/core/tasks.go:693-734` | Production duplicate between task cancel and task-run cancel flows. |
| `internal/api/core/tasks.go:517-646` | `internal/api/core/tasks.go:605-734` | Production duplicate across task-run status transition handlers. |
| `internal/api/core/tasks.go:797-825` | `internal/api/core/tasks.go:827-855` | Production duplicate in task and task-run query normalization helpers. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `WriteSSE` | `internal/api/core/sse.go:35` | Session and prompt streaming serialize every SSE event through this writer. | `BenchmarkWriteSSE` |
| `EmitObserveEvents` | `internal/api/core/sse.go:112` | Observe streams batch and serialize event summaries repeatedly during polling and SSE replay. | `BenchmarkEmitObserveEvents` |
| `SessionPayloadsFromInfos` | `internal/api/core/conversions.go:71` | Session listing and status responses repeatedly materialize payload slices from runtime session info. | `BenchmarkSessionPayloadsFromInfos` |
| `AgentEventPayloadFromEvent` | `internal/api/core/conversions.go:152` | HTTP/UDS prompt and session streaming convert every agent event, including raw payload copying. | `BenchmarkAgentEventPayloadFromEvent` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/api/...` before the conversion-path fix, followed by the same benchmark command after the fix:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkWriteSSE` | 397.7 | 304 | 395.5 | 304 | deferred — flat benchmark, no scoped improvement justified in this pass |
| `BenchmarkEmitObserveEvents` | 32850.6 | 25139 | 33131.4 | 25140 | deferred — flat benchmark, write amplification is unchanged by the conversion fix |
| `BenchmarkSessionPayloadsFromInfos` | 12037.2 | 57344 | 12075.8 | 57344 | deferred — flat benchmark, allocation profile unchanged |
| `BenchmarkAgentEventPayloadFromEvent` | 215.2 | 256 | 203.3 | 192 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/api/httpapi/server.go:460` | `(*Server).Start` | `serveDone` channel is closed after `Serve` returns; shutdown path cancels through `Shutdown`. | One goroutine owns the HTTP server accept loop. |
| `internal/api/udsapi/server.go:488` | `(*Server).Start` | `serveDone` channel is closed after `Serve` returns; shutdown path cancels through `Shutdown`. | One goroutine owns the UDS accept loop. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/api/httpapi/server.go:435` (`serveDone`) | unbuffered | HTTP server lifecycle | serve goroutine in `Start` | `Start`, `Shutdown` | Bounded server completion signal. |
| `internal/api/udsapi/server.go:469` (`serveDone`) | unbuffered | UDS server lifecycle | serve goroutine in `Start` | `Start`, `Shutdown` | Bounded socket server completion signal. |
| `internal/api/core/handlers.go:132` (`cfg.StreamDone`) | unbuffered | transport wiring | caller-supplied or local fallback | session/observe/prompt streams | Installed by transports to terminate long-lived streams. |
| `internal/api/core/handlers.go:180` (`done`) | unbuffered | `StreamDoneChannel` fallback | local setup path | `StreamDoneChannel` callers | Safe closed-over fallback only when transports do not install one. |
| `internal/api/core/handlers.go:57` / `91` (`StreamDone` / `streamDone`) | read-only external channel | transport/server lifecycle | transport owner | stream handlers | Read-only shutdown bridge injected into `BaseHandlers`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/api/httpapi/server.go:34` (`mu`) | write-heavy | HTTP server listener state and lifecycle fields | Serializes startup/shutdown transitions. |
| `internal/api/udsapi/server.go:51` (`mu`) | write-heavy | UDS server listener state and lifecycle fields | Serializes startup/shutdown transitions. |
| `internal/api/core/handlers.go:91` (`settingsMu`) | read-heavy | `BaseHandlers` stream-done bridge and HTTP port settings | Guards concurrent transport configuration reads/writes. |

### Concurrency — Select Audit

- `internal/api/httpapi/prompt.go:111` includes `c.Request.Context().Done()`, `h.StreamDoneChannel()`, and the prompt event channel.
- `internal/api/udsapi/prompt.go:41` includes `c.Request.Context().Done()`, `h.StreamDoneChannel()`, and the prompt event channel.
- `internal/api/core/session_stream.go:79` includes `ctx.Done()` while polling/stitching stream batches.
- `internal/api/core/handlers.go:626` includes `c.Request.Context().Done()` and `h.StreamDoneChannel()` during observe SSE polling.
- `internal/api/core/bridges.go:201` includes `c.Request.Context().Done()` and `h.StreamDoneChannel()` during bridge-health SSE polling.
- `internal/api/httpapi/server.go:530` and `internal/api/udsapi/server.go:594` are bounded waits on `serveDone` vs `ctx.Done()` in shutdown paths.
- No blocking production `select` in `internal/api/` was found without either context cancellation, transport shutdown bridging, or a bounded completion channel.

### Security — Threat Model

- Trust boundaries:
  - Local HTTP and UDS clients send JSON bodies, query params, path params, and headers into `internal/api` transport handlers.
  - `internal/api/core` validates and normalizes request data before delegating to session, workspace, automation, bridge, memory, network, task, bundle, and extension services.
  - SSE endpoints expose continuously streamed event data back to local clients and therefore must obey caller cancellation and transport shutdown.
- Attacker capabilities:
  - A local client can send malformed JSON, oversized webhook bodies, path/query/header values, replayed SSE cursors, and arbitrary prompt/session/task payloads.
  - A remote webhook sender can control the request body and webhook headers for automation endpoints, but not the daemon-side secret material used by validation downstream.
- In-scope assets:
  - Session lifecycle operations and prompt streams.
  - Workspace and memory filesystem boundaries.
  - Task and automation mutation endpoints.
  - Bridge secret binding and delivery-test inputs.
  - Network channel creation and outbound message routing.
- Out-of-scope:
  - Trusted operator configuration of daemon services and workspace roots.
  - Internal correctness of downstream manager/service packages outside `internal/api/`.
  - Code generation quality in `internal/api/spec` where the package only exposes schema artifacts.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/api/core/handlers.go:196,217,255,266,276,287,315,350,361,401,465,494,521,547,564,586,649,676` | Session, agent, hook, observe, and daemon endpoints accept query params, JSON bodies, path IDs, and `Last-Event-ID`. | `ShouldBindJSON`, `validateCreateSessionRequest`, `ParseSessionEventQuery`, `ParseHookCatalogFilter`, `ParseHookRunsQuery`, `ParseHookEventFilter`, `ParseObserveEventQuery`, `parseLastEventID`, and `ParseObserveCursor` all fail closed on malformed input before delegating. | Session manager, agent catalog/loader, hook store, observer, and health/status payload builders. | LOW — transport layer validates and normalizes user data before handing it to daemon-owned services. |
| `internal/api/core/workspaces.go:16,57,73,95,155,171` | Workspace create/update/resolve plus path-ID reads and deletes. | `ShouldBindJSON`, workspace service resolution, and downstream workspace validation enforce IDs, roots, and agent references. | Workspace service CRUD and resolution calls. | LOW — daemon-owned workspace service remains the authority for path/root validation. |
| `internal/api/core/resources.go:136,159,182,225,262` | Resource list/get/put/delete accept query filters, path params, and JSON payloads. | `ParseResourceFilter`, `ShouldBindJSON`, and request-to-resource conversion validate kinds, scopes, owners, sources, and IDs before service calls. | Resource service list/get/put/delete. | LOW — invalid filter combinations fail with validation errors before reaching storage. |
| `internal/api/core/skills.go:15,51,77,109,146` | Skill endpoints accept workspace query filters and skill-name path params. | `strings.TrimSpace`, workspace lookup, and skill manager/service validation bound names and workspace scoping. | Skill catalog/content manager and enable/disable operations. | LOW — inputs are normalized string identifiers only. |
| `internal/api/core/bridges.go:22,55,76,108,124,156,161,166,171,230,246,262,292,308` | Bridge CRUD, instance control, SSE health, secret bindings, and delivery tests accept JSON bodies, path IDs, binding names, and stream cursors. | `ShouldBindJSON`, request converters (`ToCreateInstanceRequest`, `ToUpdateInstanceRequest`, `ToBridgeSecretBinding`, `ToResolveDeliveryTargetRequest`), trimmed IDs, and SSE shutdown/cursor parsing constrain input before bridge service calls. | Bridge manager/provider/catalog methods and bridge-health streams. | LOW — secrets stay in bridge service storage; transport layer rejects malformed identifiers and payloads. |
| `internal/api/core/memory.go:26,37,60,92,114,227,358,363,400,417` | Memory list/read/write/delete/consolidate accept scope/workspace query, filename path params, and JSON bodies. | `resolveMemoryLocation`, `resolveMemoryWriteScope`, `ParseOptionalMemoryScope`, and `resolveMemoryWorkspace` validate scope/workspace binding; downstream memory store rejects path separators in filenames. | Memory store read/write/delete/consolidate and health-workspace resolution. | LOW — filename traversal is blocked and scope/workspace mismatches fail closed. |
| `internal/api/core/automation.go:36,64,103,125,187,215,230,259,281,315,330,387,415,444,465,489,665` | Automation job/trigger/run CRUD plus webhook delivery accept query params, JSON bodies, path IDs, webhook path params, headers, and request bodies. | `ParseAutomationJobListQuery`, `ParseAutomationTriggerListQuery`, `ParseAutomationRunQuery`, `ShouldBindJSON`, request validation, `automationpkg.ValidateScopeBinding`, `automationpkg.ParseWebhookEndpoint`, required signature/timestamp/delivery headers, and `http.MaxBytesReader` on webhook bodies. | Automation manager CRUD/run methods and `HandleWebhook`. | LOW — webhook entrypoint has the strictest validation path and fails closed before manager dispatch. |
| `internal/api/core/tasks.go:77,105,143,171,220,264,308,357,395,429,473,517,561,605,649,693,737,781` | Task and task-run endpoints accept query filters, JSON bodies, and path IDs for task/run lifecycle mutations. | `ShouldBindJSON`, `requiredPathID`, normalized `Scope/Status/OwnerKind`, optional limit parsers, and workspace/network filter validation reject malformed input before reaching task services. | Task service CRUD, dependency mutation, enqueue/claim/start/attach/complete/fail/cancel calls. | LOW — handlers are thin validators over downstream task ownership and state-machine checks. |
| `internal/api/core/network.go:30,40,108,124,156` | Network status/peers/channels/send/inbox endpoints accept query filters and JSON send requests. | `networkServiceRequired`, `ShouldBindJSON`, `NetworkSendRequestFromPayload`, and session/channel normalization guard outbound calls. | Network service status/list/send/inbox methods. | LOW — invalid channel/session/message payloads fail with `StatusForNetworkError`. |
| `internal/api/core/network_details.go:40,114,141,218` | Network channel create/detail/messages/peer endpoints accept JSON bodies, path channel IDs, peer IDs, and message limits. | `ShouldBindJSON`, `resolveCreateNetworkChannelRequest`, `normalizeNetworkChannel`, `ParseOptionalInt`, and trimmed peer IDs validate request shape before service access. | Session creation rollback flow and network detail/message/peer service lookups. | LOW — transport layer normalizes all identifiers before downstream use. |
| `internal/api/core/bundles.go:18,31,45,62,76,89,111,123` | Bundle catalog, preview, activation CRUD, and network-settings endpoints accept path IDs and JSON activation requests. | `ShouldBindJSON`, `strings.TrimSpace`, and downstream bundle service validation constrain IDs and activation options. | Bundle service preview/get/activate/update/deactivate/network-settings. | LOW — no direct filesystem or process sink in transport code. |
| `internal/api/httpapi/prompt.go:76` | HTTP prompt streaming accepts JSON bodies and session path IDs. | `ShouldBindJSON`, `extractPromptMessage` filters message parts down to user text, and stream loop obeys caller/shutdown cancellation. | `Sessions.Prompt` event stream and SSE writer. | LOW — prompt text is an intended session input, not executed by transport code. |
| `internal/api/httpapi/sessions.go:13` | HTTP approval endpoint accepts JSON approval decisions and session path IDs. | `ShouldBindJSON` plus `acp.ApproveRequest.Validate` reject malformed decisions before permission approval. | `Sessions.ApprovePermission`. | LOW — handler only forwards validated approval state. |
| `internal/api/httpapi/middleware.go:36,61` | Browser-originating requests control the `Origin` header. | `resolveAllowedOrigin` canonicalizes scheme/host/port and only permits same-origin, loopback-equivalent, or exact bound-host matches. | `Access-Control-Allow-Origin` response header or `403`. | LOW — cross-origin requests fail closed instead of falling back to wildcard CORS. |
| `internal/api/udsapi/prompt.go:17` | UDS prompt streaming accepts JSON bodies and session path IDs. | `ShouldBindJSON`, non-empty message check, and stream select on caller/shutdown completion. | `Sessions.Prompt` event stream and SSE writer. | LOW — same transport-only prompt forwarding model as HTTP. |
| `internal/api/udsapi/extensions.go:15,34,69,73,77,102` | UDS extension install/enable/disable/status endpoints accept JSON path/checksum payloads and extension-name path params. | `ShouldBindJSON`, trimmed path/checksum/name, and extension-service status code mapping reject malformed install or toggle requests. | Extension service list/install/status/enable/disable methods. | LOW — extension validation and checksum enforcement remain in the service layer. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/api/core/conversions.go:152` | `AgentEventPayloadFromEvent` converted `event.Raw` through a string path, forcing avoidable trim/validate/copy work on every streamed event payload. | fixed |
| 02 | refactoring-analysis | medium | `internal/api/core/tasks.go:517` | Task-run lifecycle handlers remain duplication-heavy, with near-identical status transition blocks across start/complete/fail/cancel flows. | deferred |
| 03 | refactoring-analysis | low | `internal/api/core/automation.go:187` | Job and trigger delete handlers are duplicated, but the shared extractable logic is small compared with the risk of broad churn in this pass. | wontfix |
| 04 | extreme-software-optimization | low | `internal/api/core/sse.go:35` | `WriteSSE`, `EmitObserveEvents`, and `SessionPayloadsFromInfos` benchmark as active paths but did not yet justify a scoped optimization beyond the conversion fix. | deferred |

## Per-Skill Notes

### refactoring-analysis

- The highest production complexity is concentrated in request parsing and transport orchestration rather than one obviously broken helper.
- `internal/api/core/tasks.go` is the strongest follow-up candidate because it combines large size, duplication, and state-transition branching in one file.
- `internal/api/core/automation.go` duplication is real, but the duplicated delete handlers are short enough that extracting a helper in this pass would be stylistic churn more than risk reduction.

### extreme-software-optimization

- Added `BenchmarkAgentEventPayloadFromEvent` so every hot-path candidate identified in this pass now has a benchmark in `internal/api/core/perf_bench_test.go`.
- Fixed the event conversion path by introducing `payloadJSONBytes`, which preserves valid JSON payload passthrough without bouncing `[]byte` through `string` first.
- The fixed benchmark improved from `215.2 ns/op, 256 B/op` to `203.3 ns/op, 192 B/op`, and allocs dropped from `3 allocs/op` to `2 allocs/op`.
- `WriteSSE`, `EmitObserveEvents`, and `SessionPayloadsFromInfos` were rerun after the fix and stayed effectively flat, so they remain benchmarked-but-deferred rather than speculative optimization targets.

### ubs

- `not-run` due to missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer

- No deadlock or goroutine-leak finding was confirmed after auditing the server lifecycle goroutines, transport shutdown bridge channels, and stream `select` sites.
- The runtime goroutines in `internal/api/` are few, owned by transport/server lifecycle, and each has an explicit shutdown signal.

### security-review

- No high-confidence security finding was confirmed inside `internal/api/` after enumerating the request entrypoints and validation boundaries.
- The highest-risk surface is webhook delivery, and that path already validates scope binding, endpoint format, required headers, and request-body size before delegating to the automation manager.
- Memory filename traversal remains blocked downstream because the memory store rejects path separators in filenames.

## Deferred Items (carry forward)

- **02** — Split task-run lifecycle mutation handlers into shared validation/transition helpers if a future task can absorb a broader refactor inside `internal/api/core/tasks.go`.
- **04** — Revisit `WriteSSE`, `EmitObserveEvents`, and `SessionPayloadsFromInfos` only if future profiling shows they dominate end-to-end request latency rather than just package-local microbenchmarks.

## `make verify`

Fresh verification command: `make verify`

```text
0 issues.
✓  internal/api/contract (1.075s)
✓  internal/api/core (1.427s)
✓  internal/api/udsapi (1.698s)
✓  internal/api/httpapi (1.79s)
✓  internal/daemon (7.862s)
✓  internal/cli (8.237s)

DONE 4426 tests in 9.996s
OK: all package boundaries respected
```
