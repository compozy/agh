# Native Tools

## Contents

- Operating rule
- Discovery and catalog toolsets
- Runtime and workspace tools
- Skills and memory tools
- Network tools
- Task and autonomy tools
- Config, hooks, automation, extensions, and auth tools
- Descriptor discipline

## Operating Rule

Agents running inside AGH should prefer daemon-native tools over shelling out when a dedicated agh\_\_ tool is visible and callable. Native tools are policy-filtered, structured, auditable, and redaction-aware. Shell commands remain valid when a native tool is absent, denied, too narrow for the task, or when the user explicitly asks for CLI output.

Never guess a tool schema from this reference. Use agh\_\_tool_info for the exact descriptor, input schema, risks, and availability diagnostics before the first call.

## Discovery And Catalog Toolsets

- Toolset agh**bootstrap: agh**tool_list, agh**tool_search, agh**tool_info.
- Toolset agh\_\_catalog: skill catalog access plus bootstrap tools.

Use:

1. agh\_\_tool_search with the domain or action.
2. agh\_\_tool_info for the selected ToolID.
3. The dedicated tool call when available.
4. CLI/API fallback only after reading denial or absence diagnostics.

## Runtime And Workspace Tools

Session tools:

- agh\_\_session_list
- agh\_\_session_status
- agh\_\_session_history
- agh\_\_session_events
- agh\_\_session_describe
- agh\_\_session_health

Authored context tools:

- agh\_\_agent_heartbeat_status
- agh\_\_agent_heartbeat_wake

Workspace tools:

- agh\_\_workspace_list
- agh\_\_workspace_info
- agh\_\_workspace_describe

Provider model tools:

- agh\_\_provider_models_list
- agh\_\_provider_models_refresh
- agh\_\_provider_models_status

## Skills And Memory Tools

Skill tools:

- agh\_\_skill_list
- agh\_\_skill_search
- agh\_\_skill_view

Use agh\_\_skill_view with a file/resource argument when reading skills/agh/references/\*.md from inside AGH.

Memory tools:

- agh\_\_memory_list
- agh\_\_memory_show
- agh\_\_memory_search
- agh\_\_memory_propose
- agh\_\_memory_note

Memory admin tools include health, scope, reindex, promote, reset, reload, decisions, recall traces, dreams, daily logs, extractor, provider, and session-ledger operations under the agh\__memory_ namespace. Inspect descriptors before using admin tools because they are broader than normal memory reads.

## Network Tools

Coordination tools:

- agh\_\_network_status
- agh\_\_network_channels
- agh\_\_network_inbox
- agh\_\_network_peers
- agh\_\_network_send
- agh\_\_network_threads
- agh\_\_network_thread_messages
- agh\_\_network_directs
- agh\_\_network_direct_resolve
- agh\_\_network_direct_messages
- agh\_\_network_work

Use these only inside a policy scope that permits network coordination. Read references/network.md before sending or interpreting network messages.

## Task And Autonomy Tools

Task tools:

- agh\_\_task_list
- agh\_\_task_read
- agh\_\_task_create
- agh\_\_task_child_create
- agh\_\_task_update
- agh\_\_task_cancel
- agh\_\_task_run_list
- agh\_\_task_run_review_request
- agh\_\_task_run_review_list
- agh\_\_task_run_review_show
- agh\_\_task_execution_profile_get
- agh\_\_task_execution_profile_set
- agh\_\_task_execution_profile_delete
- agh\_\_task_notification_subscribe
- agh\_\_task_notification_list
- agh\_\_task_notification_show
- agh\_\_task_notification_delete

Session-bound autonomy tools:

- agh\_\_task_run_claim_next
- agh\_\_task_run_heartbeat
- agh\_\_task_run_complete
- agh\_\_task_run_fail
- agh\_\_task_run_release
- agh\_\_task_run_review_submit

Autonomy tools are bound to the caller session. Do not substitute general task mutation tools for session-bound lease operations. Read references/tasks-and-orchestration.md before claiming, heartbeating, completing, failing, releasing, or submitting review verdicts.

## Config, Hooks, Automation, Extensions, And Auth Tools

Config tools live under agh\__config_\* and include show, list, get, set, unset, diff, and path.

Hook tools live under agh\__hooks_\* and include list, info, events, runs, create, update, delete, enable, and disable. Hooks are typed dispatch, not an event bus.

Automation tools live under agh\__automation_\* and cover jobs, triggers, run records, history, enable/disable, and manual triggering.

Extension tools live under agh\__extensions_\* and cover search, list, info, install, update, remove, enable, and disable.

MCP auth exposes agh\_\_mcp_auth_status for redacted diagnostics. Browser/OAuth login and raw auth material remain management-surface operations unless AGH exposes a scoped tool for them.

## Descriptor Discipline

This reference gives the stable map. The live descriptor gives the contract:

- exact input schema
- output shape
- read/write/destructive risk flags
- availability reason codes
- policy and dependency diagnostics

If a descriptor is unavailable or denied, do not retry blindly. Choose a narrower tool, read-only status path, or CLI/operator surface based on the reason code.
