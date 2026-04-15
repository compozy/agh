# TechSpec: Provider-Scoped Bridge Adapters (Bridge V1)

## Executive Summary

This change delivers eight production bridge provider extensions aligned with AGH's bridge architecture: **Slack, Microsoft Teams, Google Chat, Discord, GitHub, Linear, Telegram, and WhatsApp**. Each provider lives under `extensions/bridges/<platform>/` and runs as a single provider-scoped subprocess that multiplexes multiple `BridgeInstance` records and tenants internally.

The original "full chat-sdk parity" goal is intentionally narrowed. Bridge v1 focuses on strong conversational flows plus a small typed interaction subset instead of promising every platform-specific UI and lifecycle feature from day one.

The shared `internal/bridgesdk` package remains the foundation, but the bridge runtime and Host API contract must evolve to support provider-scoped runtimes, per-instance provider configuration, hardened webhook ingress, and structured operational reporting.

## Goals

- Ship provider extensions for all eight target platforms.
- Keep `BridgeInstance` as the daemon-owned routing, health, and secret-binding unit.
- Support multi-instance and multi-tenant providers through one provider-scoped subprocess.
- Make bridge v1 production-credible with shared operational hardening.
- Preserve a path to richer interaction families later without overloading v1.

## Non-Goals

- Full feature parity with all chat-sdk adapters in v1.
- Cross-platform modal lifecycle portability.
- Cross-platform ephemeral message semantics.
- Typing indicators as a portable bridge contract.
- Approval UI flows and other platform-native confirmation surfaces.
- Credential pooling and rotation strategies in alpha.
- Two-lane reasoning-vs-answer rendering in bridge delivery.

## Approved Architecture

### Provider Runtime Model

Each platform provider remains one extension, but the subprocess model changes:

- one subprocess per provider extension
- many `BridgeInstance` records multiplexed inside that provider runtime
- one `BridgeInstance` is no longer equivalent to one launched process

`BridgeInstance` remains the daemon-owned operational object for:

- routing
- secret bindings
- status and health
- delivery defaults
- provider-specific configuration

This replaces the old one-instance-per-extension assumption, which cannot support GitHub App installations, Linear OAuth organizations, or other provider-level multi-tenant modes cleanly.

### Component Overview

```
extensions/bridges/telegram/     ─┐
extensions/bridges/slack/        ─┤
extensions/bridges/discord/      ─┤
extensions/bridges/whatsapp/     ─┤  provider-scoped subprocesses
extensions/bridges/teams/        ─┤  (one runtime per provider)
extensions/bridges/gchat/        ─┤
extensions/bridges/github/       ─┤
extensions/bridges/linear/       ─┘
        │
        │ imports
        ▼
internal/bridgesdk/              ← shared provider SDK
(peer, handshake, instance cache,
 webhook guards, dedup, batching,
 error classification, delivery)
        │
        │ JSON-RPC over stdio
        ▼
internal/daemon bridge runtime   ← daemon-owned registry/routing/health
        │
   ┌────┼────────────┐
   ▼    ▼            ▼
bridges  subprocess   extension
registry  process     manager
```

### Data Flow

1. Platform sends webhook to one provider runtime endpoint.
2. Provider runtime validates and guards the request, applies adapter-local dedup and optional batching, then maps the payload into one typed inbound bridge event tied to a `bridge_instance_id`.
3. Provider calls the daemon Host API to ingest the inbound event.
4. Daemon resolves routing and dispatches to the owning ACP session.
5. Daemon projects prompt output into bridge delivery requests tied to the target `bridge_instance_id`.
6. Provider maps the outbound request into platform API calls and acknowledges progress back to the daemon.

## Bridge V1 Scope

### Required Conversational Contract

Bridge v1 must support:

- inbound text messages
- outbound text delivery
- normalized attachments
- thread-aware routing and targeting
- progressive textual streaming
- edit and delete semantics for previously delivered messages
- auth and health reporting
- typed delivery target resolution

### Typed Optional Interaction Contract

Bridge v1 also supports a narrow typed interaction subset for platforms that can implement it:

- `command` inbound events
- `action` inbound events
- `reaction` inbound events

Providers may implement any subset of those optional families, but they are part of the bridge v1 protocol shape and must not be hidden behind opaque `metadata` blobs.

