# Refacs Loop State

> Updated: 2026-05-07
> Goal: Analyze and implement necessary refactoring/performance improvements across all Go packages under `internal/`, one package per loop iteration.

## Completed Packages

| Iteration | Package | Report | Status |
| --- | --- | --- | --- |
| 001 | `github.com/pedronauck/agh/internal/acp` | `001_report_internal_acp.md` | completed |
| 002 | `github.com/pedronauck/agh/internal/agentidentity` | `002_report_internal_agentidentity.md` | completed |
| 003 | `github.com/pedronauck/agh/internal/api/contract` | `003_report_internal_api_contract.md` | completed |
| 004 | `github.com/pedronauck/agh/internal/api/core` | `004_report_internal_api_core.md` | completed |
| 005 | `github.com/pedronauck/agh/internal/api/httpapi` | `005_report_internal_api_httpapi.md` | completed |
| 006 | `github.com/pedronauck/agh/internal/api/spec` | `006_report_internal_api_spec.md` | completed |
| 007 | `github.com/pedronauck/agh/internal/api/testutil` | `007_report_internal_api_testutil.md` | completed |
| 008 | `github.com/pedronauck/agh/internal/api/udsapi` | `008_report_internal_api_udsapi.md` | completed |
| 009 | `github.com/pedronauck/agh/internal/automation` | `009_report_internal_automation.md` | completed |
| 010 | `github.com/pedronauck/agh/internal/automation/model` | `010_report_internal_automation_model.md` | completed |
| 011 | `github.com/pedronauck/agh/internal/bridges` | `011_report_internal_bridges.md` | completed |
| 012 | `github.com/pedronauck/agh/internal/bridgesdk` | `012_report_internal_bridgesdk.md` | completed |
| 013 | `github.com/pedronauck/agh/internal/bundles` | `013_report_internal_bundles.md` | completed |
| 014 | `github.com/pedronauck/agh/internal/bundles/model` | `014_report_internal_bundles_model.md` | completed |
| 015 | `github.com/pedronauck/agh/internal/cli` | `015_report_internal_cli.md` | completed |
| 016 | `github.com/pedronauck/agh/internal/cli/docpost` | `016_report_internal_cli_docpost.md` | completed |
| 017 | `github.com/pedronauck/agh/internal/codegen/openapits` | `017_report_internal_codegen_openapits.md` | completed |
| 018 | `github.com/pedronauck/agh/internal/codegen/sdkts` | `018_report_internal_codegen_sdkts.md` | completed |
| 019 | `github.com/pedronauck/agh/internal/config` | `019_report_internal_config.md` | completed |
| 020 | `github.com/pedronauck/agh/internal/coordinator` | `020_report_internal_coordinator.md` | completed |
| 021 | `github.com/pedronauck/agh/internal/daemon` | `021_report_internal_daemon.md` | completed |
| 022 | `github.com/pedronauck/agh/internal/diagnostics` | `022_report_internal_diagnostics.md` | completed |
| 023 | `github.com/pedronauck/agh/internal/e2elane` | `023_report_internal_e2elane.md` | completed |
| 024 | `github.com/pedronauck/agh/internal/extension` | `024_report_internal_extension.md` | completed |
| 025 | `github.com/pedronauck/agh/internal/extension/contract` | `025_report_internal_extension_contract.md` | completed |

## Current State

- Last completed iteration: 025
- Last completed package: `internal/extension/contract`
- Next package: `github.com/pedronauck/agh/internal/extension/protocol`

## Deferred Cross-Package Findings

- Iteration 018 found that `internal/api/spec` has a hard-coded `hookEventFamilyValues` list similar to the stale list fixed in `internal/codegen/sdkts`. It was intentionally not changed in iteration 018 because the loop is scoped to one Go package per run. Revisit before declaring the overall refacs goal complete.
- Iteration 019 found that `internal/hooks` matcher validation still allocates on the valid hook-declaration path. It is on the `internal/config.HookDeclarations` hot path, but the implementation belongs to `internal/hooks`; revisit during that package's iteration.
- Iteration 021 found that daemon native-extension source error classification still relies on string matching. A complete fix should introduce typed marketplace/source errors in `internal/extension` and then update the daemon classifier.
- Iteration 022 found that some non-diagnostics callsites discard `RegisterDynamicSecret` cleanup functions. Revisit per-call MCP registrations and daemon/CLI lifetime semantics during the owning package iterations.
- Iteration 023 found that `internal/testutil/acpmock.DefaultDriverPath` also builds into a temp directory without a cleanup handle. Revisit during the `internal/testutil/acpmock` package iteration.
- Iteration 023 found duplicated E2E binary env var names across Mage, Go testutil, and web E2E fixtures. Revisit only as a broader harness-contract cleanup because the duplication crosses Go and TypeScript surfaces.
- Iteration 024 confirmed baseline failures in the Teams and Telegram provider integration tests before the local `internal/extension` refactor work. The focused post-change probe still failed in `TestTeamsProviderLaunchNegotiatesBridgeRuntime`, `TestTeamsProviderIngressAndDeliveryConformance`, and `TestTelegramProviderLaunchNegotiatesBridgeRuntime`; Slack, WhatsApp, and Linear provider launch probes passed. Revisit through the provider/reference-extension harness rather than treating this as a regression from the manager/tool-provider cleanup.
- Iteration 024 deferred a registry context API hard-cut. `internal/extension.Registry` still uses the package-local `registryContext()` helper; replacing that with context-bearing registry methods crosses many callsites and should be handled as a broader package API redesign.
- Iteration 024 deferred the typed marketplace/source error hard-cut that iteration 021 identified because the complete fix must update both `internal/extension` and the daemon native-extension classifier.
- Iteration 024 deferred `HostAPIHandler` domain decomposition. The handler remains broad, but splitting it cleanly crosses host API task/resource/network surfaces and should be driven by a dedicated structural pass after the package-level lifecycle bugs are closed.
- Iteration 025 found that runtime Host API dispatch in `internal/extension` can still drift from the canonical `internal/extension/contract.HostAPIMethodSpecs()` registry because dispatch keys live in a separate handler map. The fix belongs to a cross-package parity pass that compares `HostAPIHandler.MethodHandlers()` with the generated contract registry and replaces remaining raw string keys with constants.
- Iteration 025 found that `internal/extension/contract` is an intentionally broad SDK aggregator over multiple internal domain structs. A structural split into explicit host API DTOs, SDK root DTOs, hook registry, and protocol registry remains deferred because it is larger than this package iteration and requires generated-contract snapshot strategy.

