---
status: completed
title: "ACP SDK Upgrade and Config Options"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 6: ACP SDK Upgrade and Config Options

## Overview
This task upgrades ACP support so active sessions use ACP `configOptions` as the source of truth for model and reasoning controls. It keeps the legacy `set_model` path only as a fallback where config options are absent.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST upgrade `github.com/coder/acp-go-sdk` from `v0.6.3` to `v0.12.2` using `go get`.
- MUST produce `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md` before migrating code, enumerating every changed ACP symbol AGH uses.
- MUST capture ACP `configOptions` from `session/new`, `session/load`, and `config_option_update`.
- MUST prefer `session/set_config_option` for model changes when a model config option exists.
- MUST prefer `session/set_config_option` for reasoning effort when a reasoning config option exists.
- MUST keep `session/set_model` only as fallback when config options are absent and legacy model state supports it.
- MUST remove provider-level `supports_reasoning_effort` gating from session start behavior.
- MUST match config option IDs conservatively and never invent reasoning levels from catalog metadata.
- MUST expose active session config options through a named API/contract payload consumed by web and co-ship generated contract outputs when the API contract changes.
</requirements>

## Subtasks
- [x] 6.1 Audit ACP SDK `v0.6.3` to `v0.12.2` breaking changes and write `analysis/acp-sdk-breaking-changes.md`.
- [x] 6.2 Upgrade ACP SDK dependency with `go get` and resolve the audited compile/API changes.
- [x] 6.3 Add session config option state capture on new/load/update paths.
- [x] 6.4 Implement conservative model/reasoning option detection and set-config behavior.
- [x] 6.5 Preserve legacy model-state fallback where config options are absent.
- [x] 6.6 Add `SessionConfigOptionPayload` contract exposure for active session config options and run `make codegen` / `make codegen-check` when contract output changes.
- [x] 6.7 Add ACP fixture tests for config options, updates, fallback, resume/load behavior, and no invented reasoning levels.

## Implementation Details
Follow `_techspec.md` section `ACP Session Config Options`. Use Zed and Harnss references for active session config controls, but keep AGH behavior conservative.

### Relevant Files
- `go.mod` - ACP SDK dependency.
- `go.sum` - ACP SDK checksums.
- `internal/acp/client.go` - session new/load, caps capture, model application.
- `internal/acp/client_test.go` - ACP protocol fixtures and helper state.
- `internal/session/manager_start.go` - preferred model/reasoning session start flow.
- `internal/session/session.go` - session state shape if config options are exposed.
- `.resources/zed/crates/agent_ui/src/config_options.rs` - active config option UI/reference behavior.
- `.resources/zed/crates/acp_thread/src/connection.rs` - ACP set config option reference.
- `.resources/zed/crates/agent_servers/src/acp.rs` - ACP server-side config option and protocol reference.
- `.resources/harnss/src/types/window.d.ts` - ACP config option cache/set IPC reference.

### Dependent Files
- `internal/api/contract/contract.go` - add `SessionConfigOptionPayload` for active session config option exposure.
- `openapi/agh.json` - regenerate if `SessionConfigOptionPayload` affects API contract output.
- `web/src/generated/agh-openapi.d.ts` - regenerate if generated session types change.
- `web/src/systems/session/` - Task 09 consumes active session config option state.
- `openapi/agh.json` - regenerated in Task 10 if contract shape changes.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - separates active ACP session config from pre-session catalog truth.

### Web/Docs Impact
- `web/`: active session controls in `web/src/systems/session` must prefer config options in Task 09.
- `packages/site`: ACP model/reasoning behavior documented in provider/session docs in Task 10.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: no extension API added here; active ACP config semantics may affect extension-authored session tooling indirectly.
- Agent manageability: session start via CLI/HTTP/UDS must report deterministic errors when requested config options are unavailable.
- Config lifecycle: reasoning support is session-config driven, not provider-level `supports_reasoning_effort`.

## Deliverables
- ACP SDK upgraded to `v0.12.2`.
- Active session config options captured and updated.
- `.compozy/tasks/provider-model-catalog/analysis/acp-sdk-breaking-changes.md` with the audited v0.6.3-to-v0.12.2 symbol impact.
- `SessionConfigOptionPayload` contract exposure for web/session consumers.
- Generated OpenAPI/web types for session config option payloads when contract output changes.
- Model/reasoning set-config-option behavior with legacy fallback.
- ACP/session tests with 80%+ coverage for changed behavior **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] `session/new` captures config options with model and reasoning entries.
  - [x] `session/load` captures config options with model and reasoning entries.
  - [x] `config_option_update` mutates active session config option state.
  - [x] preferred model uses `session/set_config_option` when model option exists.
  - [x] reasoning effort uses `session/set_config_option` when reasoning option exists.
  - [x] legacy `session/set_model` is used only when config options are absent.
  - [x] unknown reasoning effort is not sent when no reasoning config option exists.
- Integration tests:
  - [x] session start with model + reasoning override preserves existing mode negotiation.
  - [x] existing ACP sessions without config options continue to start with legacy model fallback.
  - [x] generated contract or contract-source tests include `SessionConfigOptionPayload` and Task 09 can reference it by name.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `go test ./internal/acp ./internal/session` passes.
- Active ACP config options override catalog assumptions only for their session.
