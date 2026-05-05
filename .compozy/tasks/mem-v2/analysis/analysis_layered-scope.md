# Analysis: Layered Filesystem-Scope Design for AGH Memory v2

**Date:** 2026-05-04
**Scope:** opinionated layered-scope design (`global` / `workspace` / `agent`) for memory v2. Decides on-disk layout, precedence model, lifecycle, and migration delta. Read-only; cites `path:line` and prior analyses.
**Decision frame:** "híbrido escopado" model — Markdown autoritativo for curated memory, append-only event log for signals/audit, SQLite catalog as derived index.

---

## 1. TL;DR

Adopt **three first-class scopes** (`global`, `workspace`, `agent`) plus a **derived `session` slice** that is part of the event log, not the file tree. Layout is uniform across scopes:

```
<scope_root>/memory/
  MEMORY.md                              ← projection (rendered from catalog)
  <type>_<slug>.md                       ← curated topic file with frontmatter
  daily/YYYY-MM-DD.md                    ← per-scope append-only daily log
  _system/
    dreaming/YYYYMMDD-<slug>.md          ← consolidation outputs
    extractor/YYYY-MM-DD.md              ← extraction logs / failures
    ad_hoc/<ts>-<slug>.md                ← agent/operator delta notes
```

`<scope_root>` is `$AGH_HOME` for global, `<workspace>/.agh` for workspace, **`<workspace>/.agh/agents/<agent>/`** for agent (option B — workspace-bound; defended in §8). Single `<workspace>/.agh/memory/agh.db` per workspace holds the catalog + event log; global catalog stays at `$AGH_HOME/agh.db`. Agent-scope catalog rows live in the workspace DB (no separate per-agent DB).

**Precedence (read-side):** `agent ▸ workspace ▸ global`. Deeper scope **shadows-by-id**, never merges silently. **Default write scope** is declared on `AgentDef.memory.scope`; explicit per-call override via `--scope` / `scope:` field. **`AGENTS.md`** stays in instruction hierarchy (Codex root→leaf walk), separate from memory tree (P11). **Skills** stay in their own roots; memory references them by import statement, not by colocation.

**Why not the RFC 001 path `.agents/<name>/memory/`?** Because RFC 001 wrote that line before the workspace lifecycle was settled. OpenClaw's per-agent dir is the production-tested shape; mounting it under `<workspace>/.agh/agents/<agent>/` keeps agent memory naturally co-versioned with the workspace it was learned in, while preserving `$AGH_HOME/agents/<agent>/` for **agent identity / runtime state** (not memory).

---

## 2. Comparative on-disk layout (one row = scope semantics)

Scopes existing in each system, where they live, and the precedence rule. All paths absolute or relative-to-root as the source uses them.