## Package Order Source

Package order is deterministic from:

```bash
rtk go list ./internal/...
```

## Validation Summary For Iteration 001

```bash
rtk make verify
rtk go test ./internal/acp -count=1
rtk go test -tags integration ./internal/acp -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/acp -count=1
rtk golangci-lint run ./internal/acp
rtk proxy go test -run '^$' -bench . -benchmem ./internal/acp -count=1
rtk proxy env GOOS=windows GOARCH=amd64 go test -c -o /tmp/acp-windows.test.exe ./internal/acp
```

## Validation Summary For Iteration 002

```bash
rtk make verify
rtk go test ./internal/agentidentity -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/agentidentity -count=1
rtk golangci-lint run ./internal/agentidentity
rtk proxy go test ./internal/agentidentity -cover -count=1
rtk go test ./internal/agentidentity ./internal/api/core ./internal/api/udsapi ./internal/cli -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/agentidentity/identity_test.go
rtk proxy go test -run '^$' -bench . -benchmem ./internal/agentidentity -count=1
```

## Validation Summary For Iteration 003

```bash
rtk make verify
rtk go test ./internal/api/contract -count=1
rtk go test -tags integration ./internal/api/contract -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/contract -count=1
rtk golangci-lint run ./internal/api/contract
rtk make codegen-check
rtk go test ./internal/api/contract ./internal/api/core ./internal/extension ./internal/cli ./internal/daemon -count=1
rtk proxy go test ./internal/api/contract -cover -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/contract/json_safety_bench_test.go
rtk proxy go test -run '^$' -bench 'Benchmark(ContainsRawClaimTokenFieldNestedPayload|ValidateAuthoredContextRedactedNestedPayload)' -benchmem ./internal/api/contract -count=5
```

## Validation Summary For Iteration 004

```bash
rtk make verify
rtk go test ./internal/api/core -count=1
rtk go test ./internal/api/contract ./internal/api/core ./internal/testutil/e2e -count=1
rtk go test -tags integration ./internal/api/core -count=1
rtk go test -tags integration ./internal/daemon -run 'TestDaemonE2ENetwork(DirectReplyLifecycleWithMockAgents|WhoisAndCapabilityExchange)' -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/core -count=1
rtk golangci-lint run ./internal/api/core
rtk make codegen-check
rtk proxy go test ./internal/api/core -cover -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/network_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/coverage_helpers_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/perf_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/testutil/e2e/runtime_harness_helpers_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/daemon_network_collaboration_integration_test.go
rtk rg -n "NetworkChannelMessagePayload|NetworkChannelMessagePayloadFrom" internal web packages openapi cmd
rtk proxy go test -run '^$' -bench 'Benchmark(EmitObserveEvents|PromptStreamEncoderEmit)$' -benchmem ./internal/api/core -count=5
```

## Validation Summary For Iteration 005

```bash
rtk make verify
rtk go test ./internal/api/httpapi -run 'Test(CanonicalHostNormalizesBoundHostPorts|LoopbackGuardsHandleBoundHostPorts|CORSMiddlewareAllowsPatchPreflight|HTTPAgentKernelRoutesMatchDocumentedSpecOperations|ServerHandlerConfigIncludesCoordinatorConfig|StaticRoutes)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/httpapi/middleware_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/httpapi/routes_refac_test.go
rtk golangci-lint run ./internal/api/httpapi
rtk go test ./internal/api/httpapi -count=1
rtk go test ./internal/api/httpapi ./internal/api/udsapi ./internal/api/spec ./internal/daemon -run 'Test(HTTPAgentKernelRoutesMatchDocumentedSpecOperations|ServerHandlerConfigIncludesCoordinatorConfig|CoordinatorConfig|RegisterTaskRoutesUseSharedHandlerBindings|UDSTransportTaskSurfaceMatchesHTTPAndDocumentedSpecOperations)' -count=1
rtk proxy go test ./internal/api/httpapi -run '^TestStaticRoutes' -count=1 -memprofile /tmp/httpapi-static-after.mem
rtk go tool pprof -top /tmp/httpapi-static-after.mem
rtk go test -tags integration ./internal/api/httpapi -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/httpapi -count=1
rtk go test ./internal/daemon -count=1
rtk proxy go test ./internal/api/httpapi -cover -count=1
```

## Validation Summary For Iteration 006

```bash
rtk make verify
rtk go test ./internal/api/spec -run '^TestOperationsReturnDefensiveCopies$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/spec/operations_refac_test.go
rtk golangci-lint run ./internal/api/spec
rtk go test ./internal/api/spec -count=1
rtk go test -tags integration ./internal/api/spec -count=1
rtk make codegen-check
rtk go test ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi ./internal/codegen/openapits ./internal/codegen/sdkts -run 'Test(OperationsReturnDefensiveCopies|OperationsRemainUniqueWithExpandedTaskSurface|Document|Codegen|OpenAPI|Transport|Resource)' -count=1
rtk hyperfine --warmup 3 --runs 10 'rtk go test ./internal/api/spec -run . -count=1'
rtk proxy go test ./internal/api/spec -run '^TestDocumentTracksRequiredFieldsAndEnums$' -count=30 -cpu=1 -outputdir /tmp -cpuprofile /tmp/api-spec-006.cpu -memprofile /tmp/api-spec-006.mem -memprofilerate=1
rtk go tool pprof -top /tmp/api-spec-006.cpu
rtk go tool pprof -top /tmp/api-spec-006.mem
rtk make codegen
rtk go test ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi ./internal/codegen/openapits ./internal/codegen/sdkts -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/spec -count=1
rtk proxy go test ./internal/api/spec -cover -count=1
rtk make web-test
rtk make web-typecheck
```

