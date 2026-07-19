# Native Tools

## Contents

- Operating rule
- Discovery and catalog toolsets
- Runtime and workspace tools
- Skills and memory tools
- Network tools
- Task and autonomy tools
- Loop tools
- Config, hooks, automation, marketplace, extensions, bundles, resources, and MCP tools
- Observability and bridge tools
- CLI/HTTP-only management surfaces
- Descriptor discipline
- Descriptor and skill co-ship

## Operating Rule

Inside AGH, prefer callable daemon-native tools over shelling out. They are policy-filtered, structured, auditable, and redaction-aware. Use shell only when a native tool is absent, denied, too narrow, or explicitly requested.

`agh__*` strings are canonical ToolIDs for registry, policy, CLI, descriptors, and `tool_id`; harnesses may wrap call names. Resolve by capability plus canonical ID, then call the returned reference exactly.

Never guess a tool schema from this reference. Resolve canonical `agh__tool_info` for the exact descriptor, input schema, risks, and availability diagnostics before the first call.

Management-only surfaces include diagnostics, support bundles, scheduler controls, task inspection/pause/force recovery, notification presets, config apply history, and some session repair/recap/approval flows.

## Discovery And Catalog Toolsets

- Toolset `agh__bootstrap`: `agh__tool_list`, `agh__tool_search`, `agh__tool_info`.
- Toolset `agh__catalog`: skill catalog access plus bootstrap tools.

Bare managed sessions receive the full availability-gated callable catalog through hosted MCP. AGH
does not add an automatic bootstrap/catalog allowlist over the hosted projection. Explicit agent
`tools`, `toolsets`, `deny_tools`, session lineage, disabled sources, and approval/risk gates still
apply.

The discovery loop and denial-handling rules live in the preceding tools-and-skills prompt section; this reference supplies the stable tool map.

## Runtime And Workspace Tools

Session tools: `agh__session_list`, `agh__session_status`, `agh__session_history`, `agh__session_events`, `agh__session_describe`, `agh__session_health`.

`agh__session_list` returns one counted catalog page and accepts workspace, exact state, exact session `type`, exact agent, search, resumability, health, sort, cursor, and limit inputs. Use `type: "user"` when a workflow needs operator-created sessions without daemon-managed dream, system, coordinator, or spawned sessions.

Authored context tools: `agh__agent_heartbeat_status`, `agh__agent_heartbeat_wake`.

Workspace tools: `agh__workspace_list`, `agh__workspace_info`, `agh__workspace_describe`, `agh__agent_create`. `agh__agent_create` authors one public `AGENT.md` at `global` or `workspace` scope; provide `scope`, `name`, `provider`, `prompt`, and `workspace` for workspace scope. Reserved internal names such as `onboarding` are rejected.

Fresh daemon boot registers the operator `$HOME` as the default workspace through the resolver, so `agh__workspace_list` should return at least that workspace on a clean install.

A successful workspace catalog read reconciles registered roots before returning: entries whose directories no longer exist are durably unregistered, while other filesystem or deletion failures fail the read instead of hiding uncertain state. `agh__workspace_list`, `agh workspace list`, and HTTP/UDS `GET /api/workspaces` share this catalog.

Workspace unregister is atomic with credential cleanup: it removes workspace-scoped MCP OAuth rows and their encrypted access/refresh values, preserves global and sibling-workspace credentials, and leaves all state intact when cleanup fails.

The managed `onboarding` agent is internal to first-run setup and is not granted the full workspace or coordination toolsets. It receives only `agh__workspace_list`, `agh__workspace_describe`, `agh__network_channels`, `agh__network_channel_create`, and `agh__agent_create`.

Provider model tools: `agh__provider_models_list`, `agh__provider_models_curate`, `agh__provider_models_refresh`, `agh__provider_models_status`.