| System | Global / user scope | Workspace / project scope | Agent / per-agent scope | Daily / log scope | Precedence rule (read) |
|---|---|---|---|---|---|
| **AGH today** | `$AGH_HOME/memory/{MEMORY.md, *.md, .consolidate-lock}` (`internal/memory/store.go:22-30`, `:1216-1223`) | `<workspace>/.agh/memory/{MEMORY.md, *.md}` | **none** (`analysis_agh-current.md` §3, §16 — RFC 001 promised `.agents/<name>/memory/` never landed) | none | Both indexes injected together via `Assembler` (`internal/memory/assembler.go:98-117`); no precedence — concatenated |
| **Codex** | `~/.codex/memories/{MEMORY.md, memory_summary.md, raw_memories.md, rollout_summaries/, skills/, extensions/ad_hoc/notes/<ts>-<slug>.md}` + `~/.codex/AGENTS.md` (`analysis_codex.md` §3.1, §4.4) | per-cwd `AGENTS.md` walk root→leaf to nearest `.git` (`analysis_codex.md` §4.1) — **no project memory**; only project instructions | none (sub-agents share parent prompt; no per-agent memory dir) | rollouts under `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` (transcript, not memory) | AGENTS.md root→leaf concatenation — model applies precedence (`analysis_codex.md` §4.3); deeper > shallower in prompt comment, not enforced in code |
| **Claude Code** | `~/.claude/CLAUDE.md` + `~/.claude/projects/<sanitized-cwd>/memory/{MEMORY.md, *.md}` (auto-memdir, `analysis_claude-code.md` §2 / `paths.ts:getAutoMemPath`) + `/etc/claude-code/CLAUDE.md` (managed) | `<cwd>/CLAUDE.md` + `.claude/CLAUDE.md` + `CLAUDE.local.md` (instructions); **memory dir is per-`canonical-git-root`, not per-cwd** | `<cwd>/.claude/agent-memory/<agentType>/` or `<memoryBase>/agent-memory/<agentType>/` per declared scope `'user'/'project'/'local'` (`analysis_claude-code.md` §8) | none (sessions in `<projectDir>/<sessionId>.jsonl`) | CLAUDE.md: enterprise > project > user > local, last-wins; memdir: separate, always-loaded; **agent-memory shadows auto-memdir on `@agent-X` mention** (`attachments.ts:2196`) |
| **Hermes** | `~/.hermes/memories/{MEMORY.md, USER.md}` + `~/.hermes/skills/<name>/SKILL.md` (`analysis_hermes.md` §3) | none (Honcho plugin can do `per-directory` SHA256 hash — provider concern) | none in built-in store; per-agent state via `agents.id` rows in single `~/.hermes/state.db` | none filesystem (sessions in `~/.hermes/state.db`) | Snapshot-frozen at session start; tool replies show live state. No multi-scope precedence — single global file |
| **OpenClaw** | none (no global memory dir; QMD is sidecar) | `<workspaceDir>/{MEMORY.md, memory/YYYY-MM-DD.md, DREAMS.md, memory/dreaming/{light,deep,rem}/YYYY-MM-DD.md, memory/.dreams/*.json}` (`analysis_openclaw.md` §2 / `extensions/memory-core/`) | `~/.openclaw/agents/<agentId>/sessions/<sid>.jsonl` (transcripts; not durable memory) + `~/.openclaw/memory/<agentId>.sqlite` (per-agent index DB) | `<workspaceDir>/memory/YYYY-MM-DD.md` (workspace-scoped daily log) | Dreaming pipeline: light (recall stats) → REM (narrative) → deep (write to `MEMORY.md`); promotion gates score/recallCount/uniqueQueries before crossing scope boundary |
| **OpenFang** | `<workspace>/{SOUL.md, USER.md, MEMORY.md, IDENTITY.md, AGENTS.md, BOOTSTRAP.md, HEARTBEAT.md, context.md}` per-agent workspace (`analysis_openfang.md` §3.2) — **all 32 KB capped** | same files (workspace ≡ agent workspace) | per-agent workspace dir is the agent scope; every file mtime-cached on `WorkspaceContext` | `memory/<YYYY-MM-DD>.md` daily log per agent (`kernel.rs:449-475`, capped 1 MB) | Files are **ordered prompt sections** (`prompt_builder.rs`), not precedence-merged. `BOOTSTRAP.md` only when user_name unknown; `HEARTBEAT.md` only autonomous |
| **paperclip / PARA** | none | `$AGENT_HOME/life/{projects,areas,resources,archives}/<name>/` PARA convention (`analysis_goclaw-paperclip-multica.md` §2) — folders are *taxonomic*, not *scope* | `$AGENT_HOME` is the per-agent root; `life/projects/<name>/` hosts active project notes; `archives/` is supersession sink | daily notes inside `life/areas/<area>/daily/` by convention (qmd-readable) | "no deletion, only supersede" — `life/archives/` is the cemetery; access-count decay |
| **goclaw** | none (DB-resident) | `(agent_id, user_id IS NULL)` rows in `memory_chunks`; per-tenant via `tenant_id` | `(agent_id, user_id)` rows in `memory_chunks`; `WithSharedMemory(ctx)` flag opens cross-user view (`analysis_goclaw-paperclip-multica.md` §1.3) | none filesystem; episodic via `episodic_summaries` table | Per-user 1.2× score boost over global; sharing flag flips filter; **all in DB, no scope dirs** |

**Read across the row:** every file-first system has a global root + workspace/project root; only Claude Code and OpenClaw ship a real *agent* directory and only OpenClaw mounts it under workspace lineage in a useful way. Codex famously has only the global handbook — and explicitly admits this is a known weak spot for per-cwd memory (`analysis_codex.md` §20.2). DB-only systems (Hermes built-in, goclaw, OpenFang) collapse "scope" into composite-key columns, paying with worse human-editability. AGH's hybrid (markdown authoritative + catalog index + event log) gives us the best of both: scopes are filesystem-real, but agent-manageable and auditable.

---

## 3. Recommended AGH v2 layout

Full tree from `$AGH_HOME` and `<workspace>/.agh/`. Single SQLite catalog per scope-root: `agh.db` at `$AGH_HOME` (global+catalog) and `agh.db` at `<workspace>/.agh/` (workspace+agent catalog). Event log lives in the same DBs as a `memory_events` table.

### 3.1 `$AGH_HOME` (global)

```
$AGH_HOME/                                 ← from cfg.HomeDir, default ~/.agh
├── agh.db                                 ← global catalog DB (memory_catalog_entries scope=global, memory_events, observability spine)
├── agh.db-wal, agh.db-shm
├── memory/                                ← global scope root (memory.global_dir)
│   ├── MEMORY.md                          ← projection, rendered from catalog (≤ 200 lines / 25 KB, banner on truncation)
│   ├── user_<slug>.md                     ← curated user-type files (preferences, persona)
│   ├── feedback_<slug>.md                 ← curated feedback rules
│   ├── reference_<slug>.md                ← cross-workspace reference notes (rare in global)
│   ├── daily/
│   │   └── YYYY-MM-DD.md                  ← per-day append-only journal (extractor + agent ad-hoc)
│   └── _system/                           ← never injected into MEMORY.md prompt
│       ├── dreaming/
│       │   └── YYYYMMDD-<llm-slug>.md     ← consolidation synthesis (goclaw _system/dreaming pattern)
│       ├── extractor/
│       │   ├── YYYY-MM-DD.jsonl           ← extractor decisions (incl. NOOPs, rejects)
│       │   └── failures/<run_id>.md       ← retryable consolidation failures (DLQ)
│       └── ad_hoc/
│           └── <iso8601>-<slug>.md        ← Codex-style delta notes from agents/operators
├── agents/<agent>/                        ← AGENT IDENTITY/RUNTIME, not memory; kept for parity with RFC 001 wording
│   ├── identity.toml                      ← (existing or future) — out of scope for memory v2
│   └── memory/                            ← INTENTIONALLY ABSENT (see §8)
├── sessions/<session_id>/                 ← unchanged; transcript only (events.db is here)
├── skills/                                ← unchanged (procedural memory; cross-referenced, not authoritative-here)
└── AGENTS.md                              ← global instruction file (Codex pattern), NOT memory; resolver in instruction layer
```

