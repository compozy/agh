# Design: Bridge Adapters V1

**Date:** 2026-04-15
**Status:** Approved
**Authors:** Pedro Nauck + Codex

## Summary

This design replaces the original "full chat-sdk parity" bridge-adapter direction with a bridge v1 that is implementable in the current AGH architecture without relying on opaque metadata or contradictory runtime assumptions.

The approved direction is:

- one provider-scoped subprocess per bridge provider extension
- many `BridgeInstance` records multiplexed inside that provider runtime
- per-instance `provider_config` stored separately from `delivery_defaults`
- a bridge v1 protocol focused on strong conversational flows
- typed optional interaction events for `commands`, `actions`, and `reactions`
- hardened operational behavior in the shared bridge SDK

The goal is not to shrink bridges into a toy system. The goal is to define a production-credible first version that fits AGH's core architecture and leaves room for richer interaction families later.

## Problem

The previous bridge-adapter tech spec had three structural contradictions:

- it assumed multi-tenant providers while the current runtime only supports one enabled bridge instance per extension session
- it promised full chat-sdk parity while the current bridge transport only models conversational message delivery
- it required provider-specific runtime configuration without defining where that configuration lives

Those are architecture decisions, not implementation details. Leaving them implicit would force workaround-shaped code into `internal/daemon`, `internal/bridges`, and every adapter implementation.

## Approved Direction

### 1. Runtime ownership

Each platform provider remains one extension under `extensions/bridges/<platform>/`, but the daemon runs one subprocess per provider extension, not one subprocess per bridge instance.

`BridgeInstance` remains the daemon-owned operational object for:

- routing
- secret bindings
- status and health
- delivery defaults
- per-tenant/provider configuration

The provider runtime multiplexes any number of enabled bridge instances that belong to that provider.

### 2. Configuration model

`BridgeInstance` gains a dedicated provider-owned config payload, for example `provider_config`.

That payload holds adapter-specific operational configuration such as:

- provider mode (`comments`, `agent-sessions`, `bot`, `app`)
- public webhook URL information
- enterprise API base URLs
- DM policy
- debounce tuning
- provider-specific tenant identifiers or flags

`delivery_defaults` remains narrow and only governs canonical outbound targeting behavior.

Extension manifests remain static provider metadata. They do not carry per-instance runtime settings.

### 3. Bridge v1 protocol scope

Bridge v1 is not "everything in chat-sdk". It is:

- inbound conversational messages
- outbound conversational delivery
- attachments
- threading
- typed delivery targets
- streaming textual updates
- edit/delete semantics
- health and auth reporting

In addition, v1 includes typed optional interaction families for:

- command invocation
- action submission
- reaction events

The following stay out of scope for v1:

- modal lifecycle orchestration
- ephemeral message portability
- typing indicators as a cross-platform contract
- approval UI flows
- multi-lane reasoning vs final-answer rendering
- credential pool rotation

### 4. Shared SDK responsibilities

The shared bridge SDK or provider runtime scaffold must own the operational behaviors that are common across providers:

- webhook signature verification hooks
- webhook request guards
- adapter-local deduplication
- inbound batching/debounce
- platform error classification
- retry/backoff decisions for retryable failures
- structured degradation reporting

This avoids repeating production hardening logic in every provider adapter.

## Approved Operational Additions

### 1. Webhook defenses

Every provider webhook endpoint should include:

- HTTP method validation
- content-type validation where applicable
- body size limits
- fixed-window or token-bucket rate limiting
- in-flight concurrency limits

Signature verification remains mandatory, but it is not the only guard.

### 2. Adapter-local dedup

Providers should maintain a short-lived in-memory dedup cache keyed by platform delivery identity before forwarding to AGH.

This does not replace daemon-level `IngestDedupRecord`; it reduces unnecessary daemon round-trips caused by platform retries and duplicate webhook deliveries.

### 3. Message batching

Providers may coalesce short bursts of inbound user messages before ingestion using a small debounce window.

This behavior must be provider-configurable per bridge instance and should preserve sender, thread, and ordering semantics.

### 4. Error classification

The shared SDK should classify outbound provider errors into actionable classes such as:

- `auth`
- `rate_limit`
- `timeout`
- `transient`
- `permanent`

Recovery behavior is then defined per class:

- retry with backoff
- mark `Degraded`
- mark `AuthRequired`
- abort without retry

### 5. DM policy

Direct-message authorization is explicit bridge-instance configuration, not ad hoc provider code.

Supported v1 policy modes:

- `open`
- `allowlist`
- `pairing`

Providers enforce the configured policy before creating or resuming daemon-side routes.

## Consequences

### Positive

- the bridge design now matches the intended multi-tenant providers
- v1 scope is ambitious but implementable
- provider-specific config has a clear home
- production hardening behavior becomes a shared runtime concern instead of copy-paste adapter logic

### Negative

- bridge runtime and host API contracts must evolve; "reuse as-is" is no longer accurate
- the spec no longer claims full parity with every chat-sdk feature in v1
- provider manifests need a stronger metadata contract around secret slots and config schema hints

## Follow-up

The next documentation changes should update:

- `.compozy/tasks/bridge-adapters/_techspec.md`
- `adr-001.md`
- `adr-002.md`
- `adr-003.md`

to reflect this approved design.
