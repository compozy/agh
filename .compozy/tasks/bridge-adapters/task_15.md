---
status: completed
title: "Implement the GitHub provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 15: Implement the GitHub provider extension

## Overview

Implement the GitHub provider to validate provider-scoped multi-instance behavior in a platform with App installations and comment-thread semantics. This task proves the architecture works beyond chat-style adapters and can still preserve the approved bridge v1 contract.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a GitHub provider using the shared provider runtime and provider-scoped Host API contract.
2. MUST support GitHub webhook ingress and outbound comment-style delivery behavior within the approved bridge v1 scope.
3. MUST use provider config and secret slots to distinguish GitHub App and PAT modes, including installation-oriented multi-instance behavior where applicable.
4. SHOULD validate the provider-scoped runtime against one of the clearest multi-tenant pressures in the approved design.
</requirements>

## Subtasks
- [x] 15.1 Create the GitHub provider runtime and manifest on top of `internal/bridgesdk`
- [x] 15.2 Implement GitHub webhook mapping for comments and related bridge-event identities
- [x] 15.3 Implement GitHub outbound delivery, App or PAT mode behavior, and state reporting
- [x] 15.4 Add conformance and multi-installation coverage for GitHub

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "GitHub", and "Operational Requirements". This task should keep GitHub inside the approved bridge v1 scope while using provider config for App or PAT mode differences.

### Relevant Files
- `internal/bridges/types.go` — GitHub needs the bridge identity model to represent comment threads and installation-scoped ownership
- `internal/extension/protocol/host_api.go` — GitHub runtime depends on the provider-scoped Host API methods
- `internal/extensiontest/bridge_adapter_harness.go` — GitHub should pass the shared conformance harness
- `internal/bridges/delivery_types.go` — GitHub outbound delivery must fit the approved bridge v1 contract

### Dependent Files
- `extensions/bridges/github/*` — New GitHub provider package tree should live here if the repo follows the TechSpec layout
- `internal/daemon/bridges_test.go` — GitHub can later serve as a multi-instance integration target
- `docs/plans/2026-04-15-bridge-adapters-design.md` — GitHub may inform later notes about installation-oriented multiplexing

### Reference Sources (.resources/)
- `.resources/chat/packages/adapter-github/src/index.ts` — **Primary reference**: Chat-SDK GitHub adapter; HMAC-SHA256 webhook verification (`x-hub-signature-256`), `issue_comment` and `pull_request_review_comment` events, PAT vs GitHub App modes, multi-tenant installation caching per `{owner}/{repo}`, review comment threading via `in_reply_to_id`, streaming accumulation strategy (GitHub rejects rapid edits with 422), emoji-to-reaction mapping
- `.resources/chat/packages/adapter-github/src/format.ts` — GitHub GFM format converter (cards → GFM markdown tables)
- `.resources/hermes/gateway/platforms/webhook.py` — Hermes generic webhook adapter with GitHub-specific HMAC validation and event-type filtering via `X-GitHub-Event` header; relevant for webhook handler design
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "github" --topic chat-sdk` for GitHub-specific patterns

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — GitHub is a primary justification for provider-scoped runtimes because of App installations
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — GitHub implementation must stay inside the approved bridge v1 surface

## Deliverables
- Production GitHub provider extension built on the shared bridge substrate
- GitHub webhook and delivery mapping within the approved bridge v1 contract
- Provider-specific conformance and multi-installation coverage for GitHub
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for GitHub provider behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] GitHub webhook payloads map issue or pull-request comment activity into the expected bridge routing identity
  - [x] GitHub provider config validation distinguishes App mode and PAT mode settings correctly
  - [x] GitHub outbound delivery mapping preserves the target context needed to post follow-up comments
- Integration tests:
  - [x] a provider-scoped GitHub runtime ingests webhook events for different owned bridge instances without process-level isolation
  - [x] GitHub outbound delivery posts comment-style responses and reports state transitions through the shared runtime path
  - [x] GitHub provider passes the shared conformance harness plus App-installation or multi-instance scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- GitHub validates the provider-scoped runtime under installation-oriented multi-instance pressure
- The bridge substrate works for comment-thread style providers as well as chat-style adapters
