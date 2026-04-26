# Skills & Tools Registry — Autonomy Gap Analysis

Slice: how an AGH agent discovers, acquires, and advertises capabilities (skills, MCP tools, ACP-exposed tools, CLI subcommands).

## 1. TL;DR

- AGH ships a mature **Skills registry** (`internal/skills/`) with multi-source precedence, frontmatter parsing, MCP sidecars, hash provenance, watcher-driven reload, workspace overlay caching, and a marketplace installer (`internal/registry/`). It is wired into prompt assembly through a static `<available-skills>` catalog injected into the system prompt at session start.
- The **Tool registry** (`internal/tools/`) is a thin record type (`Tool{Name, Description, InputSchema, ReadOnly, Source}`) with a `ToolProvider` interface and a desired-state resource sync (`internal/daemon/tool_mcp_resources.go`). There is **no central runtime tool catalog the agent can query**, no per-agent allowlist semantics, no usage telemetry.
- **`internal/toolruntime/`** is a process-tracking registry for live tool subprocesses (interrupt scopes, ownership), **not** a discovery surface.
- An agent inside an ACP session today only knows: (a) the static skill catalog (XML metadata block) injected once, and (b) whatever tools its underlying ACP runtime exposes (Claude Code, Codex, etc.). AGH does not let the agent ask "what skills/tools does AGH have?", "load skill X on demand", or "which capabilities do my peers expose?" via uniform tool calls.
- Network capability advertising exists for **agent-level capability docs** (`config.CapabilityCatalog`), surfaced as `PeerCard.Capabilities` and a `whois`-pulled rich catalog. Skills and tools are not part of that brief — the loaded skill set is invisible to peers.
- For autonomy we need: a queryable skills-and-tools API the agent reaches via tool/CLI, on-demand body load (already half there for skills), per-role/per-session scoping, peer-skill query, and execution telemetry feeding back into the catalog.

## 2. Current Skill / Tool Model

### 2.1 Skills

