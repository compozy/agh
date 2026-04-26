---
status: pending
title: Agent Self And Channel Verbs
type: backend
complexity: high
dependencies:
  - task_04
  - task_05
---

# Task 06: Agent Self And Channel Verbs

## Overview
Expose the first agent-facing self and channel commands over UDS/CLI. Agents should be able to inspect their identity/context and participate in local channels through `agh me`, `agh me context`, and `agh ch ...` without operator-only flags or shell snippets.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-007, and ADR-010 before changing channel verbs
- REFERENCE TECHSPEC for command names and endpoint mapping
- FOCUS ON "WHAT" - agent self/context and local channel operations
- MINIMIZE CODE - reuse existing network services and CLI client patterns
- TESTS REQUIRED - CLI, UDS, long-poll/stream behavior, and identity checks must be covered
- NO WORKAROUNDS - do not hard-code delivery shell snippets instead of first-class commands
</critical>

<requirements>
- MUST add `/agent/me` and `/agent/context` UDS endpoints and `agh me`, `agh me context` commands.
- MUST add `/agent/channels`, recv, send, and reply endpoints plus `agh ch list`, `agh ch recv --wait`, `agh ch send`, and `agh ch reply --to-message`.
- MUST infer caller identity through task_05 and use caller session/workspace/channel context.
- MUST support stable `-o json` output and `-o jsonl` for wait/stream operations.
- MUST preserve existing operator `agh network ...` commands.
- MUST not introduce cross-daemon swarm or broad network protocol changes in the MVP.
</requirements>

## Subtasks
- [ ] 6.1 Add UDS handlers for `/agent/me` and `/agent/context`.
- [ ] 6.2 Add UDS handlers for agent channel list/recv/send/reply using existing network service.
- [ ] 6.3 Add `agh me` and `agh me context` commands with JSON output.
- [ ] 6.4 Add `agh ch` commands with identity inference and JSON/JSONL output.
- [ ] 6.5 Add CLI/UDS/network tests for valid identity, denied identity, reply-by-message, waiting receive, and operator command regression.
- [ ] 6.6 Run contract/codegen/web checks if endpoint specs or generated DTOs change.

## Implementation Details
Prefer thin command wrappers over existing network service methods. Agent channel verbs should bind caller identity automatically; operator `network send/inbox/channels` remains explicit and unchanged.

### Relevant Files
- `internal/api/udsapi/routes.go` - route registration.
- `internal/api/udsapi/network_test.go` - UDS network behavior tests.
- `internal/cli/network.go` - operator network command precedent.
- `internal/cli/client.go` - UDS client methods.
- `internal/network/*` - channel, inbox, and envelope logic.
- `internal/daemon/task_runtime.go` - existing network channel validator wiring.
- `.resources/multica/packages/core/inbox/ws-updaters.ts` - reference for inbox update semantics.
- `.resources/claude-code/commands.ts` - reference for command surface organization.
- `.resources/paperclip/doc/execution-semantics.md` - reference for explicit execution semantics.

### Dependent Files
- `packages/site/content/runtime/cli-reference/` - task_16 adds CLI references.
- `packages/site/content/runtime/core/network/*` - task_16 documents channel semantics.
- `internal/api/spec/spec.go` - if HTTP/OpenAPI parity is added.

### Related ADRs
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - canonical CLI names.
- [ADR-007: Minimal Network Evolution for Local Autonomy](adrs/adr-007.md) - local network MVP boundary.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - operator network flows remain explicit.

## Deliverables
- Agent self/context UDS endpoints and CLI commands.
- Agent channel UDS endpoints and CLI commands.
- Tests for agent identity, channel send/receive/reply, output formats, and operator command regression.
- Unit tests with 80%+ coverage for command/handler helpers **(REQUIRED)**.
- Integration tests through UDS/network service for channel flows **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] `agh me -o json` returns self/session/workspace identity for a valid caller.
  - [ ] `agh me context -o json` returns stable section ordering from task_04.
  - [ ] `agh ch send` rejects missing channel/body and invalid caller identity.
  - [ ] `agh ch reply --to-message` resolves reply metadata without requiring source session flags.
  - [ ] JSONL receive output emits one valid object per message.
- Integration tests:
  - [ ] A local session sends and receives through a channel using only agent env identity.
  - [ ] `agh ch recv --wait` wakes on a new message without arbitrary sleep in the implementation.
  - [ ] Existing `agh network send --session ...` continues to work as an operator command.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- User-started agents can inspect self/context and communicate locally without operator-only flags.
- Network scope remains local-first and minimal.
