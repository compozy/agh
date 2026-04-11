# TechSpec: Channel Adapters

## Executive Summary

AGH gains real-time messaging channel adapters through a **hybrid architecture**: the daemon owns a small channel substrate in `internal/channels/`, while platform-specific adapters such as Telegram, Slack, and Discord remain extension subprocesses. The channel substrate owns the load-bearing invariants that must be consistent across every adapter: channel instance registry, scoped routing, delivery-target resolution, credential binding, and the negotiated outbound delivery stream. Extensions continue to own platform transport, upstream authentication, formatting, media handling, and channel-specific UX.

The implementation strategy replaces the previous 100% extension-based design. The daemon no longer treats channels as opaque integrations. Instead, it treats them as a first-class operational surface with a stable registry and routing model, while preserving extension-based platform adapters. The primary trade-off is clear: AGH adds a small core subsystem to reduce protocol and lifecycle risk. This increases daemon scope, but it avoids pushing routing, credential mediation, and streaming semantics into an RPC boundary that is too important to remain ad hoc.

## System Architecture

### Component Overview

The design introduces four daemon-owned channel components:

- `Registry`
  Persists channel instances, secret bindings, and route ownership.
- `Router`
  Builds canonical routing keys from instance policy and resolves them to ACP sessions.
- `DeliveryBroker`
  Projects session output into a delivery-oriented stream for channel-capable extensions.
- `TargetResolver`
  Resolves typed delivery targets for automation, manual sends, and future outbound workflows.

The extension boundary remains platform-specific:

- Channel extension subprocesses receive platform-native inbound events.
- They normalize those events into a typed ingest request and call the daemon Host API.
- They consume the negotiated delivery stream and translate it into platform sends, edits, typing indicators, attachments, or fallback messages.

Data flow:

1. Operator or API creates a `channel_instance`.
2. Daemon validates scope, extension ownership, routing policy, and secret bindings.
3. Extension Manager launches the adapter subprocess with instance-scoped bound secrets.
4. Platform event arrives at the adapter.
5. Adapter calls `channels/messages/ingest`.
6. Daemon validates the instance, builds the routing key, resolves or creates the ACP session, and prompts it.
7. Session output is projected into the negotiated `channels/deliver` stream.
8. Adapter acknowledges deliveries and maps them to platform-native message IDs.
9. Automation and manual outbound sends resolve typed `delivery_target` objects through the same registry.

External system interactions:

- Messaging platform APIs are accessed only by the adapter extension.
- Secure secret storage remains daemon-owned.
- Automation integrates through typed delivery targets rather than direct platform calls.

## Implementation Design

### Core Interfaces

```go
type Registry interface {
    CreateInstance(ctx context.Context, req CreateInstanceRequest) (*ChannelInstance, error)
    GetInstance(ctx context.Context, id string) (*ChannelInstance, error)
    UpdateInstanceState(ctx context.Context, req UpdateInstanceStateRequest) error
    ResolveRoute(ctx context.Context, key RoutingKey) (*ChannelRoute, error)
    UpsertRoute(ctx context.Context, route ChannelRoute) error
}
```

```go
type DeliveryBroker interface {
    Deliver(ctx context.Context, evt DeliveryEvent) error
    Snapshot(ctx context.Context, deliveryID string) (*DeliverySnapshot, error)
}
```

```go
type RoutingPolicy struct {
    IncludePeer   bool `json:"include_peer"`
    IncludeThread bool `json:"include_thread"`
    IncludeGroup  bool `json:"include_group"`
}
```

Error handling conventions:

- Validation failures return typed invalid-parameter errors.
- Unknown or disabled instances return typed not-found or unavailable errors.
- Delivery backpressure returns a typed rate-limited or queue-saturated error.
- Routing conflicts are daemon-owned consistency errors and must be surfaced explicitly.

### Data Models

Core persisted entities:

- `ChannelInstance`
  `id`, `scope`, `workspace_id?`, `platform`, `extension_name`, `display_name`, `enabled`, `status`, `routing_policy`, `delivery_defaults`, `created_at`, `updated_at`
- `ChannelSecretBinding`
  `channel_instance_id`, `binding_name`, `vault_ref`, `kind`, `created_at`, `updated_at`
- `ChannelRoute`
  `routing_key_hash`, `scope`, `workspace_id?`, `channel_instance_id`, `peer_id?`, `thread_id?`, `group_id?`, `session_id`, `agent_name`, `last_activity_at`, `created_at`, `updated_at`

Typed runtime models:

- `RoutingKey`
  Fixed base:
  `scope`, `workspace_id?`, `channel_instance_id`
  Policy-controlled dimensions:
  `peer_id?`, `thread_id?`, `group_id?`
- `DeliveryTarget`
  `channel_instance_id`, `peer_id?`, `thread_id?`, `group_id?`, `mode`
- `InboundMessageEnvelope`
  `channel_instance_id`, `scope`, `workspace_id?`, `peer_id?`, `thread_id?`, `group_id?`, `platform_message_id`, `received_at`, `sender`, `content`, `attachments`, `idempotency_key`
- `DeliveryEvent`
  `delivery_id`, `channel_instance_id`, `routing_key`, `delivery_target`, `seq`, `event_type`, `content`, `final`, `metadata`
- `IngestDedupRecord`
  `idempotency_key`, `channel_instance_id`, `received_at`, `expires_at`

Database schema outline:

```sql
CREATE TABLE channel_instances (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL,
    workspace_id TEXT,
    platform TEXT NOT NULL,
    extension_name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    status TEXT NOT NULL,
    routing_policy TEXT NOT NULL,
    delivery_defaults TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE channel_secret_bindings (
    channel_instance_id TEXT NOT NULL,
    binding_name TEXT NOT NULL,
    vault_ref TEXT NOT NULL,
    kind TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (channel_instance_id, binding_name)
);

CREATE TABLE channel_routes (
    routing_key_hash TEXT PRIMARY KEY,
    scope TEXT NOT NULL,
    workspace_id TEXT,
    channel_instance_id TEXT NOT NULL,
    peer_id TEXT,
    thread_id TEXT,
    group_id TEXT,
    session_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    last_activity_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE channel_ingest_dedup (
    idempotency_key TEXT PRIMARY KEY,
    channel_instance_id TEXT NOT NULL,
    received_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
```

### API Endpoints

