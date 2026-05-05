---
status: pending
title: Native Agent Tools and Hosted Tool Schemas
type: backend
complexity: high
dependencies:
  - task_08
---

# Task 10: Native Agent Tools and Hosted Tool Schemas

## Overview

Update AGH native tools so agents can operate on public threads, direct rooms, and work metadata without shelling out to CLI. This task aligns tool descriptors, hosted/MCP schemas, and daemon dispatch with the new contract model.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and task_08 before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for tool IDs and schema rules.
- FOCUS ON native and hosted tool surfaces; extension Host API is task_11.
- TESTS REQUIRED for closed schemas, legacy rejection, raw-token rejection, dispatch, and parity with HTTP validation.
- NO WORKAROUNDS: do not overload old inbox or peer tools to pretend they are thread/direct tools.
</critical>

<requirements>
- MUST update `agh__network_send` to accept `surface`, `thread_id`, `direct_id`, and `work_id`.
- MUST reject `interaction_id` through closed schemas.
- MUST add `agh__network_threads`, `agh__network_thread_messages`, `agh__network_directs`, `agh__network_direct_resolve`, `agh__network_direct_messages`, and `agh__network_work`.
- MUST preserve `additionalProperties:false` behavior where existing tool schemas enforce it.
- MUST preserve raw claim-token rejection in tool inputs, results, prompts, and logs.
- MUST ensure hosted/MCP schemas expose the same shapes as native tools.
</requirements>

## Subtasks

- [ ] 10.1 Update native tool IDs and descriptor registration.
- [ ] 10.2 Update `agh__network_send` schema and handler.
- [ ] 10.3 Add thread/direct/work tool schemas and daemon dispatch.
- [ ] 10.4 Align hosted/MCP tool descriptors with native schemas.
- [ ] 10.5 Add schema, dispatch, parity, and redaction tests.

## Implementation Details

Tool descriptions must make direct-room visibility explicit: restricted to the two room peers plus runtime/audit access, not cryptographic privacy.

### Relevant Files

- `internal/tools/builtin_ids.go` - tool IDs.
- `internal/tools/builtin/network.go` - network tool descriptors.
- `internal/tools/builtin/toolsets.go` - toolset registration.
- `internal/daemon/native_tools.go` - daemon tool handlers.
- `internal/tools/builtin/builtin_test.go` - descriptor tests.
- `internal/tools/native_test.go` - native tool validation tests.
- `internal/daemon/native_tools_test.go` - daemon handler tests.
- `internal/daemon/tools_transport_parity_test.go` - transport parity where applicable.

### Dependent Files

- `internal/skills/bundled/skills/agh-network/SKILL.md` - task_12 prefers native tools where available.
- `packages/site/content/runtime/core/network/*` - task_16 documents tool usage.
- `web/e2e/fixtures/runtime-seed.ts` - task_17 may use native tool scenarios.

### Related ADRs

- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - tool field names.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - send schema.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: hosted tool descriptors should remain stable and reusable by MCP sidecars.
- Agent manageability: native tools are the primary agent path for network operations.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: no direct web code change.
- Docs impact: task_16 documents native/hosted tool schemas and examples.

## Deliverables

- Updated and new native network tools.
- Hosted/MCP schema alignment.
- Closed-schema rejection of legacy fields.
- Tests for dispatch, validation, parity, and redaction.

## Tests

- Unit tests:
  - [ ] Tool schemas reject `interaction_id`.
  - [ ] Tool schemas reject missing matching container fields.
  - [ ] Tool schemas reject raw `claim_token` inputs.
  - [ ] Tool descriptions do not teach `kind:"direct"` or cryptographic privacy.
- Integration tests:
  - [ ] `agh__network_direct_resolve` creates or returns the same direct room idempotently.
  - [ ] `agh__network_send` matches HTTP validation for thread and direct surfaces.
  - [ ] Work lookup returns the same state as HTTP/UDS.
  - [ ] Hosted/MCP descriptor output matches native schemas.
- Test coverage target: >=80% for touched tool/daemon packages.
- All tests must pass.

## Success Criteria

- Agents can inspect and send network messages through native tools with no legacy ambiguity.
- Tool schemas fail closed for deleted fields and unsafe token material.
- Prompt/skill tasks can rely on final tool names and schemas.