### 3.2 `<workspace>/.agh/` (workspace + agent)

```
<workspace>/.agh/                          ← from aghconfig.DirName
├── agh.db                                 ← workspace+agent catalog DB
├── agh.db-wal, agh.db-shm
├── memory/                                ← workspace scope root
│   ├── MEMORY.md
│   ├── project_<slug>.md
│   ├── reference_<slug>.md
│   ├── daily/YYYY-MM-DD.md
│   └── _system/{dreaming,extractor,ad_hoc}/
└── agents/<agent>/                        ← AGENT SCOPE (option B — see §8)
    └── memory/
        ├── MEMORY.md                      ← agent-scope projection
        ├── feedback_<slug>.md             ← agent-private feedback
        ├── user_<slug>.md                 ← rare; agent-specific user notes
        ├── daily/YYYY-MM-DD.md
        └── _system/{dreaming,extractor,ad_hoc}/
```

### 3.3 Where things live (cross-reference cheatsheet)

- **AGENTS.md** — `$AGH_HOME/AGENTS.md` (global instructions) + `<workspace>/AGENTS.md` (project instructions, root→leaf walk to repo root, Codex pattern). **Not under `memory/`. Not part of any scope precedence in this design.** Instruction-resolver TechSpec is a separate workstream (P11, `analysis.md` §3 P11).
- **Skills** — unchanged: `<bundled>` ▸ `$AGH_HOME/skills/` ▸ `<workspace>/.agh/skills/` ▸ agent-local. Memory may import skill snippets by reference; skills are the procedural-memory home, never duplicated under `memory/`.
- **Catalog DB** — global `agh.db` at `$AGH_HOME/agh.db` (already wired in `internal/daemon/boot.go:285-288`). Workspace+agent catalog rows go in **per-workspace** `<workspace>/.agh/agh.db` (NEW). Both DBs carry `memory_catalog_entries`, `memory_events`, `memory_recall_signals`, `memory_consolidations` tables. CHECK constraint extends from `('global','workspace')` to `('global','workspace','agent','session')`.
- **Event log** — `memory_events` table (append-only, append-only via insert-only DAO). Lives **in the same DB as the catalog it derives** (so a `BEGIN IMMEDIATE` write of an event + projected catalog row happens atomically).
- **Session-scope memory (derived)** — there is no `session/` directory. Session-scope writes are events with `scope=session`, `session_id=<uuid>`; they project into ephemeral catalog rows that auto-purge on `OnSessionEnd` unless a controller decision promotes them to a persistent scope.

---

## 4. Precedence model

### 4.1 Read-side merge order

**Rule: agent ▸ workspace ▸ global. Deeper scope shadows by `<type>_<slug>`. Never merge silently.**

When the recall layer or `Assembler` builds the prompt context for a turn:

1. Resolve `(active_agent_name, workspace_id)` from session context.
2. Fetch curated index entries from each scope independently:
   - `agent` (if `active_agent_name` is bound and `workspace_id` is bound)
   - `workspace` (if `workspace_id` is bound)
   - `global` (always)
3. Merge by `(<type>, <slug>)` key. **Deeper scope wins**. Shadowed entries are emitted as a `memory_events` row `op=shadowed` with `winner_scope`, `loser_scope` fields (audit trail; equivalent to Claude Code "explicit user override is *not* a bypass" — `analysis_claude-code.md` §13.7). Shadowed entries do **not** appear in the prompt index.
4. Render shadow-aware `MEMORY.md` projection per scope; concatenate scopes into the prompt as labelled sections (`## Agent Memory Index`, `## Workspace Memory Index`, `## Global Memory Index`). Order in the prompt is **least-specific first** (global → workspace → agent) so the deeper scope appears last (mimics Codex hierarchical-AGENTS.md ordering, `analysis_codex.md` §4.3 — "the one located deeper in the directory structure overrides the higher-level file"). LLMs read top-to-bottom and weight recency; this order keeps the most-specific instructions closest to the user message.
5. Recall pipeline (FTS5 + trigram per `analysis_hermes.md` §2.5 / `analysis.md` §5 Eixo 1) queries all three scopes' catalog rows, applies the same shadow rule, and ranks. **Cross-scope candidates are not boosted** (goclaw's 1.2× per-user boost is rejected; see §11.2).

### 4.2 Write-side default + override

**Default scope per memory type** (existing in `internal/memory/types.go:237-248` for global+workspace) extends to:

```go
// Type → default scope
user, feedback     → ScopeGlobal       (unchanged)
project, reference → ScopeWorkspace    (unchanged)
// + new: AgentDef.memory.scope OVERRIDES the type default when present
```

The agent definition (currently lacks the field — `analysis_agh-current.md` §3) gains a `Memory` block:

```toml
[memory]
scope = "workspace"        # one of: agent | workspace | global
                           # absent → fall back to DefaultScopeForType(type)
```

When agent declares `memory.scope = "agent"`, **all writes default to agent scope** (regardless of memory type), unless the call carries an explicit `--scope` (CLI), `scope:` field (HTTP/UDS), or controller override. This mirrors Claude Code's per-agent `memory: 'user' | 'project' | 'local'` (`analysis_claude-code.md` §8) but with stable AGH scope names.