| Concern | File:Line | Behavior |
|---|---|---|
| Skill type | `/Users/pedronauck/Dev/compozy/agh/internal/skills/types.go:21` | `Skill{Meta, Source, Dir, FilePath, Enabled, MCPServers, Hooks, Provenance, InstalledFrom}` — metadata only, body lives on disk and loads via `Registry.LoadContent`. |
| Source precedence | `/Users/pedronauck/Dev/compozy/agh/internal/skills/types.go:33-47` | 5 tiers: `Bundled < Marketplace < User < Additional < Workspace` (lowest first → highest wins). |
| Registry | `/Users/pedronauck/Dev/compozy/agh/internal/skills/registry.go:29-88` | `Registry` keeps `globalSkills map[string]*Skill`, `resourceWorkspaces` overlay, `wsCache` per workspace, `globalVersion atomic.Int64` for invalidation. |
| Loader (FS scan) | `/Users/pedronauck/Dev/compozy/agh/internal/skills/loader.go:138-200` | Walks `~/.agh/skills`, `~/.agh/agents`, `<workspace>/.agh/skills`, `<workspace>/.agents`; depth ≤4, ≤300 candidates, snapshot-based dedupe. |
| Bundled skills | `/Users/pedronauck/Dev/compozy/agh/internal/skills/bundled/embed.go:1-17` + `bundled/skills/` (`agh-agent-setup`, `agh-memory-guide`, `agh-network`, `agh-session-guide`) | `embed.FS` of starter SKILL.md instructions. |
| Workspace overlay | `/Users/pedronauck/Dev/compozy/agh/internal/skills/registry.go:147-200` + `registry_workspace_cache.go` | `ForWorkspace(ctx, resolved)` merges global + workspace skills, cached by `(workspaceID, paths)` key with 10-min TTL. |
| Resource projection | `/Users/pedronauck/Dev/compozy/agh/internal/skills/resource.go:13-87` | `SkillResourceSpec` (kind=`skill`) lets desired-state machinery promote registry to authority. `ApplyResourceRecords` flips `resourceAuthority=true`. |
| Watcher | `/Users/pedronauck/Dev/compozy/agh/internal/skills/watcher.go:46-118` | Polls roots every 3s (default), diffs filesnap snapshots, calls `RefreshGlobal` on change. |
| Verifier | `/Users/pedronauck/Dev/compozy/agh/internal/skills/verify.go` (referenced by `registry.go:504`) | Critical-severity warnings drop the skill silently. |
| Marketplace install | `/Users/pedronauck/Dev/compozy/agh/internal/registry/installer.go:21-86` | Downloads archive, validates SKILL.md or extension.toml at root, runs prompt-injection regex blocklist, writes provenance sidecar. |
| MCP sidecar merge | `/Users/pedronauck/Dev/compozy/agh/internal/skills/mcp_sidecar.go:15-89` | Reads `mcp.json` next to SKILL.md, overrides skill-frontmatter `metadata.agh.mcp_servers` entries last-wins. |
| MCP resolver | `/Users/pedronauck/Dev/compozy/agh/internal/skills/mcp.go:36-100` | Aggregates MCP servers from active skills, applies marketplace allowlist (`SkillsConfig.AllowedMarketplaceMCP`), passes to ACP `StartOpts.MCPServers`. |
| Catalog provider (prompt) | `/Users/pedronauck/Dev/compozy/agh/internal/skills/catalog.go:36-110` | `CatalogProvider.PromptSection` builds an XML `<available-skills>` block of `name` + 200-char description + footer "Use `agh skill view <name>` to load full instructions." |
| Session wiring | `/Users/pedronauck/Dev/compozy/agh/internal/session/manager.go:79-148` + `manager_lifecycle.go:184-206` | `WithSkillRegistry` + `WithMCPResolver` are required together; `resolveStartMCPServers` calls `skillRegistry.ForWorkspace(...)` once per session start, merges with agent-declared MCP servers. |
| HTTP API | `/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:182-188` + `internal/api/core/skills.go:15-180` | `GET /api/skills?workspace=`, `GET /api/skills/:name`, `GET /api/skills/:name/content`, `POST /api/skills/:name/(enable|disable)`. |
| CLI surface | `/Users/pedronauck/Dev/compozy/agh/internal/cli/skill_commands.go:15-318` | `agh skill list/view/info/create/search/install/remove/update`. |
| Web UI | `/Users/pedronauck/Dev/compozy/agh/web/src/systems/skill/lib/query-options.ts:6-32` | Three queries: list, detail (metadata), content (lazy, gated on `enabled`). |

### 2.2 Tools

| Concern | File:Line | Behavior |
|---|---|---|
| Tool record | `/Users/pedronauck/Dev/compozy/agh/internal/tools/tool.go:91-97` | `Tool{Name, Description, InputSchema json.RawMessage, ReadOnly, Source}`. |
| Tool source enum | `/Users/pedronauck/Dev/compozy/agh/internal/tools/tool.go:14-25` | `builtin`, `mcp`, `extension`, `dynamic`. |
| Provider interface | `/Users/pedronauck/Dev/compozy/agh/internal/tools/tool.go:133-136` | `ToolProvider.Tools(ctx) ([]Tool, error)`. **No `Call`, no `IsAvailable`, no permission hooks.** |
| Resource codec | `/Users/pedronauck/Dev/compozy/agh/internal/tools/resource.go:12-61` | `ToolResourceKind = "tool"`, validates name, source, JSON-object `input_schema`. |
| Daemon sync | `/Users/pedronauck/Dev/compozy/agh/internal/daemon/tool_mcp_resources.go:257-353` | `toolMCPSourceSyncer.Sync` collects `ToolProvider` outputs into desired-state records under daemon source. Reconciles tool + MCP resources. |
| Agent payload | `/Users/pedronauck/Dev/compozy/agh/internal/api/contract/contract.go:120-129` | `AgentPayload.Tools []string` — opaque names only, no schema, no namespace, no availability flag. |
| **No HTTP/CLI for tool list** | n/a | Search shows no `/api/tools` route, no `agh tool` cobra command. Tools only exist as resource records and as part of agent payloads. |