## Validation Summary For Iteration 007

```bash
rtk make verify
rtk go test ./internal/api/testutil -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/testutil/apitest_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/tools_test.go
rtk rg -n "ConfigWithDisabledNetwork\\(testutil\\.NewTestHomePaths\\(t\\)\\)" internal
rtk go test ./internal/api/core -run '^TestTool' -count=20 -cpuprofile=/tmp/agh-api-testutil-tools-after.cpu -memprofile=/tmp/agh-api-testutil-tools-after.mem -memprofilerate=1
rtk go tool pprof -list='github.com/pedronauck/agh/internal/api/testutil.NewTestHomePaths' /tmp/agh-api-testutil-tools-after.cpu
rtk go tool pprof -list='github.com/pedronauck/agh/internal/api/testutil.NewTestHomePaths' /tmp/agh-api-testutil-tools-after.mem
rtk go test ./internal/api/testutil ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1
rtk golangci-lint run ./internal/api/testutil ./internal/api/core
rtk env CGO_ENABLED=1 go test -race ./internal/api/testutil -count=1
rtk go test -tags integration ./internal/api/testutil -count=1
rtk go test ./internal/daemon -run '^TestTool' -count=1
rtk go test ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/daemon -count=1
rtk proxy go test ./internal/api/testutil -cover -count=1
rtk go test -tags integration ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/api/testutil -count=1
```

## Validation Summary For Iteration 008

```bash
rtk make verify
rtk go test ./internal/api/udsapi -run 'Test(ServerStart|HostedMCPStreamErrorData|ExtensionStatusCodeMappings|RegisterNetworkRoutesMatch|UDSTransportTaskSurface)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/server_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/server_env_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/extensions_additional_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/hosted_mcp_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/udsapi/helpers_test.go
rtk go test ./internal/api/udsapi -count=1
rtk golangci-lint run ./internal/api/udsapi
rtk go test -tags integration ./internal/api/udsapi -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/udsapi -count=1
rtk proxy go test ./internal/api/udsapi -cover -count=1
rtk go test ./internal/api/core ./internal/api/httpapi ./internal/api/spec ./internal/api/udsapi ./internal/cli ./internal/daemon -count=1
rtk go test -tags integration ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi -count=1
rtk go test ./internal/api/udsapi -run '^TestServerStartRejectsRestartDuringShutdown$' -count=20
rtk go test ./internal/api/udsapi -run '^TestServerStartDuplicateKeepsActiveSocket$' -count=20
rtk proxy go test ./internal/api/udsapi -run '^(TestNew|TestPath|TestServer|TestEnsure|TestSocket)' -count=20 -memprofile=/tmp/udsapi-server-after.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/udsapi-server-after.mem
```

## Validation Summary For Iteration 009

```bash
rtk make verify
rtk go test ./internal/automation -run 'Test(TriggerFilterPathMatching|TriggerDispatchSnapshotIsolation|MergedRuntimeContextNilParent)$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/trigger_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/manager_refac_test.go
rtk go test ./internal/automation -count=1
rtk golangci-lint run ./internal/automation
rtk proxy go test ./internal/automation -run '^$' -bench 'Benchmark(TriggerEngineFireMatchingRegistrations|ExactFilterMatchNestedData|RenderTriggerPromptStatic|RenderTriggerPromptTemplate)$' -benchmem -count=5
rtk go test -tags integration ./internal/automation -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/automation -count=1
rtk proxy go test ./internal/automation -cover -count=1
rtk go test ./internal/automation ./internal/bundles ./internal/api/contract ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/daemon ./internal/extension ./internal/store/globaldb ./internal/testutil/e2e -count=1
```

## Validation Summary For Iteration 010

```bash
rtk make verify
rtk go test ./internal/automation/model -run '^TestValidateTriggerFilter$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/validate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/template_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/automation/model/template_bench_test.go
rtk go test ./internal/automation/model -count=1
rtk golangci-lint run ./internal/automation/model
rtk proxy go test ./internal/automation/model -cover -count=1
rtk proxy go test ./internal/automation/model -run '^$' -bench 'BenchmarkValidateTriggerPromptTemplate' -benchmem -count=10
rtk go test ./internal/automation -run 'Test(ValidateTriggerFilter|ValidateTriggerPromptTemplate|ParseTriggerPromptTemplate|TriggerPromptTemplate)' -count=1
rtk go test ./internal/automation ./internal/automation/model -count=1
rtk golangci-lint run ./internal/automation/model ./internal/automation
rtk proxy go test ./internal/automation -run '^$' -bench 'BenchmarkRenderTriggerPrompt(Static|Template)|BenchmarkExactFilterMatchNestedData' -benchmem -count=10
rtk go test ./internal/automation/model ./internal/automation ./internal/config ./internal/store/globaldb ./internal/settings ./internal/api/core ./internal/api/contract ./internal/cli ./internal/daemon -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/automation/model -count=1
rtk go test -tags integration ./internal/automation ./internal/automation/model -count=1
rtk proxy go test ./internal/automation ./internal/automation/model -coverpkg=./internal/automation/model -coverprofile=/tmp/automation-model-combined-cover.out -count=1
rtk go tool cover -func=/tmp/automation-model-combined-cover.out
```

## Validation Summary For Iteration 011

