---
status: completed
title: Add Agent-Operable CLI Commands
type: backend
complexity: high
dependencies:
  - task_11
---

# Task 12: Add Agent-Operable CLI Commands

## Overview

Add CLI commands that let agents and operators inspect and manage Soul, Heartbeat, and session health through the same UDS/API behavior exposed by task_11. This task completes the agent-manageable surface for local automation and structured JSON workflows.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, every ADR, and existing CLI command patterns before editing.
- REFERENCE TECHSPEC for command names, JSON output, errors, workspace scoping, and naming exclusions.
- FOCUS ON WHAT must be operable: Soul authoring, Heartbeat authoring/status, session health/status/inspect, and explicit Soul refresh.
- MINIMIZE CODE in task notes; implement through existing CLI client/core route patterns.
- TESTS REQUIRED for human output, `--json`, `--workspace`, errors, CAS conflicts, and no forbidden command names.
- NO WORKAROUNDS: do not add `agh session heartbeat`; use `agh session health`, `agh session status`, and `agh session inspect`.
</critical>

<requirements>
- MUST activate `agh-code-guidelines`, `golang-pro`, and `bubbletea` only if TUI behavior is touched.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST add `agh agent soul` subcommands for inspect/validate/write/delete/history/rollback as specified.
- MUST add `agh session soul refresh` for explicit session snapshot refresh.
- MUST add `agh agent heartbeat` subcommands for inspect/validate/write/delete/history/rollback/status.
- MUST add `agh session health`, `agh session status`, and `agh session inspect` commands for runtime health and wake eligibility.
- MUST support deterministic `--json`, `--workspace`, exit codes, redacted errors, and machine-readable conflict diagnostics.
- MUST avoid direct filesystem mutation from CLI; commands must call managed API/UDS behavior.
</requirements>

## Subtasks
- [x] 12.1 Add CLI command tree for `agh agent soul` and `agh session soul refresh`.
- [x] 12.2 Add CLI command tree for `agh agent heartbeat`.
- [x] 12.3 Add CLI commands for `agh session health`, `agh session status`, and `agh session inspect`.
- [x] 12.4 Wire commands through existing clients to UDS/API routes without duplicating service logic.
- [x] 12.5 Add CLI tests for JSON/human output, workspace scoping, errors, and CAS conflicts.
- [x] 12.6 Regenerate CLI docs metadata if required by the existing CLI doc pipeline.

## Implementation Details

Preserve existing CLI conventions for command construction, client access, `--json`, workspace flags, redaction, and errors. The CLI is an agent-operable management surface, not a direct file editor.

### Relevant Files
- `internal/cli/agent.go` - agent command tree destination.
- `internal/cli/session.go` - session command tree destination.
- `internal/cli/client.go` - UDS/HTTP client behavior.
- `internal/cli/doc.go` - CLI docs metadata if command docs are generated.
- `internal/api/contract/` - request/response types from task_10.
- `internal/api/udsapi/` - route behavior consumed by CLI.

### Dependent Files
- `internal/cli/*_test.go` - CLI command behavior and output tests.
- `packages/site/content/runtime/cli-reference/` - generated or updated by task_15 after CLI behavior lands.
- `.compozy/tasks/agent-soul/task_15.md` - docs update depends on final command behavior.
- `.compozy/tasks/agent-soul/task_17.md` - QA execution uses CLI flows as primary real-scenario evidence.

### Related ADRs
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - requires CLI manageability for Soul.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - requires CLI manageability for Heartbeat and session health.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - forbids confusing session health with authored Heartbeat policy.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: CLI commands become stable local automation examples for extensions, tools, SDK docs, and agent-authored capabilities.
- Agent manageability: completes CLI inspect/manage/status operations with structured JSON and deterministic errors.
- Config lifecycle: commands must surface disabled/config-bound states and effective config digests without adding new keys.

### Web/Docs Impact
- Web impact: no UI change; generated contract consumers remain task_14 responsibility.
- Docs impact: task_15 must update CLI reference and examples after these commands are implemented.

## Deliverables
- `agh agent soul` command group with managed authoring and inspect verbs.
- `agh session soul refresh` explicit refresh command.
- `agh agent heartbeat` command group with managed authoring, inspect, and status verbs.
- `agh session health`, `agh session status`, and `agh session inspect` commands.
- CLI tests for structured JSON, human output, workspace scoping, deterministic errors, and forbidden naming.
- CLI docs metadata updates if required by the repository pipeline.

## Tests
- Unit tests:
  - [x] `agh agent soul inspect --json` returns contract-shaped redacted data.
  - [x] `agh agent soul write` sends `expected_digest` and reports stale conflicts deterministically.
  - [x] `agh agent heartbeat status --json` includes policy, config digest, wake status, and session health where applicable.
  - [x] `agh session health --json` and `agh session inspect --json` use closed state/reason values.
  - [x] No `agh session heartbeat` command exists in command help or docs metadata.
  - [x] `--workspace` scopes every command consistently.
- Integration tests:
  - [x] CLI commands work against an isolated daemon through UDS.
  - [x] Human and JSON outputs are redacted and stable for automation.
  - [x] CLI routes return the same semantic errors as HTTP/UDS tests from task_11.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec.md` - agent manageability and route parity requirements.
- `_techspec_soul.md` - Soul CLI requirements.
- `_techspec_heartbeat.md` - Heartbeat and session health CLI requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - gateway manageability precedent.
- `.resources/paperclip/cli/src/commands/heartbeat-run.ts:85-103` - CLI heartbeat contrast.
- `.resources/openclaw/docs/gateway/protocol.md:313-438` - protocol command/status precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agents can manage authored context and inspect session health through CLI JSON without direct file writes.
- No command naming confuses session health with `HEARTBEAT.md` policy.