#### HTTP / UDS API

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/channels` | List channel instances |
| `POST` | `/api/channels` | Create a channel instance |
| `GET` | `/api/channels/:id` | Get one channel instance |
| `PATCH` | `/api/channels/:id` | Update mutable instance fields |
| `POST` | `/api/channels/:id/enable` | Enable an instance |
| `POST` | `/api/channels/:id/disable` | Disable an instance |
| `POST` | `/api/channels/:id/restart` | Restart an instance |
| `POST` | `/api/channels/:id/test-delivery` | Send a test message through the adapter |
| `GET` | `/api/channels/:id/routes` | Inspect active routes for an instance |

#### Extension Host API

| Method | Direction | Description |
|---|---|---|
| `channels/messages/ingest` | Extension -> AGH | Submit one normalized inbound platform message |
| `channels/instances/get` | Extension -> AGH | Fetch instance metadata relevant to the running adapter |
| `channels/instances/report_state` | Extension -> AGH | Report adapter-observed instance state transitions |
| `channels/deliver` | AGH -> Extension | Negotiated delivery stream request for outbound channel events |

Request design notes:

- `channels/messages/ingest` must accept a typed envelope, not arbitrary JSON blobs.
- `channels/deliver` is negotiated during `initialize` and is not a generic hook or raw session subscription mechanism.
- `channels/deliver` acknowledgements may return `remote_message_id` and `replace_remote_message_id` for progressive delivery.

## Integration Points

### Extension Protocol

Channel-capable extensions negotiate the `channels/deliver` service during `initialize`. The extension protocol must be updated to declare this service explicitly rather than relying on an unregistered daemon-to-extension method. Ordering is guaranteed per routing key. Backpressure is bounded per channel instance and per routing key.

### Session Manager

The channel router resolves or creates ACP sessions and remains the daemon-side owner of session continuity. The adapter does not persist local `chat_id -> session_id` authority. Session output is projected into channel delivery events rather than exposing raw ACP internals.

### Automation

Automation integrates through typed `delivery_target` objects, not through raw platform strings. Trigger ingress from channel messages still enters through extension-to-daemon ingest. Outbound automation deliveries resolve through `TargetResolver` and the channel registry.

### Secret Storage

The daemon remains the owner of secure secret storage. Extensions do not receive arbitrary `vault/get` access. Instead, the daemon resolves `channel_secret_bindings` and injects only the credentials attached to the channel instance at launch time.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/channels/` | new | New governance-focused substrate for registry, routing, delivery, and targets | Add narrow core package |
| `internal/extension/` | modified | Handshake and capability negotiation must support channel delivery service | Extend protocol and runtime |
| `internal/session/` | modified | Channel router resolves sessions and projects output into delivery events | Add channel-aware projection hooks |
| `internal/store/globaldb/` | modified | New tables for channel instances, bindings, and routes | Add schema and store helpers |
| `internal/api/httpapi/` | modified | Add channel instance management endpoints | Add handler and routing layer |
| `internal/api/udsapi/` | modified | Mirror channel instance management for CLI | Add handler wiring |
| `internal/cli/` | modified | Add channel management commands | Add CLI surface |
| `internal/observe/` | modified | Add per-instance health and backlog visibility | Extend health and metrics |
| `channel adapter extensions` | modified | Adapters now integrate with ingest + negotiated delivery service | Update extension contract |
| `automation` | modified | Outbound delivery resolves typed targets against the channel registry | Add target resolution integration |

## Testing Approach

### Unit Tests

- Validate `RoutingPolicy` behavior across peer, thread, and group combinations.
- Validate `RoutingKey` construction and hashing.
- Validate scope rules for `global` and `workspace` instances.
- Validate secret binding resolution and least-privilege injection.
- Validate `DeliveryTarget` resolution and defaulting behavior.
- Validate idempotency handling on repeated inbound events.

### Integration Tests

- `create instance -> launch extension -> ingest inbound message -> resolve route -> prompt session -> deliver outbound stream -> ack`
- `restart extension -> preserve routes -> resume deliveries`
- `disable instance -> block new ingest -> preserve prior route history`
- `automation run -> resolve delivery_target -> deliver outbound message`
- `duplicate inbound webhook -> no duplicate prompt or route mutation`

Environment notes:

- Use real SQLite via `t.TempDir()`.
- Use a test extension subprocess that implements the negotiated delivery service.
- Keep one reference adapter integration path in scope for v1, preferably Telegram.

## Development Sequencing

### Build Order

1. Add channel persistence models and store helpers for instances, secret bindings, and routes — no dependencies.
2. Add the core channel registry and routing-key builder — depends on step 1.
3. Add typed delivery-target model and target resolution — depends on steps 1 and 2.
4. Extend the extension handshake and protocol to negotiate `channels/deliver` — depends on step 2.
5. Add `channels/messages/ingest` and instance-state Host API methods — depends on steps 2 and 4.
6. Add delivery-broker projection from session output to channel delivery events — depends on steps 2 and 4.
7. Add HTTP, UDS, and CLI surfaces for channel instance lifecycle — depends on steps 1, 2, and 3.
8. Add per-instance health and observability — depends on steps 2, 5, and 6.
9. Implement the reference Telegram adapter against the new substrate — depends on steps 4, 5, 6, 7, and 8.
10. Integrate automation delivery-target resolution with the channel registry — depends on steps 3 and 7.

### Technical Dependencies

