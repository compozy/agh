# Native Tools

## Contents

- Operating rule
- Discovery and catalog toolsets
- Runtime and workspace tools
- Skills and memory tools
- Network tools
- Task and autonomy tools
- Config, hooks, automation, extensions, bundles, resources, and MCP tools
- Observability and bridge tools
- CLI/HTTP-only management surfaces
- Descriptor discipline
- Descriptor and skill co-ship

## Operating Rule

Agents running inside AGH should prefer daemon-native tools over shelling out when a dedicated `agh__*` tool is visible and callable. Native tools are policy-filtered, structured, auditable, and redaction-aware. Shell commands remain valid when a native tool is absent, denied, too narrow for the task, or when the user explicitly asks for CLI output.

Never guess a tool schema from this reference. Use `agh__tool_info` for the exact descriptor, input schema, risks, and availability diagnostics before the first call.

Not every management surface has a native tool. Diagnostics, support bundles, scheduler controls, task inspection/pause/force recovery, notification preset management, config apply history, and some session repair/recap/approval flows are intentionally CLI/HTTP surfaces today.

## Discovery And Catalog Toolsets

- Toolset `agh__bootstrap`: `agh__tool_list`, `agh__tool_search`, `agh__tool_info`.
- Toolset `agh__catalog`: skill catalog access plus bootstrap tools.

Use:

1. `agh__tool_search` with the domain or action.
2. `agh__tool_info` for the selected ToolID.
3. The dedicated tool call when available.
4. CLI/API fallback only after reading denial or absence diagnostics.

## Runtime And Workspace Tools

Session tools:

- `agh__session_list`
- `agh__session_status`
- `agh__session_history`
- `agh__session_events`
- `agh__session_describe`
- `agh__session_health`

Authored context tools:

- `agh__agent_heartbeat_status`
- `agh__agent_heartbeat_wake`

Workspace tools:

- `agh__workspace_list`
- `agh__workspace_info`
- `agh__workspace_describe`
- `agh__agent_create` — authors one public `AGENT.md` definition at `global` or `workspace` scope (mutating; reuses the same writer as `POST /api/agents`). Provide `scope`, `name`, `provider`, and `prompt`; workspace scope also needs `workspace`. Reserved internal managed names such as `onboarding` are rejected.

The managed `onboarding` agent is internal to first-run setup and is not granted the full workspace or coordination toolsets. It receives only `agh__workspace_list`, `agh__workspace_describe`, `agh__network_channels`, `agh__network_channel_create`, and `agh__agent_create`.

Provider model tools:

- `agh__provider_models_list`
- `agh__provider_models_refresh`
- `agh__provider_models_status`

## Skills And Memory Tools

Skill tools:

- `agh__skill_list`
- `agh__skill_search`
- `agh__skill_view`

Use `agh__skill_view` with a file/resource argument when reading `skills/agh/references/*.md` from inside AGH.

Memory tools:

- `agh__memory_list`
- `agh__memory_show`
- `agh__memory_search`
- `agh__memory_propose`
- `agh__memory_note`

Memory admin tools include health, scope, reindex, promote, reset, reload, decisions, recall traces, dreams, daily logs, extractor, provider, and session-ledger operations under the `agh__memory_*` namespace. Inspect descriptors before using admin tools because they are broader than normal memory reads.

## Network Tools

Coordination tools:

- `agh__network_status`
- `agh__network_channels`
- `agh__network_channel_create` — registers one channel with a stated `purpose` for a workspace (mutating). Channel names are lowercase `[a-z0-9][a-z0-9_-]{0,63}`.
- `agh__network_inbox`
- `agh__network_peers`
- `agh__network_send`
- `agh__network_threads`
- `agh__network_thread_messages`
- `agh__network_directs`
- `agh__network_direct_resolve`
- `agh__network_direct_messages`
- `agh__network_work`

Use these only inside a policy scope that permits network coordination. Read references/network.md before sending or interpreting network messages.

## Task And Autonomy Tools

Task tools:

