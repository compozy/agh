# AGH Feature Inventory (from specs)

Source corpus mined:
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/<slug>/_techspec.md` (active + `_archived/`)
- `/Users/pedronauck/Dev/compozy/agh/CLAUDE.md`, `/Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/_memory/standing_directives.md`, `/Users/pedronauck/Dev/compozy/agh/docs/_memory/glossary.md`
- `git log` (commit titles confirm ship status)
- `internal/<pkg>/` directory existence + key files (registry, dispatch, policy, capability, peer)

Status taxonomy:
- shipped — TechSpec archived under `_archived/`, code present in `internal/`, and a feature commit exists. Or: active task slug whose code has merged (e.g. `unified-capabilities` and `tools-registry` both have `feat:` merge commits in `git log`).
- in-progress — active task slug, partial code, follow-up TechSpec describes remaining work (e.g. `tools-refac`).
- planned — task slug, no code yet, or RFC-only (e.g. AGH Network v1 verified-format identity).

---

## Headline features (shipped + flagship)

### 1. Autonomy Kernel (coordinator + claim/lease + safe spawn)
- Status: **shipped** — active dir `.compozy/tasks/autonomous/` (still has reviews-002), `internal/coordinator/`, `internal/scheduler/`, `internal/task/` (with `ClaimNextRun`), `internal/situation/` all exist; commit `205a6dda feat: autonomous system (#75)`; package layout confirms (`internal/CLAUDE.md` lines 77, 98, 109, 117).
- Description: AGH ships a daemon-owned autonomy kernel that lets multiple ACP-based agent sessions coordinate on a single task queue. A coordinator agent decomposes work into `task_runs`; idle agent sessions atomically claim runs through `ClaimNextRun` and hold a lease they must heartbeat. Spawning child sessions is gated by lineage, TTL, hard caps and permission narrowing. Manual operator control stays first-class — user-created tasks and coordinator-created tasks share the same queue, claim-token, and lease contracts.
- Differentiator: most agent harnesses fork-and-pray when delegating. AGH has a real durable scheduler with token-fenced ownership, lease-recovery on restart, and one shared task queue between humans and agents. Permission narrowing and child-cannot-widen-parent are enforced in code, not in the prompt.
- Direct quote (`autonomous/_techspec.md:7`): "The implementation strategy is to connect existing AGH substrate instead of replacing it... New autonomy behavior is added through four coordinated layers: Situation Surface, Agent Kernel CLI, Autonomy Kernel, and Memory/Self-Correction."
- Direct quote (`autonomous/_techspec.md:29`): "`ClaimNextRun(criteria)` atomically selects and claims one queued run inside SQLite, returns a `claim_token`, and requires that token for heartbeat, completion, failure, and release operations."
- User-visible outcome: `agh task next --wait` returns a claimed run with a coordination channel; the agent calls `agh task heartbeat|complete|fail|release` with its claim token; if the agent dies the lease expires and another idle agent picks the run up. Operator UI shows publish/enqueue/coordinator-trigger as distinct stages.

### 2. AGH Network — agent-to-agent protocol (v0)
- Status: **shipped** — `_archived/20260412-040024-network/`, `_archived/20260412-040024-channels/`, `internal/network/` packed with `manager.go`, `peer.go`, `lifecycle.go`, `delivery.go`, `envelope.go`, `audit.go`, `capability_catalog.go`. Commits `6e9d088e feat: add network implementation (#15)`, `20bad010 feat: add channels (#14)`, `7a225bd3 refactor: rename spaces to channels (#17)`, `25db48fa feat: redesign network workspace (#59)`, `68cc7df4 refactor: enable AGH network by default for new installs (#57)`.
- Description: AGH Network is the embedded agent-to-agent communication layer. Each active session is a peer (`{agent_name}.{session_id}`). The daemon embeds a NATS server as its wire transport, manages peer lifecycle off session join/leave, routes envelopes by space/peer route-token, and auto-prompts the receiving agent with queued messages between turns. Out: agents call `agh network send`. In: typed envelope kinds (`greet`, `whois`, `direct`, `say`, `receipt`, `trace`, `capability`).
- Differentiator: this is the load-bearing claim. Other harnesses don't have a multi-agent wire protocol at all — they are one-process-one-agent runners. AGH spec is explicitly designed to remain implementable outside AGH: "AGH competes on runtime, SDK, observability, DX, and integration depth — NOT the wire protocol. The protocol must remain implementable outside AGH" (`docs/_memory/glossary.md:216`).
- Direct quote (`_archived/.../network/_techspec.md:11-17`): "Embedded NATS server in the daemon binary (single-binary, local-first). Each active session is a unique peer with identity `{agent_name}.{session_id}`. Network Manager as boot-phase observer of session lifecycle (not a property of sessions)."
- User-visible outcome: in one terminal `agh start --agent coder`, in another `agh start --agent reviewer`, both see each other in `agh network peers`, `agh network send --to reviewer.<id>` reaches the other agent's prompt. Network is on by default for new installs.

### 3. Capabilities — unified network capability artifact
- Status: **shipped** — active dir `.compozy/tasks/unified-capabilities/`, `internal/network/capability_catalog.go`, `internal/registry/`. Commits `0bdd840b feat: agent capabilities (#49)`, `731414f4 feat: unify capability (#53)`, `a768e13a feat: replace recipe wire kind with capability`, `223ed391 feat: canonicalize unified capabilities`, `caeee323 feat: align discovery contracts with unified capabilities`, `323ee740 feat: render unified capabilities in network peer UI`.
- Description: A capability is the single canonical name for "what an agent can do for another agent" — authored locally in `capabilities.toml` or `capabilities/<id>.toml`, projected into network discovery (`greet` brief, `whois` rich), and transferable across the wire as `kind: "capability"` envelopes with a runtime-computed canonical digest. Replaces the prior `capabilities + recipes` split.
- Differentiator: capabilities are interpretive, structured (id/summary/outcome/context_needed/artifacts_expected/execution_outline/constraints/examples/requirements), digest-stable, and travel between peers. Other agent systems either ship opaque tool blobs or have no transferable artifact at all. Vocabulary discipline is intentional: "A capability is **interpretive**, not deterministic — it tells an agent what is available, not how to execute a deterministic program" (`docs/_memory/glossary.md:13`).
- Direct quote (`unified-capabilities/_techspec.md:5-7`): "A capability becomes the only authored delegation artifact, the only rich discovery artifact, and the only transferable procedural artifact on the wire. The current `recipe` protocol kind and runtime model are removed."
- User-visible outcome: drop a `capabilities.toml` next to your agent. Other peers see your capabilities listed in `greet`; explicit `whois` returns the rich form; `agh network send --kind capability ...` ships the actual artifact.

### 4. Tool Registry (canonical agent tool surface)
- Status: **shipped** (foundation) + **in-progress** (canonical surface refac). Foundation: commit `f0c53baf feat: tools registry (#85)`; `internal/tools/` with `dispatch.go`, `policy.go`, `policy_resolver.go`, `mcp.go`, `native.go`, `availability.go`, `approval_token.go`. Active follow-up: `.compozy/tasks/tools-refac/`.
- Description: A daemon-owned registry that unifies tool identity (canonical `ToolID`), discovery, availability, policy, execution, hooks, telemetry, extension descriptors, MCP adapters, and session-visible exposure through one central dispatch. Three executable backend kinds: `native_go` (compiled-in AGH built-ins), `extension_host` (third-party Go/TS extension subprocesses), `mcp` (daemon-owned client to external MCP servers). AGH-native tools become session-callable through an AGH-hosted MCP proxy that uses `mark3labs/mcp-go`.
- Differentiator: ACP doesn't define a tool registry, MCP defines protocol but not policy. AGH owns the policy engine (ACP approval mode + session lineage + agent policy + source/risk + registry allow/deny + toolsets + availability + hooks → structured effective decisions). Same tool surface across CLI/HTTP/UDS/hosted MCP.
- Direct quote (`tools-registry/_techspec.md:5`): "AGH's Tool Registry as a daemon-owned runtime service, not as a static list of built-in commands. The registry will unify tool identity, discovery, availability, policy, execution, hooks, telemetry, extension descriptors, MCP adapters, and session-visible exposure through one central dispatch pipeline."
- Direct quote (`tools-refac/_techspec.md:9-12`): "The current branch already ships the registry core, policy engine, hosted MCP transport, approval bridge, CLI/HTTP/UDS tool surfaces, and an intentionally narrow built-in MVP subset. The remaining problem is surface ambiguity..."
- User-visible outcome: an agent sees the same tool by the same canonical ID whether it arrived via native built-in, extension, or external MCP server. `agh tool list/search/info/invoke` work everywhere; policy decisions surface as structured reasons, not opaque denies.

### 5. Hooks platform — typed lifecycle dispatch
- Status: **shipped** — `_archived/20260410-021708-hooks/`, `internal/hooks/`. ~24 typed events organized into sync-pipeline / async-only families.
- Description: First-class hooks platform with a typed lifecycle taxonomy (session.*, input.*, prompt.*, agent.*, turn.*, message.*, tool.*, permission.*, context.*, event.* + autonomy hooks `coordinator.*`, `spawn.*`, `task.run.*`). Sync hooks compose as a sequential pipeline (each receives previous patch); async hooks run on a worker pool. Hooks come from native Go, settings/config, agent-definition, skill, and extension manifests in deterministic precedence order.
- Differentiator: not a generic event bus. Typed payloads/patches per event, dispatch-depth guard, and a hard rule that `permission.*` hooks can deny but cannot escalate deny→allow. Skills and extensions can declare hooks in metadata frontmatter without changing core packages.
- Direct quote (`_archived/.../hooks/_techspec.md:7`): "The implementation strategy is to create a dedicated `internal/hooks` package that exposes typed dispatch functions (not a generic event bus)... Because AGH is greenfield alpha, the system should define the full contract now rather than evolve through incompatible one-off seams."
- Direct quote (`internal/CLAUDE.md:25`): "Hooks are typed dispatch, not an event bus. Dispatch at the call site that owns the state transition. Never tail event/log tables to fire hooks. Hooks may deny/narrow/annotate but cannot bypass safety primitives."
- User-visible outcome: drop a hook in a skill's frontmatter or an extension manifest, see it run on the relevant lifecycle event with a typed payload, observe its patch in the audit telemetry.

### 6. Memory — dual-scope persistent + Dream consolidation
- Status: **shipped** — `_archived/20260405-031926-agh-memory-extensibility/`, `_archived/mem-improvs/`, `internal/memory/` and `internal/memory/consolidation/` with `runtime.go`, `runtime_test.go`, `perf_bench_test.go`. Commit `8ef47401 refactor: memory improvements (#35)`.
- Description: A file-based persistent memory store with global + workspace + agent scopes and MEMORY.md indexes injected into prompts at session start. Dream consolidation runs on a Time → Sessions → Lock cascade (default 24h, 5 touched sessions, file-lock) — when all three gates pass, AGH spawns an ephemeral ACP session with approve-all permissions that reads recent session events, synthesizes durable facts, and writes back to memory files. Four types: `user | feedback | project | reference`.
- Differentiator: most harnesses dump conversation history into a vector DB. AGH has typed memory scopes, a written taxonomy, frontmatter-tagged files agents can edit themselves, and a literal "dream" gate-based consolidation that doesn't fire until the cost of a real LLM call is justified. Memory consolidation is a runtime gate, not a heuristic.
- Direct quote (`_archived/.../agh-memory-extensibility/_techspec.md:5`): "(1) a file-based persistent memory store (memdir) with dual global/workspace directories and MEMORY.md indexes; (2) a dream consolidation service that spawns ephemeral ACP agent sessions to synthesize session transcripts into durable memory files; and (3) team memory via workspace-scoped files with agent metadata for cross-agent knowledge sharing."
- Direct quote (`internal/CLAUDE.md:127-128`): "Memory taxonomy: `user | feedback | project | reference` types; scopes `agent | workspace | global`. Memory consolidation gates: Time → Sessions → Lock cascade ordered by computational cost. Default gates: 24h, 5 touched sessions, file-lock. Never replace gates with naive heuristics."
- User-visible outcome: `agh memory list/read/write/delete/consolidate`, `agh memory health`, `agh memory history`. Memory written by one agent is visible to other workspace agents; dream session runs nightly and updates the canonical files.

### 7. Extensions — three-dimensional packages with bidirectional Host API
- Status: **shipped** — `_archived/20260411-014454-ext-architecture/`, `internal/extension/` with bridge_delivery_*.go, bundle.go, capability.go, describe.go, contract subpkg. Commits `8cfc9a42 refactor: add extensions gaps (#21)`, `ed1688cf feat: add ext refac and sandbox (#25)`, `132648c6 feat: add extension tool manifest reconciliation`, `f88f47b9 feat: add extension tool runtime sdk`, `58ad2dba feat: add public Go extension SDK`.
- Description: Extensions ship as 3-D packages — they bundle **resources** (agents, skills, hooks, MCP configs), provide **capabilities** (agent drivers, memory backends, observe exporters, channel adapters), and perform **actions** through a bidirectional Host API (sessions/*, memory/*, skills/*, observe/*, tasks/*). Two execution tiers: L1 native Go for first-party compiled-in code, L3 JSON-RPC subprocess for third-party Go OR TypeScript extensions. Capability-scoped security at the Host API boundary.
- Differentiator: most agent stacks have either no extension model or a one-way "plugin loads, plugin runs". AGH extensions are bidirectional (extension → daemon RPC), can declare capabilities they provide AND consume, and use the same hooks/skills/MCP/tools surfaces that native code uses. Both Go and TypeScript SDKs ship.
- Direct quote (`_archived/.../ext-architecture/_techspec.md:5-7`): "Extensions are modeled as **three-dimensional packages** that bundle **resources** (agents, skills, hooks, MCP configs), provide **capabilities** (agent drivers, memory backends, observe exporters), and perform **actions** via a bidirectional Host API."
- User-visible outcome: `agh extension install <path>`; the extension's manifest declares capabilities; the daemon subprocess-launches it with capability-negotiation handshake; failures are scoped, not global.

---

## Supporting features (shipped but secondary)

### Channels (real-time messaging adapters)
- Status: shipped (`_archived/.../channels/`, `internal/bridges/`, commits `20bad010 feat: add channels`, `7a225bd3 refactor: rename spaces to channels`).
- Hybrid: daemon owns channel registry, scoped routing, delivery-target resolution, secret binding; extensions own platform transport (Telegram/Slack/Discord). One stream contract for outbound delivery: `channels/deliver`.
- Differentiator vs competitors: most agent products treat Slack/Discord as an afterthought hardcoded in the app shell. AGH has a typed channel substrate so any extension can provide a platform adapter that participates in routing, retries, and outbound streaming.

### Automation system (schedules + triggers)
- Status: shipped (`_archived/.../automation/`, `internal/automation/`, commit `9b6c7d76 feat: add automation (#16)`, `99d38914 refactor: automation improvements pass`). Hermes hardening adds **durable scheduler state** for at-most-once dispatch (`hermes/_techspec.md` Track 3, ADR-002).
- Cron + interval + one-shot + event-driven triggers (session events, webhook with HMAC, memory consolidation, hook events). Unified `Dispatcher` so both time and event paths produce identical session creation. TOML declarative + API/CLI dynamic; both persisted in SQLite. Global and workspace scope.
- Differentiator: most agent harnesses are interactive-only. AGH agents can be triggered by cron, by webhooks, or by other agents' events out of the box, with persisted scheduler state that survives daemon restart without duplicate fires.

### Skills v2 — MCP lazy-load + lifecycle + marketplace seam
- Status: shipped (`_archived/20260408-201357-skills-v2/`, `internal/skills/`, `internal/skills/bundled/`).
- Skills declare MCP servers in `metadata.agh.mcp_servers`; daemon provisions them at session start with trust-tiered consent (bundled/user auto-approved, marketplace requires consent). Skills declare lifecycle hooks in `metadata.agh.hooks`. Five-source precedence: Bundled → Marketplace → User → Additional → Workspace; agent-local overrides all.
- Differentiator: most "skills" systems are just markdown prompts. AGH skills carry typed metadata that wires them into the runtime's MCP, hooks, memory, and provenance machinery — and load-time `VerifyContent` security scanning runs on every load (not just install).

### Sandbox profiles (local + Daytona)
- Status: shipped (`internal/sandbox/` with `daytona/`, `local/`, `providertest/`, `registry.go`; commit `ed1688cf feat: add ext refac and sandbox`).
- Workspace/session-selected execution boundary. Two providers (`local`, `daytona`); lifecycle hooks `sandbox.prepare/ready/sync.before/sync.after/stop`; Host API surfaces `sandbox/list/info/exec`. Glossary intentionally separates `sandbox` from `environment` (process-level env vars).

### Per-session ACP provider override
- Status: shipped (active `.compozy/tasks/session-driver-override/`, commit `2e8b2450 feat: session provider selection (#60)`).
- One agent can keep its default ACP runtime while individual sessions pick a different provider. Provider becomes part of session identity — persisted on disk and in the global session index, validated on resume, no silent fallback.
- Differentiator: most harnesses are single-provider per agent at config-edit time. AGH lets you flip provider per session with deterministic resume.

### Hermes hardening (production-grade runtime)
- Status: shipped (active `.compozy/tasks/hermes/`, commit `803d0fdc refactor: hermes adjustments (#69)`, `713c9f1f feat: production grade adjustments (#66)`). Six tracks: state/migrations/retention/retry foundations, ACP/session lifecycle hardening (typed `FailureKind`, agent probes, crash bundles), durable automation scheduler, MCP OAuth 2.1 + PKCE, memory CLI health/history, CLI/setup/release hardening (`agh config show|get|set|edit|path|check`, `agh mcp auth login|status|logout`, completion, uninstall, update, install script, `.env` repair).
- Differentiator: this is the unglamorous-but-load-bearing list — numbered SQL migrations, retention sweeps, agent probes, crash bundles, OAuth with redaction, symlink escape protection. Most agent products skip this and accept silent corruption.

### Tasks UI + Settings UI + Network workspace UI
- Status: shipped (`_archived/.../tasks-ui/`, `_archived/.../settings-ui/`, `_archived/.../network-workspace`, commits `1fe58fff feat: tasks ui (#36)`, `39cc6696 feat: settings ui (#37)`, `25db48fa feat: redesign network workspace (#59)`). Web at `web/src/systems/{tasks,settings,session,network}/`.
- Task-native dashboard, inbox, list (split view), kanban, detail (events SSE), run detail, multi-agent live screens. Live SSE-driven views with task tree, claim/lease projections, coordinator state.

### Observability spine
- Status: shipped (`internal/observe/`, `internal/store/sessiondb/`, `internal/store/globaldb/`).
- Append-only event store (`runtime.db`) is the canonical operational ledger; per-session DBs (`events.db`) are projections. Live broadcasters publish only after durable append; reconnect/replay uses `after_seq`. Coverage matrix tests fail if any required lifecycle path skips its canonical event.
- Direct quote (`internal/CLAUDE.md:51`): "Append-only event store (`runtime.db`) is the canonical operational ledger; session DBs are projections, not authority. Live broadcasters publish only after durable append; reconnect/replay uses `after_seq`."

### Documentation site (Fumadocs at agh.network)
- Status: shipped (`_archived/.../site/`, `packages/site/`, commit `e479c328 feat: add site (#26)`).
- Two doc collections: AGH Runtime (operators, ~50 pages) + AGH Network Protocol (implementers, ~13 pages). CLI reference auto-generated from Cobra (`make cli-docs`). Shared `@agh/ui` package across `web/` and `packages/site/`.

---

## Planned / in-progress

- **Tools-refac canonical surface** (active `tools-refac/`) — extends the shipped foundation with default discovery toolsets, a tools startup prompt section, bundled `agh-tools-guide`, dynamic policy-input resolution, expanded built-in coverage, session-bound autonomy execution tools, and status-only MCP auth visibility.
- **AGH Network v1 verified-format identity** (`docs/_memory/glossary.md:70-74`) — `nickname@fingerprint` plus `proof` strip-defense classification. Today the wire ships v0.
- **Eval/replay harness** (`autonomous/_techspec.md:534`, step 14 post-MVP) — recorded ACP/session trajectories, YAML cases, deterministic replay assertions.
- **Memory provenance + session summaries** (`autonomous/_techspec.md:532`, step 12 post-MVP) — broader recall provenance and session-end summaries before broad turn/network extraction.
- **WASM extension tier (L2)** (`_archived/.../ext-architecture/_techspec.md:5`) — designed as a future seam, deferred until hook latency or sandbox requirements justify it.
- **AGENT.md format** (`docs/_memory/glossary.md:45`) — proposed in RFC 001, "not yet fully implemented."
- **Coordinator dashboards / spawn lineage tree / autonomy alerts UI** (`autonomous/_techspec.md:535`, step 15 post-MVP).
- **Bridge SDK executable tool adapters, remote peer tool execution over AGH Network** (`tools-registry/_techspec.md:25-29` post-MVP).
- **Marketplace registry full implementation** (`_archived/.../skills-v2/`) — pluggable interface shipped, ClawHub backend in flight.

---

## Cross-cutting themes

These are NOT features — they are invariants that span every shipped feature and form the most accurate positioning surface for a marketing site.

### A. "Agent-manageable by default" (SD-011)
- Source: `docs/_memory/standing_directives.md` SD-011, `internal/CLAUDE.md` line 26, `CLAUDE.md` line 7.
- Direct quote (`CLAUDE.md:7`): "**Core product premise**: every capability must be both extensible by the runtime and manageable by agents. Features are incomplete if they only work through internal Go calls or the web UI."
- Meaning: every public AGH surface is reachable via CLI verbs with `-o json/jsonl`, HTTP/UDS parity, and is documented for the agent path. Agents can configure, start, stop, claim, release, and repair through structured commands. UI-only manageability is incomplete by definition.
- Why this matters for the site: the current site markets UI-first. The actual win is "your agents drive the system using the same surfaces you do."

### B. Single-binary local-first daemon
- Source: `internal/CLAUDE.md` "Runtime" section (lines 39-46), `CLAUDE.md` line 3.
- Direct quote (`internal/CLAUDE.md:41`): "Single-binary and local-first. Sidecars or external control planes require a written techspec."
- Meaning: no cloud, no required external services, no NATS cluster, no vector DB. The daemon embeds its NATS server (network-only boundary exception) and uses SQLite for everything durable. Runs in background by default, no foreground terminal required.

### C. Detached execution lifetime (SD-010)
- Source: `docs/_memory/standing_directives.md` SD-010, `internal/CLAUDE.md` lines 33-35.
- Meaning: prompts, network sends, and automation jobs detach via `context.WithoutCancel(ctx)` so client disconnect kills streaming, not execution. Long-running agent work survives operator browser refreshes.
- Why this matters: real differentiator vs harnesses where closing the tab kills the agent's work mid-flight.

### D. Truthful UI > plausible UI (SD-007)
- Source: `docs/_memory/standing_directives.md` SD-007.
- Direct quote: "UI must reflect actual backend support. Don't render controls or metrics the runtime doesn't model."
- Why this matters: anti-pattern for the site itself. The current site claims things ("replayable" stuff that doesn't matter) while ignoring shipped features.

### E. Greenfield alpha — zero legacy tolerance (SD-002)
- Source: `CLAUDE.md` lines 9-17, `docs/_memory/standing_directives.md` SD-002.
- Implication for marketing: AGH is alpha; honest framing wins trust here.

### F. Composition root discipline + extensible by default
- Source: `internal/CLAUDE.md` lines 11-14, SD-008, SD-011.
- Direct quote (`internal/CLAUDE.md:11`): "Designed for incremental extension — new capabilities arrive as new packages wired into `daemon/`, without modifying existing packages. Small interfaces + dependency injection. Every capability plan decides which extension points, hooks, capabilities, tools/resources, bundles, registries, bridge SDKs, and docs must be added, updated, or removed."

### G. Strong vocabulary discipline
- Source: `docs/_memory/glossary.md`.
- `capability` is the only canonical name — `recipe`/`workflow`/`procedure`/`playbook` are forbidden synonyms.
- `sandbox` is for execution boundaries, not env vars.
- `AGENT.md` (single agent definition) ≠ `AGENTS.md` (project file).
- Marketing copy must mirror this — using the wrong word in copy is a self-inflicted credibility hit.

### H. Security invariants are first-class
- Source: `internal/CLAUDE.md` lines 54-62.
- `claim_token` redaction non-negotiable, never on the wire / in logs / in memory.
- Symlink escape hardening on every load (skills, sidecars, bundles).
- Path security via `sanitizePathKey + realpathDeepestExisting`.
- Identity proof-stripping defense (network v1).
- Load-time skill content scan via `VerifyContent` on every load (not just install).
- External-call timeouts mandatory; `http.DefaultClient` forbidden in production.

---

## Implicit positioning statements found in specs

These are the lines closest to "this is what makes AGH different" — quote them, don't paraphrase.

1. `CLAUDE.md:3` — "AGH is an Agent Operating System — a Go single-binary daemon that manages AI agent sessions via ACP (Agent Client Protocol)."
2. `CLAUDE.md:5` — "**Goals**: daemon single-binary in background, strong observability, agent-first system (agents manipulate via CLI + REST), highly extensible, highly configurable."
3. `CLAUDE.md:7` — "**Core product premise**: every capability must be both extensible by the runtime and manageable by agents. Features are incomplete if they only work through internal Go calls or the web UI."
4. `docs/_memory/glossary.md:216` — "AGH **competes on runtime, SDK, observability, DX, and integration depth — NOT the wire protocol.** The protocol must remain implementable outside AGH."
5. `docs/_memory/glossary.md:212-215` — "AGH is **not a workflow engine**. Capabilities are interpretive, not deterministic programs. AGH is **not a federation protocol**. AGH is **not an MCP replacement**. MCP integrates _into_ AGH skills via `metadata.agh.mcp_servers`. AGH is **not an A2A replacement**."
6. `internal/CLAUDE.md:11` — "Designed for incremental extension — new capabilities arrive as new packages wired into `daemon/`, without modifying existing packages."
7. `internal/CLAUDE.md:26` — "Agent-manageable by default. User-visible runtime capabilities must expose stable machine-readable control surfaces for agents... UI-only manageability is incomplete."
8. `autonomous/_techspec.md:7-9` — "...autonomy behavior is added through four coordinated layers: Situation Surface, Agent Kernel CLI, Autonomy Kernel, and Memory/Self-Correction... Autonomy extensibility is a first-class requirement."
9. `unified-capabilities/_techspec.md:5` — "A capability becomes the only authored delegation artifact, the only rich discovery artifact, and the only transferable procedural artifact on the wire."
10. `_archived/.../network/_techspec.md:11-17` — "Embedded NATS server in the daemon binary (single-binary, local-first). Each active session is a unique peer with identity `{agent_name}.{session_id}`."
11. `tools-registry/_techspec.md:5` — "AGH's Tool Registry as a daemon-owned runtime service, not as a static list of built-in commands."
12. `_archived/.../hooks/_techspec.md:7` — "...typed dispatch functions (not a generic event bus)... define the full contract now rather than evolve through incompatible one-off seams."
13. `docs/_memory/standing_directives.md` SD-011 — "AGH is not only a daemon with UI. It is an extensible runtime that agents must be able to inspect, configure, operate, and repair through structured surfaces."
14. `docs/_memory/standing_directives.md` SD-005 — "real-scenario QA against a multi-agent / multi-channel / multi-task workspace catches drift `make verify` misses."

---

## Marketing-ready highlights (top 7)

Ranked by differentiation strength × shipping confidence × user-visible payoff. One-line hook each.

1. **An open workplace for AI agents.** (previously phrased as *"Your agents can finally talk to each other"*.) AGH Network — embedded peer-to-peer protocol, every session is a peer, agents discover each other via `greet`/`whois` and exchange typed envelopes (`direct`, `say`, `capability`). Single binary. Local-first. On by default for new installs.
2. **A real autonomy kernel, not a fork-and-pray loop.** Token-fenced task claim/lease, durable scheduler, coordinator agent for semantic decomposition, mechanical scheduler for sweep/recovery, safe spawn with TTL and permission narrowing — all sharing one queue with manual operator control.
3. **Capabilities — what your agents can do, expressed once and shared.** A canonical structured artifact (id/summary/outcome/context/artifacts/outline/constraints/examples/requirements) authored locally, projected into network discovery, and transferable on the wire with a runtime-computed digest. Replaces ad-hoc "tool descriptions in the prompt."
4. **One tool registry across native, extension, and MCP.** Same canonical `ToolID`, same policy engine, same hosted MCP exposure — whether the tool is compiled-in Go, a third-party Go/TS extension, or an external MCP server. Policy decisions return structured reasons, not opaque denies.
5. **Memory that consolidates only when it should.** Dual-scope persistent memory + Dream gate-cascade (Time → Sessions → Lock). When all three pass, AGH spawns an ephemeral session that synthesizes recent events into durable typed memory files. Agents write memory themselves via CLI.
6. **Extensions are bidirectional, three-dimensional packages — in Go or TypeScript.** Bundle resources (agents/skills/hooks/MCP), provide capabilities (drivers/backends/exporters), and call back into the daemon through a capability-scoped Host API. Same surface as native code.
7. **Agent-manageable by default.** Every capability has a CLI verb with `-o json/jsonl` and an HTTP/UDS path. Your agents drive AGH using the same surfaces you do. UI is observability; the system is operated through structured, deterministic commands.

Honorable mentions worth a feature-page tile each (not a hero):
- Channels (Slack/Telegram/Discord adapters as extensions, daemon-owned routing)
- Automation (cron + webhook + event triggers, durable at-most-once scheduling)
- Hooks platform (typed dispatch, deterministic precedence, can deny but cannot escalate)
- Sandbox profiles (local + Daytona, lifecycle hooks, Host API)
- Per-session ACP provider override (deterministic, persisted, no silent fallback)
- Observability spine (append-only event ledger, SSE replay via `after_seq`, coverage matrix tests)

---

## Anti-marketing: things to STOP saying

- "Replayable" as a hero claim — `after_seq` reconnect-replay is a correctness invariant, not a customer benefit. It is implementation discipline, not a marketable feature.
- "Workflow"/"recipe"/"procedure"/"playbook" — forbidden vocabulary per `docs/_memory/glossary.md:15`. Use `capability`.
- "Federation" or "trust network" — RFC 004 is explicit: AGH Network is self-certified pairwise, not federated.
- "MCP replacement" — MCP integrates INTO AGH skills via `metadata.agh.mcp_servers`. Don't position as a competitor.
- "A2A replacement" — same: industry standard that can coexist.
- "Production-ready" — AGH is greenfield alpha by explicit policy. Honest framing wins trust at this stage.
- UI-first framing — violates the agent-manageable-by-default product premise.
