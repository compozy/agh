# Final-QA Backlog Plan — Clusters A, B, D

## Summary

- Ship in this order: `A1 contract/migration -> A2 emitters/projections -> B agent-local resolution -> D13/D18 -> D03 tool interception -> D12 minimal closure`.
- Hard cuts in this round: `actor_ref -> actor_id`, canonical agent roots under `.agh/agents/<name>/`, removal of legacy `.agents` skill roots, and `AUT-03` as AGH-owned tool interception at the execution boundary.
- Acceptance remains the same: rerun only the 9 reopened final-qa rows with real public-surface evidence.

## Cluster A — Canonical Event Correlation Contract

- Add canonical top-level correlation fields across contract, store, observe, CLI, HTTP/UDS, and web: `coordinator_session_id`, `scheduler_reason`, `hook_event`, `hook_name`, `actor_kind`, `actor_id`, `release_reason`, plus the already required `task_id`, `run_id`, `workflow_id`, `claim_token_hash`, and `lease_until`.
- Emit `hook.dispatch.start`, `hook.dispatch.complete`, and `settings.changed` on the unified observe/session envelope. `settings.changed` stays on the normal observe stream; there is no dedicated settings SSE surface.
- Ship one numbered migration that adds the missing correlation columns to event summaries and renames persisted `actor_ref` columns to `actor_id`.
- Regenerate contract/codegen/web artifacts in the same PR. Reopen `API-17`, `OBS-02`, and `CFG-04` with corpus-based tests proving the new keys appear on the public evidence surfaces.

## Cluster B — Agent-Local Skill Resolution and Shadowing

- Canonical roots are `~/.agh/agents/<name>/` for global scope and `<workspace-or-additional-root>/.agh/agents/<name>/` for workspace/additional scope. `<name>` is the exact logical `AgentDef.Name` after `TrimSpace`, matched bytewise and case-sensitively, with no slugification or lowercasing. Path-shaping names are rejected as validation errors.
- This round removes `.agents/<name>` and `~/.agents/skills` entirely from runtime/docs/tests. The same hard cut updates glossary, RFC 001, memory/spec notes, config helpers, and any runtime paths that still keep `.agents` alive.
- `ForAgent(ctx, resolved, agentName)` resolves the winning agent definition using the existing agent precedence `workspace -> additional -> global`. Agent-local skills come only from that winning agent directory’s `skills/` subtree. There is no same-name merge across multiple agent roots.
- Without `workspace`, `for_agent=<name>` is valid and resolves the global agent at `~/.agh/agents/<name>/`. Workspace only changes resolution context; it does not force storage into the workspace.
- Merge semantics are operational and fixed: start from the normal effective skill set `bundled -> marketplace -> user -> additional -> workspace`, or its resource-backed equivalent, then apply agent-local as a final overlay. Non-colliding agent-local skills append. Same-name collisions replace the current winner. There is no separate “agent-only mode”.
- Agent-scoped disable state is persisted only in the resolved `AGENT.md` frontmatter as `skills.disabled: []string`. This round adds only `skills.disabled`; `skills.inherit` and `skills.extra_sources` stay out of scope.
- Disable semantics are logical, not physical. If both workspace `review` and agent-local `review` exist, disabling `review` for agent `foo` yields one effective `review` row for `foo` with `enabled=false` and no fallback to the workspace copy.
- Disabling an absent skill name is allowed and creates a proactive tombstone in `skills.disabled`. If that skill appears later in the same resolution scope, it resolves as disabled immediately. Enabling removes the tombstone even when the skill is not currently present.
- Agent-scoped mutations write to the winning resolved `AGENT.md`, even when `workspace_id` is present and the winning agent lives in `~/.agh/agents/<name>/AGENT.md`. Workspace constrains resolution, not write location. Settings responses should reuse existing `write_target` reporting to show the actual file being edited.
- `resourceAuthority` remains valid only for the base skill layers. Agent definitions and agent-local overlays remain file-backed in this round. Agent-local always sits above the resource-projected base set and is never written back through resource projection.
- Full public parity ships in the same round. `GET /api/skills`, `GET /api/skills/:name`, `GET /api/skills/:name/content`, `POST /api/skills/:name/enable`, and `POST /api/skills/:name/disable` all accept optional `for_agent=<name>` and optional `workspace=<ref>`.
- `GET /api/skills` with no `workspace` is valid and returns the global base set. Adding `for_agent` returns the effective global-agent set.
- CLI mirrors that contract exactly with `agh skill list|info|view|enable|disable --for-agent <name>`, and global scope is the default when `--workspace` is omitted.
- Settings also gains full parity in this round. `GET/PATCH /api/settings/skills` supports `scope=agent&agent_name=<name>&workspace_id=<optional>`. `agent_name` is required for `scope=agent`; `workspace_id` is optional and only affects resolution context.
- Runtime behavior must hard-cut to the new resolver in the same round. Every agent-composition consumer that currently uses `ForWorkspace()` must switch to `ForAgent()` whenever agent identity exists, including session prompt composition, hook bridge delivery, host API agent context, situation service, and provider-native tool capability composition.
- Cache keys become agent-aware and must include workspace discriminator, canonical agent name, resolution mode, base-layer snapshots, the winning `AGENT.md` snapshot, agent `skills/` snapshots, resource revision, global version, and an agent-disabled revision. Different agents in the same workspace never share cache entries.
- Watchers must observe `~/.agh/agents`, every workspace/additional `.agh/agents` root, the winning `AGENT.md`, and `skills/**` sidecars. Create/remove/modify invalidates only the affected agent entries.
- Invalid agent-local state is fail-closed. Invalid `AGENT.md`, invalid `skills.disabled`, unreadable or malformed agent-local skills, or critical verification failures abort agent-aware resolution and mutation for that agent. There is no silent fallback to workspace/global in that case.
- Error mapping is fixed: invalid `agent_name` or `for_agent` is `400`; unknown agent in the selected resolution context is `404`; invalid resolved agent layer is `422`; workspace lookup errors keep the existing workspace status mapping.
- Emit `skills.shadow` once per effective rebuild per actual collision, never on cache hits. Payload includes `skill_name`, `old_source`, `new_source`, `old_path`, `new_path`, `layer_pair`, `resolution_scope`, optional `agent_name`, optional `workspace_id`, and `shadow_kind`.
- Emit `skills.load_failed` once per failed agent rebuild with `agent_name`, `source=agent-local`, `path`, `error_code`, and redacted `error_detail`.

