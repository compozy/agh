---
status: completed
title: "Build shared internal/bridgesdk runtime core and ingress hardening"
type: backend
complexity: critical
dependencies:
  - task_02
  - task_03
  - task_04
---

# Task 05: Build shared internal/bridgesdk runtime core and ingress hardening

## Overview

Create the shared substrate that every bridge provider will import instead of copying logic from `telegram-reference`. This task centralizes provider runtime boot, instance cache management, webhook defense, local dedup, batching, error classification, and shutdown/health helpers.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST create `internal/bridgesdk` as the shared provider runtime package described in ADR-001 and the TechSpec "Shared SDK Requirements" section.
2. MUST include reusable webhook request guards covering method checks, content-type checks where applicable, body-size limits, rate limiting, and in-flight concurrency limits in addition to signature-verification hooks.
3. MUST include adapter-local dedup, optional inbound batching/debounce, and actionable error classification with recovery mapping for `auth`, `rate_limit`, `timeout`, `transient`, and `permanent`.
4. SHOULD provide provider runtime helpers for instance-cache synchronization, health reporting, graceful shutdown, and delivery acknowledgments so providers only implement platform-specific behavior.
</requirements>

## Subtasks
- [x] 5.1 Create the shared bridge SDK runtime scaffold and provider-owned instance cache
- [x] 5.2 Add reusable webhook server guards, dedup cache, and batching helpers
- [x] 5.3 Add provider error classification and retry or status-mapping helpers
- [x] 5.4 Add health, shutdown, and delivery-ack helpers plus conformance-friendly test seams

## Implementation Details

Follow the TechSpec sections "Shared SDK Requirements", "Operational Requirements", "Webhook Defense", "Adapter-Local Dedup", "Inbound Batching", and "Error Classification". This task should build the shared substrate only; it should not implement any platform-specific provider behavior yet.

### Relevant Files
- `sdk/examples/telegram-reference/main.go` — Current reference adapter contains bootstrap logic that should move into shared runtime primitives
- `internal/extensiontest/bridge_adapter_harness.go` — Conformance harness needs stable hooks into the shared runtime behavior
- `internal/extension/protocol/host_api.go` — Shared SDK Host API client must speak the provider-scoped bridge Host API methods
- `internal/subprocess/handshake.go` — Shared runtime initialization must align with the provider-scoped handshake contract

### Dependent Files
- `sdk/examples/telegram-reference/README.md` — Reference documentation later needs to point at the shared SDK or legacy replacement path
- `internal/extension/telegram_reference_integration_test.go` — Existing integration tests will later validate the shared runtime behavior
- `extensions/bridges/*` — All real provider binaries later depend on this package

### Reference Sources (.resources/)

**Webhook defense and ingress hardening:**
- `.resources/openclaw/src/plugin-sdk/webhook-ingress.ts` — OpenClaw 6-layer webhook defense: method → content-type → body size → rate limit → in-flight → anomaly tracking; primary reference for webhook guard design
- `.resources/openclaw/src/plugin-sdk/webhook-memory-guards.ts` — OpenClaw `FixedWindowRateLimiter` and bounded counter implementations; reference for rate-limiting helpers
- `.resources/hermes/gateway/platforms/webhook.py` — Hermes webhook adapter: HMAC validation, idempotency cache, rate limiting, async fire-and-forget; complete production webhook handler reference

**Dedup and batching:**
- `.resources/hermes/gateway/platforms/helpers.py` — Hermes `TextBatchAggregator` (debounce 600ms, split threshold 4000 chars) and `MessageDeduplicator` (TTL-based, 2000 entries); direct implementation reference for batching and dedup
- `.resources/goclaw/internal/channels/channel.go` — GoClaw dedup via `sync.Map` and history flushing patterns

**Error classification and retry:**
- `.resources/hermes/agent/error_classifier.py` — Hermes `FailoverReason` enum and `ClassifiedError` dataclass with pattern-based detection for billing, rate limit, context overflow, auth; primary error classification reference
- `.resources/hermes/agent/retry_utils.py` — Hermes `jittered_backoff()` with decorrelated seeding; reference for retry helpers
- `.resources/openclaw/src/infra/retry.ts` — OpenClaw composable retry with `shouldRetry`, `retryAfterMs` (Retry-After header), and `onRetry` hooks
- `.resources/openclaw/src/infra/backoff.ts` — OpenClaw `BackoffPolicy` with factor and jitter
- `.resources/openclaw/src/infra/errors.ts` — OpenClaw `ErrorKind` classification (`refusal`, `timeout`, `rate_limit`, `context_length`)
- `.resources/goclaw/internal/providers/retry.go` — GoClaw Go retry implementation with jitter and `IsRetryableError()`; closest Go-native reference

**Runtime and lifecycle:**
- `.resources/goclaw/internal/channels/channel.go` — GoClaw `Channel` and `StreamingChannel` interfaces with `Start/Stop/Send/IsRunning` lifecycle; Go-native channel runtime reference
- `.resources/goclaw/internal/providers/adapter_registry.go` — GoClaw `AdapterRegistry` with `Register/Get` factory pattern
- `.resources/hermes/gateway/platforms/base.py` — Hermes `BasePlatformAdapter` with `connect/disconnect/send/send_typing/edit_message` interface; shows the adapter abstraction layer

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Defines the SDK surface and provider/runtime split
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Defines the ingress hardening and error-handling behavior the SDK must own

## Deliverables
- New shared `internal/bridgesdk` runtime package with provider-scoped boot and instance-cache helpers
- Reusable webhook guards, local dedup, batching, and error-classification helpers
- Shared health, shutdown, and delivery-ack abstractions for provider runtimes
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for runtime boot, ingress hardening, and retry behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] webhook guards reject unsupported methods, oversized bodies, and invalid content types before reaching provider handlers
  - [x] adapter-local dedup suppresses repeated idempotency keys within TTL and releases them after expiry
  - [x] batching coalesces a short burst of messages under one routing identity while preserving ordering
  - [x] error classification maps representative provider errors into the expected recovery classes
- Integration tests:
  - [x] a provider runtime built on `internal/bridgesdk` boots against the provider-scoped handshake and can ingest through the Host API client
  - [x] ingress hardening rejects invalid requests without invoking downstream provider mapping
  - [x] classified auth and rate-limit failures produce the expected retry or status-transition behavior in the shared runtime helpers
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Shared bridge runtime concerns live in one reusable package instead of provider-local copies
- Future providers can focus on platform-specific mapping rather than runtime hardening