### Deferred Beyond V1

The following remain out of scope:

- modal lifecycle orchestration
- ephemeral delivery portability
- typing indicators
- approval requests and platform-native approval widgets
- reasoning-lane versus answer-lane dual streaming
- credential pool rotation

## Data Model Changes

### Bridge Instance

`BridgeInstance` needs one additional provider-owned config payload, for example `provider_config`.

`provider_config` holds instance-specific operational settings such as:

- provider mode (`comments`, `agent-sessions`, `bot`, `app`)
- webhook public URL information
- enterprise API base URLs
- DM policy
- batching and dedup tuning
- provider-specific tenant identifiers or flags

`delivery_defaults` remains narrow and only governs outbound destination defaults such as:

- `peer_id`
- `thread_id`
- `group_id`
- `mode`

It must not absorb provider configuration.

### Provider Manifest

Provider manifests remain static extension metadata and should declare:

- `bridge.platform`
- `bridge.display_name`
- required secret slots
- optional provider config schema/version hints

They do not carry per-instance runtime configuration.

### Secret Slots

Each provider declares named secret slots. Example slots:

| Provider | Slot | Purpose |
|----------|------|---------|
| Telegram | `bot_token` | Bot API token |
| Telegram | `webhook_secret` | Optional webhook secret token |
| Slack | `bot_token` | Bot OAuth token |
| Slack | `signing_secret` | Request signature verification |
| Discord | `bot_token` | Bot token |
| Discord | `public_key` | Interaction signature verification |
| WhatsApp | `access_token` | Cloud API token |
| WhatsApp | `app_secret` | Webhook signature verification |
| WhatsApp | `verify_token` | Webhook challenge verification |
| Teams | `app_id` | Bot identity |
| Teams | `app_password` | Bot secret |
| Teams | `app_tenant_id` | Optional tenant pinning |
| Google Chat | `credentials_json` | Service account credentials |
| Google Chat | `project_number` | JWT verification context |
| GitHub | `webhook_secret` | Webhook verification |
| GitHub | `token` | PAT mode |
| GitHub | `app_id` | App mode |
| GitHub | `private_key` | App private key |
| Linear | `webhook_secret` | Webhook verification |
| Linear | `api_key` | Single-tenant mode |
| Linear | `client_id` | OAuth mode |
| Linear | `client_secret` | OAuth mode |

## Host API and Runtime Changes

The current bridge runtime handshake is too instance-centric for provider-scoped runtimes. The spec therefore requires provider-scoped runtime negotiation.

### Runtime Handshake

The bridge initialize payload must change from "one assigned instance" to "provider runtime context". That context should include:

- provider identity
- platform
- supported protocol version
- zero or more managed bridge instance snapshots
- resolved secret bindings per instance

The provider runtime is then responsible for keeping its internal instance cache current through explicit Host API calls or daemon-driven updates.

### Required Host API Surface

At minimum, provider runtimes need Host API methods that let them:

- list or fetch the bridge instances they own
- ingest inbound bridge events tied to a `bridge_instance_id`
- report per-instance state and degradation

The old "single authorized instance bound to the process" model is not sufficient for the approved architecture.

### Outbound Delivery

`bridges/deliver` remains the negotiated daemon -> extension service, but every delivery request continues to carry:

- `bridge_instance_id`
- routing key
- delivery target

This keeps the daemon in charge of route ownership while the provider runtime selects the correct platform tenant configuration internally.

## Shared SDK Requirements

`internal/bridgesdk` must own the common operational machinery that every provider would otherwise reimplement:

- JSON-RPC peer and handshake helpers
- provider runtime instance cache
- webhook server scaffolding
- webhook request guards
- adapter-local dedup cache
- inbound batching and debounce
- error classification and retry decisions
- health probe helpers
- delivery acknowledgment helpers
- graceful shutdown

## Operational Requirements

### Webhook Defense

Provider webhook endpoints must implement:

- method validation
- content-type validation where applicable
- body size limits
- rate limiting
- in-flight concurrency limits
- signature verification

Signature verification alone is not sufficient.

### Adapter-Local Dedup

Providers must maintain a short-lived in-memory dedup cache to suppress duplicate platform retries before calling the daemon.