```bash
rtk make verify
rtk go test ./internal/bridges -count=1
rtk go test -tags integration ./internal/bridges -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bridges -count=1
rtk golangci-lint run ./internal/bridges
rtk proxy go test ./internal/bridges -cover -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridges/resource_projection_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridges/delivery_projection_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridges/registry_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridges/json_equal_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridges/resource_projection_bench_test.go
rtk go test ./internal/bridges -run 'Test(ResourceProjectionRollbackPlanRefacs|BrokerProjectEventRefacs|RegistryContextRefacs|BridgeProviderConfigRefacs)$' -count=1
rtk go test ./internal/bridges -run 'Test(BridgeInstanceValidateProviderConfigDMPolicyAndDegradation|BridgeResourceProjectionIgnoresSemanticallyEquivalentJSON|BridgeResourceBuildComputesDeltaWithoutApplyingSideEffects|BridgeResourceProjectionRemovesLegacyRowsWhenSnapshotIsEmpty|RegistryGuardClauses)$' -count=1
rtk go test ./internal/bridges -run '^TestBrokerProjectEventDeduplicatesAndFailsSession$' -count=1
rtk rg -n "ManagedSync|NewManagedSync|NewManagedResourceSync|WithManagedResourceSync|WithManagedSync|ManagedResourceSyncService|ManagedSyncService|type DeliveryBroker interface|var _ DeliveryBroker" internal/bridges internal/daemon internal/bundles internal/extension internal/api internal/store
rtk proxy go test ./internal/bridges -run '^$' -bench 'Benchmark(SemanticJSONEqual|BuildResourceState|Broker)' -benchmem -count=10
rtk go test ./internal/bridges ./internal/bridgesdk ./internal/bundles ./internal/api/contract ./internal/api/core ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/daemon ./internal/extension ./internal/observe ./internal/store/globaldb ./internal/testutil/e2e -count=1
rtk golangci-lint run ./internal/bridges ./internal/bundles ./internal/daemon ./internal/extension ./internal/observe
```

Additional wide integration command investigated but not used as package gate:

```bash
rtk go test -tags integration ./internal/bridges ./internal/bundles ./internal/daemon ./internal/extension -count=1
```

Result: failed outside `internal/bridges`; see `011_report_internal_bridges.md` for classification.

## Validation Summary For Iteration 012

```bash
rtk make verify
rtk go test ./internal/bridgesdk -run '^TestPeerResponseRefacs$' -count=1
rtk go test ./internal/bridgesdk -run '^TestPeer' -count=1
rtk proxy go test ./internal/bridgesdk -run '^$' -bench 'BenchmarkPeerCallRoundTrip' -benchmem -count=5
rtk go test ./internal/bridgesdk -run '^TestHostAPIClientRefacs$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(HostAPIClientRefacs|PeerResponseRefacs)$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(HostAPI|Peer)' -count=1
rtk go test ./internal/bridgesdk -run '^Test(InboundBatcherRefacs|RuntimeRefacs|WebhookRefacs|RetryDoRefacs|PeerResponseRefacs|HostAPIClientRefacs)$' -count=1
rtk go test ./internal/bridgesdk -run '^Test(RuntimeServeInitializeDeliverHealthShutdownAndSync|InboundBatcherCoalescesShortBurstAndPreservesOrdering|WebhookHandlerWritesHTTPErrorFromProviderMapping|InstanceCacheSnapshotAndListReturnClones|RetryDo)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/batching_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/runtime_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/webhook_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/errors_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/peer_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bridgesdk/hostapi_refac_test.go
rtk go test ./internal/bridgesdk -count=1
rtk go test -tags integration ./internal/bridgesdk -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bridgesdk -count=1
rtk golangci-lint run ./internal/bridgesdk
rtk proxy go test ./internal/bridgesdk -cover -count=1
rtk proxy go test ./internal/bridgesdk -run '^$' -bench 'Benchmark(InboundBatchKey|InstanceCacheSnapshot|FixedWindowRateLimiterAllow|PeerCallRoundTrip)$' -benchmem -count=5
rtk go test ./internal/bridgesdk ./internal/extension ./internal/extensiontest -count=1
rtk go test ./extensions/bridges/discord ./extensions/bridges/gchat ./extensions/bridges/github ./extensions/bridges/linear ./extensions/bridges/slack ./extensions/bridges/teams ./extensions/bridges/telegram ./extensions/bridges/whatsapp ./sdk/examples/telegram-reference -count=1
rtk golangci-lint run ./internal/bridgesdk ./extensions/bridges/discord ./extensions/bridges/gchat ./extensions/bridges/github ./extensions/bridges/linear ./extensions/bridges/slack ./extensions/bridges/teams ./extensions/bridges/telegram ./extensions/bridges/whatsapp ./sdk/examples/telegram-reference
```

Observed results:

- Focused refactor tests: `20 passed`.
- Existing focused regression set: `14 passed`.
- Package tests: `84 passed`.
- Integration-tag package tests: `87 passed`.
- Race package tests: passed.
- Package lint: no issues.
- Package coverage after edits: `81.2%` statements.
- Direct internal dependent set: `626 passed in 3 packages`.
- Provider/example dependent set: `191 passed in 9 packages`.
- Provider/example lint set: no issues.
- Final benchmarks:
  - `BenchmarkInboundBatchKey`: about `129.8-135.0 ns/op`, `80 B/op`, `1 alloc/op`.
  - `BenchmarkInstanceCacheSnapshot`: about `948.8-971.2 ns/op`, `3280 B/op`, `11 allocs/op`.
  - `BenchmarkFixedWindowRateLimiterAllow`: about `49.86-52.23 ns/op`, `0 B/op`, `0 allocs/op`.
  - `BenchmarkPeerCallRoundTrip`: about `8763-8972 ns/op`, `2361-2364 B/op`, `54 allocs/op`.

## Validation Summary For Iteration 013