## Cluster D — ACP Supervision and Tool-Execution Interception

- `ACP-12` stays test-scope only. `ACP-13` adds a single absolute `prompt_deadline` carried across detached prompt paths and surfaced as `stop_reason="timeout"` with `stop_detail="prompt_deadline_exceeded"`. `ACP-18` adds only `claim_token_hash` to synthetic metadata.
- `AUT-03` is an AGH-owned tool-execution gateway at the provider-native tool execution boundary, not a session proxy. Prompts, assistant text, reasoning, heartbeats, and all other ACP traffic stay on the normal session path.
- Every provider-native tool call that is subject to policy or hook enforcement must pass through AGH before any real side effect. `deny` means the tool does not execute. `allow` forwards execution and publishes the normal result on the existing session/event flow.
- The interception requirement applies to every enforced provider-native tool class, not only `Write`. No per-tool bypass is allowed.

## Test Plan

- Reopen and rerun only `API-17`, `OBS-02`, `CFG-04`, `CFG-06`, `SKL-03`, `ACP-12`, `ACP-13`, `ACP-18`, and `AUT-03`.
- Add regressions for: global agent scope without workspace, workspace-context writes that resolve to global `AGENT.md`, proactive tombstones for absent skills, no-fallback agent disable, `422` invalid agent-local state, `skills.shadow` pair coverage, `skills.load_failed`, AGENT.md-triggered watcher invalidation, and runtime consumers switching from `ForWorkspace()` to `ForAgent()`.
- `make codegen-check` is mandatory in the same PR as contract changes, and `make verify` remains the blocking completion gate.

## Locked Defaults

- `AUT-03` is tool interception at execution boundary, not a session proxy.
- Agent-local canonical roots are under `.agh/agents`, never `.agents`.
- Legacy `~/.agents/skills` is removed in this round.
- CLI defaults to global skill scope when `--workspace` is omitted.
- Invalid agent-local state is fail-closed.
- Agent-scope disable uses logical tombstones stored in resolved `AGENT.md` and allows absent-name tombstones.