**Explicit override** is mandatory at every surface:
- CLI: `agh memory write --scope <agent|workspace|global>`
- HTTP/UDS: body field `"scope": "agent"`
- Native tool: `agh__memory_propose({scope: "agent", ...})`
- Host API extension contract: `Write(scope, filename, content)` (existing — extends to new scope)

### 4.3 Defense vs Codex's root→leaf concatenation

Codex chose **concatenation in document order** with the model interpreting precedence (`analysis_codex.md` §4.3: "When two AGENTS.md files disagree, the one located deeper... overrides"). For instructions this is fine; the model has full text and can reason. **For memory it is wrong** — it doubles token cost (both the global and shadowed entry get injected) and lets contradictory facts survive in-prompt. AGH's curated memory uses real shadowing in code, dropping the loser before render. The Codex pattern is preserved only for `AGENTS.md` (instructions, separate resolver).

### 4.4 Defense vs Claude Code enterprise>project>user

Claude Code's CLAUDE.md hierarchy is **last-wins concatenation** with five layers (managed/user/project/local/additional, `analysis_claude-code.md` §5). It works because CLAUDE.md is small and human-edited. Memory v2 has thousands of records over time — the sane scaling is shadow-by-id, not last-wins-on-concat. We borrow the **labelling and audit-trail** discipline (`analysis_claude-code.md` §13.7 "explicit user override is *not* a bypass"); we reject the global-merge pattern.

---

## 5. Lifecycle per scope

### 5.1 `global`

| Phase | Behavior |
|---|---|
| Create | Daemon `boot.go` calls `EnsureDirs` on `$AGH_HOME/memory/` (existing — `internal/memory/store.go:101-117`); add `daily/` and `_system/{dreaming,extractor,ad_hoc}/` to the ensure list. Idempotent. |
| Read | Always loaded into prompt at session start (frozen snapshot). Recall pipeline queries global catalog rows on every turn. |
| Write | All four memory types accepted with `--scope global` (default scope: `user`, `feedback`). Mediated by controller (`analysis.md` §5 Slice 1). |
| Discovery | Static path from `cfg.Memory.GlobalDir` (already in config). No discovery walk. |
| Hot-reload | `agh memory reload` invalidates **next-turn** snapshot, never current turn (Hermes pattern, `analysis_hermes.md` §3.2 frozen-snapshot). File-watcher OFF by default; opt-in `cfg.Memory.WatchGlobal = true` for daemon-resident operators. |
| Workspace move | N/A (workspace-independent). |
| Agent removal | N/A (cross-cutting). |
| `agh memory reset --scope global` | Truncates all `<type>_<slug>.md`, clears catalog rows, emits `op=reset` events, removes `MEMORY.md` projection. `daily/` and `_system/` preserved unless `--include-system` passed. |

### 5.2 `workspace`

| Phase | Behavior |
|---|---|
| Create | First daemon boot inside a workspace (`workspace.Resolver` first-touch) creates `<workspace>/.agh/memory/` + `daily/` + `_system/`. Same stable-workspace_id resolution as today (`internal/memory/store.go:1186-1195` `canonicalWorkspaceRoot`); v2 adds a `workspace_id` (UUID) generated on first touch and stored in `<workspace>/.agh/workspace.toml`. Path moves do not orphan rows because catalog rows key on `workspace_id`, not path. |
| Read | Loaded only when active workspace matches. Frozen at session boot like global. |
| Write | Default for `project`, `reference` types. Workspace-scope writes touch only the workspace DB. |
| Discovery | `workspace.Resolver` (existing) walks ancestors looking for `.agh/` marker; deepest `.agh/` wins (Codex root-marker pattern, `analysis_codex.md` §4.1). Tie with `.git/`-only ancestors: AGH creates `.agh/` lazily on first memory write. |
| Hot-reload | Same as global; per-workspace flag. |
| Workspace move (path change) | `workspace_id` is stable → catalog rows survive. New `realpath` triggers a reconciliation `op=workspace_relocated` event. **Critical fix vs current code**: today `workspace_root` is a path (`internal/memory/catalog.go:1260-1270` `deriveWorkspaceRoot`), so a `mv` orphans rows. v2 stores `workspace_id` in catalog and resolves path → ID via `.agh/workspace.toml`. |
| Agent removal in workspace | Removes only that agent's subdir; workspace memory untouched. |
| `agh memory reset --scope workspace` | Same shape as global, scoped to one workspace. |

### 5.3 `agent`

| Phase | Behavior |
|---|---|
| Create | Lazy. First write to `agent` scope creates `<workspace>/.agh/agents/<agent>/memory/` (mkdir + ensure subdirs). No bootstrap on agent registration — empty state is OK. |
| Read | Loaded only when `active_agent_name` matches AND workspace is bound. Frozen snapshot at session boot. Sub-agents: read parent agent's snapshot (defense in depth: sub-agent controller is `Mode=ReadOnly`, P12 in `analysis.md` §3). |
| Write | Default scope only when `AgentDef.memory.scope = "agent"`. Otherwise must be explicit. Type-default routing (`DefaultScopeForType`) is overridden by agent's declared scope (§4.2). |
| Discovery | `(workspace_id, agent_name)` lookup. No filesystem walk; agent name comes from session context. |
| Hot-reload | Same as workspace. |
| Workspace move | `workspace_id`-keyed → survives. |
| Agent removal | Hard-delete `<workspace>/.agh/agents/<agent>/memory/` IF operator runs `agh memory reset --scope agent --agent <name>` OR `agh agents remove <name> --purge-memory`. Otherwise the directory survives — agent definitions are mutable; memory is durable. Catalog cleanup via `op=agent_purged` event. |
| `agh memory reset --scope agent --agent <name>` | Truncates only that agent's curated tree + catalog rows. `_system/` preserved unless `--include-system`. |
| Cross-workspace agent reuse | Each workspace has its own agent directory. **No cross-workspace agent memory** in v2 — see §8 for the rationale (workspace-bound is the right primitive). If operator wants cross-workspace agent memory, that's a `global`-scope record under `agent_name`-namespaced filenames; not a separate scope. |

