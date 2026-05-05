# Analysis: Agent-Definition Scope vs Agent-Memory Scope (mem-v2)

**Date:** 2026-05-04
**Scope:** Resolve the contradiction in `analysis_layered-scope.md` §8: an agent can be defined globally OR per-workspace, but the previous recommendation pinned agent memory to the workspace. Decide where agent memory actually lives, in code, with competitor evidence.
**Read-only.** Cites `path:line`. Five candidates analyzed; one verdict picked.

---

## 1. TL;DR

**Adopt C3 — two-tier agent memory: a global baseline at `$AGH_HOME/agents/<agent>/memory/` plus a workspace override at `<workspace>/.agh/agents/<agent>/memory/`. Catalog both rows with `(scope='agent', scope_owner='global'|'workspace_id', agent_name)`. Resolution at read time: workspace-agent shadows global-agent shadows workspace shadows global, by `<type>_<slug>`.**

Rationale, three lines:

1. **Competitor evidence forces it.** OpenClaw treats the agent as the primary key for memory at `~/.openclaw/memory/<agentId>.sqlite` (`packages/memory-host-sdk/src/host/config-utils.ts:270-290`); Claude Code ships THREE agent-memory scopes — `'user'` (global), `'project'` (workspace, VCS-shared), `'local'` (workspace, gitignored) — `tools/AgentTool/agentMemory.ts:12-65`. Hermes scopes via `HERMES_HOME` profile, where memory IS the home — agent-bound by definition. Codex has no agent memory (only sub-agent isolation). Three of the four "real" systems give the agent its own durable namespace, and Claude Code explicitly ships both the global-baseline AND workspace-override shapes.
2. **AGH's own reality forces it.** AGH agents already live at BOTH `$AGH_HOME/agents/<name>/AGENT.md` AND `<workspace>/.agh/agents/<name>/AGENT.md` (`internal/config/agent.go:163-169` `AgentsDir()`); `internal/CLAUDE.md` "Memory & Skills Runtime" guarantees "agent-local overriding all". Pinning memory to workspace alone breaks the "global agent retains learnings across projects" use case (the scenario the user explicitly flagged); pinning to global alone breaks the "feedback rule learned in workspace A leaks to workspace B" safety property. The choice is not either/or — it is **both, with deterministic precedence**.
3. **Agent-manageability needs the verb, not just the location.** Add `agh memory promote --scope agent-global` and `agh memory demote --scope agent-workspace` so operators choose movement explicitly, matching the greenfield "no compat shim" rule. Default *write* scope per `AgentDef.memory.scope = "agent-workspace" | "agent-global"` (workspace as default — safer for multi-project users).

Migration delta: extends `analysis_layered-scope.md` §10 with one extra scope-owner discriminator (`scope_owner TEXT`) on the catalog and one extra path on disk (`$AGH_HOME/agents/<agent>/memory/`). Touches the contract, CLI, HTTP/UDS, native tools, extension host_api, dream worker — single atomic change per greenfield rule.

---

## 2. The dilemma stated precisely

### 2.1 Concrete scenario

Operator defines `pedro-reviewer` once:

```
$AGH_HOME/agents/pedro-reviewer/AGENT.md       # global definition
```

Operator runs sessions in two workspaces:

- **Workspace A**: `~/dev/agh` (Go monorepo). Agent learns: *"feedback: pedro reviews require race-detector evidence on concurrency PRs"*.
- **Workspace B**: `~/dev/portfolio` (static-site project). Agent learns: *"user: pedro prefers Tailwind over plain CSS"*.

The contradiction (`analysis_layered-scope.md` §8 Candidate B):

> Per the recommendation, both lessons live at `<workspace-A>/.agh/agents/pedro-reviewer/memory/` and `<workspace-B>/.agh/agents/pedro-reviewer/memory/`.
>
> - The race-detector rule is **correctly** workspace-bound — it only applies in the Go repo.
> - The Tailwind preference is **incorrectly** workspace-bound — it's a stable user fact about Pedro that the agent should retain in workspace C, D, E, F.

Workspace-only memory **starts blank** every time the agent runs in a new workspace. Global-only memory **leaks** workspace-private context across workspaces. Neither single-tier choice is correct; the right primitive is two-tier with explicit promotion.

### 2.2 Why this is load-bearing for AGH

