---
status: pending
title: CLI Network Thread, Direct, Work, and Send Commands
type: backend
complexity: high
dependencies:
  - task_08
---

# Task 09: CLI Network Thread, Direct, Work, and Send Commands

## Overview

Update the agent-operable CLI for the new conversation model. This task adds thread, direct-room, work, and revised send commands while deleting legacy `--interaction-id` and `--kind direct` behavior.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and task_08 before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `bubbletea` only if TUI surfaces are touched, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for command names, flags, and output shapes.
- FOCUS ON CLI control plane; native tools and Host API are separate tasks.
- TESTS REQUIRED for JSON/jsonl/toon output where supported and for legacy flag rejection.
- NO WORKAROUNDS: do not keep hidden aliases for `--interaction-id` or `--kind direct`.
</critical>

<requirements>
- MUST add `agh network threads list/show/messages`.
- MUST add `agh network directs list/resolve/show/messages`.
- MUST add `agh network work` lookup/status if exposed by task_08.
- MUST revise `agh network send` to accept `--surface`, `--thread`, `--direct`, and `--work`.
- MUST remove `--interaction-id`.
- MUST reject `--kind direct`.
- MUST preserve `-o json`, `-o jsonl` for list/message streams where useful, and `-o toon` where existing CLI supports it.
- MUST preserve deterministic errors and raw claim-token rejection.
</requirements>

## Subtasks

- [ ] 9.1 Update CLI client methods for thread, direct, work, and send routes.
- [ ] 9.2 Add thread list/show/messages commands and structured output.
- [ ] 9.3 Add direct list/resolve/show/messages commands and structured output.
- [ ] 9.4 Update `network send` flags and validation.
- [ ] 9.5 Add CLI unit/integration tests for success paths and hard-cut rejections.

## Implementation Details

The CLI must remain usable by agents in scripts. Prefer stable JSON field names that match contract DTOs and avoid prose-only outputs for machine workflows.

### Relevant Files

- `internal/cli/network.go` - command definitions, flags, output.
- `internal/cli/client.go` - API/UDS client helpers.
- `internal/cli/network_test.go` - command validation and output tests.
- `internal/cli/network_client_test.go` - client behavior tests.
- `internal/cli/cli_integration_test.go` - end-to-end CLI coverage.

### Dependent Files

- `packages/site/content/runtime/cli-reference/network` - task_16 regenerates docs.
- `internal/skills/bundled/skills/agh-network/SKILL.md` - task_12 uses final CLI examples.
- `web/e2e/fixtures/*` - task_17 may use CLI flows for fixture seeding.

### Related ADRs

- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - CLI flag rename.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - send flags.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: CLI output should match contract DTOs so extensions and scripts can reuse examples.
- Agent manageability: CLI is a primary agent-operable surface for creating and inspecting network conversations.
- Config lifecycle: no new config keys; status output may show aggregate counts from task_07.

### Web/Docs Impact

- Web impact: no direct web change.
- Docs impact: CLI reference and examples must be regenerated/updated in task_16.

## Deliverables

- Updated CLI commands and flags.
- Structured output for thread/direct/work/read/send flows.
- Negative tests for deleted flags and kinds.
- CLI integration evidence against daemon-served routes where existing patterns support it.

## Tests

- Unit tests:
  - [ ] `agh network send --surface thread --thread ...` builds the expected payload.
  - [ ] `agh network send --surface direct --direct ... --work ...` builds the expected payload.
  - [ ] `--interaction-id` is rejected.
  - [ ] `--kind direct` is rejected.
  - [ ] JSON/jsonl/toon output shapes match the contract.
- Integration tests:
  - [ ] CLI list/show/messages flows match HTTP/UDS responses for the same persisted state.
  - [ ] Direct resolve through CLI is idempotent for the same peer pair.
  - [ ] Raw claim-token payloads are rejected.
- Test coverage target: >=80% for touched CLI code.
- All tests must pass.

## Success Criteria

- Agents can manage public threads, direct rooms, work lookup, and send flows through CLI.
- Legacy CLI affordances fail loudly.
- CLI examples are ready for prompt and docs tasks.