### 5.4 `agh memory reset` semantics

`agh memory reset [--scope <s>] [--agent <a>] [--workspace <ws>] [--include-system] [--include-daily] [--dry-run]`. Always emits `memory_events` audit rows. Default preserves `_system/` and `daily/` (forensic). `--dry-run` prints the affected file count + catalog row count without mutating.

---

## 6. `_system/` namespace rule

**`_system/` is never injected into MEMORY.md prompt.** This is the v2 invariant.

- Files under `<scope>/memory/_system/` are excluded from `Scan` for prompt assembly (extends `shouldSkipFile` — `internal/memory/store.go:1173-1175` — to skip `_system/` directory entirely).
- Catalog rows for `_system/` artifacts get `injection = false` flag; recall pipeline filters them out unless explicit `--include-system` is passed.
- Subdir taxonomy:
  - **`_system/dreaming/YYYYMMDD-<llm-slug>.md`** — consolidation synthesis output, owned by the dedicated dreaming agent (`analysis.md` §5 Slice 1 Eixo 2). Browsable for forensics; `agh memory dream show <date>` surfaces it.
  - **`_system/extractor/YYYY-MM-DD.jsonl`** — per-day rolling log of extractor decisions (ADD/UPDATE/NOOP/REJECT); one JSONL line per decision with full provenance. Mirrors `memory_events` but on disk for human grepping.
  - **`_system/extractor/failures/<run_id>.md`** — DLQ for failed consolidations (goclaw P10 lesson — `analysis_goclaw-paperclip-multica.md` §1.4 dreamingWorker DLQ gap). Manual retry: `agh memory dream retry <run_id>`.
  - **`_system/ad_hoc/<iso8601>-<slug>.md`** — direct port of Codex's `extensions/ad_hoc/notes/` mutation channel (`analysis_codex.md` §8.2). Agents and operators may write delta notes here mid-session; the next dreaming run reconciles them into curated entries. **This is the only place agents may write outside of controller-mediated curated entries.** All other writes go through the controller (P8 in `analysis.md` §3).

**Cite:** Codex `~/.codex/memories/extensions/ad_hoc/notes/<ts>-<slug>.md` is the canonical pattern (`analysis_codex.md` §1, §8.2). AGH adopts the path verbatim, ports the lifecycle: agent writes a delta note → next dreaming run reads `_system/ad_hoc/`, reconciles into proper curated entries, deletes processed notes. Direct edits of `MEMORY.md` or `<type>_<slug>.md` by an in-flight agent are **forbidden by the controller**.

---

## 7. Daily-log strategy

**Decision: per-scope `daily/YYYY-MM-DD.md`. Not a single shared daily log.**

| Argument | Source | Verdict |
|---|---|---|
| OpenClaw ships `<workspace>/memory/YYYY-MM-DD.md` (workspace daily log) and the `/new` `/reset` hook writes `<workspace>/memory/YYYY-MM-DD-<slug>.md` snapshots (`analysis_openclaw.md` §1.7) | Confirms workspace-scope daily log works in production | Adopt |
| OpenFang ships `memory/<YYYY-MM-DD>.md` per agent (`analysis_openfang.md` §3.2) — capped 1 MB | Confirms per-agent daily log works | Adopt for agent scope |
| Codex has no daily log (rollouts are JSONL, transcripts not memory) | Codex bets on the consolidation pipeline doing daily aggregation | Reject for AGH — daily log is cheap insurance + agent-readable |
| Claude Code's KAIROS mode appends to `~/.claude/projects/<sanitized-cwd>/memory/logs/YYYY/MM/YYYY-MM-DD.md` (`analysis_claude-code.md` §2 `paths.ts:getAutoMemDailyLogPath`) | Daily log is the *source* for nightly `/dream` distillation in Claude Code | Confirm pattern: daily log feeds dreaming worker |

**Per-scope rationale:**
- A global-scope daily log captures cross-workspace ad-hoc agent observations.
- A workspace-scope daily log captures workspace-tagged work.
- An agent-scope daily log captures agent-specific lessons that are too narrow for the workspace.

A single shared daily log would cross scope boundaries and re-introduce the same drift we're avoiding in the curated tree. Three small files per day cost nothing; the dreaming worker walks the right one per scope.

**Hard caps:** daily file ≤ 1 MB (OpenFang `kernel.rs:449-475`). On overflow, rotate to `daily/YYYY-MM-DD.<seq>.md`.

---

## 8. `agent` scope on disk — final decision

Three candidates; pick one.

### Candidate A — `$AGH_HOME/agents/<agent>/memory/` (RFC 001 wording)

