---
status: completed
title: "Extend bridge core models, persistence, and provider manifests"
type: backend
complexity: critical
dependencies: []
---

# Task 01: Extend bridge core models, persistence, and provider manifests

## Overview

Bring the daemon-owned bridge model up to the shape approved in the TechSpec before any runtime work begins. This task introduces `provider_config`, provider-declared secret/config metadata, DM policy, and structured degradation data in the core bridge domain and persistence layers.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend the daemon-owned bridge domain in `internal/bridges` so `BridgeInstance` can carry provider-owned configuration separately from `delivery_defaults`, as described in the TechSpec "Data Model Changes" section.
2. MUST update global persistence and validation so bridge instances, provider metadata, and structured degradation fields round-trip through storage without overloading opaque transport-only blobs.
3. MUST extend extension manifest parsing and validation so bridge-capable providers can declare required secret slots and optional config schema/version hints in static provider metadata.
4. SHOULD model DM policy and structured degradation reasons as typed, reusable bridge concepts instead of provider-local strings.
</requirements>

## Subtasks
- [x] 1.1 Add `provider_config`, DM policy, and structured degradation fields to the bridge core types and validators
- [x] 1.2 Update global DB schema and CRUD helpers for the new bridge-instance and provider metadata shape
- [x] 1.3 Extend manifest parsing and validation for bridge secret-slot declarations and config schema hints
- [x] 1.4 Add unit and persistence coverage for the new bridge model and manifest contract

## Implementation Details

Follow the TechSpec sections "Data Model Changes", "Provider Manifest", "Secret Slots", and "Operational Requirements". This task should stop at model, manifest, and persistence concerns; it should not redesign the subprocess runtime or Host API yet.

### Relevant Files
- `internal/bridges/types.go` — Current bridge model lacks `provider_config`, DM policy, and structured degradation fields
- `internal/store/globaldb/global_db_bridge.go` — Bridge instance persistence must round-trip the expanded schema
- `internal/extension/manifest.go` — Bridge manifest metadata is currently limited to `platform` and `display_name`
- `internal/api/contract/bridges.go` — Shared transport payloads will need to map the expanded bridge instance shape later

### Dependent Files
- `internal/subprocess/handshake.go` — Later runtime negotiation depends on the expanded bridge instance model
- `internal/extension/host_api_bridges.go` — Host API authorization and state reporting later consume the new fields
- `web/src/systems/bridges/types.ts` — Web bridge management later needs typed access to provider metadata and config

### Reference Sources (.resources/)
- `.resources/chat/packages/chat/src/types.ts` — Chat-SDK core `Adapter` interface with `lockScope` (DM policy analog), per-adapter config shape, and `persistMessageHistory` flag; primary reference for how adapters declare capabilities
- `.resources/goclaw/internal/channels/channel.go` — GoClaw `Channel` interface with `DMPolicy` and `GroupPolicy` enums (`"pairing"`, `"allowlist"`, `"open"`, `"disabled"`); directly informs the DM policy model
- `.resources/hermes/gateway/config.py` — Hermes `PlatformConfig` dataclass: `token`, `api_key`, `home_channel`, `extra` dict, `reply_to_mode`; reference for `provider_config` shape separation from delivery defaults
- `.resources/openclaw/src/channels/plugins/types.core.ts` — OpenClaw `ChannelSecurityDmPolicy` discriminated union (`"pairing"`, `"open"`, `"implicit"`); alternative DM policy design
- `.resources/hermes/agent/error_classifier.py` — Hermes `FailoverReason` enum (`auth`, `rate_limit`, `timeout`, `overloaded`, etc.) with `ClassifiedError` dataclass; informs structured degradation reason model
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Indexed codebase vault; use `kb inspect` and `kb search` for symbol lookups

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Explains why `BridgeInstance` must remain daemon-owned while runtimes multiplex multiple instances
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Defines DM policy and provider configuration as v1 concerns

## Deliverables
- Expanded bridge core types and validation for provider-owned config, DM policy, and degradation metadata
- Updated global DB schema and persistence helpers for the new bridge-instance shape
- Extended bridge provider manifest metadata for secret slots and config schema hints
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for persistence and manifest validation behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `BridgeInstance.Validate()` accepts valid `provider_config` payloads and rejects malformed JSON independently from `delivery_defaults`
  - [ ] DM policy validation accepts only the supported v1 values and rejects unknown modes
  - [ ] bridge manifest validation requires `bridge.platform`, `bridge.display_name`, and validates secret-slot declarations for bridge adapters
  - [ ] structured degradation reasons normalize and reject empty or unsupported values where the model requires them
- Integration tests:
  - [ ] a persisted bridge instance with `provider_config` and `delivery_defaults` round-trips through `globaldb` unchanged
  - [ ] bridge providers with declared secret slots and config schema hints round-trip through manifest loading and validation
  - [ ] legacy bridge instances without provider config still load under the new schema contract if the TechSpec allows empty config
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The daemon has one authoritative persisted bridge model for provider-scoped runtimes
- Provider configuration is structurally separated from outbound delivery defaults