### 2.3 Tool runtime (orthogonal)

`/Users/pedronauck/Dev/compozy/agh/internal/toolruntime/registry.go:128-565` — a *process* registry. It checkpoints subprocess records (PID, owner, source) and supports scoped interrupts (by session/turn/tool-call/extension/hook). It does not enumerate tool definitions.

### 2.4 Capability surfaces (network)

| Concern | File:Line | Behavior |
|---|---|---|
| Agent capability catalog | `/Users/pedronauck/Dev/compozy/agh/internal/config/capabilities.go:28-44` | `CapabilityDef` (id, summary, outcome, version, context_needed, artifacts_expected, execution_outline, constraints, examples, requirements) loaded from `<agent>/capabilities.toml`/`.json` or `<agent>/capabilities/*.toml`. SHA-256 digest computed. |
| Session-level capabilities | `/Users/pedronauck/Dev/compozy/agh/internal/session/network_peer.go:9-31` | `networkPeerCapabilities(catalog)` projects from `AgentDef.Capabilities`. Skill set is **not** included. |
| Peer card | `/Users/pedronauck/Dev/compozy/agh/internal/network/envelope.go:227-235` | `PeerCard.Capabilities []string` (just IDs); rich entries live in `Ext` under `agh.capability_brief` (`/Users/pedronauck/Dev/compozy/agh/internal/network/capability_brief.go:11-51`). |
| Whois rich catalog | `/Users/pedronauck/Dev/compozy/agh/internal/network/capability_catalog.go:62-109` | `whois` requesters can include `agh.include = ["capability_catalog"]` (+ optional `agh.capability_ids`) and receive the full per-capability documents. |

### 2.5 Scope summary

Today the registry scope axes are **(global vs workspace) × (skill source tier)**. There is no per-session, per-agent, or per-role scoping — every active skill applies to every session within its workspace. Session-time customization is limited to `enabled/disabled` overlays in workspace config (`SkillsConfig.DisabledSkills`, `manager_test.go:2125-2237`).

## 3. What an Autonomous Agent Actually Needs

For a session to be self-sufficient about its capabilities it must be able to:

1. **Browse the catalog** — list all installed skills (metadata-only) plus all available tools (built-in + MCP-proxied + extension + CLI subcommands), with descriptions, schemas, and availability flags.
2. **Load skills on demand** — fetch full `SKILL.md` body (and supporting files in the skill dir) when the metadata snippet is insufficient — *without* pre-injecting every body into the system prompt.
3. **Load tools on demand** — request a tool's full input schema only when about to call it, the way Claude Code's `ToolSearch` returns `tool_reference` blocks the API expands inline (`/Users/pedronauck/Dev/compozy/agh/docs/ideas/from-claude-code/analysis_tool_system.md:219-261`).
4. **Filter by role/scope** — when this session plays role `reviewer`, only see safe tools; when it plays `master`, see catalog/playbook tools too. Currently AGH has agent definitions but no per-role tool allowlists outside ACP-runtime control (`our_system_kernel.md:336-342` describes the deprecated `agentTypeTools` map).
5. **Acquire (install) skills at runtime** — call into `agh skill install` from inside a session for itself or for a peer that lacks it. Currently install is CLI-only, not exposed as a session-callable tool.
6. **Advertise its loaded skills to peers** — append a "skills bundle" projection to `PeerCard` so peers can discover not only declarative capabilities but the actionable procedures this session knows.
7. **Query peer skills** — extend `whois` so a master session can ask "do you have skill `terraform-review`?" before delegating.
8. **Observe usage** — record `skill.viewed`, `skill.invoked`, `tool.called` events with success/error, latency, and outcome so the catalog can self-rank ("which skills actually move the needle?").
9. **Receive availability gating** — when an MCP server is offline, when an env var is missing, when a binary is absent, the tool/skill should disappear from the agent's view ("fail-closed availability"; cf. Hermes `check_fn` (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/tools/registry.py:258-286`) and `analysis_hermes.md:128-129`).
10. **Compose toolsets** — request a named toolset (`coding`, `research`) rather than enumerating tools (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/toolsets.py:31-454`, AGH `analysis.md:99-124`).

## 4. Gaps

### G1 — No agent-facing discovery API for skills/tools
- Skills are queryable over HTTP/CLI, but **the running agent itself cannot call those**: there is no AGH-provided ACP tool, no MCP tool, no built-in CLI subcommand surface beyond what the bundled `agh-network` skill hints at (`agh skill view`). The catalog block injected into the system prompt is a frozen list (`internal/skills/catalog.go:96-110`) — no live re-query.
- There is no `/api/tools` endpoint, no `agh tool list`, and `Tool` records exist only as desired-state resources. The agent has no programmatic way to ask "what tools live in this daemon?".

### G2 — Skills are statically materialized into the prompt
- `manager_lifecycle.go:196` calls `skillRegistry.ForWorkspace` *once* during `startSession`. The catalog injected at start (and the resolved MCP servers passed to `acp.StartOpts.MCPServers`) is the agent's permanent view for the session. There is no mid-session "skill loaded" event, no way to add/remove a skill without restarting the agent.
- Any skill enable/disable POST (`internal/api/core/skills.go:109-179`) only affects *future* sessions; live sessions don't refresh.

### G3 — No on-demand body injection mechanism
- `Registry.LoadContent` exists (`internal/skills/registry.go:127-144`) and the HTTP `GET /api/skills/:name/content` is wired (`routes.go:186`), but there is no path inside an active prompt turn for the LLM to *request* that content. The system prompt nudges "Use `agh skill view <name>`", placing the burden on the ACP runtime's terminal tool. AGH itself doesn't gate or stream skill bodies as ACP tool results, doesn't budget them, doesn't persist them as messages, and doesn't deduplicate.

### G4 — Tool registry has no `Call`, no `IsAvailable`, no permissions
- `tools.Tool` is purely descriptive (`internal/tools/tool.go:91-136`). `ToolProvider.Tools()` enumerates; there is no `Call`, `CheckAvailability`, `IsConcurrencySafe`, `IsDestructive`, `CheckPermissions` (cf. Claude Code `Tool.ts`, `analysis_tool_system.md:5-40`).
- AGH delegates execution entirely to the ACP agent process. That is fine for ACP-native tools, but there is no surface for AGH-owned tools (e.g., `network_send`, `task_claim`, `skill_install`) that should be available across all ACP runtimes.

### G5 — No per-role / per-session tool scoping
- `AgentDef.Tools []string` (`internal/config/agent.go:22`) is an unstructured list. Nothing enforces "reviewer only sees read-only tools". The "tool allowlists per type" listed in `our_system_kernel.md:336-342` is documentation of the *old* implementation; the current system has no equivalent.
- Workspace `SkillsConfig.DisabledSkills` is the only filter, applied at registry-load time, not at request time.

### G6 — Skills are absent from peer discovery
- `PeerCard.Capabilities`/`Ext.agh.capability_brief` list only `CapabilityDef` IDs from `capabilities.toml` (`network_peer.go:9-31`). The actual skill set the session loaded is *not* projected anywhere on the wire. A peer cannot know which skills the other side has.
- There is no `whois` extension for skills (analogous to the existing `whois` capability_catalog mechanism in `capability_catalog.go:62-109`).

### G7 — No cross-session / cross-peer skill query
- Even locally, one session cannot ask "does session B have skill X loaded?" — sessions are siloed and the registry is daemon-global. To support delegation ("send this to a peer that knows `terraform-review`") we need both the local query API and the wire-protocol extension.

### G8 — No usage telemetry
- `internal/observe` records prompt/tool events via the per-session SQLite store (`session/manager_lifecycle.go` notifier). Nothing slices by *skill name* or *tool name* to answer "which skills are actually invoked?", "which fail?", "how much do they cost?". The dream-consolidation pipeline could feed off such telemetry but currently has no input.
- Skill registration emits a single info log (`registry.go:705-715`) and verification warnings; runtime invocation produces nothing skill-attributed.

### G9 — No availability gating on skills/tools
- `MCPResolver.Resolve` filters marketplace skills by allowlist (`internal/skills/mcp.go:131-150`) but does not check whether the MCP server binary actually exists, whether env vars resolve, or whether previous spawn attempts failed.
- Tool `IsAvailable()` is missing (G4). The ACP agent is therefore likely to advertise tools that error out at call time — exactly the failure mode `analysis_hermes.md:128-129` calls "the single most important reliability property" to fix.

### G10 — No skill-acquire from inside a session
- `agh skill install` is in `cli/skill_commands.go:253` but is CLI-only. There's no API endpoint (`/api/skills` is read-only — no POST install). An autonomous agent can install only by shelling out to the binary itself, racing the daemon's own watcher, and waiting for the next snapshot poll (3s default).

### G11 — No version pinning / signed bundles
- `Provenance{Hash, Registry, Slug, Version, InstalledAt}` (`internal/skills/types.go:58-64`) is a SHA-256 of skill files + source URL only. There is no signature, no transparency log, no CVE lookup. Marketplace allowlists are name-string matches.
- `installerVerificationRules` (`internal/registry/installer.go:42-100`) is a hardcoded prompt-injection regex set. No update path, no per-org policy.

### G12 — `Tool.Source = "dynamic"` is unused
- The enum has a slot for runtime-assembled tools (`internal/tools/tool.go:24`), but no producer in the codebase. There is no notion of "this tool was synthesized by skill X right now".

## 5. Reference Comparisons

### 5.1 Claude Code (`/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/`, doc `analysis_tool_system.md`)
- **`buildTool()` factory** with fail-closed defaults (`isReadOnly: false`, `isConcurrencySafe: false`, `checkPermissions: allow`) — every tool is the same shape and security-relevant flags must be explicitly opted in (`analysis_tool_system.md:5-42`).
- **`ToolSearch` deferred loading** — tools sent to the API with `defer_loading: true`; the model emits `tool_reference` blocks via ToolSearch and the API expands them inline. Discovered tools persist via `extractDiscoveredToolNames()` (`analysis_tool_system.md:219-260`). Direct analog of what AGH needs for >40-tool catalogs.
- **MCP and skills as first-class registry citizens** — `services/plugins/`, `skills/loadSkillsDir.ts:67-94` (5 settings sources: managed, user, project, plugin, mcp). Bundle of bundles approach, but every loaded skill participates in the same dispatch.
- **`maxResultSizeChars` per tool with disk persistence** for large outputs (`analysis_tool_system.md:184-195`) — the agent never blows context with a huge tool result.

### 5.2 Hermes (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/`)
- **Single-process `ToolRegistry` singleton** (`tools/registry.py:100-227`) where each tool calls `registry.register(name, toolset, schema, handler, check_fn, requires_env, is_async, ...)` at module import. Toolset names auto-classify (`mcp-*` are MCP-proxied; `mcp-`-prefix collisions are allowed since they overwrite each other; non-MCP collisions are rejected as shadowing).
- **`check_fn` availability gating** (`tools/registry.py:258-286`) — `get_definitions` skips tools whose check fails. AGH's missing equivalent is G4/G9.
- **`toolsets.py` recursive composition** (`toolsets.py:504-554`) — named toolsets compose other toolsets; the `all`/`*` aliases project across every registered toolset. This is exactly the per-role scoping AGH lacks (G5).
- **Skill manager tool** (`tools/skill_manager_tool.py:32-200`) — agent-callable CRUD on skills (create/edit/patch/delete/write_file/remove_file). Includes optional `_security_scan_skill` gating (`tools/skill_manager_tool.py:72-96`). Direct analog of G10.
- **Skill validation: name regex, frontmatter required fields, file/byte caps, allowed subdirs** (`skill_manager_tool.py:118-200`). AGH has the load-time verifier but no agent-callable create path.