- **Pros:** matches RFC 001 literal text (`internal/CLAUDE.md` "Memory & Skills Runtime"); single agent, single home, no per-workspace fan-out; consistent with skills layout (`$AGH_HOME/skills/`); cross-workspace agent memory comes for free.
- **Cons:** an agent learning lessons in workspace A leaks them into workspace B by default — wrong default for multi-tenant or multi-project users; Claude Code learned this the hard way and made `'project'` scope opt-in (`analysis_claude-code.md` §8 — three explicit scopes); leaks workspace-private context across workspaces (security smell — `internal/CLAUDE.md` Security Invariants emphasis on scope isolation).

### Candidate B — `<workspace>/.agh/agents/<agent>/memory/` (workspace-bound)

- **Pros:** agent memory is naturally co-versioned with the workspace it was earned in; matches OpenClaw's production-tested `~/.openclaw/agents/<agentId>/sessions/` (workspace-bound pattern, `analysis_openclaw.md` §1.6); zero cross-workspace leakage by default; `workspace_id`-keyed catalog rows survive workspace renames; clean delete semantics on workspace removal; integrates with stable `workspace_id` (§5.2); preserves agent identity at `$AGH_HOME/agents/<agent>/` for runtime/identity (so RFC 001's spirit survives — agents have a global home for *config*, not for *memory*).
- **Cons:** an agent that genuinely should have cross-workspace memory pays a small migration cost (operator promotes the entry to global scope — `agh memory promote --scope global`); requires v2 to write `workspace_id` to `.agh/workspace.toml`.

### Candidate C — Both (global agent + workspace agent overrides)

- **Pros:** maximally flexible.
- **Cons:** four-way precedence (`agent@workspace ▸ agent@global ▸ workspace ▸ global`) is genuinely confusing; doubles the surface area in CLI, HTTP, controller, observability events; only Claude Code does this and it's the most-criticised part of their memory UX (`analysis_claude-code.md` §13 — "no first-class delete/forget UX, no in-UI list/search/edit/delete, power-users only").

### Verdict: **B**

Pick **Candidate B**: agent scope at `<workspace>/.agh/agents/<agent>/memory/`. Workspace-bound by default. Cross-workspace agent memory is a deliberate `agh memory promote` action, not a default.

**Defense vs RFC 001:** RFC 001's `.agents/<name>/memory/` was written before the workspace lifecycle was settled. The intent was "agent has its own memory namespace" — fulfilled by candidate B. The literal path was a naming sketch, not an architectural commitment. RFC 001's mandate (`internal/CLAUDE.md` "Five-layer skill/memory/agent precedence: Bundled → Marketplace → User → Additional → Workspace, with agent-local overriding all") survives intact: agent overrides workspace overrides global, regardless of which directory holds the bytes.

**Cross-reference to existing AGH:** `$AGH_HOME/agents/<agent>/` stays reserved for agent-identity / per-agent runtime state (e.g. learned model preferences, default permission profile). Memory does not live there. This separation is what RFC 001 was reaching for — config and memory have different lifecycles.

---

## 9. Conflicts / shadowing rules

### 9.1 Same `<type>_<slug>.md` exists in multiple scopes

Example: `feedback_pedro_ci.md` exists at both global and workspace level.

**Rule: deeper scope wins; shadowed entry is logged + audit-trailed but NOT deleted.**

1. Recall pipeline returns the workspace entry only.
2. `Assembler` injects only the workspace version into the prompt.
3. A `memory_events` row of shape `op=shadowed scope_winner=workspace scope_loser=global filename=feedback_pedro_ci.md workspace_id=... agent_name=...` is written.
4. `agh memory list --show-shadowed` reveals shadowed entries.
5. `agh memory show feedback_pedro_ci --scope global` still works; `--scope` is mandatory when ambiguous (default to deepest).

### 9.2 Why not merge?

- **Doubles token budget.** Both entries fight for the same prompt slot.
- **Risks contradiction.** `feedback_pedro_ci.md` at global says "always run with race"; workspace override says "skip race for slow integration tests" — merged, the model gets a contradiction.
- **Codex hierarchical-AGENTS.md does merge** (`analysis_codex.md` §4.3) — it works for *instructions* because the model can reason about scope precedence in-prompt; for *memory facts* it fails because facts don't carry scope cues.

### 9.3 Why not auto-promote?

We considered "if global and workspace agree, drop the workspace copy as redundant." Rejected — it requires content-similarity reasoning and confuses operators ("I just wrote this here, where did it go?"). Promotion is an explicit verb: `agh memory promote --scope global feedback_pedro_ci`.

### 9.4 Why not error?

Erroring on conflict means an agent writing locally fails because of an unrelated global entry it has never seen. Bad UX. Logging + shadowing is the right tradeoff.

### 9.5 Defense vs Claude Code's "explicit user override is *not* a bypass" semantics

Claude Code says deeper scope wins **and** that win goes through the regular permission/policy chain (`analysis_claude-code.md` §13.7). AGH inherits the spirit: shadowing is auditable; an agent cannot silently override a global feedback rule without the operator seeing the audit row. Operators can configure a `memory.shadow_policy = "warn" | "allow" | "deny"` on `<type>` per scope (default: `allow` everywhere except `feedback`-type, where the default is `warn` — operators usually want to see when a feedback rule is being locally overridden).

---

## 10. Migration path from AGH today

