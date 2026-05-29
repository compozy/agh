# Capabilities And Bundles

## Contents

- Capability vocabulary
- Extensibility surfaces
- Cross-surface impact audit
- Agent manageability
- Bundles
- Hooks
- Config lifecycle
- Settings apply lifecycle

## Capability Vocabulary

The canonical AGH artifact name is capability. Do not use recipe, workflow, procedure, or playbook for current AGH behavior unless quoting historical material.

A capability should be discoverable, manageable by agents, and represented through public runtime surfaces. It is incomplete if it only works through internal Go calls or the web UI.

## Extensibility Surfaces

When adding or changing AGH behavior, decide which surfaces are affected:

- extensions and extension resources
- hooks
- skills and capabilities
- tools and toolsets
- bundles
- registries
- bridge SDKs
- MCP sidecars
- CLI, HTTP, and UDS APIs
- docs and generated references

No-impact is acceptable only when there is evidence.

## Cross-Surface Impact Audit

For any feature, bug fix, refactor, contract/API/CLI/native-tool/config/docs update, or runtime behavior change, record the AGH impact decision before claiming the change complete:

- Native tools: tool IDs, toolsets, descriptors, input/output schemas, schema digests, risk flags, availability diagnostics, capability gates, and agent CLI/API fallbacks.
- Extensibility and hooks: extension resources, hook taxonomy and dispatch call sites, skills/capabilities, tools/resources, bundles, registries, bridge SDKs, MCP sidecars, config lifecycle, docs, and tests.
- Workspace data isolation: whether data is global, workspace-scoped, session-scoped, or agent-scoped; `workspace_id` flow through CLI/HTTP/UDS/core/store/web/SSE/cache/events; and cross-workspace leak tests for list/read/cache/event paths.
- Official AGH skill: `skills/agh/SKILL.md` and `skills/agh/references/*.md` guidance that must change when public behavior or agent-operable surfaces change.

Use `no impact` only with checked-surface evidence. QA/worktree isolation and workspace data isolation are separate decisions.

## Agent Manageability

Every user-visible runtime capability needs an agent-operable path:

- CLI with -o json or -o jsonl where relevant
- HTTP/UDS parity when state crosses the daemon boundary
- discoverable status/config output
- deterministic errors and reason codes
- docs that describe the agent path

UI-only management is incomplete.

## Bundles

Bundles activate related runtime resources together. Treat bundle projection as daemon-owned state. Do not make a bundle depend on prompt prose for authority.

When changing bundle behavior, update resources, registries, config docs, CLI/API surfaces, and tests in the same change. Greenfield AGH favors hard cuts over compatibility bridges.

## Hooks

Hooks are typed dispatch at the owning state transition. They are not a generic event bus and must not tail event/log tables to infer work.

Hooks may deny, narrow, annotate, or observe. They must not bypass safety primitives such as claim tokens, leases, TTL, lineage, spawn caps, or permission narrowing.

Skill-declared hooks are part of the skill contract. Keep hook declarations structured and validated, not buried in prose.

## Config Lifecycle

Any feature or refactor must state whether config.toml keys, defaults, docs, and examples are added, changed, or removed. In greenfield alpha, delete obsolete config paths instead of creating aliases or fallback bridges.

If a rename touches code, storage, APIs, CLI, extensions, specs, docs, and task artifacts, update them together.

`[autonomy.scheduler]` tunes the mechanical scheduler's convergence escalation ladder for starved runs. Keys are wake-cycle counts that must stay positive and monotonic (`fan_out_after` ≤ `spawn_after` ≤ `event_after` ≤ `needs_attention_after`) plus a `min_queued_age` duration. Defaults: `fan_out_after = 2`, `spawn_after = 4`, `event_after = 6`, `needs_attention_after = 10`, `min_queued_age = "2m"`. Validation rejects non-monotonic or non-positive values at load.

## Settings Apply Lifecycle

`config.toml` is desired state. Runtime truth advances only when `ConfigApplyService` applies that desired change to the daemon active generation or records why it cannot.

Agent-manageable settings changes must surface lifecycle status, not just file writes. The public contract names are:

- `SettingsApplyTargetName`: `general`, `memory`, `skills`, `automation`, `network`, `observability`, `hooks-extensions`, `providers`, `mcp-servers`, `sandboxes`, and `hooks`.
- `SettingsMutationBehavior`: `applied_now`, `restart_required`, or `action_trigger`.
- `SettingsApplyLifecycle`: `live`, `live-add`, `live-remove-if-unused`, `restart-required`, or `session-rebind`.
- `ConfigApplyStatus`: `pending_apply`, `applied`, `blocked`, or `failed`.
- `SettingsApplyNextAction`: `none`, `restart-daemon`, `new-session`, or `retry`.

Use `agh config reload -o json` to reconcile edited desired state with the active generation. Use `agh config apply-history -o json` or `GET /api/settings/apply` to inspect persisted apply records. A settings write is incomplete if agents cannot see whether it applied live, requires a daemon restart, affects only new sessions, or failed with retryable diagnostics.

Codegen owns the lifecycle matrix documentation. When config lifecycle rules change, update the source matrix and run `make codegen`; do not hand-edit generated lifecycle docs.
