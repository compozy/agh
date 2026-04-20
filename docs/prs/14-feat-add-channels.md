# PR #14: feat: add channels

- **URL**: https://github.com/compozy/agh/pull/14
- **Author**: @pedronauck
- **State**: merged
- **Created**: 2026-04-11T12:16:28Z
- **Merged**: 2026-04-12T03:09:03Z

## Summary by CodeRabbit

- **New Features**
  - Full channel management API (create/update/enable/disable/restart/list/get/routes/test-delivery) and runtime integration for adapters and delivery.

- **Delivery**
  - New delivery broker: ordered prompt deliveries with retries, snapshots, resume support, coalescing, and delivery telemetry.

- **CLI**
  - New channel commands: list/get/create/update/enable/disable/restart/routes/test-delivery.

- **Observability**
  - Channel health, aggregated status counts and per-instance delivery metrics exposed.

- **Tests**
  - Extensive unit, integration, and conformance test coverage for channels, routing, delivery, and adapters.

## Walkthrough

Adds end-to-end channel support: domain models, routing and target resolution, a delivery broker, registry service, daemon/runtime orchestration, extension host API and manager delivery integration, HTTP/UDS API + CLI surfaces, OpenAPI spec updates, and extensive unit/integration tests and adapter conformance harnesses.

## Changes

| Cohort / File(s)                                                                                                                                                                                                                                          | Summary                                                                                                                                                                                                     |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Domain & core types** <br> `internal/channels/...` <br> `internal/channels/doc.go`, `internal/channels/types.go`, `internal/channels/types_test.go`, `internal/channels/dimensions.go`, `internal/channels/routing.go`                                  | New channels package: enums, sentinel errors, ChannelInstance/Route/Delivery/Target types, normalization/validation, routing-key canonicalization, routing-dimension helpers, and unit tests.               |
| **Registry & persistence** <br> `internal/channels/registry.go`, `internal/channels/registry_test.go`, `internal/channels/registry_integration_test.go`                                                                                                   | Registry service and store interface: create/get/list/update instances, lifecycle transitions, route resolve/upsert/list, canonicalization and persistence tests.                                           |
| **Delivery pipeline & metrics** <br> `internal/channels/delivery_types.go`, `internal/channels/delivery_broker.go`, `internal/channels/delivery_broker_test.go`, `internal/channels/delivery_projection_test.go`, `internal/channels/delivery_metrics.go` | Delivery abstractions, Broker with per-route workers, queueing/backpressure, snapshot/ack semantics, metrics model and extensive concurrency tests.                                                         |
| **Target resolution & lifecycle** <br> `internal/channels/target.go`, `internal/channels/target_test.go`, `internal/channels/target_integration_test.go`, `internal/channels/lifecycle.go`                                                                | ResolveDeliveryTarget and BuildDeliveryTarget merging instance defaults with overrides, DeliveryMode validation, and instance lifecycle transition validation.                                              |
| **Daemon runtime & orchestration** <br> `internal/daemon/...` <br> `boot.go`, `channels.go`, `daemon.go`, `channels_test.go`, integration tests                                                                                                           | Channel runtime composition, secret binding resolution, broker exposure, instance lifecycle methods (Start/Stop/Restart), boot wiring and integration tests.                                                |
| **Extension host API & manager** <br> `internal/extension/...` <br> `host_api.go`, `host_api_channels.go`, `channel_delivery_notifier.go`, `manager.go`, `protocol/host_api.go`, `contract/*`, tests...                                                   | Host API handlers for ingest/get/report_state, Manager.DeliverChannel implementation, capability↔service-method negotiation, notifier projecting agent events to the broker, and related tests/integration. |
| **API contract, handlers & conversions** <br> `internal/api/contract/channels.go`, `internal/api/core/channels.go`, `internal/api/core/conversions.go`, `internal/api/core/errors.go`, `internal/api/core/interfaces.go`, tests...                        | New contract DTOs (create/update/test delivery), HTTP handlers for channels/routes/lifecycle/test-delivery, error→HTTP status mapping, observer health conversion additions, and tests.                     |
| **HTTP/UDS servers, routes & spec** <br> `internal/api/httpapi/...`, `internal/api/udsapi/...`, `internal/api/spec/spec.go`, `spec_test.go`                                                                                                               | Route registration and server wiring (WithChannelService), OpenAPI additions for channel endpoints/enums, and handler + integration tests.                                                                  |
| **API test utilities** <br> `internal/api/testutil/apitest.go`                                                                                                                                                                                            | Test stubs: StubObserver channel-health hook and new StubChannelService to exercise handlers.                                                                                                               |
| **CLI & client** <br> `internal/cli/...` <br> `internal/cli/channel.go`, `client.go`, tests...                                                                                                                                                            | CLI commands and output for channels (list/get/create/update/enable/disable/restart/routes/test-delivery), DaemonClient channel methods, request/response type aliases, and tests.                          |
| **Extension test harness & conformance** <br> `internal/extensiontest/...`, `internal/extension/telegram_reference_integration_test.go`                                                                                                                   | New harness and marker-based conformance validator for channel adapters, scripted prompt driver, and adapter integration tests.                                                                             |
| **Routes / registration updates** <br> `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, `internal/api/httpapi/handlers.go`, `internal/api/udsapi/server.go`                                                                             | Registered /api/channels endpoints across HTTP and UDS handlers and propagated ChannelService through server/handler configs.                                                                               |
| **Many tests & integrations** <br> multiple \*\_test.go across packages                                                                                                                                                                                   | Extensive unit, integration, concurrency, and end-to-end tests exercising routing, delivery, ingestion, host API and adapter conformance.                                                                   |

## Sequence Diagram

mermaid
sequenceDiagram
participant CLI as CLI / Client
participant API as API Server (HTTP/UDS)
participant Reg as Channel Registry (Service)
participant DB as Persistence (store)
participant Broker as Delivery Broker
participant Ext as Extension Process / Adapter
participant Obs as Observer

    CLI->>API: create/update/get/test-delivery requests
    API->>Reg: CreateInstance / UpdateInstance / ResolveDeliveryTarget / ListRoutes
    Reg->>DB: Insert/Update/Get ChannelInstance & routes
    DB-->>Reg: persisted data
    Reg-->>API: channel instance / resolved delivery target
    API-->>CLI: 201/200 + payload

    alt ingest/delivery flow
        Ext->>API: channels/messages/ingest (host API)
        API->>Reg: BuildRoutingKey / ResolveOrCreateRoute
        Reg->>DB: Resolve/Create route
        Reg-->>API: route / session info
        API->>Broker: RegisterPromptDelivery / ProjectEvent
        Broker->>Ext: DeliverChannel RPC (start/delta/final/resume)
        Ext-->>Broker: DeliveryAck
        Broker->>Obs: update delivery metrics / health
    end