Concrete delta from current 2-scope (global+workspace) to v2 3-scope (global+workspace+agent), plus stable `workspace_id`. Hard cuts where the greenfield rule applies (`internal/CLAUDE.md`).

### 10.1 Filesystem migration

| Today | v2 | Action |
|---|---|---|
| `$AGH_HOME/memory/` | `$AGH_HOME/memory/` (unchanged) | + add `daily/`, `_system/{dreaming,extractor,ad_hoc}/` on first v2 boot |
| `$AGH_HOME/memory/.consolidate-lock` | `$AGH_HOME/memory/_system/dreaming/.lock` | Move; emits `op=migration` event |
| `<workspace>/.agh/memory/` | `<workspace>/.agh/memory/` (unchanged) | + add `daily/`, `_system/`; create `.agh/workspace.toml` with new UUID `workspace_id` |
| `(no agent dir)` | `<workspace>/.agh/agents/<agent>/memory/` | Lazy-create on first agent-scope write |
| `$AGH_HOME/agh.db` (global catalog) | `$AGH_HOME/agh.db` (unchanged) | Schema migration: extend `scope` CHECK to `('global','workspace','agent','session')`; add `workspace_id TEXT` column; add `agent_name TEXT` column on catalog + events; add new tables `memory_events`, `memory_recall_signals`, `memory_consolidations`. Run via numbered migration in catalog registry (`agh-schema-migration` skill mandatory). |
| (no per-workspace DB) | `<workspace>/.agh/agh.db` | NEW — created on first workspace-scope write. Holds workspace + agent rows. Schema same as global. |

### 10.2 Code rename / move list (atomic single change, per greenfield-delete rule)

| File:Symbol today | v2 | Action |
|---|---|---|
| `internal/memory/types.go:28-33` `ScopeGlobal/ScopeWorkspace` | + `ScopeAgent`, `ScopeSession` | Add constants; update `Scope.Validate` (`types.go:256-265`); update `DefaultScopeForType` to honor `AgentDef.Memory.Scope` first |
| `internal/memory/store.go:38-46` `Store{globalDir, workspaceDir}` | `Store{globalDir, workspaceDir, agentDir}` + `(workspaceID, agentName)` accessors | `ForWorkspace(workspaceRoot)` → `ForWorkspace(workspaceID, root)`; new `ForAgent(workspaceID, agentName, root)` cloning method; `dirForScope` extends to `case ScopeAgent` |
| `internal/memory/store.go:522-546` `dirForScope` | + agent case | Returns `<workspace_root>/.agh/agents/<agent>/memory/` |
| `internal/memory/store.go:1216-1223` `workspaceMemoryDir` | + `agentMemoryDir(workspaceRoot, agentName)` | New helper |
| `internal/memory/catalog.go:34-87` schema | numbered migration v3 `add_agent_scope` | Extends scope CHECK; adds `workspace_id`, `agent_name` columns; backfills existing rows with `workspace_id` from `.agh/workspace.toml` (created on migration); rebuilds FTS5 shadow tables |
| `internal/memory/catalog.go:1260-1270` `deriveWorkspaceRoot` | replaced by `resolveWorkspaceID(workspaceRoot)` | Reads `.agh/workspace.toml`; creates if missing on first write |
| `internal/config/agent.go` `AgentDef` | + `Memory MemoryConfig` block with `Scope string` | Wire through `internal/config/agent_resource.go` |
| `internal/api/contract` memory types | + `Scope = "agent"` enum value | Triggers codegen co-ship (`agh-contract-codegen-coship`) |
| `internal/api/core` memory handlers | + agent-scope filters in list/read/write/search/reindex/history | One pass per handler |
| `internal/cli/memory*.go` | + `--agent <name>` flag wherever scope is | Mandatory when `--scope agent` |
| `cmd/agh/native_tools` | + `agh__memory_propose` reads agent scope | Already passes scope through; just enable enum |
| `internal/extension/contract/host_api.go` | + agent scope in `Write/Read/List` | Extension contract change; existing extensions panic on unknown scope — caught by tests |
| `internal/memory/dream.go` | workspace iteration includes agent dirs | `prepareWorkspace` walks `<workspace>/.agh/agents/*/memory/` for agent-scope dreaming candidates |
| `internal/memory/consolidation/runtime.go` | per-(workspace,agent) consolidation cursors | Extends `resolveWorkspaces` to emit `(workspace_id, agent_name)` pairs |
| `docs/_memory/glossary.md:138-144` | already lists `agent` scope | No change — code now matches docs |

### 10.3 What gets deleted (greenfield-delete)

- The entire `deriveWorkspaceRoot(path)` legacy resolution (replaced by `workspace_id` keys). No fallback.
- Any test that locks `workspace_root TEXT NOT NULL DEFAULT ''` shape in the catalog. Tests get rewritten, not bridged.
- Compatibility code for "MEMORY.md is source-of-truth" pattern — v2 makes it a projection unconditionally.

### 10.4 What stays unchanged

- Frontmatter format (`internal/memory/types.go:50-58`).
- `MEMORY.md` 200-line / 25 KB cap (`internal/memory/store.go:24-26`).
- `Time → Sessions → Lock` consolidation cascade (`analysis.md` §3 P7 — proven safe; replace only after v2 worker proves itself).
- Atomic write (`fileutil.AtomicWriteFile` in `store.go:171`).
- The four memory types (`user|feedback|project|reference`) — closed taxonomy.
- Bundled `globalDir` config key.