### 5.3 OpenClaw / Hermes / GoClaw cross-cutting (`/Users/pedronauck/Dev/compozy/agh/docs/ideas/extensability/analysis.md:309-324`)
- Patterns observed in 4+ frameworks: uniform tool interface with JSON Schema; manifest-first plugin discovery; tool namespacing (`mcp__server__tool`, `ext__name__tool`); availability gating; non-blocking fan-out for events; subprocess env-var allowlist isolation. AGH already has the env allowlist (`hookEnvAllowlist`) and progressive disclosure for skills, but lacks namespacing, availability gating, and manifest-first plugin discovery.

## 6. Concrete Proposals

The following are deliberately small, additive packages and APIs. Numbering reflects the order in which they unblock the next.

### P1 — Promote `tools.Tool` to a runtime tool with `Call`, `Availability`, `Permissions`
- Extend `internal/tools/tool.go` (or split into `tool_def.go` / `tool_handle.go`):
  - Add `Aliases []string`, `Namespace string` (e.g. `agh.skill`, `agh.network`, `mcp.<server>`, `ext.<name>`), `IsConcurrencySafe`, `IsDestructive`, `MaxResultBytes int`.
  - Add `Availability func(ctx) AvailabilityResult` returning `{Available bool, Reason string}` so the registry can drop tools whose deps are missing (mirror Hermes `check_fn`).
  - Add `Call func(ctx, json.RawMessage) (ToolResult, error)` plus `CheckPermission` hook fed into the existing `acp.ApproveRequest` flow.