- `agh__task_list`
- `agh__task_read`
- `agh__task_create`
- `agh__task_child_create`
- `agh__task_update`
- `agh__task_cancel`
- `agh__task_run_list`
- `agh__task_run_review_request`
- `agh__task_run_review_list`
- `agh__task_run_review_show`
- `agh__task_execution_profile_get`
- `agh__task_execution_profile_set`
- `agh__task_execution_profile_delete`
- `agh__task_notification_subscribe`
- `agh__task_notification_list`
- `agh__task_notification_show`
- `agh__task_notification_delete`

Session-bound autonomy tools:

- `agh__task_run_claim_next`
- `agh__task_run_heartbeat`
- `agh__task_run_complete`
- `agh__task_run_fail`
- `agh__task_run_release`
- `agh__task_run_review_submit`

Autonomy tools are bound to the caller session. Do not substitute general task mutation tools for session-bound lease operations. Read references/tasks-and-orchestration.md before claiming, heartbeating, completing, failing, releasing, or submitting review verdicts.

## Config, Hooks, Automation, Extensions, Bundles, Resources, And MCP Tools

Config tools live under `agh__config_*` and include show, list, get, set, unset, diff, and path.

Hook tools live under `agh__hooks_*` and include list, info, events, runs, create, update, delete, enable, and disable. Hooks are typed dispatch, not an event bus.

Automation tools live under `agh__automation_*` and cover jobs, triggers, run records, history, enable/disable, and manual triggering.

Extension tools live under `agh__extensions_*` and cover search, list, info, install, update, remove, enable, and disable.

Bundle tools live under `agh__bundles_*` and include list, info, activate, deactivate, and status.

Resource tools live under `agh__resources_*` and include list, info, and snapshot for desired-state resources.

MCP tools expose `agh__mcp_status` and `agh__mcp_auth_status` for redacted diagnostics. Browser/OAuth login and raw auth material remain management-surface operations unless AGH exposes a scoped tool for them.

## Observability And Bridge Tools

Runtime log inspection is available through `agh__logs`. Metrics and redacted event search are available through `agh__observe_metrics` and `agh__observe_search`.

Bridge inspection is available through `agh__bridges_list` and `agh__bridges_status`. Bridge lifecycle, route mutation, test delivery, and secret binding management remain CLI/HTTP surfaces unless a scoped native tool is present in the live descriptor.

## CLI/HTTP-Only Management Surfaces

Use CLI or HTTP/UDS with structured output for these current management surfaces:

- Runtime diagnostics: `agh status -o json`, `agh doctor -o json`, and `GET /api/status` / `GET /api/doctor`.
- Session repair and recap: `agh session recap`, `agh session repair`, `agh session approve`, `agh session inspect`, and `agh session soul refresh`.
- Task management gaps: `agh task inspect`, `agh task pause`, `agh task resume`, forced `agh task release`, and forced `agh task fail --reason`.
- Scheduler controls: `agh scheduler status`, `pause`, `resume`, `drain`, and `backlog`.
- Config apply lifecycle history: `agh config reload`, `agh config apply-history`, and `GET /api/settings/apply`.
- Notification preset management: `agh notifications presets list` and `agh notifications preset show/create/enable/disable/delete`.
- Support bundles: `agh support bundle --yes` plus the HTTP create/get/download support-bundle endpoints.

Task notification subscription tools are native, but notification preset management is not. Do not invent `agh__scheduler_*`, `agh__support_*`, `agh__doctor`, `agh__status`, `agh__task_inspect`, or `agh__notifications_*` calls unless the live registry exposes them.

## Descriptor Discipline

This reference gives the stable map. The live descriptor gives the contract:

- exact input schema
- output shape
- read/write/destructive risk flags
- availability reason codes
- policy and dependency diagnostics

If a descriptor is unavailable or denied, do not retry blindly. Choose a narrower tool, read-only status path, or CLI/operator surface based on the reason code.

## Descriptor And Skill Co-Ship

Changing native tools is a public agent contract change. When an AGH change adds, removes, renames, or changes an `agh__*` tool ID, toolset, descriptor, input/output schema, schema digest, risk flag, availability diagnostic, capability gate, or CLI/API fallback, update the Official AGH skill references under `skills/agh/` in the same change or record explicit no-impact evidence naming the checked tool surfaces.