`agh__provider_models_list` accepts `view=curated|all` and defaults to curated; the CLI equivalents are `agh provider models list` and `agh provider models list --all`. `agh__provider_models_curate` is mutating, requires `providers.models.write`, and accepts required `provider_id`/`model_id` plus optional `hidden`, `featured`, `deprecated`, and `default_effort`. Its CLI fallback is `agh provider models set`. Treat `model_not_found` and `reasoning_effort_unsupported` as terminal input diagnostics; when the descriptor reports the settings backend unavailable, do not retry blindly.
For providers with an explicit curated set, the default view contains visible explicit or featured rows; live-only rows appear there only through the no-explicit-set fallback.

## Skills And Memory Tools

Skill tools: `agh__skill_list`, `agh__skill_search`, `agh__skill_view`.

Resolve canonical `agh__skill_view`, then use its returned tool reference with a file/resource argument when reading `skills/agh/references/*.md` from inside AGH.

Memory tools: `agh__memory_list`, `agh__memory_show`, `agh__memory_search`, `agh__memory_propose`, `agh__memory_note`.

Memory admin tools include health, scope, reindex, promote, reset, reload, decisions, recall traces, dreams, daily logs, extractor, provider, and session-ledger operations under the `agh__memory_*` namespace. Inspect descriptors before using admin tools because they are broader than normal memory reads.

## Network Tools

Coordination tools: `agh__network_status`, `agh__network_channels`, `agh__network_channel_create`, `agh__network_channel_update`, `agh__network_inbox`, `agh__network_peers`, `agh__network_send`, `agh__network_threads`, `agh__network_thread_messages`, `agh__task_promote_from_thread`, `agh__network_subscriptions`, `agh__network_subscribe`, `agh__network_mute`, `agh__network_unmute`, `agh__network_directs`, `agh__network_direct_resolve`, `agh__network_direct_messages`, `agh__network_work`.

Channel create/update are mutating. Channel names are lowercase `[a-z0-9][a-z0-9_-]{0,63}`;
coordinator routing metadata requires `coordinator_peer_id`. Routing metadata never enrolls or wakes
an execution.

The coordination toolset is projected only when the caller's immutable participation snapshot is
Live, then narrowed by policy and dependency gates. Daemon availability alone never exposes it. A
Local caller receives `not_participating`; create a new explicitly Live execution instead of
retrying. The reserved onboarding agent has only its documented first-run provisioning subset.

Read references/network.md before sending or interpreting messages. Direct/mention `say` is the
only current model-wake path; other messages may persist without activation. Use `agh network usage
-o json` through CLI/HTTP/UDS for usage because it is a management read, not a native coordination
tool ID.

## Task And Autonomy Tools

Task tools: `agh__task_list`, `agh__task_read`, `agh__task_create`, `agh__task_child_create`, `agh__task_update`, `agh__task_cancel`, `agh__task_promote_from_thread`, `agh__task_fanout_runs`, `agh__task_run_list`, `agh__task_run_review_request`, `agh__task_run_review_list`, `agh__task_run_review_show`, `agh__task_execution_profile_get`, `agh__task_execution_profile_set`, `agh__task_execution_profile_delete`, `agh__task_notification_subscribe`, `agh__task_notification_list`, `agh__task_notification_show`, `agh__task_notification_delete`.

Session-bound autonomy tools: `agh__task_run_claim_next`, `agh__task_run_heartbeat`, `agh__task_run_complete`, `agh__task_run_fail`, `agh__task_run_release`, `agh__task_run_review_submit`.

Autonomy tools are bound to the caller session. Do not substitute general task mutation tools for session-bound lease operations. Read references/tasks-and-orchestration.md before claiming, heartbeating, completing, failing, releasing, or submitting review verdicts.

## Loop Tools

Toolset `agh__loops` (16 tools):
`agh__loop_list/_inspect/_validate/_status/_runs/_create/_run/_configure/_pause/_resume/_approve/_stop/_delete`,
`agh__goal_get`, `agh__goal_report`, and `agh__loop_turns`.

No `agh__loop_edit`. See references/loops.md for publishing, approval/self-approval, and Goal report
binding semantics.

## Config, Hooks, Automation, Marketplace, Extensions, Bundles, Resources, And MCP Tools