- `internal/CLAUDE.md` "Memory & Skills Runtime" promises "agent-local overriding all" — without an agent-global tier, the only place agent memory can live is workspace, which means cross-workspace agent learnings are silently dropped. That contradicts both the user/feedback memory taxonomy (`internal/CLAUDE.md` "Memory taxonomy: `user | feedback | project | reference` types") and the agent-manageable principle.
- Greenfield rule (`CLAUDE.md` "Greenfield Alpha — Zero Legacy Tolerance") means we cannot bridge later — the v2 schema must commit to the right tier count *now*.

---

## 3. Per-competitor matrix (code-cited)

### 3.1 Hermes

| Question | Answer | Evidence |
|---|---|---|
| Where is the **agent definition**? | No file-based agent definitions. Identity is the `agent_identity` kwarg passed to `MemoryProvider.initialize()` plus a profile-aware `HERMES_HOME`. | `agent/memory_provider.py:62-82` (kwargs include `hermes_home`, `agent_identity`, `agent_workspace`); `hermes_cli/main.py:96-163` (profiles set `HERMES_HOME` to `<root>/profiles/<name>` before module import). |
| Where is the **agent memory**? | `$HERMES_HOME/memories/{MEMORY.md, USER.md}` and `$HERMES_HOME/state.db` (sessions). Memory IS the home — there is no separate agent root. | `analysis_hermes.md` §3; `agent/memory_provider.py:69-71` ("hermes_home: The active HERMES_HOME directory path. Use this for profile-scoped storage instead of hardcoding `~/.hermes`."); `hermes_state.py:43-72` (no `agents` table — `sessions` keyed by `id` only, no `agent_id` foreign key). |
| Shared across uses or context-scoped? | **Context-scoped via `HERMES_HOME` selection.** Different profile = different home = different memory + state. Within one profile, memory is shared across all sessions of that profile. | `agent/memory_provider.py:77-79` ("agent_identity: Profile name (e.g. 'coder'). Use for per-profile provider identity scoping."); `hermes_cli/main.py:147-163` (profile flag mutates `HERMES_HOME` before any module imports). |
| Resolution rule | Process-launch decision: `HERMES_HOME` is set once, before module imports run. No multi-scope merge — single root. | `hermes_cli/main.py:94-101` ("Many modules cache HERMES_HOME at import time (module-level constants). … every subsequent `os.getenv('HERMES_HOME', ...)` resolves correctly."). |

**Verdict:** Hermes implements **C1 (memory follows agent definition)** by collapsing the question — the profile *is* both the definition and the home. No multi-tier; no workspace-vs-global split. Workable for a single-user CLI; insufficient for AGH's multi-workspace daemon model.

### 3.2 OpenClaw

| Question | Answer | Evidence |
|---|---|---|
| Where is the **agent definition**? | Per-agent dir at `$STATE_DIR/agents/<agentId>/agent/` (config, auth-profiles, tools). `agentDir` field in config; canonical path `~/.openclaw/agents/<agentId>/agent`. | `src/agents/agent-scope.ts` (referenced at `src/extensionAPI.ts:20`); `src/config/normalize-paths.test.ts:42` (`agentDir: "~/.openclaw/agents/main"`); `src/infra/state-migrations.ts:727` (`stateDir/agents/<targetAgentId>/agent`). |
| Where is the **agent memory**? | `~/.openclaw/memory/<agentId>.sqlite` (per-agent SQLite memory index) + a **per-agent workspace dir** that holds `MEMORY.md`. | `src/config/schema.help.ts:1044` ("Sets where the SQLite memory index is stored on disk for each agent. Keep the default `~/.openclaw/memory/{agentId}.sqlite`"); `packages/memory-host-sdk/src/host/config-utils.ts:270-290` `resolveAgentWorkspaceDir` — per-agent workspace, default `<stateDir>/workspace-<agentId>` for non-default agents, `resolveDefaultAgentWorkspaceDir(env)` for default. |
| Shared across uses or context-scoped? | **Per-agent global.** Memory is keyed by `agentId`; no notion of "the same agent in another workspace" — agents own their workspace. | `src/config/config.multi-agent-agentdir-validation.test.ts:8-22` (refuses to run two agents with the same `agentDir` — agent ID owns its dir exclusively). |
| Resolution rule | `(agentId)` → `agentDir` (definition) + `memoryStorePath` (sqlite) + `workspaceDir` (markdown). All three are per-agent-global; collisions across agents are rejected at config validation. | `src/config/agent-dirs.ts:85-110` ("Each agent must have a unique agentDir; sharing it causes auth/session state collisions"). |