- Compile-time check (`var _ ToolDriver = (*xType)(nil)`) per AGH conventions.
- Status: **proposed**. (Today only the `Tool` record + `ToolProvider.Tools(ctx)` exist.)

### P2 — Daemon-owned `Catalog` package (`internal/catalog/`)
- New thin coordinator that watches both registries and exposes one read API:
  - `Catalog.ListSkills(ctx, scope)` — returns metadata view.
  - `Catalog.LoadSkill(ctx, name, scope, file)` — returns full body / referenced file with byte cap.
  - `Catalog.ListTools(ctx, scope, filter)` — returns built-in + MCP-projected + extension tools after availability gating.
  - `Catalog.SearchByKeyword(ctx, q)` — TF-IDF over name+description (Claude Code `ToolSearch` analog).
- Backed by `skills.Registry` (existing, file-watched) and a new `tools.Registry` aggregating `ToolProvider` instances (subprocess providers register at boot via the existing `toolMCPSourceSyncer`).
- Adds `internal/api/contract` types `CatalogQuery`, `SkillSummary`, `ToolSummary` so HTTP/UDS share the same payloads.
- Status: **proposed** — registry pieces exist but are not unified.

### P3 — Built-in AGH tools surfaced through the Catalog
Names land under namespace `agh.*` so they cannot collide with ACP-native or MCP tools.
- `agh.skill.list`, `agh.skill.view`, `agh.skill.search`, `agh.skill.install` — each maps to existing CLI logic but reachable from inside the session as a real tool the LLM emits.
- `agh.tool.list`, `agh.tool.search` — exposes the catalog itself.
- `agh.network.peers`, `agh.network.send` — subset of the existing `agh-network` bundled skill, callable directly.
- `agh.task.*` — pairs with the task-discovery slice.
Implementation pattern: register in `daemon/` composition root after extension/skill loading; pass `Catalog` and `network.Service` etc. via functional options.
- Status: **proposed**.

