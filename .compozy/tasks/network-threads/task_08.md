---
status: pending
title: Public Contracts, HTTP/UDS Parity, and Codegen
type: backend
complexity: critical
dependencies:
  - task_05
  - task_06
---

# Task 08: Public Contracts, HTTP/UDS Parity, and Codegen

## Overview

Expose the new conversation model through shared public contracts, HTTP routes, UDS routes, agent-native UDS routes, and generated OpenAPI/TypeScript artifacts. This task makes public ingress reject legacy fields and keeps transport parity strict.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and the `agh-contract-codegen-coship` skill before editing contracts.
- ACTIVATE `agh-contract-codegen-coship`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for route map, payload shapes, and validation.
- FOCUS ON contracts and transport parity; CLI/native/extensions/web implementation are later tasks.
- TESTS REQUIRED for HTTP/UDS parity, agent-native UDS paths, codegen drift, and legacy rejection.
- NO WORKAROUNDS: do not accept `interaction_id`, `--interaction-id` equivalents, or `kind:"direct"` through public contracts.
</critical>

<requirements>
- MUST update `NetworkSendRequest` and response payloads with `surface`, `thread_id`, `direct_id`, and `work_id`.
- MUST add thread list/show/messages endpoints.
- MUST add direct list/resolve/show/messages endpoints.
- MUST add work lookup endpoint.
- MUST delete or replace old primary flat message paths as specified by the TechSpec.
- MUST update `/agent/channels/:channel/send`, `/agent/channels/:channel/recv`, and `/agent/channels/reply` semantics if they currently encode `KindDirect` or `InteractionID`.
- MUST keep HTTP and UDS route registration in parity and fail tests if parity drifts.
- MUST run `make codegen` or the approved generator path and include generated `openapi/agh.json` plus generated web TypeScript outputs.
</requirements>

## Subtasks

- [ ] 8.1 Update contract DTOs and validation for send, thread, direct, work, and message payloads.
- [ ] 8.2 Add shared core handlers for thread/direct/work read paths and direct resolve.
- [ ] 8.3 Register equivalent HTTP and UDS routes.
- [ ] 8.4 Update agent-native UDS channel send/receive/reply behavior.
- [ ] 8.5 Regenerate OpenAPI and generated TypeScript consumers with the approved codegen path.
- [ ] 8.6 Add parity, contract, and legacy rejection tests.

## Implementation Details

Generated files must be regenerated, not hand-edited. The current worktree may already contain unrelated contract/codegen edits; do not revert or overwrite unrelated changes.

### Relevant Files

- `internal/api/contract/contract.go` - shared DTOs.
- `internal/api/core/interfaces.go` - store/runtime dependencies.
- `internal/api/core/network.go` - core network handlers.
- `internal/api/core/network_details.go` - network details/read paths.
- `internal/api/core/agent_channels.go` - agent-native UDS channel behavior.
- `internal/api/httpapi/routes.go` - HTTP route registration.
- `internal/api/udsapi/routes.go` - UDS route registration.
- `internal/api/spec/spec.go` - OpenAPI source.
- `openapi/agh.json` - generated OpenAPI output.
- `web/src/generated/agh-openapi.d.ts` - generated web contract output.

### Dependent Files

- `web/src/systems/network/adapters/network-api.ts` - task_13 consumes generated types.
- `internal/cli/network.go` - task_09 consumes routes.
- `internal/tools/builtin/network.go` - task_10 consumes DTO shapes.
- `internal/extension/contract/host_api.go` - task_11 mirrors DTOs where practical.

### Related ADRs

- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - public field rename.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - public validation.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: shared DTOs should be reusable by Host API, SDK, tools, and docs generation.
- Agent manageability: HTTP and UDS become the public management surfaces for agents and operators.
- Config lifecycle: no new config keys; status payloads may add aggregate counts only.

### Web/Docs Impact

- Web impact: generated web types and route operations must be regenerated in this task. The generated DTOs feed the query-key shape and component file map prescribed in `_design.md` §11; field naming must align with that map so task_13 consumers do not need to reshape generated types.
- Docs impact: generated API docs and CLI docs are updated in task_16, but OpenAPI must be current here.

## Deliverables

- Updated public contract DTOs.
- HTTP and UDS routes for threads, direct rooms, work, and send.
- Updated agent-native UDS channel behavior.
- Regenerated OpenAPI and TypeScript contract artifacts.
- Parity and validation tests.

## Tests

- Unit tests:
  - [ ] Contract decoding rejects `interaction_id`.
  - [ ] Contract decoding rejects `kind:"direct"`.
  - [ ] Send validation enforces matching `surface` and container fields.
  - [ ] `receipt` and `trace` require `work_id`.
- Integration tests:
  - [ ] HTTP and UDS expose the same thread/direct/work routes.
  - [ ] Agent-native UDS send/receive/reply no longer uses direct kind or interaction ID.
  - [ ] Direct resolve can create or return an existing direct room through HTTP and UDS.
  - [ ] `make codegen-check` passes after regeneration.
- Test coverage target: >=80% for touched API packages.
- All tests must pass.

## Success Criteria

- Public API surfaces expose the new conversation model with strict hard-cut validation.
- HTTP, UDS, generated OpenAPI, and generated web types are in sync.
- Later CLI, native tools, extensions, and web tasks can build on the same contracts.