```bash
rtk make verify
rtk go test ./internal/bundles -run 'Test(ServiceRefacs|StableIDGoldenValues|FindBundleResourceRecordIndexedNormalizesLookupKeys|BundleActivationBuildComposesTypedBundleDependency)$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/service_refac_test.go
rtk go test ./internal/bundles -count=1
rtk golangci-lint run ./internal/bundles
rtk proxy go test ./internal/bundles -cover -count=1
rtk go test -tags integration ./internal/bundles -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bundles -count=1
rtk go test ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon -count=1
rtk golangci-lint run ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(ListActivationsLargeCatalog|BuildLargeCatalog)$' -benchmem -count=5
rtk proxy go test ./internal/bundles -run '^$' -bench '^BenchmarkServiceBuildLargeCatalog$' -benchmem -benchtime=200x -count=1 -memprofile /tmp/agh-bundles-013-build-after.mem -memprofilerate=1
rtk proxy go test ./internal/bundles -run '^$' -bench '^BenchmarkServiceListActivationsLargeCatalog$' -benchmem -benchtime=200x -count=1 -memprofile /tmp/agh-bundles-013-list-after.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/agh-bundles-013-build-after.mem
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/agh-bundles-013-list-after.mem
```

Observed results:

- Focused refactor tests: `13 passed in 1 packages`.
- Package tests: `66 passed in 1 packages`.
- Integration-tag package tests: `69 passed in 1 packages`.
- Race package tests: passed.
- Package lint: no issues.
- Direct dependent set: `1827 passed in 5 packages`.
- Direct dependent lint set: no issues.
- Coverage after edits: `80.6%` statements.
- Final benchmarks:
  - `BenchmarkServiceListActivationsLargeCatalog`: about `0.93-1.01 ms/op`, `1.539 MB/op`, `10641-10644 allocs/op`.
  - `BenchmarkServiceBuildLargeCatalog`: about `1.01-1.08 ms/op`, `1.449 MB/op`, `9160-9162 allocs/op`.

## Validation Summary For Iteration 014

```bash
rtk make verify
rtk go test ./internal/bundles/model -count=1
rtk go test ./internal/bundles -run 'Test(ServiceRefacs|Scope|Activation|Inventory)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/model/model_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/service_refac_test.go
rtk golangci-lint run ./internal/bundles/model
rtk go test ./internal/bundles/model ./internal/bundles -count=1
rtk golangci-lint run ./internal/bundles/model ./internal/bundles
rtk proxy go test ./internal/bundles/model -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bundles/model ./internal/bundles -count=1
rtk go test -tags integration ./internal/bundles/model ./internal/bundles -count=1
rtk go test ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon -count=1
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(Build|ListActivations)LargeCatalog$' -benchmem -count=5
```

Observed results:

- Model package tests: `34 passed in 1 packages`.
- Focused parent regression tests: `6 passed in 1 packages`.
- Combined package tests: `101 passed in 2 packages`.
- Integration-tag package tests: `104 passed in 2 packages`.
- Race package tests: passed.
- Package lint: no issues.
- Package-local model coverage: `98.1% of statements`.
- Direct dependent set: `1828 passed in 5 packages`.
- `make verify`: passed.
- Caller benchmark check remained in the iteration-013 range:
  - `BenchmarkServiceListActivationsLargeCatalog`: about `0.97-1.43 ms/op`, `1.539 MB/op`, `10643 allocs/op`.
  - `BenchmarkServiceBuildLargeCatalog`: about `1.03-1.08 ms/op`, `1.449 MB/op`, `9160-9161 allocs/op`.

## Validation Summary For Iteration 015

```bash
rtk make verify
rtk go test ./internal/cli -run 'TestParseRequiredJSONRawMessage|TestWaitForDaemonStart|TestRunDaemonDetachedReturnsReadyStatus|Test.*Render|Test.*Output|Test.*Human|Test.*Toon' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/json_flags_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/daemon_wait_refac_test.go
rtk go test ./internal/cli -count=1
rtk golangci-lint run ./internal/cli
rtk proxy go test ./internal/cli -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/cli -count=1
rtk go test ./cmd/agh ./internal/cli -count=1
rtk go test -tags integration ./internal/cli -run '^TestCLITaskRunLifecycleIntegration$' -count=1
rtk go test -tags integration ./internal/cli -count=1
rtk proxy go test ./internal/cli -run '^$' -bench 'Benchmark(RenderHumanTableLarge|RenderToonArrayLarge|DecodeSSELargeStream|DoRequestPostJSON)$' -benchmem -count=5
```

Observed results:

- Focused unit/output/refac tests: `136 passed in 1 packages`.
- New AGH test-shape checks: passed for `internal/cli/json_flags_test.go` and `internal/cli/daemon_wait_refac_test.go`.
- Package tests: `690 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `74.8%` statements.
- Race package tests: passed.
- Direct entrypoint dependent set: `693 passed in 2 packages`.
- Focused lifecycle integration test: passed.
- Full integration-tag package command: `741 passed, 10 failed`; remaining failures are historical channel/presence read-model failures outside CLI request shaping and are classified in `015_report_internal_cli.md`.
- `make verify`: passed.
- Final benchmarks:
  - `BenchmarkRenderHumanTableLarge`: about `47.8-49.3 us/op`, `127112-127113 B/op`, `22 allocs/op`.
  - `BenchmarkRenderToonArrayLarge`: about `50.8-51.8 us/op`, `84752 B/op`, `19 allocs/op`.
  - `BenchmarkDecodeSSELargeStream`: about `112.8-114.9 us/op`, `246408 B/op`, `4099 allocs/op`.
  - `BenchmarkDoRequestPostJSON`: about `1191-1208 ns/op`, `2641 B/op`, `29 allocs/op`.

## Validation Summary For Iteration 016

```bash
rtk make verify
rtk go test ./internal/cli/docpost -count=1
rtk golangci-lint run ./internal/cli/docpost
rtk proxy go test ./internal/cli/docpost -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/cli/docpost -count=1
rtk go test -tags integration ./internal/cli/docpost -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/docpost/docpost_refac_test.go
rtk go test ./internal/cli/docpost ./internal/cli -run 'Test(NewDocCommand|ProcessInputRefacs|LinkRewriteRefacs|BuildTargetMap|RemapLinks|RewriteLinks)' -count=1
rtk make cli-docs
rtk make site-build
```

Observed results:

- Package tests: `68 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `89.5% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `68 passed in 1 packages`.
- New AGH test-shape check: passed for `internal/cli/docpost/docpost_refac_test.go`.
- Direct dependent/focused command tests: `24 passed in 2 packages`.
- Generated CLI docs: regenerated successfully, with no final persistent diff under `packages/site/content/runtime/cli-reference`.
- Site build: passed.
- `make verify`: passed.

## Validation Summary For Iteration 017

```bash
rtk go test ./internal/codegen/openapits -count=1
rtk golangci-lint run ./internal/codegen/openapits
rtk proxy go test ./internal/codegen/openapits -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/openapits -count=1
rtk go test -tags integration ./internal/codegen/openapits -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/openapits/generate_test.go
rtk make codegen-check
rtk go test ./internal/codegen/openapits ./cmd/agh-codegen -count=1
rtk golangci-lint run ./internal/codegen/openapits ./cmd/agh-codegen
rtk proxy go test -tags mage . -count=1
rtk make verify
```

Observed results:

- Package tests: `23 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `81.4% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `23 passed in 1 packages`.
- New AGH test-shape check: passed for `internal/codegen/openapits/generate_test.go`.
- `make codegen-check`: passed.
- Direct dependent codegen package set: `64 passed in 2 packages`.
- Direct dependent lint set: no issues.
- Mage-tag root tests: passed.
- `make verify`: passed.

## Validation Summary For Iteration 018

```bash
rtk go test ./internal/codegen/sdkts -count=1
rtk golangci-lint run ./internal/codegen/sdkts
rtk proxy go test ./internal/codegen/sdkts -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/sdkts -count=1
rtk go test -tags integration ./internal/codegen/sdkts -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/generate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/perf_bench_test.go
rtk go run ./cmd/agh-codegen sdk-contracts
rtk make codegen-check
rtk go run ./cmd/agh-codegen check
rtk go test ./internal/codegen/sdkts ./cmd/agh-codegen -count=1
rtk golangci-lint run ./internal/codegen/sdkts ./cmd/agh-codegen
rtk make bun-typecheck
rtk make bun-test
rtk proxy go test -tags mage . -count=1
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench . -benchmem -count=5
rtk make verify
```

Observed results:

- Package tests: `29 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `91.8% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `29 passed in 1 packages`.
- AGH test-shape checks: passed for `internal/codegen/sdkts/generate_test.go` and `internal/codegen/sdkts/perf_bench_test.go`.
- `make codegen-check`: passed.
- `go run ./cmd/agh-codegen check`: passed.
- Direct dependent codegen package set: `70 passed in 2 packages`.
- Direct dependent lint set: no issues.
- Bun typecheck: passed.
- Bun tests: `357` files and `2233` tests passed.
- Mage-tag root tests: passed.
- `make verify`: passed.
- Final benchmarks:
  - `BenchmarkGenerate`: about `590-648 us/op`, `684533-684542 B/op`, `1161 allocs/op`.
  - `BenchmarkStructFieldsPromptPayload`: mostly `52-64 us/op` with one noisy outlier, `134696 B/op`, `58 allocs/op`.

## Validation Summary For Iteration 019

```bash
rtk go test ./internal/config -count=1
rtk golangci-lint run ./internal/config
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/config/config_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/config/perf_bench_test.go
rtk proxy go test ./internal/config -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/config -count=1
rtk go test -tags integration ./internal/config -count=1
rtk proxy go test ./internal/config -run '^$' -bench . -benchmem -count=5
rtk proxy go test ./internal/config -run '^$' -bench 'Benchmark(ResolveAgentMergedMCPServers|HookDeclarationsNormalization)$' -benchmem -count=5
rtk go test ./internal/config ./internal/skills ./internal/daemon ./internal/cli -count=1
rtk go test -tags integration ./internal/config ./internal/skills -count=1
rtk rg -n "_\\s*=\\s*.*(Close|Remove|Write|Sync|Read|Run|Do|Wait)|_\\s*=\\s*[^,]+" internal/config --glob '*.go' --glob '!*_test.go'
rtk make verify
```

Observed results:

- Package tests: `505 passed in 1 packages`.
- Package lint: no issues.
- New AGH test-shape check: passed for `internal/config/config_refac_test.go`.
- Existing benchmark test-shape check: passed for `internal/config/perf_bench_test.go`.
- Package coverage: `80.7% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `515 passed in 1 packages`.
- Full benchmark suite: passed.
- Focused final benchmarks:
  - `BenchmarkResolveAgentMergedMCPServers`: about `43.7-64.4 us/op`, `89811-89817 B/op`, `486 allocs/op`.
  - `BenchmarkHookDeclarationsNormalization`: about `49.9-50.9 us/op`, `57704-57706 B/op`, `337 allocs/op`.
- Direct dependent package set (`config`, `skills`, `daemon`, `cli`): `1989 passed in 4 packages`.
- Direct integration dependent set (`config`, `skills`): `698 passed in 2 packages`.
- Production `_ =` cleanup discard scan for `internal/config`: no matches.
- `make verify`: passed after the final file split.

## Validation Summary For Iteration 020

```bash
rtk go test ./internal/coordinator -count=1
rtk golangci-lint run ./internal/coordinator
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/coordinator/coordinator_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/coordinator/coordinator_bench_test.go
rtk proxy go test ./internal/coordinator -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/coordinator -count=1
rtk go test -tags integration ./internal/coordinator -count=1
rtk proxy go test ./internal/coordinator -run '^$' -bench . -benchmem -count=5
rtk go test ./internal/daemon -run 'TestCoordinatorRuntime' -count=1
rtk go test -tags integration ./internal/daemon -run 'TestDaemonE2E.*Coordinator|TestCoordinatorRuntime' -count=1
rtk go test ./internal/daemon ./internal/task ./internal/session -run 'Coordinator|coordinator|Bootstrap|bootstrap|ExecutableRun|HealthySession' -count=1
rtk rg -n "_\\s*=" internal/coordinator --glob '*.go'
rtk make verify
```