- Extension protocol update for negotiated channel delivery service
- New global DB schema for channel persistence
- Session-output projection layer for delivery events
- Reference adapter test harness for conformance
- Automation integration for typed delivery targets

## Monitoring and Observability

Operational visibility must be per `channel_instance`, not only per extension process.

Key metrics:

- `agh_channel_instances_total{platform,status}`
- `agh_channel_ingest_total{platform,status}`
- `agh_channel_deliveries_total{platform,event_type,status}`
- `agh_channel_delivery_backlog{channel_instance_id}`
- `agh_channel_delivery_dropped_total{channel_instance_id,reason}`
- `agh_channel_route_count{channel_instance_id}`
- `agh_channel_auth_failures_total{channel_instance_id}`

Log events:

- channel instance created
- channel instance enabled or disabled
- channel instance entered `auth_required`
- route created or rebound
- delivery started
- delivery coalesced due to pressure
- delivery failed terminally
- adapter restart and recovery result

Health surface:

- extend `/api/observe/health` with aggregated channel metrics
- add per-instance detail in a dedicated channel API
- report `disabled`, `starting`, `ready`, `degraded`, `auth_required`, `error`

## Technical Considerations

### Key Decisions

- **Hybrid architecture**: the daemon owns channel governance while adapters remain extensions.
- **Core-owned routing**: session continuity belongs to the daemon, not to extension-local SQLite.
- **Scoped instances**: channels support both `global` and `workspace` scope.
- **Policy-driven routing key**: fixed base plus instance-selected dimensions.
- **Typed delivery targets**: automation and manual sends reference a canonical target object.
- **Negotiated delivery stream**: real-time outbound channel delivery is a negotiated extension service, not hooks or polling.
- **Bound secrets**: adapters receive only the secrets explicitly attached to their channel instance.

### Known Risks

- The core substrate may grow beyond governance if platform specifics leak into it.
- Delivery-stream semantics may become too complex if v1 tries to model every platform quirk.
- Routing-policy misconfiguration could merge or fragment conversations incorrectly.
- Cross-platform routing dimension semantics are underspecified: `peer_id`, `thread_id`, and `group_id` may map to different platform concepts across adapters (e.g., Slack channels vs. Discord forums vs. Telegram supergroup topics), producing semantically incompatible results when queried across platforms.
- Secret-binding UX may feel opaque if CLI and API flows are not explicit and operator-friendly.
- Restart semantics for long-running streaming deliveries need careful conformance testing.

Mitigations:

- Keep `internal/channels/` narrow and policy-focused.
- Limit v1 delivery semantics to text-first conversational delivery plus attachment references.
- Add strong config validation and route-inspection tools.
- Provide a reference adapter and a reusable adapter conformance harness.
- Require a written per-platform dimension mapping contract before each adapter ships, documenting which platform concept maps to `peer_id`, `thread_id`, and `group_id`.

## Architecture Decision Records

Current ADRs:
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Move routing, registry, targets, and channel governance into the daemon while keeping platform adapters as extensions.
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Persist channel instances and routes in the daemon and make routing key construction a core responsibility.
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — Replace ad hoc session subscriptions with a negotiated delivery-oriented extension service.
- [ADR-008: Bound Secret Injection per Channel Instance](adrs/adr-008.md) — Replace arbitrary vault reads with daemon-resolved secret bindings per channel instance.

Superseded historical ADRs:
- [ADR-001: 100% Extension-Based Channel Adapters](adrs/adr-001.md) — Superseded after review showed that channels need a daemon-owned governance substrate.
- [ADR-002: Persistent Session Per Chat with Serial Queue](adrs/adr-002.md) — Superseded by daemon-owned scoped routing and policy-driven routing keys.
- [ADR-003: Session Event Subscription via Host API](adrs/adr-003.md) — Superseded by the negotiated delivery-stream service.
- [ADR-004: Minimal Credential Vault for Extension Secrets](adrs/adr-004.md) — Superseded by instance-scoped secret binding instead of arbitrary extension vault access.
