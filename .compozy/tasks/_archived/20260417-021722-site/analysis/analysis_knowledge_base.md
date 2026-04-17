# Knowledge Base Sources Analysis for AGH Documentation Site

**Date**: 2026-04-15
**Scope**: QMD collections, Obsidian knowledge vault, KB CLI, project docs, sandbox/site task artifacts
**Purpose**: Catalog all knowledge sources, extract documentation-relevant content, and organize findings for site planning

---

## 1. Knowledge Sources Found and Their Status

### 1.1 QMD Collections (15 total)

QMD is installed and operational at `/Users/pedronauck/.local/share/mise/installs/node/24.14.0/bin/qmd`. BM25 search works; hybrid/vector search (`qmd query`) fails due to a `sqlite-vec` module error in the local Bun SQLite build.

| Collection          | Files | Status                         | AGH Relevance                                                                                             |
| ------------------- | ----- | ------------------------------ | --------------------------------------------------------------------------------------------------------- |
| `agh-compozy`       | 0     | Empty, not indexed             | Direct -- intended for AGH project docs, not yet populated                                                |
| `agh-docs`          | 0     | Empty, not indexed             | Direct -- intended for AGH documentation, not yet populated                                               |
| `ai-harness`        | 201   | Active, heavily used           | High -- 24 wiki articles (~99K words) covering agent harness architecture, tools, memory, security, evals |
| `agent-networks`    | 106   | Active                         | High -- 33 wiki articles (~106K words) on agent-to-agent protocols, discovery, trust, distributed systems |
| `claude-code`       | 74    | Active                         | Medium -- 15 wiki articles (~48K words) on Claude Code harness architecture                               |
| `hermes`            | 93    | Active                         | Medium -- 23 wiki articles (~66K words) on Hermes agent architecture                                      |
| `openfang`          | 82    | Active                         | Medium -- 22 wiki articles (~65K words) on OpenFang agent runtime                                         |
| `openclaw`          | 58    | Active                         | Medium -- 21 wiki articles (~55K words) on OpenClaw architecture                                          |
| `goclaw`            | 60    | Active                         | Medium -- 21 wiki articles (~51K words) on GoClaw architecture                                            |
| `pi-mono`           | 74    | Active                         | Medium -- 13 wiki articles (~44K words) on Pi monorepo runtime                                            |
| `ai-memory`         | 173   | Active, compiled stage pending | Medium -- 15 wiki articles on memory architectures for agents                                             |
| `go-best-practices` | 149   | Active, compiled stage pending | Low -- Go performance optimization (useful for implementation, not docs site)                             |
| `rewrite-qa`        | 27    | Active                         | Low -- QA process documentation                                                                           |
| `kodebase-vault`    | 0     | Empty                          | N/A                                                                                                       |
| `kodebase-go`       | 0     | Empty                          | N/A                                                                                                       |

**Key finding**: The two AGH-specific collections (`agh-compozy`, `agh-docs`) are empty. All documentation-relevant knowledge currently lives in the domain-topic collections and in project artifacts. Populating these collections should happen after the site taxonomy is approved.

### 1.2 Obsidian Knowledge Vault

Located at `/Users/pedronauck/Dev/knowledge/`. This is a multi-topic Obsidian vault following the Karpathy Knowledge Base Pattern. The vault has `.obsidian/` config, a `.kb/vault/` symlink tree for KB CLI access, and a structured README.

**Vault topics (12):**