### 10.5 Migration ordering (one PR; the greenfield rule forbids bridges)

1. Schema migration (catalog v3 + new tables).
2. Code (Scope enum extension, Store/catalog API, agentDir handling).
3. Contract (`internal/api/contract` + codegen co-ship).
4. CLI/HTTP/UDS/native-tools.
5. Tests (every package). All `make verify` must pass; one commit per remediation batch (`internal/CLAUDE.md` Commit style).
6. Docs (`docs/_memory/glossary.md`, RFC 001 update, CLAUDE.md skills RFC).

---

## 11. Open decisions for the TechSpec to confirm

Six sharp questions the spec author must answer in writing.

### 11.1 Does session-scope live in the file tree at all?

This analysis recommends **NO**: session scope is event-only, with ephemeral catalog rows that auto-purge on `OnSessionEnd`. Alternative is a `<workspace>/.agh/sessions/<session_id>/memory/` shadow tree (Codex pattern, `analysis_codex.md` §3.1 rollouts in `~/.codex/sessions/YYYY/MM/DD/`). **Spec must commit one way or the other.** If "yes filesystem", define purge cadence and disk-budget caps. If "no filesystem", define event TTL.

### 11.2 Per-user / per-actor 1.2× boost across scopes?

goclaw applies a 1.2× score boost to per-user chunks (`analysis_goclaw-paperclip-multica.md` §1.5), and itself flags this as questionable in shared mode. v2 must answer: does **agent-scope** memory get a boost over workspace-scope at recall time? This analysis recommends **NO** — shadowing already gives deeper scope priority by exact-match; soft-boosting at recall time on top of shadowing is double-counting. Spec must confirm.

### 11.3 Cross-workspace agent memory (the candidate-A escape valve)

If users complain about workspace-bound agent scope (§8 Verdict B), what's the operator workflow? Recommendation: explicit `agh memory promote --scope global` for individual entries, no implicit cross-workspace agent scope. Spec must confirm + document the migration path for users with one agent across many workspaces.

### 11.4 Is `workspace_id` allocation tied to `.agh/workspace.toml` or to `.git/`?

This analysis recommends `.agh/workspace.toml` (works for non-git workspaces; matches Claude Code's `findCanonicalGitRoot` worktree-dedup spirit but doesn't require git). Alternative: derive `workspace_id` deterministically from canonical git root. Spec must commit. Implication: non-git workspaces lose stability if we go git-only.

### 11.5 Daily-log retention + dreaming window

Daily logs grow unbounded across years. Spec must specify: (a) how many days back the dreaming worker reads; (b) how long daily files survive on disk; (c) cold-storage migration to `_system/archive/YYYY-MM/`. Recommendation: dreaming reads last 14 days; daily files survive 90 days; cold-storage afterwards. Tunable in `cfg.Memory.Daily.RetentionDays`.

### 11.6 Native tool / agent write surface for `_system/ad_hoc/`

Codex's mutation channel (`extensions/ad_hoc/notes/`) is the **only** in-flight write surface for agents in their model (`analysis_codex.md` §8.2 — "do not try to edit the memory files yourself"). v2 inherits this. But: do AGH agents call a `agh__memory_note(scope, content, slug?)` native tool, or do they `Write` to a path? Recommendation: dedicated `agh__memory_note` native tool with controller-mediated write to `_system/ad_hoc/`. Reason: bypasses path-validation pitfalls (path traversal, wrong scope) and gives the controller a clean dispatch point. Spec must commit + define tool schema.

---

## Appendix — Citation index

- AGH today: `internal/memory/store.go`, `internal/memory/types.go`, `internal/memory/catalog.go`, `internal/daemon/boot.go:285-288`, `internal/CLAUDE.md` "Memory & Skills Runtime", `docs/_memory/glossary.md:127-144`.
- Forensic source paths: `analysis_agh-current.md` §3 (taxonomy gaps), §4 (filesystem layout), §16 (drift summary).
- Codex AGENTS.md walk: `analysis_codex.md` §4 (`codex-rs/core/src/agents_md.rs:213-303`).
- Codex memory tree: `analysis_codex.md` §3.1, §1, §8.2 (`~/.codex/memories/`, `extensions/ad_hoc/notes/<ts>-<slug>.md`).
- Claude Code memdir: `analysis_claude-code.md` §2 (`memdir/paths.ts:getAutoMemPath`), §5 (precedence), §8 (per-agent memory).
- Hermes single-file approach: `analysis_hermes.md` §3 (frozen-snapshot), §2.5 (FTS5 + trigram).
- OpenClaw workspace + per-agent dirs: `analysis_openclaw.md` §1.6, §2 (`<workspaceDir>/memory/YYYY-MM-DD.md`, `~/.openclaw/agents/<agentId>/sessions/`).
- OpenFang workspace identity files: `analysis_openfang.md` §3.2 (`SOUL.md/USER.md/MEMORY.md/IDENTITY.md/AGENTS.md/BOOTSTRAP.md/HEARTBEAT.md`, prompt-builder ordering).
- Paperclip PARA: `analysis_goclaw-paperclip-multica.md` §2 (`$AGENT_HOME/life/{projects,areas,resources,archives}/<name>/`).
- goclaw per-user boost + recall feedback: `analysis_goclaw-paperclip-multica.md` §1.3, §1.5, §1.6.
- Architectural pillars: `analysis.md` §3 P1 / P11 / P12, §5 Slice 1.