Observed results:

- Package tests: `15 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `internal/coordinator/coordinator_test.go` and `internal/coordinator/coordinator_bench_test.go`.
- Package coverage: `88.8% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `15 passed in 1 packages`.
- Package benchmarks: passed.
- Final benchmarks:
  - `BenchmarkPromptOverlay`: about `683 ns/op` to `2190 ns/op`, `2288 B/op`, `6 allocs/op`.
  - `BenchmarkPermissionPolicy`: about `1013-1178 ns/op`, `432 B/op`, `4 allocs/op`.
  - `BenchmarkLineage`: about `721-873 ns/op`, `456 B/op`, `4 allocs/op`.
- Focused daemon coordinator runtime tests: `11 passed in 1 packages`.
- Focused daemon coordinator integration tests: `11 passed in 1 packages`.
- Focused daemon/task/session dependent set: `24 passed in 3 packages`.
- Production/test `_ =` scan in `internal/coordinator`: no matches.
- `make verify`: passed.

## Validation Summary For Iteration 021

```bash
rtk go test ./internal/daemon -run 'TestToolMCP|TestResourceAgentCatalogLookupReturnsDefensiveCopy|TestStopSkillsWatcherRespectsShutdownContext' -count=1
rtk go test -tags integration ./internal/daemon -run 'TestToolMCP|TestAgentSkillResources|TestDaemonE2ENetworkDirectReplyLifecycleWithMockAgents' -count=1
rtk golangci-lint run ./internal/daemon
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/agent_skill_resources_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/skills_watcher_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/perf_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/tool_mcp_resources_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/tool_mcp_resources_integration_test.go
rtk proxy go test ./internal/daemon -run '^$' -bench 'Benchmark(ResourceAgentCatalogResolveAgentWorkspaceHit|AgentSkillSourceSyncerSyncNoop|ToolMCPSourceSyncerSyncNoop)$' -benchmem -count=5
rtk go test ./internal/daemon -count=1
rtk proxy go test ./internal/daemon -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/daemon -count=1
rtk rg -n "context\\.Background\\(\\)|_\\s*=" internal/daemon/boot.go internal/daemon/daemon.go internal/daemon/info.go internal/daemon/lock.go internal/daemon/agent_skill_resources.go internal/daemon/tool_mcp_resources.go internal/daemon/perf_bench_test.go internal/daemon/agent_skill_resources_refac_test.go internal/daemon/skills_watcher_refac_test.go
rtk make verify
```

Observed results:

- Focused package tests: `16 passed in 1 packages`.
- Focused integration-tag package tests: `14 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for all touched daemon test/benchmark files.
- Focused benchmarks:
  - `BenchmarkResourceAgentCatalogResolveAgentWorkspaceHit`: `5790-5828 ns/op`, `752 B/op`, `8 allocs/op`.
  - `BenchmarkAgentSkillSourceSyncerSyncNoop`: `370310-374668 ns/op`, `325765-325854 B/op`, `5134 allocs/op`.
  - `BenchmarkToolMCPSourceSyncerSyncNoop`: `435737-441716 ns/op`, `421308-421330 B/op`, `5667 allocs/op`.
- Full package tests: `626 passed in 1 packages`.
- Package coverage: `72.8% of statements`.
- Race package tests: passed.
- Scoped `context.Background()` / `_ =` scan over touched daemon production/refac files: no matches.
- `make verify`: passed.

## Validation Summary For Iteration 022

```bash
rtk go test ./internal/diagnostics -count=1
rtk golangci-lint run ./internal/diagnostics
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/diagnostics/redact_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/diagnostics/redact_bench_test.go
rtk proxy go test ./internal/diagnostics -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/diagnostics -count=20
rtk go test -tags integration ./internal/diagnostics -count=1
rtk proxy go test ./internal/diagnostics -run '^$' -bench 'BenchmarkRedact(Static|Dynamic)Secrets$' -benchmem -count=5
rtk go test ./internal/session ./internal/api/core ./internal/api/contract ./internal/soul ./internal/heartbeat ./internal/mcp ./internal/cli -run 'Redact|redact|Secret|secret|Failure|Diagnostic|MCPAuth' -count=1
rtk rg -n "_\\s*=|context\\.Background\\(|strings\\.Contains\\(.*err\\.Error\\(\\)" internal/diagnostics --glob '*.go'
rtk make verify
```

Observed results:

- Package tests: `28 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `internal/diagnostics/redact_test.go` and `internal/diagnostics/redact_bench_test.go`.
- Package coverage: `90.5% of statements`.
- Race package tests with `-count=20`: passed.
- Integration-tag package tests: `28 passed in 1 packages`.
- Focused benchmarks:
  - `BenchmarkRedactStaticSecrets`: `9872-9974 ns/op`, `1467-1469 B/op`, `20 allocs/op`.
  - `BenchmarkRedactDynamicSecrets`: `10178-10287 ns/op`, `944-946 B/op`, `11 allocs/op`.
- Caller smoke package set (`session`, `api/core`, `api/contract`, `soul`, `heartbeat`, `mcp`, `cli`): `96 passed in 7 packages`.
- Scoped production/test scan for ignored errors, `context.Background`, and `strings.Contains(err.Error())`: no matches.
- `make verify`: passed.

## Validation Summary For Iteration 023

```bash
rtk go test ./internal/e2elane -count=1
rtk go test ./internal/e2elane -count=20
rtk proxy go test ./internal/e2elane -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/e2elane -count=1
rtk go test -tags integration ./internal/e2elane -count=1
rtk golangci-lint run ./internal/e2elane
rtk proxy go test ./internal/e2elane -run '^$' -bench . -benchmem -count=3
rtk go test -tags mage . -count=1
rtk env CGO_ENABLED=1 go test -race -tags mage . -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/e2elane/command_wiring_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/e2elane/lanes_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py magefile_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py magefile_lane_binary_test.go
rtk make test-e2e-runtime
rtk make verify
```