| Topic               | Domain         | Wiki Articles        | Raw Sources                         | Key Content                                                                         |
| ------------------- | -------------- | -------------------- | ----------------------------------- | ----------------------------------------------------------------------------------- |
| `ai-harness`        | AI             | 24 (~99K words)      | 133 articles + 26 bookmark clusters | Agent harness patterns, tool design, memory, observability, competitive landscape   |
| `agent-networks`    | Agent Networks | 33 (~106K words)     | 66 articles + 1 snapshot            | Protocol landscape (ANP, A2A, MCP), identity, trust, discovery, distributed systems |
| `claude-code`       | Tool           | 15 (~48K words)      | 50 articles                         | Claude Code internals: harness architecture, hooks, skills, prompt layering         |
| `hermes`            | Hermes         | 23 (~66K words)      | 65 articles                         | Python hub-and-spoke agent gateway, skills pipeline, adapters                       |
| `openfang`          | OpenFang       | 22 (~65K words)      | 55 articles                         | Rust daemon, workspace sandboxing, knowledge graph, channel adapters                |
| `openclaw`          | OpenClaw       | 21 (~55K words)      | 32 articles                         | Gateway/assistant split, sessions, DMs, memory, skills marketplace                  |
| `goclaw`            | GoClaw         | 21 (~51K words)      | 34 articles                         | Go agent runtime, WebSocket RPC, multi-tenant PostgreSQL, BM25+pgvector skills      |
| `pi-mono`           | Pi Mono        | 13 (~44K words)      | 56 articles                         | TypeScript monorepo, minimal extensible runtime, dual system skills                 |
| `ai-memory`         | AI Memory      | 15 (pending compile) | Pending                             | Memory consolidation, forgetting, episodic/semantic memory, production patterns     |
| `go-best-practices` | Go             | 18 (pending compile) | 76 articles + 50 videos             | Go performance optimization, profiling, concurrency                                 |
| `multica`           | Unknown        | Exists but minimal   | Pending                             | Newly scaffolded                                                                    |
| `paperclip`         | Unknown        | Exists but minimal   | Pending                             | Newly scaffolded                                                                    |
| `_inbox`            | Inbox          | Capture notes        | Ideas                               | Quick capture items pending triage                                                  |

**Key finding**: The vault contains ~545K words of compiled wiki content across 8 active topics, plus ~600+ raw source documents. The `ai-harness` and `agent-networks` topics are the highest-value sources for AGH documentation and positioning.

### 1.3 KB CLI

Installed at `/Users/pedronauck/.local/share/mise/installs/node/24.14.0/bin/kb`. The CLI scaffolds topic-based knowledge vaults, manages QMD collections, and supports search. However, `kb topic list` and `kb search` fail from the AGH repo root because the `.kb/vault/` directory only exists in the knowledge vault root at `/Users/pedronauck/Dev/knowledge/`.

The KB vault has 8 symlinked topics at `/Users/pedronauck/Dev/knowledge/.kb/vault/`: claude-code, goclaw, hermes, multica, openclaw, openfang, paperclip, pi-mono.

### 1.4 Project Documentation

All located under `/Users/pedronauck/Dev/compozy/agh/`:

- **README.md**: Product overview, architecture diagram, feature list, CLI tree, quick start
- **CLAUDE.md**: Engineering guide with build commands, architecture principles, package layout
- **docs/rfcs/**: 5 RFCs (agent-md, skills, network v0, network v1, network old)
- **docs/plans/**: 5 design plans (workspace, network, RFC examples, automation, bridges)
- **docs/ideas/**: 8 directories covering network drafts, Claude Code reverse-engineering, extensibility, orchestration, competitive analysis
- **docs/\_refacs/**: Bundle/bridge reconciliation analysis
- **docs/design/**: 8 JPEG design mockup images
- **.compozy/tasks/\_archived/**: 27 archived task directories covering the full development history

### 1.5 Compozy Task Artifacts

- **Sandbox task** (`_idea.md`, `_techspec.md`, `adrs/`): Execution environment abstraction for local/Daytona/E2B providers
- **Site task** (`analysis/`): 8 analysis files plus main synthesis document
- **Archived tasks**: 27 prior task specs covering every major AGH subsystem

### 1.6 Competitor Resources

Located at `/Users/pedronauck/Dev/compozy/agh/.resources/`:

- Full codebase snapshots of OpenClaw, OpenCode, OpenFang, GoClaw, Hermes, Pi-Mono, ACPX, T3Code, Harnss, Collaborator AI, Chat SDK
- Used for competitive analysis and documentation taxonomy benchmarking

---

## 2. Key Insights Extracted from Each Source

### 2.1 From QMD `ai-harness` Collection

**The Agent Harness article** (most relevant single article):

- Defines the agent harness as "everything between a foundation model and the end user"
- Identifies 5 core components: agentic loop, tool dispatch, permission model, context management, adversarial/security layer
- Documents 3 harness categories: IDE-integrated, sandbox-based, CLI-based
- Establishes that the harness is model-agnostic -- value is in the runtime, not the model
- Frames "configuration-as-behavior" -- the same harness creates different products through configuration surfaces
- Calls out the "harness as platform" thesis: skills, hooks, extensions, MCP servers, dashboards

**State of AI Agent Harnesses (2025-2026) briefing**:

- Three major shifts: prompt engineering to context engineering, single agents to multi-agent orchestration, ad-hoc tool calling to standardized protocols (MCP, A2A, ACP)
- The prototype-to-production gap is the dominant challenge -- reliability, observability, governance, cost
- Agent harness is now recognized as a distinct engineering discipline
- Coding agents are the proving ground for harness architecture

**Skill Systems Comparison across 6 harnesses**:

- 4 of 6 harnesses converge on AgentSkills standard (YAML + Markdown)
- Skill format convergence makes skills portable across harnesses
- Three distribution models: marketplace/registry, filesystem, compiled-in
- AGH's skill system should leverage this convergence for ecosystem compatibility

**Workspace and Directory Access comparison**:

- Detailed analysis of how 6 harnesses handle workspace resolution, sandboxing, and file access
- OpenFang's daemon model most similar to AGH's architecture

### 2.2 From QMD `agent-networks` Collection

**Agent Network Protocol (ANP) article**:

- Three-layer architecture: identity/secure communication, meta-protocol negotiation, application/discovery
- DID-based, peer-to-peer, trustless by design
- Positioned as "the HTTP of the agentic web era"
- Contrasts with A2A's enterprise trust assumptions

**Agent-to-Agent Protocol Landscape article**:

- Market has converged on protocol layers: MCP for tools, A2A for agent delegation, ACP for graph-based workflows
- Documentation must clarify where each protocol sits in the stack
- AGH Network needs to position within this layered ecosystem

**The MCP-A2A Composition Pattern**:

- The 2026 agent stack is layered: MCP for tools, A2A/agent protocols for agent-to-agent communication
- AGH should position its network protocol inside the layered stack, not as a generic runtime feature

### 2.3 From Project RFCs and Plans

**AGH Network Design (docs/plans/2026-04-08)**:

- Approved name: "AGH Network"
- Intentionally opinionated but not captive -- semantic core is transport-agnostic
- NATS is first normative transport profile
- Product moat is runtime, SDK, observability, and DX -- not protocol lock-in
- Lightweight task lifecycle, not a workflow engine

**RFC 003 (Network v0)**:

- Full wire format with 7 message kinds
- Interaction lifecycle with correlation
- NATS transport binding

**RFC 004 (Network v1)**:

- Ed25519 trust profile
- Conformance levels (basic, standard, verified)
- Extension model processing

**Orchestration patterns analysis (docs/ideas/orchestration/)**:

- Network protocol is a wire layer, not a workflow engine
- Multi-agent orchestration, state handoff, compensation, and observability belong in the daemon/runtime layer
- This is exactly the runtime/protocol split the site needs to reflect

### 2.4 From Sandbox Task

**\_idea.md** (execution environment research):

- AGH's execution model is currently hardcoded to local subprocess + local filesystem + daemon-owned PTYs
- The correct abstraction is "execution environment" with local as first-class provider, Daytona/E2B as optional
- Recommendation: Daytona first (official Go SDK, workspace-like lifecycle), E2B second (transient automation)
- Critical insight: remote execution changes launch, filesystem, terminal execution, permissions, sync, and resume -- not just the subprocess command

**\_techspec.md** (execution environment implementation):

- Three interfaces: Provider, Launcher, ToolHost
- SSH as primary transport for Daytona (clean stdio for ACP)
- Session-scoped sandbox with copy-on-start/collect-on-stop
- Workspace-scoped environment selection consistent with existing config model

### 2.5 From Existing Site Analysis Files

**analysis.md** (main synthesis):

- Two first-class product surfaces: AGH Runtime and AGH Network Protocol
- Homepage should answer "why AGH if many harnesses exist?" with a two-part story
- Docs should follow Diataxis inside domain-first navigation
- Next artifact: PRD first, then TechSpec

**analysis_network_protocol.md**:

- AGH Network should be positioned as a protocol product, not a runtime feature
- The protocol is the unique differentiator -- most harnesses have a runtime but not an open network surface
- Clean boundary table between Runtime owns (daemon, sessions, CLI, storage) and Protocol owns (envelope, message kinds, trust, transport)
- Adoption story: keep your runtime, map to AGH envelopes, implement smallest core first

**analysis_runtime_capabilities.md**:

- 7 launch-worthy differentiators: single daemon, unified operator surface, durable sessions, persistent memory with consolidation, skills as runtime assets, workspace-aware operations, advanced operations in one place
- Terminology note: standardize on "Memory" (not "Knowledge") as canonical user-facing term

**analysis_resources_docs.md**:

- 7 reusable documentation patterns from reference projects
- 7 reusable homepage/positioning patterns
- Key recommendation: docs home should route, not explain

**analysis_competitor_docs.md**:

- 11 competitor projects analyzed across 3 documentation maturity tiers
- Universal section taxonomy identified (16 sections ranked by frequency)
- OpenClaw has the most mature docs (Mintlify, 200+ pages, 13 locales)

**analysis_existing_knowledge.md**:

- Complete content inventory of all AGH documentation artifacts
- 5 RFCs, 5 design plans, 8 idea directories, 27 archived task specs

**analysis_project_features.md**:

- Comprehensive feature catalog with 7 core value propositions
- Detailed breakdown of sessions, ACP, memory, skills, workspaces, automation, bridges, extensions

---

## 3. Documentation-Relevant Content Organized by Topic

### 3.1 Product Identity and Positioning

**Sources**: analysis.md, analysis_network_protocol.md, ai-harness briefing, README.md

- AGH is an "Agent Operating System" -- not just another agent harness
- Two-pillar product: AGH Runtime (local daemon) + AGH Network (open protocol)
- Single binary. No sidecars. No external control plane.
- Model-agnostic -- harness value is independent of any model provider
- Configuration-as-behavior -- the same runtime becomes different products through config
- Local-first posture -- workspace identity is local and canonical even when execution is remote

### 3.2 Core Runtime Concepts

**Sources**: analysis_runtime_capabilities.md, analysis_project_features.md, CLAUDE.md, README.md

- **Sessions**: Full state machine (starting/active/stopping/stopped), resume support, types (user/dream/system), turn sources (user/network)
- **ACP Driver**: JSON-RPC over stdio, subprocess spawning, event streaming, permission policies
- **Memory**: Dual-scope (global + workspace), dream consolidation, read/write/delete operations
- **Skills**: Bundled/user/workspace/marketplace, MCP sidecars, AgentSkills format convergence
- **Workspaces**: Registration, resolution, config overlays, workspace-aware runtime behavior
- **Automation**: Jobs, triggers, runs, scheduled/event-driven execution
- **Bridges**: Instances, routes, delivery behavior, messaging platform integration
- **Extensions**: Go native + JSON-RPC subprocess two-tier architecture
- **Observability**: Health metrics, event recording, transcript replay, token cost tracking

### 3.3 Network Protocol Concepts

**Sources**: analysis_network_protocol.md, RFCs 003/004, network design plan, agent-networks vault

- **Envelope**: Canonical message wrapper with correlation, lineage, and routing
- **Message Kinds**: greet, whois, say, direct, receipt, trace, recipe (7 kinds in v0)
- **Interaction Lifecycle**: Lightweight task lifecycle, not a workflow engine
- **Discovery**: Capability signaling, whois semantics
- **Trust**: Claimed vs verified identity, Ed25519 baseline trust profile, conformance levels (basic/standard/verified)
- **Transport Profiles**: Core is transport-agnostic, NATS as first normative binding
- **Conformance**: Defined classes for implementer compatibility
- **Translation Boundary**: Local runtime events become protocol messages at the harness boundary; protocol messages become prompts/queue entries after validation

### 3.4 Execution Environment (Upcoming)

**Sources**: sandbox \_idea.md, sandbox \_techspec.md, sandbox ADRs

- Environment abstraction: local, Daytona, E2B as first three backends
- Local path vs runtime path separation (critical for remote execution)
- Provider lifecycle: prepare, sync-to, sync-from, destroy
- SSH as primary transport for Daytona (clean stdio for ACP, no PTY)
- Workspace-scoped environment selection through config profiles

### 3.5 Competitive Landscape

**Sources**: analysis_competitor_docs.md, ai-harness vault, docs/ideas/market-pair/

- 6 reference harnesses deeply analyzed: Claude Code, OpenClaw, OpenFang, GoClaw, Hermes, Pi-Mono
- AGH differentiates through: open network protocol, daemon architecture, workspace-aware memory, single-binary posture
- Documentation maturity benchmark: OpenClaw (Tier 1), OpenCode (Tier 1), GoClaw/OpenFang (Tier 2)
- AGH should target Tier 1 documentation quality from launch

---

## 4. Messaging and Positioning Themes Found

### 4.1 Primary Positioning

1. **"Runtime + Protocol" split**: AGH is two products -- a runtime you install and a protocol other harnesses can implement. This is the most consistent theme across all analysis documents.

2. **"Local-first agent runtime"**: Single binary, no sidecars, no external control plane. The operational simplicity is a key differentiator in a landscape of complex multi-service architectures.

3. **"The harness is the product"**: The ai-harness briefing establishes that harness quality differentiates products more than model capability. AGH should lean into this.

### 4.2 Secondary Positioning

4. **"Configuration-as-behavior"**: The same runtime becomes different products through workspace config, skills, hooks, and automation. This is the platform thesis.

5. **"Durable sessions, not ephemeral chats"**: Resume, replay, transcript, event persistence. This is the operational maturity story.

6. **"Open protocol, not captive ecosystem"**: AGH Network can be implemented by third parties. The moat is runtime quality, not protocol lock-in.

7. **"Workspace-aware everything"**: Memory, skills, config, agents -- all scoped to the workspace context.

### 4.3 Guardrails (Things NOT to Claim)

- Do not imply broad external protocol adoption before implementations ship
- Do not collapse runtime messaging into protocol messaging
- Do not claim the protocol defines orchestration or global federation
- Do not make NATS sound like the whole protocol (it is a transport profile)
- Do not describe v0/v1 RFCs as final standards
- Do not call the web UI network view "the protocol"

---

## 5. Technical Concepts That Need User-Facing Explanations

### 5.1 Core Concepts (Must Explain)

| Concept              | Internal Term                                       | Explanation Challenge                                                                                                      |
| -------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| Session lifecycle    | State machine with starting/active/stopping/stopped | Users need to understand resume, persistence, and stop reasons without seeing the state machine internals                  |
| ACP protocol         | Agent Client Protocol over JSON-RPC/stdio           | Users need to know AGH works with multiple agents (Claude Code, Codex, Gemini CLI) without understanding JSON-RPC          |
| Dual-scope memory    | Global + workspace memory with dream consolidation  | "Dream consolidation" needs a clear user-facing metaphor -- it is background memory refinement                             |
| Workspace resolution | Config overlay, agent discovery, skill loading      | Users register a workspace directory and AGH loads the right config/agents/skills -- explain the benefit, not the resolver |
| Skills               | AgentSkills format, MCP sidecars, marketplace       | Users need to understand install, enable, and use -- not the reconciliation pipeline                                       |
| Automation           | Jobs/triggers/runs/webhooks                         | Framing should be "scheduled and event-driven agent work" not "cron system"                                                |
| Bridges              | Bridge instances, routes, targets                   | Framing should be "connect agents to Slack/Discord/Telegram" not "bridge adapter architecture"                             |
| Extensions           | Two-tier: Go native + JSON-RPC subprocess           | Users need "install, enable, configure" -- not the process model                                                           |
| Observability        | Event streams, health, reconciliation, audits       | Frame as "see what your agents are doing" with concrete UI and CLI paths                                                   |

### 5.2 Protocol Concepts (Must Explain Separately)

| Concept               | Explanation Challenge                                                                 |
| --------------------- | ------------------------------------------------------------------------------------- |
| Envelope              | The universal message wrapper -- needs a simple diagram showing fields                |
| Message kinds         | 7 kinds in v0 -- explain when each is used in practical terms                         |
| Interaction lifecycle | Lightweight correlation -- not a workflow engine, just tracking a conversation thread |
| Trust model           | Claimed vs verified -- "you can check who sent a message"                             |
| Transport profiles    | "The protocol works over different transports, NATS is the first one"                 |
| Conformance           | "How to know if your implementation is compatible"                                    |

### 5.3 Advanced Concepts (Can Defer)

| Concept                                 | Why It Can Wait                                                                        |
| --------------------------------------- | -------------------------------------------------------------------------------------- |
| Execution environments (Daytona/E2B)    | Not yet shipped; explain when Phase 2 launches                                         |
| Extension Host API                      | Developer-facing, not operator-facing                                                  |
| Dream consolidation internals           | The user-facing story is "memory improves over time"                                   |
| Network runtime vs protocol distinction | Important for the site, but internal to AGH's architecture -- users just see "network" |

---

## 6. Existing Documentation Drafts That Could Be Adapted

### 6.1 Directly Adaptable

| Source                                        | Adapt Into                                | Notes                                                        |
| --------------------------------------------- | ----------------------------------------- | ------------------------------------------------------------ |
| `README.md`                                   | Getting Started, Architecture overview    | Already has quick start, CLI tree, architecture diagram      |
| `docs/plans/2026-04-08-agh-network-design.md` | Protocol Overview page                    | Clean summary of approved protocol direction                 |
| `docs/rfcs/003_agh-network-v0.md`             | Protocol Reference (v0 wire format)       | Needs editorial pass for external audience                   |
| `docs/rfcs/004_agh-network-v1.md`             | Protocol Reference (v1 trust/conformance) | Needs editorial pass                                         |
| `analysis.md` (site analysis synthesis)       | Site PRD input                            | Homepage strategy, docs taxonomy, and CTAs are ready for PRD |
| `analysis_runtime_capabilities.md`            | Runtime Overview page                     | 7 launch-worthy differentiators already drafted              |
| `analysis_network_protocol.md`                | Protocol landing page                     | Runtime vs protocol split already articulated                |

### 6.2 Reference Material (Extract and Rewrite)

| Source                                                | Extract For                          | Notes                                              |
| ----------------------------------------------------- | ------------------------------------ | -------------------------------------------------- |
| `docs/rfcs/001_agent-md-with-skills-memory.md`        | Agents concept page                  | Agent definition format and scoping                |
| `docs/rfcs/002_skills-system-final.md`                | Skills concept page                  | Lifecycle hooks, MCP bridge, marketplace model     |
| `docs/plans/2026-04-10-automation-techspec-design.md` | Automation concept page              | Cron, triggers, webhooks design                    |
| `docs/plans/2026-04-15-bridge-adapters-design.md`     | Bridges concept page                 | Channel integration model                          |
| `docs/plans/2026-04-06-workspace-entity-design.md`    | Workspaces concept page              | Resolver and entity management                     |
| `docs/ideas/market-pair/`                             | Competitive differentiation inputs   | Gap analysis vs OpenClaw, OpenFang, GoClaw, Hermes |
| `docs/ideas/from-claude-code/`                        | Architecture influence docs          | How Claude Code patterns influenced AGH            |
| `.compozy/tasks/sandbox/_techspec.md`                 | Future "Execution Environments" page | When feature ships                                 |

### 6.3 Knowledge Vault Articles (Rewrite for AGH Context)

| Article                                                                                  | Rewrite For                                        | Value                                                |
| ---------------------------------------------------------------------------------------- | -------------------------------------------------- | ---------------------------------------------------- |
| `ai-harness/wiki/concepts/The Agent Harness.md`                                          | "What is an Agent Harness?" explainer or blog post | Provides the category definition AGH operates in     |
| `ai-harness/wiki/concepts/Memory Systems for Agents.md`                                  | Memory concept page background                     | Taxonomy and lifecycle model                         |
| `agent-networks/wiki/concepts/Agent-to-Agent Protocol Landscape.md`                      | Protocol positioning context                       | Where AGH Network fits in the ecosystem              |
| `ai-harness/outputs/briefings/state-of-ai-agent-harnesses-2025-2026.md`                  | Blog post or "Why AGH" page                        | Industry context for AGH's positioning               |
| `ai-harness/outputs/queries/2026-04-06-skill-systems-comparison-across-six-harnesses.md` | Skills concept page, competitive context           | Shows AGH's skill system in context of the ecosystem |

---

## 7. Recommended Next Steps

1. **Populate `agh-compozy` QMD collection** with the approved site content taxonomy after the PRD is approved. This makes the documentation searchable and maintainable through QMD.

2. **Populate `agh-docs` QMD collection** once documentation pages are written. This becomes the canonical docs search index.

3. **Create a site PRD** (`.compozy/tasks/site/_prd.md`) incorporating the two-pillar positioning, homepage strategy, and docs taxonomy from the synthesis analysis.

4. **Reuse the knowledge vault** as background context for documentation writing, not as copy. The vault articles are written as domain-neutral reference material; AGH docs need product-specific framing.

5. **Seed the competitor docs benchmark** from `analysis_competitor_docs.md` -- target OpenClaw-level maturity (Tier 1: dedicated framework, tab-based navigation, i18n support).

---

## Evidence and Source Paths

### QMD CLI

- `qmd collection list` -- 15 collections inventoried
- `qmd ls ai-harness` -- 12 output files
- `qmd search` -- BM25 search operational
- `qmd query` -- hybrid search broken (sqlite-vec module error)

### Obsidian Vault

- `/Users/pedronauck/Dev/knowledge/` -- vault root
- `/Users/pedronauck/Dev/knowledge/.kb/vault/` -- 8 symlinked KB topics
- `/Users/pedronauck/Dev/knowledge/CLAUDE.md` -- vault conventions
- `/Users/pedronauck/Dev/knowledge/README.md` -- topic inventory (12 topics, ~545K compiled words)

### KB CLI

- `kb --help` -- operational
- `kb topic list` -- requires `.kb/vault/` in working directory

### Project Documentation

- `/Users/pedronauck/Dev/compozy/agh/README.md`
- `/Users/pedronauck/Dev/compozy/agh/CLAUDE.md`
- `/Users/pedronauck/Dev/compozy/agh/docs/rfcs/` (5 RFCs)
- `/Users/pedronauck/Dev/compozy/agh/docs/plans/` (5 design plans)
- `/Users/pedronauck/Dev/compozy/agh/docs/ideas/` (8 idea directories)

### Task Artifacts

- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/sandbox/_idea.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/sandbox/_techspec.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/sandbox/adrs/` (3 ADRs)
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/` (8 analysis files + synthesis)

### Existing Site Analysis

- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_kb_qmd_obsidian.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_network_protocol.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_runtime_capabilities.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_resources_docs.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_competitor_docs.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_existing_knowledge.md`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/site/analysis/analysis_project_features.md`