**Verdict:** OpenClaw implements **pure C1** — agent owns one global home that contains both definition and memory. There is no workspace-bound override at all. Feasible for a tool oriented around long-lived conversational agents (Telegram bot, Discord bot); too rigid for AGH's "same agent serves multiple project workspaces" use case.

### 3.3 Claude Code

| Question | Answer | Evidence |
|---|---|---|
| Where is the **agent definition**? | Three sources merged by precedence: built-ins, plugin agents (npm/marketplace), user agents (`~/.claude/agents/*.md`), project agents (`<cwd>/.claude/agents/*.md`), policy agents (managed). | `tools/AgentTool/loadAgentsDir.ts:296-373`; `getActiveAgentsFromList()` `:193-221` (precedence: built-in < plugin < user < project < flag < managed, last-wins by `agentType`). |
| Where is the **agent memory**? | **Three explicit scopes per agent**, declared in agent frontmatter (`memory: 'user' \| 'project' \| 'local'`): `'user'` → `<memoryBase>/agent-memory/<agentType>/`, `'project'` → `<cwd>/.claude/agent-memory/<agentType>/`, `'local'` → `<cwd>/.claude/agent-memory-local/<agentType>/` (gitignored, with optional remote-mount redirect). | `tools/AgentTool/agentMemory.ts:12-65` `getAgentMemoryDir`; `:594-605` parse from frontmatter in `loadAgentsDir.ts`. |
| Shared across uses or context-scoped? | **Both, by agent's choice.** A user-scope agent persists across all projects. A project-scope agent commits its memory to VCS. A local-scope agent stays on this machine, gitignored. | `agentMemory.ts:142-156` (scope notes injected into the agent's system prompt — `'user'` says "keep learnings general since they apply across all projects"; `'project'` says "tailor your memories to this project"; `'local'` says "tailor your memories to this project and machine"). |
| Resolution rule | Agent declares ONE scope at definition time. There is no multi-tier merge — the agent reads from + writes to exactly one path. **But** a project-snapshot mechanism (`agentMemorySnapshot.ts`) seeds user-scope from a project snapshot the first time the user runs an agent that has one. | `loadAgentsDir.ts:262-294` `initializeAgentMemorySnapshots` — for `memory === 'user'` agents only, copies a project-shipped snapshot into the user's home memory dir on first run. |

**Verdict:** Claude Code implements **C5 (agent declares its own scope)** — but with three options that mirror C1/C2/C3 sub-cases:
- `memory: 'user'` ≈ C1 global-only.
- `memory: 'project'` ≈ C2 workspace-only.
- `memory: 'local'` ≈ C2 with a gitignore twist.
- The snapshot bootstrap is a **one-shot C3-ish hop**: project ships a baseline; user accumulates from there.

Critically, Claude Code does NOT do the *runtime two-tier merge*. Each agent picks one scope and stays there. The project→user snapshot bootstrap is one-time at install, not a live read precedence chain.

### 3.4 Codex