Observed results:

- Package tests: `53 passed in 1 packages`.
- Package stress rerun: `1060 passed in 1 packages`.
- Package coverage: `91.7% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `53 passed in 1 packages`.
- Package lint: no issues.
- Benchmark command: passed with no registered package benchmarks.
- Mage-tagged root tests: `18 passed in 1 packages`.
- Mage-tagged race tests: passed.
- AGH test-shape checks: passed for all touched test files.
- Runtime E2E lane passed:
  - `internal/daemon`: `24 tests`.
  - `internal/api/httpapi`: `8 tests`.
  - `internal/api/udsapi`: `14 tests`.
  - `internal/testutil/e2e`: `6 tests`.
- `make verify`: passed.

## Validation Summary For Iteration 024

```bash
rtk go test ./internal/extension -run 'TestManager(StopShutdownErrors|ResourceSourceCleanup|StopKillsHungSubprocessAfterTimeout)$' -count=1
rtk go test ./internal/extension -run 'TestExtensionToolProvider' -count=1
rtk go test ./internal/extension -count=1
rtk golangci-lint run ./internal/extension
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/manager_refac_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/perf_bench_test.go
rtk env CGO_ENABLED=1 go test -race ./internal/extension -count=1
rtk proxy go test ./internal/extension -cover -count=1
rtk go test ./internal/extension ./internal/daemon ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/cli ./internal/bundles ./internal/tools -count=1
rtk proxy go test ./internal/extension -run '^$' -bench 'Benchmark(ExtensionToolProviderListAndResolve|TaskSummaryPayloadsFromSummaries|TaskRunPayloadsFromRuns)$' -benchmem -count=5
rtk go test -tags integration ./internal/extension -run 'Test(TeamsProviderLaunchNegotiatesBridgeRuntime|TeamsProviderIngressAndDeliveryConformance|TelegramProviderLaunchNegotiatesBridgeRuntime)$' -count=3
rtk go test ./extensions/bridges/teams ./extensions/bridges/telegram -run 'Test.*InitialState|Test.*Launch|Test.*Provider' -count=1
rtk go test -tags integration ./internal/extension -run 'Test(WhatsappProviderLaunchNegotiatesBridgeRuntime|SlackProviderLaunchNegotiatesBridgeRuntime|LinearProviderLaunchNegotiatesBridgeRuntime)$' -count=1 -v
rtk make verify
```

Observed results:

- Focused manager lifecycle tests: passed.
- Focused extension tool provider tests: `18 passed in 1 packages`.
- Full package tests: `529 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `internal/extension/manager_refac_test.go` and `internal/extension/perf_bench_test.go`.
- Race package tests: passed.
- Package coverage: `76.1% of statements`.
- Direct dependent package set (`extension`, `daemon`, `api/core`, `api/httpapi`, `api/udsapi`, `cli`, `bundles`, `tools`): `3255 passed in 8 packages`.
- Focused final benchmarks:
  - `BenchmarkTaskSummaryPayloadsFromSummaries`: about `44.7-150.7 us/op`, `212992-212995 B/op`, `513 allocs/op`.
  - `BenchmarkTaskRunPayloadsFromRuns`: about `63.4-111.2 us/op`, `395648-395649 B/op`, `973 allocs/op`.
  - `BenchmarkExtensionToolProviderListAndResolve`: about `154.1-214.8 us/op`, `159038-159050 B/op`, `1852 allocs/op`.
- Known integration probe failed before and after the scoped package changes:
  - `TestTeamsProviderLaunchNegotiatesBridgeRuntime`: timed out waiting for adapter state `ready`.
  - `TestTeamsProviderIngressAndDeliveryConformance`: final adapter state was `degraded`, expected `ready`.
  - `TestTelegramProviderLaunchNegotiatesBridgeRuntime`: final adapter state was `degraded`, expected `ready`.
- Teams and Telegram provider package-level tests under `extensions/bridges/*` passed.
- WhatsApp, Slack, and Linear focused provider launch probes passed.
- `make verify`: passed.

## Validation Summary For Iteration 025

```bash
rtk go test ./internal/extension/contract -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/extension/contract -count=1
rtk golangci-lint run ./internal/extension/contract
rtk proxy go test ./internal/extension/contract -cover -count=1
rtk go test ./internal/extension/contract ./internal/extension ./internal/codegen/sdkts ./internal/api/spec -count=1
rtk golangci-lint run ./internal/extension/contract ./internal/codegen/sdkts ./internal/api/spec
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/contract/host_api_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/contract/sdk_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/generate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/perf_bench_test.go
rtk make codegen
rtk make codegen-check
rtk make bun-typecheck
rtk make bun-test
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench 'BenchmarkGenerate|BenchmarkStructFieldsPromptPayload' -benchmem -count=3
rtk make verify
```

Observed results:

- Package tests: `18 passed in 1 packages`.
- Race package tests: passed.
- Package lint: no issues.
- Package coverage: `85.7% of statements`.
- Direct dependent package set (`extension/contract`, `extension`, `codegen/sdkts`, `api/spec`): `731 passed in 4 packages`.
- Direct dependent lint set (`extension/contract`, `codegen/sdkts`, `api/spec`): no issues.
- AGH test-shape checks: passed for all touched Go test/benchmark files.
- `make codegen`: passed.
- `make codegen-check`: passed.
- `make bun-typecheck`: passed across 5 Turbo tasks.
- `make bun-test`: `357 files`, `2233 tests`, all passed.
- Focused SDK TS benchmarks passed:
  - `BenchmarkGenerate`: about `587.1-609.4 us/op`, `684550-684558 B/op`, `1161 allocs/op`.
  - `BenchmarkStructFieldsPromptPayload`: about `52.9-53.3 us/op`, `134696 B/op`, `58 allocs/op`.
- `make verify`: passed.
