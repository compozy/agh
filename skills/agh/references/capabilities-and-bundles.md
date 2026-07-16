# Capabilities And Bundles

## Contents

- Capability vocabulary
- Extensibility surfaces
- Cross-surface impact audit
- Agent manageability
- Bundles
- Extension install trust
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

Bundle activation reads expose `version`. Before confirming a changed Live requirement, read the activation with `agh bundle get <id> -o json`, then pass that value to `agh bundle update <id> --expected-version <version> --confirm-network-requirement -o json`. A `409 Conflict` means the activation changed; reread it and inspect the current digest instead of retrying with a stale version.

When changing bundle behavior, update resources, registries, config docs, CLI/API surfaces, and tests in the same change. Greenfield AGH favors hard cuts over compatibility bridges.

Activation list and detail payloads expose `spec_drift` by comparing the stored activation spec hash with the current installed bundle profile. Use `agh bundle list -o json` or the activation API to inspect it. Reapply with `agh bundle update <activation-id> -o json`; a successful reapply reconciles current resources, stores the current hash, and clears drift. Activation timestamps are informational and never signal bundle updates.

## Extension Install Trust

`agh extension install <slug> -o json` resolves curated extensions through the daemon-owned catalog. The runtime verifies the downloaded archive against the catalog-pinned SHA-256 digest before extraction, then persists separate catalog entry, archive digest, and extracted-tree checksum provenance. Inspect the decision with `agh extension provenance <name> -o json`. A curated digest mismatch is terminal and cannot be bypassed.

Non-curated side-loads require both live policy `extensions.marketplace.allow_unverified = true` and the request-level `--allow-unverified` confirmation. The policy defaults to `false`; a block returns a structured diagnostic that points to Settings › Extensions. Changing the policy applies live and does not weaken curated digest verification.

The stable block code is `extension_unverified_policy_blocked`; its evidence path is
`/settings/extensions`. The stable curated mismatch code is `extension_archive_digest_mismatch`;
the mismatch is terminal for that catalog version and has no unverified bypass. Registry tier and
digest verification are provenance signals, not safety guarantees. `extension.digest.verify` event
queries report `outcome=success` for matching bytes and `outcome=failure` for mismatches.

An extension update commits when the registry, managed directory, and runtime reload all succeed.
Post-commit backup or staging cleanup failure does not roll back or relabel that active update:
`status` remains `updated`, and `warnings[]` contains `extension_update_cleanup_failed` with the
cleanup target and residual path. Verify the active version before asking an operator to remove the
residue.

Extension removal follows the same commit boundary. After the registry, managed directory, and
runtime reload confirm removal, backup cleanup failure leaves `status` as `removed` and reports
`extension_remove_cleanup_failed` with the residual path. Treat that path as cleanup debt; do not
restore or operate the removed extension from it.

## Hooks

Hooks are typed dispatch at the owning state transition. They are not a generic event bus and must not tail event/log tables to infer work.

Hooks may deny, narrow, annotate, or observe. They must not bypass safety primitives such as claim tokens, leases, TTL, lineage, spawn caps, or permission narrowing.

Skill-declared hooks are part of the skill contract. Keep hook declarations structured and validated, not buried in prose.

Loop lifecycle hooks use the `loop.*` family for generation, gate, node-terminal, and terminal call sites. `loop.generation.pre` and `loop.gate.pre` are sync control hooks; node and terminal wake behavior is daemon-owned and fail-open.

## Config Lifecycle

Any feature or refactor must state whether config.toml keys, defaults, docs, and examples are added, changed, or removed. In greenfield alpha, delete obsolete config paths instead of creating aliases or fallback bridges.

If a rename touches code, storage, APIs, CLI, extensions, specs, docs, and task artifacts, update them together.

`[marketplace.catalog]` controls AGH's curated MCP server, extension, and skill feed projection.
`base_url` defaults to the public `compozy/agh` catalog on `main`, `ttl` defaults to `1h`, and
`timeout` defaults to `10s`; all three paths apply live to the next fetch. Use the structured config
surfaces plus `agh config reload -o json` and apply history to change or verify them. These keys do
not replace the independent `skills.marketplace.*` and `extensions.marketplace.*` registry settings.

`[autonomy.scheduler]` tunes the mechanical scheduler's convergence escalation ladder for starved runs. Keys are wake-cycle counts that must stay positive and monotonic (`fan_out_after` ≤ `spawn_after` ≤ `event_after` ≤ `needs_attention_after`) plus a `min_queued_age` duration. Defaults: `fan_out_after = 2`, `spawn_after = 4`, `event_after = 6`, `needs_attention_after = 10`, `min_queued_age = "2m"`. Validation rejects non-monotonic or non-positive values at load.

These thresholds apply only to true convergence episodes. Compatible sessions that are starting, prompting, processing another run, or reserved earlier in the scheduler cycle hold serial backlog without consuming the ladder. Policy remains serial: saturation does not start extra task-role capacity.

`[loops.defaults.delivery]` and `[loops.defaults.watch]` seed new loop effective config before per-loop `loop_config` overrides; they are desired-state defaults, not the DB-backed override plane. Delivery defaults are `iteration_cap = 50`, `no_progress.window = 3`, `gates.max_revisions = 10`, `budget.tokens = 0`, `budget.wall_clock_sec = 0`, `budget.on_exceeded = "halt"`, and `fan_out_width = 4`. Watch defaults are `iteration_cap = 0`, `no_progress.window = 2`, `budget.tokens = 0`, `budget.wall_clock_sec = 0`, `budget.on_exceeded = "halt"`, and `fan_out_width = 2`; gate revisions remain unset for watch unless configured. Both default families accept optional `model_defaults.worker` and `model_defaults.judge`; node or criterion-local models win over these effective defaults. Operator config may tighten the compile-time ceilings but must not exceed fan-out `64`, no-progress window `30`, or gate revisions `64`. `budget.on_exceeded` accepts only `halt` or `escalate`. These paths are restart-required config lifecycle entries; use `agh config reload -o json` and apply history to inspect activation.

`[goals]` sets `max_turns = 20` and `context_nudge_ratio = 0.8` for new Goals, plus the daemon-wide durable session-event relay controls `outbox_batch_size = 50` and `outbox_poll_interval = "100ms"`. The Goal defaults are global/workspace-overridable; relay controls use global config because one relay serves every workspace. All four are agent-mutable, restart-required paths. `max_turns` must be positive; the ratio accepts `0.0` through `1.0`, with zero preserved; the relay batch accepts `1` through `200`; and its poll interval must be positive. Each Run pins its resolved ratio and every Goal checkpoint copies that value, so config reload or daemon restart cannot change an active Goal. Relay settings take effect when the daemon starts.

Loop observability is durable runtime state, not a transient UI stream. `loop_run_events` persists replayable workspace-scoped events for status changes, node running/terminal outcomes, gate verdicts, generation starts, channel messages, token ticks, and needs-approval pauses. Payloads are redacted and bounded before persistence; token ticks preserve only usage counters and terminal markers.

Automation schedule catch-up policy is part of the public schedule contract. The accepted values are `skip`, `coalesce`, and `replay`. Loop targets with a `watch-source` default to `coalesce`; other scheduled Loop targets default to `skip`. Catch-up starts must carry structured automation-run metadata so agents can distinguish normal starts from replayed/coalesced starts and reason about `concurrency: forbid|queue` outcomes.

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