### P4 — Per-role / per-session tool scoping
- Extend `AgentDef` with structured tool/skill scoping:
  ```go
  type AgentDef struct {
      // ...
      ToolPolicy *ToolPolicy `yaml:"tool_policy,omitempty"`
  }
  type ToolPolicy struct {
      Allow    []string // pattern: "agh.*", "mcp.github.*"
      Deny     []string
      Toolsets []string // named bundles, resolved recursively
  }
  ```
- Add `internal/catalog/policy.go` with recursive toolset resolution (Hermes pattern, port `toolsets.py:504-554`). Toolsets live as TOML resources under `~/.agh/toolsets/*.toml` plus workspace overlays.
- `Catalog.ListTools` accepts the policy and returns the filtered list. Session-start passes the result to the ACP runtime.
- Status: **proposed**.

### P5 — Live skill/tool injection mid-session
- Make `Manager.startSession` subscribe to `Registry.GlobalVersion` (already an `atomic.Int64`, `internal/skills/registry.go:40`). On version bump for the active workspace, build a delta payload `{added, removed, updated}` and route it to the ACP driver as a synthetic system message (similar to Claude Code's `deferred_tools_delta` attachment — `analysis_tool_system.md:254-260`).
- Requires a small `AgentDriver` extension: `NotifyCatalogDelta(ctx, *AgentProcess, CatalogDelta) error`. ACP runtimes that don't implement it return `errors.ErrUnsupported` and the daemon falls back to "applies on next session".
- Status: **proposed**.

### P6 — Network-level skill projection
- Augment `session.NetworkPeerJoin` (`internal/session/interfaces.go:53-58`) with `SkillSummaries []NetworkPeerSkill{Name, Version, Source, Description}` — projected from `skillRegistry.ForWorkspace` at join time.
- Network side: add `agh.skill_brief` ext key to `PeerCard.Ext` (parallel to `agh.capabilities_brief`, `internal/network/capability_brief.go:11-51`). Define a `whois`-pull rich variant analogous to `capability_catalog.go:62-109` with key `agh.skill_catalog` so peers can request only the IDs they care about.
- Add wire validation matching `ensureEnvelopeSizeLimit` (`capability_catalog.go:299-313`) — keep brief small, push detail to whois.
- Status: **proposed** (today only `CapabilityCatalog` from `capabilities.toml` projects to peers; `skills.*` doesn't).

### P7 — Skill / tool acquisition surfaces
- HTTP: `POST /api/skills` body `{slug, version, source}` triggering `internal/registry/installer.go` flows; `DELETE /api/skills/:name`. Privileged route — gated through the existing `privilegedMutationGuard` in `httpapi/handlers.go:124`.
- UDS counterparts so an in-process tool (P3 `agh.skill.install`) doesn't have to shell out.
- `installer.Result` should publish a `skill.installed` event so the watcher's next pass becomes redundant (don't wait 3 s).
- Status: **proposed**.

### P8 — Usage telemetry feed
- Add per-skill / per-tool counters in `internal/observe`. Events: `skill.viewed`, `skill.body_loaded`, `tool.called`, `tool.failed` with `name`, `namespace`, `session_id`, `turn_id`, `latency_ms`, `result_bytes`.
- Catalog APIs (P2) accept a `?sort=usage` parameter so heavy-hitter skills float to the top of the agent's list.
- Persisted into the per-session event store (already `internal/store/sessiondb`) plus a global rollup in `globaldb`.
- Status: **proposed**.

### P9 — Availability projection on the wire
- When publishing the catalog (P2) or peer skills (P6), include an `available bool` flag computed from:
  - MCP server presence (probe binary exists / process responds within timeout).
  - Env-var requirements satisfied (declarative `requires_env` in skill frontmatter, port from Hermes `tools/registry.py:80-97`).
  - Past failure rate ≥ N% within window → mark `degraded`.
- Status: **proposed**.

## 7. Open Questions

1. **Security boundary for agent-callable `agh.skill.install`** — should installs require the user's interactive approval through the existing ACP `ApproveRequest` channel (`session/interfaces.go:225-229`)? If yes, default to `permissions: ask`; if no, the marketplace allowlist becomes the only barrier (cf. `installer.go` verification rules and `analysis_hermes.md:153-163` discussing skill auto-proposal as opt-in).
2. **Signed bundles vs. plain hash** — `Provenance.Hash` is SHA-256 today (`types.go:58`). Should we adopt sigstore/cosign so marketplace skills can be verified against the publisher key, mirroring the Wasm-extension supply-chain considerations in `analysis.md:282-294`? If yes, the verifier slot is in `internal/skills/verify.go` and `internal/registry/installer_checksum.go`.
3. **Version pinning** — should `SkillsConfig` accept `pinned_versions = { "skill-name" = "1.2.0" }`? The watcher would refuse upgrades; the marketplace installer would pull the exact version. Affects `MarketplaceConfig` in `internal/config/config.go:131-135`.
4. **Skill ↔ tool boundary** — today skills are *instructions* and may declare MCP servers (which become tools) and hooks. Should AGH ever let a skill declare a *first-class tool* implemented as a script/Wasm in the skill dir? If yes, the loader needs to register a `ToolProvider` per skill (analog of `mcpSkillBuilders.ts` in claude-code). If no, MCP remains the only path and the skill stays purely declarative.
5. **Toolset authoring location** — `~/.agh/toolsets/*.toml` (global), workspace overlay `<root>/.agh/toolsets/*.toml`, or skill-frontmatter-declared (`metadata.agh.toolset = "research"`)? The first is simplest; the third lets a skill announce membership but raises governance issues.
6. **Telemetry retention** — usage counters can grow unbounded. Where do they live? Per-session DB is wrong (sessions die). Need a `globaldb.tool_usage` table with rolling windows and a TTL.
7. **Network-level skill privacy** — should joining a public channel auto-advertise the loaded skill set? Some skills reference internal repos / proprietary tools whose mere names leak intent. Need a `skills.network_visibility = "public"|"channel"|"none"` toggle, possibly per skill via `metadata.agh.network_visible`.
8. **Catalog freshness on resume** — `Manager.Resume` (`session/manager_lifecycle.go`) re-evaluates skills via `ForWorkspace`. If P5 lands, we must decide whether the resumed agent receives a delta of changes since last `UpdatedAt` or starts fresh; this interacts with the dream-consolidation memory model.
9. **Where do CLI subcommands live in the catalog?** The `agh-network` bundled skill currently teaches the model the `agh network ...` CLI verbs as text. Should the daemon enumerate its own cobra commands as `tools.Tool` records with `Source=builtin` and `Namespace=cli.*`, so the agent doesn't depend on the prose of a skill to know they exist?
10. **MCP namespace collision** — Hermes allows `mcp-*` → `mcp-*` overwrites (`registry.py:194-203`). Should AGH adopt the same rule, or should it require explicit namespace prefixes (`mcp.<server>.<tool>`) to keep collisions impossible (claude-code `MCPTool` uses the prefix form)? Today there is no central enforcement — each ACP runtime decides.

---

**Bottom line:** AGH has the *building blocks* of a real skills-and-tools registry (file-watched skill loader, resource codec, marketplace installer, network capability brief) but it lacks the *runtime surfaces* an autonomous agent needs to use them mid-session: a callable tool catalog, on-demand skill loading via a real ACP-side tool, per-role scoping, peer-skill projection, install-from-session, and usage telemetry. P1–P3 unblock G1/G3/G4/G10; P4 unblocks G5; P5 unblocks G2; P6 unblocks G6/G7; P8 unblocks G8; P9 unblocks G9.