Config tools live under `agh__config_*` for show/list/get/set/unset/diff/path. Hook tools live under `agh__hooks_*` for list/info/events/runs/create/update/delete/enable/disable; hooks are typed dispatch, not an event bus.

Automation job/trigger catalogs are available through CLI, HTTP/UDS, and `agh__automation_jobs_list` / `agh__automation_triggers_list`. They return counted cursor pages and support scope/workspace, source, enabled, Loop-target, search, and trigger-event filters. Run/history reads remain separate uncounted `runs` collections; bound them explicitly. Other automation tools under `agh__automation_*` cover detail, mutation, enable/disable, and manual trigger.
Config- and package-backed definitions accept enabled-only updates and reject deletion; dynamic definitions accept full mutation.

`agh__marketplace_search` returns MCP, extension, skill, and bundle rows. Single-kind calls accept
`next_cursor` as `cursor`; keep kind, query, and workspace unchanged. Curated and bundle cursors
fence their projection; remote skill cursors validate the prior page boundary with bounded
look-behind. Restart from the first page when AGH rejects a continuation. Grouped searches omit it.
Extension lifecycle tools remain under `agh__extensions_*` for
list/info/install/update/remove/enable/disable; there is no extension-specific native search tool.
When `agh__extensions_update` with `all=true` stops on a later target, its error identifies the failed
extension and completed count, and every earlier committed update retains an `extension.updated`
event. Inspect those events before retrying the failed remainder.
Successful update results may also contain `extension_update_cleanup_failed`; this is cleanup debt,
not an activation failure, and the active version remains the reported latest version.
Successful `agh__extensions_remove` results may similarly contain `extension_remove_cleanup_failed`.
The removal remains committed; use the warning's residual path for operator cleanup.

Bundle tools live under `agh__bundles_*` for list/info/activate/deactivate/status. Resource tools live under `agh__resources_*` for list/info/snapshot of desired-state resources.

MCP tools expose `agh__mcp_status` and `agh__mcp_auth_status` for redacted diagnostics. Browser/OAuth login and raw auth material remain management-surface operations unless AGH exposes a scoped tool for them.
Curated MCP installation is likewise a management-surface mutation through `agh mcp install` or
`POST /api/settings/mcp-servers/install`; do not invent `agh__mcp_install`.

## Observability And Bridge Tools

Runtime log inspection is available through `agh__logs`. Metrics and redacted event search are available through `agh__observe_metrics` and `agh__observe_search`.

Bridge list/status reads return counted, filtered pages through CLI, HTTP/UDS, and `agh__bridges_list` / `agh__bridges_status`; native results are redacted. The HTTP/UDS health stream requires at most 200 IDs from the current page under the same scope/workspace boundary. Bridge lifecycle, route mutation, secret binding, `manifest`, `setup`, `verify`, real `send-test`, and webhook registration remain CLI/HTTP/UDS management surfaces unless a scoped native tool is present in the live descriptor; never invent native equivalents.

## CLI/HTTP-Only Management Surfaces

Use CLI or HTTP/UDS with structured output for diagnostics (`agh status`, `agh doctor`), session repair/recap/approval/inspect/soul refresh, task inspect/pause/resume/forced release/fail, scheduler controls, config reload/apply history, notification presets, and support bundles.

Task notification subscription tools are native, but notification preset management is not. Do not invent `agh__scheduler_*`, `agh__support_*`, `agh__doctor`, `agh__status`, `agh__task_inspect`, or `agh__notifications_*` calls unless the live registry exposes them.

## Descriptor Discipline

This reference gives the stable map. The live descriptor gives exact input schema, output shape, risk flags, availability reason codes, and policy/dependency diagnostics.

If a descriptor is unavailable or denied, do not retry blindly. Choose a narrower tool, read-only status path, or CLI/operator surface based on the reason code.

## Descriptor And Skill Co-Ship

Changing native tools is a public agent contract change. When an AGH change adds, removes, renames, or changes an `agh__*` ID, toolset, descriptor, schema digest, risk flag, availability diagnostic, capability gate, or CLI/API fallback, update `skills/agh/` in the same change or record explicit no-impact evidence.
