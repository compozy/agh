# ACP SDK v0.6.3 to v0.12.2 Breaking-Change Audit

Task: provider-model-catalog Task 06

This audit was produced before migrating AGH code from `github.com/coder/acp-go-sdk` `v0.6.3` to `v0.12.2`.

## Sources Checked

- Current AGH usage under `internal/acp`, `internal/session`, and API contract conversion code.
- Local module cache for `github.com/coder/acp-go-sdk@v0.6.3`.
- Local module cache for `github.com/coder/acp-go-sdk@v0.12.2`.
- Zed ACP references under `.resources/zed/crates/agent_ui/src/config_options.rs`, `.resources/zed/crates/acp_thread/src/connection.rs`, and `.resources/zed/crates/agent_servers/src/acp.rs`.
- Harnss ACP config cache/set reference under `.resources/harnss/src/types/window.d.ts`.

## AGH ACP Symbols Currently Used

AGH production and test code currently uses these ACP SDK symbols directly:

- `acpsdk.Agent`, `acpsdk.AgentSideConnection`, `acpsdk.NewAgentSideConnection`
- `acpsdk.ClientCapabilities`, `acpsdk.AgentCapabilities`, `acpsdk.InitializeRequest`, `acpsdk.InitializeResponse`
- `acpsdk.FileSystemCapability`
- `acpsdk.NewSessionRequest`, `acpsdk.NewSessionResponse`
- `acpsdk.LoadSessionResponse`
- `acpsdk.SessionId`
- `acpsdk.SessionModeId`, `acpsdk.SessionModeState`, `acpsdk.AvailableSessionMode`
- `acpsdk.SessionModelState`, `acpsdk.ModelInfo`, `acpsdk.ModelId`
- `acpsdk.SetSessionModeRequest`, `acpsdk.SetSessionModeResponse`
- `acpsdk.SetSessionModelRequest`, `acpsdk.SetSessionModelResponse`
- `acpsdk.CancelNotification`
- `acpsdk.PromptRequest`, `acpsdk.PromptResponse`, `acpsdk.PromptStopReason`, `acpsdk.PromptResponseStopReasonEndTurn`
- `acpsdk.SessionNotification`, `acpsdk.SessionUpdate`
- `acpsdk.RequestError`
- `acpsdk.RequestPermissionRequest`, `acpsdk.RequestPermissionToolCall`
- `acpsdk.KillTerminalCommandRequest`, `acpsdk.KillTerminalCommandResponse`
- `acpsdk.ContentBlock`, `acpsdk.ContentBlockText`
- `acpsdk.AgentMethodSessionLoad`, `acpsdk.AgentMethodSessionPrompt`, `acpsdk.AgentMethodSessionCancel`
- `acpsdk.AgentMethodSessionSetMode`, `acpsdk.AgentMethodSessionSetModel`
- `acpsdk.ClientMethodSessionUpdate`

AGH also intentionally uses a local `wireLoadSessionRequest` wrapper because AGH needs to preserve the existing `additional_dirs` wire field used by its ACP integration.

## Changed Symbols and Required AGH Impact

### Session Creation and Loading

`NewSessionResponse` and `LoadSessionResponse` now include:

- `ConfigOptions []SessionConfigOption json:"configOptions,omitempty"`
- `Meta map[string]any json:"meta,omitempty"` instead of an unconstrained `any` meta field.

AGH impact:

- `captureCaps` must accept and store `ConfigOptions` from both `session/new` and `session/load`.
- Resume/load paths must capture config options exactly like new-session paths.
- Existing mode/model capture remains valid, but config options take precedence for active session model/reasoning controls.

### Config Option Types

`v0.12.2` introduces these wire types:

- `SessionConfigOption`
- `SessionConfigOptionSelect`
- `SessionConfigOptionBoolean`
- `SessionConfigId`
- `SessionConfigValueId`
- `SessionConfigSelectOptions`
- `SessionConfigSelectOptionsUngrouped`
- `SessionConfigSelectOptionsGrouped`
- `SessionConfigSelectOption`
- `SessionConfigOptionCategory`
- `SessionConfigOptionUpdate`

AGH impact:

- Add AGH-owned session config option state rather than leaking SDK union types into public session state.
- Convert only known option shapes needed by AGH consumers. Select options are required for model/reasoning changes; boolean options should be preserved for contract visibility but not used for model/reasoning selection.
- Flatten grouped and ungrouped select values into a stable payload while preserving each option ID, label, current value, and valid values.

### Config Option Mutation Method

`v0.12.2` adds:

- `AgentMethodSessionSetConfigOption = "session/set_config_option"`
- `SetSessionConfigOptionRequest`
- `SetSessionConfigOptionResponse`
- `SetSessionConfigOptionValueId`
- `SetSessionConfigOptionBoolean`
- `ClientSideConnection.SetSessionConfigOption`

AGH impact:

- Model changes must prefer `session/set_config_option` when a conservative model config option exists and contains the requested value.
- Reasoning effort must prefer `session/set_config_option` when a conservative reasoning config option exists and contains the requested value.
- AGH should update active session config option state from the response's returned `configOptions`.

### Legacy Model Mutation

`AgentMethodSessionSetModel` remains on the wire as `session/set_model`, but the request/response symbols changed:

- Removed or renamed: `SetSessionModelRequest`
- Removed or renamed: `SetSessionModelResponse`
- New names: `UnstableSetSessionModelRequest`, `UnstableSetSessionModelResponse`
- The agent-side interface moved this handler to `AgentExperimental.UnstableSetSessionModel`.

AGH impact:

- Production fallback must use `UnstableSetSessionModelRequest` and `UnstableSetSessionModelResponse`.
- ACP test helper agents must implement `UnstableSetSessionModel` instead of `SetSessionModel`.
- Fallback must only run when config options are absent and legacy `SessionModelState.AvailableModels` advertises the requested model.

### Session Update Notifications

`SessionUpdate` now includes:

- `ConfigOptionUpdate *SessionConfigOptionUpdate`
- `SessionInfoUpdate`
- Existing `AvailableCommandsUpdate`, `CurrentModeUpdate`, and `UsageUpdate` remain.

AGH impact:

- `config_option_update` notifications must mutate the active process/session config option state even when no prompt event is currently being emitted.
- Notification translation can continue producing a system event, but state capture must not depend on prompt activity.

### Session Model and Mode State

`SessionModeState`, `AvailableSessionMode`, `SessionModelState`, and `ModelInfo` remain structurally compatible for AGH's existing needs, with meta fields now represented as `map[string]any`.

AGH impact:

- Existing mode capture and `session/set_mode` behavior can remain.
- Existing model list capture can remain as the legacy fallback surface.

### Client Capabilities

`ClientCapabilities.Fs` remains, but the filesystem capability type was renamed:

- Removed or renamed: `FileSystemCapability`
- New name: `FileSystemCapabilities`

AGH impact:

- Initialization must construct `acpsdk.FileSystemCapabilities`.

### Prompt Metadata

`PromptRequest.Meta` now uses `map[string]any`.

AGH impact:

- AGH's structured `PromptMeta` must be converted to a map before being assigned to the SDK prompt request.
- The conversion must avoid ignored marshal/unmarshal errors.

### Terminal Kill Request Types

The client-side terminal kill symbols were renamed:

- Removed or renamed: `KillTerminalCommandRequest`
- Removed or renamed: `KillTerminalCommandResponse`
- New names: `KillTerminalRequest`, `KillTerminalResponse`

AGH impact:

- The AGH terminal kill handler signature and return values must use the new symbols.

### Permission Tool Call Type

`RequestPermissionRequest.ToolCall` is now the existing `ToolCallUpdate` type instead of a dedicated `RequestPermissionToolCall` type.

AGH impact:

- Permission display helpers must accept `acpsdk.ToolCallUpdate`.

### Cancellation and Request Errors

`CancelNotification` and `RequestError` remain available. `v0.12.2` adds `NewRequestCancelled` and uses a cancellation error code constant.

AGH impact:

- Existing request error handling should continue compiling unless code referenced removed field names.
- No AGH production code currently depends on renamed cancellation fields in the audited ACP paths.

## Conservative Matching Rules for AGH

Model config option detection:

- Prefer exact config option ID `model`.
- Allow only explicitly documented local fixture IDs after `model`; do not match display names or categories.
- Send only values present in the select option's advertised values.

Reasoning config option detection:

- Prefer exact config option ID `reasoning_effort`.
- Then allow exact `effort`.
- Send only values present in the select option's advertised values.
- Never derive valid reasoning levels from catalog metadata such as `supports_reasoning`.

## Required Verification After Migration

- Focused ACP tests for `session/new`, `session/load`, `config_option_update`, set-config model, set-config reasoning, legacy fallback, and no invented reasoning levels.
- Focused session manager tests for start option propagation and legacy resume behavior.
- Contract tests proving a named `SessionConfigOptionPayload` is exposed.
- `make codegen` and `make codegen-check` if the public API contract changes.
- Full `make verify` before completion and again after the local commit.