This supplements, but does not replace, daemon-side ingest dedup.

### Inbound Batching

Providers may coalesce short bursts of inbound user messages under the same routing identity using a debounce window.

Batching is:

- configurable per bridge instance via `provider_config`
- optional per provider
- required to preserve ordering and sender/thread identity

### Error Classification

The shared SDK must classify outbound provider failures into actionable classes:

- `auth`
- `rate_limit`
- `timeout`
- `transient`
- `permanent`

The runtime then maps those classes to recovery actions such as:

- retry with backoff and jitter
- mark instance `Degraded`
- mark instance `AuthRequired`
- stop retrying and surface operator-visible failure

### DM Policy

Direct-message access control is explicit bridge-instance configuration, not implicit provider behavior.

Supported v1 policy modes:

- `open`
- `allowlist`
- `pairing`

Providers enforce the selected policy before creating or resuming routes.

### Structured Degradation Reporting

Providers must report structured degradation reasons in addition to free-form health text so the daemon and web UI can surface stable operational causes such as:

- `auth_failed`
- `rate_limited`
- `webhook_invalid`
- `provider_timeout`
- `tenant_config_invalid`

### Optional Capability Hints

Providers may report optional platform capability hints such as:

- maximum message length
- formatting mode
- attachment constraints

These hints are optional metadata for future prompt/context injection and are not required for bridge v1 correctness.

## Reference Sources

### Chat-SDK (.resources/chat)

Primary domain reference for platform API behavior, message formats, and adapter patterns. Indexed as a KB vault at `.resources/chat/.kb/vault/chat-sdk/` (2800 symbols, 7234 relations). Use `kb search "<query>" --topic chat-sdk --vault .resources/chat/.kb/vault` for lookups.

| Platform | Primary Source | Key Patterns |
|----------|---------------|--------------|
| Telegram | `packages/adapter-telegram/src/index.ts` | Bot API, webhook/polling, inline keyboards, forum topics, `lockScope: "channel"` |
| Slack | `packages/adapter-slack/src/index.ts` | Events API, Web API, signing secret, Block Kit, streaming via `chat.update`, OAuth |
| Discord | `packages/adapter-discord/src/index.ts` | Ed25519 verification, Interactions API, 3-second ACK, embeds, threads |
| WhatsApp | `packages/adapter-whatsapp/src/index.ts` | Cloud API, verify challenge, interactive buttons, 4096-char splitting |
| Teams | `packages/adapter-teams/src/index.ts` | Bot Framework, Adaptive Cards v1.4, Task Modules, Graph API reader |
| Google Chat | `packages/adapter-gchat/src/index.ts` | Dual webhook modes, JWT verification, Cards v2, Workspace Events subscriptions |
| GitHub | `packages/adapter-github/src/index.ts` | HMAC-SHA256, PR/issue comments, review threading, App installations |
| Linear | `packages/adapter-linear/src/index.ts` | Comments vs agent-sessions, LinearWebhookClient, OAuth, append-only activities |
| Core | `packages/chat/src/types.ts` | `Adapter` interface, `MessageData`, `PostableMessage`, handler registration |
| Shared | `packages/adapter-shared/src/index.ts` | Error classes, emoji converter, file helpers, card-to-fallback |

### GoClaw (.resources/goclaw)

Go-native multi-agent platform with production channel implementations. Key references for Go patterns:

- `internal/channels/channel.go` — `Channel` / `StreamingChannel` / `WebhookChannel` interfaces, `DMPolicy` / `GroupPolicy` enums
- `internal/channels/dispatch.go` — Non-blocking outbound dispatch loop via message bus
- `internal/channels/events.go` — Two-lane streaming (reasoning + answer)
- `internal/providers/retry.go` — Go retry with jitter and `IsRetryableError()`
- `internal/providers/adapter_registry.go` — `AdapterRegistry` with `Register/Get` factory
- `internal/gateway/ratelimit.go` — Per-key token bucket rate limiter

### Hermes (.resources/hermes)

Python multi-platform agent gateway with 20+ adapters. Key references for operational patterns:

- `gateway/platforms/base.py` — `BasePlatformAdapter` interface (1400+ lines), `MessageEvent` / `SendResult` dataclasses
- `gateway/platforms/helpers.py` — `TextBatchAggregator` (debounce), `MessageDeduplicator` (TTL cache), `strip_markdown()`
- `gateway/platforms/webhook.py` — Production webhook adapter: HMAC, idempotency, rate limiting, prompt templating
- `gateway/platforms/ADDING_A_PLATFORM.md` — 16-item integration checklist for new platforms
- `agent/error_classifier.py` — `FailoverReason` enum and `ClassifiedError` with pattern-based detection
- `agent/retry_utils.py` — `jittered_backoff()` with decorrelated seeding
- `agent/credential_pool.py` — Multi-credential failover strategies

### OpenClaw (.resources/openclaw)

TypeScript multi-channel assistant platform. Key references for ingress hardening and plugin architecture:

- `src/plugin-sdk/webhook-ingress.ts` — 6-layer webhook defense (method → content-type → body size → rate limit → in-flight → anomaly)
- `src/plugin-sdk/webhook-memory-guards.ts` — `FixedWindowRateLimiter`, bounded counter
- `src/infra/retry.ts` — Composable retry with `shouldRetry`, `retryAfterMs`, `onRetry` hooks
- `src/infra/errors.ts` — `ErrorKind` classification and error cause chain formatting
- `src/channels/plugins/types.plugin.ts` — `ChannelPlugin` contract with 15+ adapter facets
- `src/channels/plugins/types.core.ts` — `ChannelSecurityDmPolicy` discriminated union
- `extensions/discord/src/` — Complete plugin implementation example

## Provider Reference Notes

The `.resources/chat` package remains a product reference for platform surface area and API behavior, but not a literal v1 parity contract.

Per-provider notes that still matter in v1:

- **Telegram**: Bot API message and forum/thread behavior
- **Slack**: Events API, Web API delivery, command/action flows
- **Discord**: interaction acknowledgment deadlines
- **WhatsApp**: verify challenge and Cloud API rate-limit behavior
- **Teams**: Bot Framework activity model and service URL handling
- **Google Chat**: direct webhook versus Pub/Sub event shapes
- **GitHub**: comment threading and App installation semantics
- **Linear**: comment versus agent-session mode split

## Impact Analysis

| Component | Impact | Notes |
|-----------|--------|-------|
| `internal/bridges` | Modified | Bridge instance model and runtime assumptions must change from instance-scoped to provider-scoped |
| `internal/subprocess` | Modified | Bridge initialize payload must become provider-scoped |
| `internal/extension` | Modified | Host API authorization and runtime wiring can no longer assume one bridge instance per process |
| `internal/bridgesdk` | New | Shared provider SDK with hardening features |
| `extensions/bridges/*` | New | Eight provider extensions |
| `sdk/examples/telegram-reference` | Modified or superseded | Reference harness must either consume the shared SDK or be clearly marked as legacy-only scaffolding |
| `web/src/systems/bridges` | Modified | UI should eventually surface provider-required config and DM policy |

## Testing Approach

### Unit Tests

- `internal/bridgesdk` RPC and handshake behavior
- webhook defense helpers
- adapter-local dedup and batching behavior
- error classification and retry policy mapping
- provider-config decoding and validation
- per-provider message and interaction mapping

### Integration Tests

- provider runtime launch with multiple bridge instances owned by one extension
- ingress routing using distinct `bridge_instance_id` values under one provider process
- delivery and recovery flows under restart
- auth and degradation reporting
- DM policy enforcement
- rate-limit and retry behavior against mock provider APIs

### Verification Targets

- `make verify`
- focused integration suites for bridge runtime and provider conformance

## Development Sequencing

1. Redesign bridge runtime handshake and Host API around provider-scoped runtimes.
2. Add `provider_config` and provider metadata/schema support.
3. Implement `internal/bridgesdk` with webhook guards, dedup, batching, and error classification.
4. Migrate `telegram-reference` onto the shared SDK or replace it with a v1 provider conformance harness.
5. Implement Telegram first to validate the provider-scoped runtime model.
6. Implement Slack and Discord next to validate typed interaction events.
7. Implement WhatsApp, Teams, Google Chat, GitHub, and Linear.
8. Add provider conformance and multi-instance integration coverage.

## Architecture Decision Records

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md)
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md)
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md)