| Question | Answer | Evidence |
|---|---|---|
| Where is the **agent definition**? | Codex has **no first-class agent definitions** in the AGH/Claude-Code sense. Sub-agents are spawned in-process with isolated context windows; no per-agent durable identity. | `analysis_codex.md` §15.2 ("Sub-agents are read-only with respect to global memory. They do not generate memory; they don't inherit memory_summary in their prompt unless explicitly handed it."); `codex-rs/core/src/context/subagent_notification.rs` (sub-agent context is ephemeral). |
| Where is the **agent memory**? | Single global memory at `<codex_home>/memories/{MEMORY.md, memory_summary.md, raw_memories.md}`. Plus per-cwd `AGENTS.md` walk (instructions, not memory). | `analysis_codex.md` §3.1; `codex-rs/core/src/agents_md.rs:212-303` (AGENTS.md walk root→leaf); `codex-rs/memories/write/src/start.rs:33` (memory generation skipped for sub-agents). |
| Shared across uses or context-scoped? | Memory is **global**; there is no agent-vs-workspace partition. Sub-agents inherit nothing by default. | `analysis_codex.md` §15. |
| Resolution rule | N/A — single-tier. AGENTS.md merges root→leaf for *instructions*; memory is one file at one path. | `codex-rs/core/src/agents_md.rs:130-141` (instruction_sources is read-only, lists files but doesn't reload). |

**Verdict:** Codex chose **single-global-only** for memory; sub-agents are deliberately kept stateless. Useful as a "what NOT to do for agent-scoped memory" reference — they ship the gap, and the analysis acknowledges it (`analysis_codex.md` §20.2 — "no per-cwd memory" listed as a known weak spot).

### 3.5 Cross-competitor distillation

| System | Agent-def scope | Agent-mem scope | Cross-workspace baseline? | Workspace-private overrides? |
|---|---|---|---|---|
| Hermes | Profile (≈ global) | Profile root | Yes (one root for all uses) | No (single tier) |
| OpenClaw | Per-agent global | Per-agent global | Yes | No |
| Claude Code | Per-source (user OR project) | Per-agent declared (one of `user`/`project`/`local`) | Only when agent declares `memory: 'user'` | Only when agent declares `memory: 'project'`/`local'` |
| Codex | None | Global only | Yes (single file) | No |
| **AGH today** | Global + workspace (file-based) | Global + workspace (no agent tier) | Memory has no agent tier at all | — |

Two patterns dominate: **(A) one-tier with the agent owning a global root** (Hermes/OpenClaw) and **(B) declare-your-scope with no runtime merge** (Claude Code). Neither directly fits AGH because:

- AGH already ships the dual agent-definition layout (`$AGH_HOME/agents/` + `<workspace>/.agh/agents/`) — `(A)` would require collapsing to one. Greenfield + the user's workflow (run same agent across projects) makes that wrong.
- AGH already ships multi-scope memory with a precedence model (`agent ▸ workspace ▸ global`) — `(B)`'s "pick one scope per agent" loses the precedence chain.

The right shape for AGH is **runtime two-tier merge for the agent dimension**, scoped by `(agent_name, scope_owner)` where `scope_owner ∈ {'global', workspace_id}`. None of the competitors does exactly this — but Claude Code's three-scope frontmatter + OpenClaw's per-agent global root + AGH's existing dual agent-def layout combine into it cleanly.

---

## 4. Five candidate architectures

### C1 — Memory follows agent definition

**Rule:** Agent defined in `$AGH_HOME/agents/<n>/` → memory at `$AGH_HOME/agents/<n>/memory/`. Agent defined in `<ws>/.agh/agents/<n>/` → memory at `<ws>/.agh/agents/<n>/memory/`.

| Aspect | Verdict |
|---|---|
| Pros | RFC 001's literal wording (`internal/CLAUDE.md` "Memory & Skills Runtime"). Co-locates def + memory. Simple mental model: "where the agent lives, its memory lives". |
| Cons | If the SAME agent is defined globally and run in a workspace, ALL learnings (including workspace-private feedback) leak across workspaces. **Wrong default for any pro user with multiple projects.** Cannot represent "this is a global agent, but this lesson is workspace-private". Forces awkward duplication: copy `pedro-reviewer/AGENT.md` into each workspace to get workspace memory. |
| Complexity | Low — one path resolver. |
| Agent-manageability cost | Low — one CLI verb (`agh memory write --scope agent`). |
| Evidence supporting | OpenClaw (`packages/memory-host-sdk/src/host/config-utils.ts:270-290`) — but OpenClaw has no workspace concept distinct from the agent's home. |

### C2 — Memory always workspace-bound (`analysis_layered-scope.md` §8 verdict)

**Rule:** Regardless of where the agent is defined, agent memory lives at `<workspace>/.agh/agents/<n>/memory/`. Cross-workspace use of the same agent starts blank.

| Aspect | Verdict |
|---|---|
| Pros | Zero cross-workspace leak. Workspace-cohesive: rm-the-workspace-dir wipes everything. Single agent-scope path everywhere. |
| Cons | **Breaks the "global agent retains user preferences" use case** (the explicit user concern). Forces operators to manually re-teach `pedro-reviewer` in every workspace. Only path to share is global scope under `agent_name`-namespaced filenames — that's a hack: it abandons the agent-scope abstraction whenever cross-workspace persistence is wanted. |
| Complexity | Low — one path resolver. |
| Agent-manageability cost | Medium — operators must promote-to-global manually for cross-workspace facts. |
| Evidence supporting | None. No competitor ships this exclusively. |

### C3 — Two-tier (global agent + workspace agent override)

**Rule:** Both `$AGH_HOME/agents/<n>/memory/` (cross-workspace baseline) AND `<workspace>/.agh/agents/<n>/memory/` (workspace override) coexist. Read precedence: workspace-agent ▸ global-agent ▸ workspace ▸ global. Write default declared on `AgentDef.memory.scope`.

| Aspect | Verdict |
|---|---|
| Pros | Captures the real-world use case (some agent learnings cross workspaces, others don't). Mirrors AGH's existing dual agent-DEFINITION discovery. Mirrors the way `internal/CLAUDE.md` already names "agent-local overriding all" — agent-local CAN be agent-global or agent-workspace. Greenfield-clean: one extra column on catalog (`scope_owner`), one extra dir on disk. |
| Cons | Four-way precedence stack `workspace-agent ▸ global-agent ▸ workspace ▸ global` is more surface area than three. Two places to look for a piece of agent memory. |
| Complexity | Medium — adds `scope_owner` discriminator to catalog rows + one extra disk path + extra resolver branch. |
| Agent-manageability cost | Medium — adds two verbs: `agh memory promote --scope agent-global <slug>`, `agh memory demote --scope agent-workspace <slug>`. CLI takes both `--agent <name>` and `--scope-owner <global\|workspace>`. |
| Evidence supporting | Closest to **Claude Code's three-scope model** (`tools/AgentTool/agentMemory.ts:12-65`) — Claude Code ships `'user'`/`'project'`/`'local'` to capture exactly this; AGH collapses to two (`agent-global`/`agent-workspace`) because we don't have a separate gitignored workspace tier (`.agh/` is one tree). |

### C4 — Type-routed (memory type owns the scope)

**Rule:** Memory type drives placement. `feedback` and `user` types → global; `project` and `reference` → workspace. Agent definition has no effect.

| Aspect | Verdict |
|---|---|
| Pros | Matches the natural taxonomy: "user preferences are global, project facts are workspace". Removes agent dimension entirely — fewer paths. |
| Cons | **Discards the agent dimension** — but the user explicitly wants `pedro-reviewer` to have its OWN memory separate from `linus-reviewer`. Type alone cannot encode "this user preference is private to this reviewer agent". Reduces "agent-local overriding all" (`internal/CLAUDE.md`) to a dead promise. |
| Complexity | Low. |
| Agent-manageability cost | Low — but loses agent isolation. |
| Evidence supporting | Goclaw approximates it (`(agent_id, user_id)` composite key in `memory_chunks`, `analysis_goclaw-paperclip-multica.md` §1.3), but goclaw has no workspace concept. No competitor blends type-routing with multi-workspace. |

### C5 — Agent-declared scope (Claude Code shape)

**Rule:** `AgentDef.memory.scope ∈ {"agent-global", "agent-workspace", "follow-type"}`. Agent picks ONE; runtime never merges.

| Aspect | Verdict |
|---|---|
| Pros | Matches Claude Code's production-tested model. Per-agent flexibility. No precedence chain to debug. |
| Cons | **Forces operators to choose at agent-definition time, before they know which lessons they'll learn.** A `pedro-reviewer` that picks `agent-global` then learns a workspace-private rule has nowhere to put it. Asymmetric: an `agent-workspace` agent has no way to preserve the user-preference type. Means agent definitions become memory-policy declarations, conflating concerns. |
| Complexity | Low — one resolver dispatch. |
| Agent-manageability cost | Low — but locks operators in until they edit the agent def. |
| Evidence supporting | Claude Code (`tools/AgentTool/loadAgentsDir.ts:594-605` parse from frontmatter; `:262-294` `initializeAgentMemorySnapshots` does a one-time bootstrap from project to user — exactly the pattern needed when scope is wrong). The fact that Claude Code ALSO ships the snapshot mechanism is evidence that scope-only-declaration is incomplete. |

---

## 5. Recommended verdict — **C3, two-tier with deterministic precedence**

**Pick C3.** Defense:

1. **It's the only candidate that doesn't strand a real use case.** C2 strands cross-workspace user preferences; C1 strands workspace-private feedback; C4 strands per-agent isolation; C5 strands whichever scope the agent didn't pick. C3 has a home for every memory the user described.
2. **It composes with AGH's existing layered design.** AGH already discovers agent definitions across global + additional + workspace (`internal/config/agent.go:131-159` `WorkspaceDiscoveryRoots`). C3 mirrors the same layout for memory — one extra disk path, one extra catalog column. The precedence chain `workspace-agent ▸ global-agent ▸ workspace ▸ global` is the four-row generalization of "agent-local overriding all" that `internal/CLAUDE.md` already commits to.
3. **It harvests Claude Code's evidence without inheriting their UX gap.** Claude Code's three-scope `'user'/'project'/'local'` shape proves operators want both tiers; their snapshot bootstrap proves scope-locked is insufficient. C3 makes both tiers first-class with explicit `agh memory promote/demote` verbs (closing the "no first-class delete/forget UX" gap noted in `analysis_claude-code.md` §13).
4. **Greenfield-clean.** No bridges: one schema migration adds `scope_owner` + `agent_name` columns; the four-row CHECK constraint replaces the existing two-row CHECK in one atomic change.
5. **Defaults are safer than C1.** Default write scope is `agent-workspace` (configurable per agent). Workspace-private leak risk is the *opt-in*, not the default. Same safety property as C2, plus the recovery valve (promote when needed).

**Default write scope policy:**
```toml
# AgentDef
[memory]
scope        = "agent-workspace"   # default; one of: agent-workspace | agent-global | workspace | global
type_routing = "explicit"          # or "default" — when "default", type drives scope per DefaultScopeForType
```

**Resolution at read time:**
```
RECALL(query, ws, agent) →
  read curated entries from:
    [1] <ws>/.agh/agents/<agent>/memory/         # workspace-agent
    [2] $AGH_HOME/agents/<agent>/memory/         # global-agent
    [3] <ws>/.agh/memory/                        # workspace
    [4] $AGH_HOME/memory/                        # global
  shadow-by-(<type>_<slug>): deeper wins; emit shadow event for each silent loss.
  return projection in prompt order [4][3][2][1] (least-specific first; most-specific last so LLM weights it strongest).
```

---

## 6. AGH-specific implications

### 6.1 Filesystem layout (final)

```
$AGH_HOME/
├── agh.db                                   # global catalog DB
├── agents/<agent>/
│   ├── AGENT.md                             # agent definition (existing)
│   ├── identity.toml                        # reserved for agent identity/runtime state
│   └── memory/                              # NEW — global-agent scope (C3)
│       ├── MEMORY.md                        # projection from catalog
│       ├── user_<slug>.md
│       ├── feedback_<slug>.md
│       ├── reference_<slug>.md
│       ├── daily/YYYY-MM-DD.md
│       └── _system/{dreaming,extractor,ad_hoc}/
├── memory/                                  # global scope (existing, unchanged)
└── skills/                                  # unchanged

<workspace>/.agh/
├── agh.db                                   # workspace catalog DB (NEW per analysis_layered-scope.md)
├── workspace.toml                           # holds workspace_id (NEW)
├── agents/<agent>/
│   ├── AGENT.md                             # workspace agent override (existing)
│   └── memory/                              # workspace-agent scope (was Candidate B in §8)
│       ├── MEMORY.md
│       ├── feedback_<slug>.md
│       ├── project_<slug>.md
│       ├── daily/YYYY-MM-DD.md
│       └── _system/{dreaming,extractor,ad_hoc}/
└── memory/                                  # workspace scope (existing)
```

### 6.2 Catalog DB layout

Single `memory_catalog_entries` schema, two physical DBs (global + per-workspace):

```sql
-- Migration v3 add_agent_two_tier (replaces analysis_layered-scope.md v3 add_agent_scope)
ALTER TABLE memory_catalog_entries ADD COLUMN agent_name   TEXT;
ALTER TABLE memory_catalog_entries ADD COLUMN scope_owner  TEXT;  -- 'global' | <workspace_id> | NULL
-- Replace scope CHECK:
-- before: CHECK (scope IN ('global','workspace'))
-- after:  CHECK (scope IN ('global','workspace','agent','session'))
-- Add disambiguator:
-- when scope='agent' AND scope_owner='global'        → row in $AGH_HOME/agh.db
-- when scope='agent' AND scope_owner='<workspace_id>' → row in <ws>/.agh/agh.db
-- when scope='workspace'                              → workspace_id in scope_owner; agent_name NULL
-- when scope='global'                                 → scope_owner='global', agent_name NULL
```

`workspace_id`-NULL semantics: agent-only-global memory (scope='agent' AND scope_owner='global') has `workspace_id = NULL`. Workspace-agent memory has `workspace_id = <uuid>` AND `scope_owner = <workspace_id>` (redundant for query-shape clarity; one canonical join key).

Catalog row for the dilemma scenario:

| `scope` | `scope_owner` | `workspace_id` | `agent_name` | `type` | `slug` | DB |
|---|---|---|---|---|---|---|
| `agent` | `global` | NULL | `pedro-reviewer` | `user` | `tailwind_pref` | `$AGH_HOME/agh.db` |
| `agent` | `<wsA-uuid>` | `<wsA-uuid>` | `pedro-reviewer` | `feedback` | `race_evidence` | `<wsA>/.agh/agh.db` |

### 6.3 Precedence (4-scope, deterministic)

`workspace-agent (1) ▸ global-agent (2) ▸ workspace (3) ▸ global (4)`. Shadow rule from `analysis_layered-scope.md` §9 generalizes by adding tier 2: shadowed entry stays on disk, recall returns deepest, emit `op=shadowed scope_winner=<i> scope_loser=<j>` event for each loss.

Read order in prompt assembly (least-specific first so LLM weights deepest last): `[4] global → [3] workspace → [2] global-agent → [1] workspace-agent`.

### 6.4 CLI / HTTP / UDS / native-tool surface (delta)

- `agh memory write --scope agent --scope-owner <global|workspace> --agent <name> --type <t> --slug <s>` — explicit two-tier write.
- `agh memory promote --scope-owner global --agent <name> <slug>` — copy entry from `<ws>/.agh/agents/<n>/memory/` → `$AGH_HOME/agents/<n>/memory/`. Source is preserved unless `--remove` is passed.
- `agh memory demote --scope-owner workspace --agent <name> <slug>` — reverse.
- HTTP/UDS: body adds `"scope": "agent"`, `"scope_owner": "global"|"<workspace_id>"`, `"agent_name": "..."`. CLI flag presence detection per `agh-code-guidelines`.
- Native tool: `agh__memory_propose({scope: "agent", scope_owner: "workspace", agent_name: "pedro-reviewer", type: "feedback", slug: "race_evidence", ...})`.
- `internal/extension/contract/host_api.go` — `Write/Read/List` gain `ScopeOwner string` and `AgentName string`. Existing extensions panic on unknown enum (caught by tests; greenfield-acceptable).
- `internal/situation/service.go` `/agent/context` — agent-scope load layers both tiers; deepest wins on conflict; both injected as labelled prompt sections (`## Agent Memory (workspace)` then `## Agent Memory (global)` — LLM treats workspace as override).

### 6.5 Migration delta from AGH today

| Today | v2 (C3) | Action |
|---|---|---|
| `internal/memory/types.go:28-33` `ScopeGlobal/ScopeWorkspace` | + `ScopeAgent`, `ScopeSession`, `ScopeOwner` enum | Add `ScopeOwner` type with `ScopeOwnerGlobal`, `ScopeOwnerWorkspace(workspaceID)`. |
| `internal/memory/store.go:38-46` `Store{globalDir, workspaceDir}` | `Store{globalDir, workspaceDir, agentDir}` + agent-tier resolver | `dirForScope(scope, owner, agentName)` — case `ScopeAgent`+`ScopeOwnerGlobal` → `$AGH_HOME/agents/<name>/memory/`; `ScopeAgent`+`ScopeOwnerWorkspace(id)` → `<ws>/.agh/agents/<name>/memory/`. |
| `internal/memory/catalog.go:34-87` schema | numbered migration v3 `add_agent_two_tier` | Adds `scope_owner`, `agent_name` columns; extends scope CHECK; backfills NULL for existing rows; rebuilds FTS5 shadow tables. Per `agh-schema-migration` skill. |
| `internal/config/agent.go` `AgentDef` | + `Memory MemoryConfig{Scope, ScopeOwner, TypeRouting}` | Wire through `internal/config/agent_resource.go`; default `Scope = "agent-workspace"`. |
| `internal/api/contract` memory types | + `Scope = "agent"` enum, `ScopeOwner` field | Triggers codegen co-ship per `agh-contract-codegen-coship`. |
| `internal/cli/memory*.go` | + `--agent`, `--scope-owner` flags | Mandatory when `--scope agent`. |
| `internal/memory/dream.go` | walks both `<ws>/.agh/agents/*/memory/` AND `$AGH_HOME/agents/*/memory/` | Per-tier consolidation candidates; promotion gate honors tier transition. |
| `internal/registry/` (multi-source agent discovery) | unchanged for definition; memory resolution adds tier picker | Definition layering already exists; memory tier picker follows `(agent_def_source, AgentDef.Memory.Scope)` — global agent defaults to `agent-global`, workspace agent defaults to `agent-workspace`, override-able per-call. |

### 6.6 Hard cuts (greenfield rule)

- Delete the `analysis_layered-scope.md` §8 Candidate B-only resolution code path before writing v2.
- No `scope='agent'` rows without `scope_owner` and `agent_name` set — CHECK constraint enforces.
- No alias verbs: `agh memory promote` is the only verb that moves entries across tiers; no fallback "auto-promote" heuristic.

---

## 7. Open sub-questions for the TechSpec

1. **Default `AgentDef.memory.scope` per agent-definition origin.** A workspace-defined agent should default to `agent-workspace`. A globally-defined agent — should it default to `agent-workspace` (safer, requires explicit promote) or `agent-global` (matches RFC 001 spirit)? **Pre-recommendation: `agent-workspace` even for global defs — least-leakage default.**
2. **Sub-agent memory inheritance.** When a parent agent spawns a sub-agent, does the sub-agent read the parent's agent-tier memory? Per Codex (`analysis_codex.md` §15.2) sub-agents should be read-only. Per `internal/CLAUDE.md` Concurrency, sub-agents inherit context but writes are gated. **TechSpec must answer: do sub-agents inherit the parent's `(agent_name, scope_owner)` for *recall*? `analysis.md` P12 says sub-agent controller is `Mode=ReadOnly` — confirm this means READ both tiers, WRITE neither.**
3. **Promotion semantics — copy or move?** When `agh memory promote --scope-owner global` runs, does the workspace tier keep the entry? **Pre-recommendation: copy by default (with shadow-flag), `--remove` for explicit move.** Avoid silent removal.
4. **Workspace-id-less agents.** When the runtime resolves an agent without an active workspace (e.g., daemon-startup health check, network channel from a peer), agent-tier writes are valid only in `agent-global`. **TechSpec must confirm: workspace-agent writes outside an active workspace are rejected with explicit error, not silently routed to global.**
5. **Cross-tier conflict during dreaming.** If the `dreaming` worker's consolidation produces a curated entry that already exists in the OTHER tier, does it write into the workspace tier (default), promote into global (if score crosses threshold), or stay put with a shadow event? Goclaw promotion gates (`analysis_goclaw-paperclip-multica.md` §1.3) suggest score/recallCount thresholds; reuse those for cross-tier promotion. **TechSpec must define the threshold + the operator override path.**

---

## 8. Citations summary

- AGH agent-def discovery: `internal/config/agent.go:131-169` `WorkspaceDiscoveryRoots` + `AgentsDir`; `:180-223` `LoadWorkspaceAgentDefs`; `:163-169` global vs workspace branch.
- AGH "agent-local overriding all": `internal/CLAUDE.md` "Memory & Skills Runtime" five-layer precedence.
- AGH skill-level agent override pattern (template for memory): `internal/skills/registry_agent.go:82-119` `SetEnabledForAgent` — agent-scope tombstones via in-place AGENT.md edit.
- Hermes profile-aware HOME: `.resources/hermes/hermes_cli/main.py:94-163`; memory_provider kwargs: `.resources/hermes/agent/memory_provider.py:62-82`; no agents table: `.resources/hermes/hermes_state.py:43-72`.
- OpenClaw per-agent global: `.resources/openclaw/packages/memory-host-sdk/src/host/config-utils.ts:270-290` `resolveAgentWorkspaceDir`; `.resources/openclaw/src/config/schema.help.ts:1044` memory store path; `.resources/openclaw/src/infra/state-migrations.ts:727` agent dir layout.
- Claude Code three-scope agent memory: `.resources/claude-code/tools/AgentTool/agentMemory.ts:12-65` `getAgentMemoryDir`; `:594-605` frontmatter parse in `loadAgentsDir.ts`; `:262-294` snapshot bootstrap.
- Codex no-agent-memory: `.compozy/tasks/mem-v2/analysis/analysis_codex.md` §15.2 (sub-agents read-only); `.resources/codex/codex-rs/memories/write/src/start.rs:33` (memory generation skipped).
- Prior verdict to override: `.compozy/tasks/mem-v2/analysis/analysis_layered-scope.md` §8 Candidate B (workspace-bound agent scope).
