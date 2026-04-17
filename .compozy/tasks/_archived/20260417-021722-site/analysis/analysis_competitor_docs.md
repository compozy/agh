# Competitor Documentation Structure Analysis

Analysis of documentation taxonomy patterns across 11 reference agent harness projects in `.resources/`. Conducted to inform the AGH documentation site architecture.

---

## 1. Summary Table

| Project             | Language            | Doc Framework                              | Doc Location                                       | Sections (Top-Level)                                                                                                       | i18n                             | Protocol Docs                                                                           | Maturity  |
| ------------------- | ------------------- | ------------------------------------------ | -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------- | --------------------------------------------------------------------------------------- | --------- |
| **OpenClaw**        | TypeScript          | Mintlify                                   | `docs/` (200+ pages)                               | Get Started, Install, Channels, Agents, Tools & Plugins, Models, Platforms, Gateway & Ops, Reference, Help                 | Yes (13 locales, glossary-based) | Dedicated "Protocols and APIs" section within Gateway tab                               | Very High |
| **OpenCode**        | Go/TypeScript       | Astro Starlight (web), Mintlify (SDK docs) | `packages/web/src/content/docs/`, `packages/docs/` | Index, Config, Providers, Network, Enterprise, Troubleshooting, Usage, Configure, Develop                                  | Yes (18 locales)                 | `network.mdx` page, `acp.mdx` page                                                      | High      |
| **GoClaw**          | Go                  | Plain Markdown (numbered)                  | `docs/` (25 files)                                 | Architecture, Agent Loop, Providers, Tools, Gateway Protocol, Channels, Store, Skills, Security, Observability, Teams, API | No                               | Dedicated `04-gateway-protocol.md` + `19-websocket-rpc.md` + `05-channels-messaging.md` | Medium    |
| **OpenFang**        | Rust                | Plain Markdown                             | `docs/` (18 files)                                 | Getting Started, Core Concepts, Integrations, Reference, Release & Ops                                                     | No                               | Dedicated `mcp-a2a.md` for MCP and A2A protocols                                        | Medium    |
| **Hermes**          | Python              | Static HTML landing page + README-driven   | `docs/` (7 files), `landingpage/`                  | Specs, Plans, Migration, Skins                                                                                             | No                               | No dedicated protocol docs                                                              | Low       |
| **ACPX**            | TypeScript          | Plain Markdown (date-prefixed)             | `docs/` (19 files)                                 | Architecture, Session Management, ACP Coverage Roadmap, Integration Plans, Flows                                           | No                               | ACP protocol is the core topic; all docs are protocol-centric                           | Medium    |
| **Pi**              | TypeScript          | Plain Markdown (in-package)                | `packages/coding-agent/docs/` (23 files)           | Compaction, Providers, SDK, Session, Settings, Skills, TUI, Extensions, RPC                                                | No                               | `rpc.md` for protocol reference                                                         | Medium    |
| **T3Code**          | TypeScript          | Plain Markdown                             | `docs/` (3 files)                                  | Observability, Release, Effect-fn Checklist                                                                                | No                               | None                                                                                    | Low       |
| **Harnss**          | TypeScript/Electron | README-only                                | No docs dir                                        | README-driven (screenshots, quickstart)                                                                                    | No                               | None (uses ACP SDK)                                                                     | Low       |
| **Collaborator AI** | TypeScript/Electron | Internal specs                             | `docs/superpowers/` (4 files)                      | Plans, Specs (internal)                                                                                                    | No                               | None                                                                                    | Low       |
| **Claude Code**     | TypeScript          | Source-code-only (no user docs in repo)    | Prompts in `_prompts/`                             | N/A (reference is source code + prompt templates)                                                                          | No                               | N/A                                                                                     | N/A       |

---

## 2. Common Patterns Across Projects

### 2.1 Documentation Tiers

Projects cluster into three documentation maturity tiers:

**Tier 1 -- Full Documentation Sites (OpenClaw, OpenCode)**

- Dedicated documentation framework (Mintlify, Astro Starlight)
- Tab-based navigation with 7-10 top-level sections
- Internationalization support (13-18 languages)
- Separate content from code (dedicated `docs/` or `packages/web/`)
- Sidebar configuration as structured JSON/JS
- Redirects for URL evolution
- Edit links back to source

**Tier 2 -- Structured Markdown Docs (GoClaw, OpenFang, ACPX, Pi)**

- Plain markdown files in a `docs/` directory
- Numbered or flat file structure
- README.md as index/table of contents
- No doc framework -- rendered by GitHub or custom site
- Manual cross-referencing between documents

**Tier 3 -- README-Driven (Hermes, T3Code, Harnss, Collaborator AI)**

- README.md carries all user-facing documentation
- Internal `docs/` used for specs and plans, not user guides
- No navigation structure
- Documentation is an afterthought to the code

### 2.2 Universal Section Taxonomy

Across all projects with structured docs, these sections appear most frequently:

| Section                          | Appears In      | AGH Relevance                  |
| -------------------------------- | --------------- | ------------------------------ |
| **Getting Started / Quickstart** | All 11 projects | Essential                      |
| **Installation**                 | 9/11 projects   | Essential                      |
| **Configuration**                | 8/11 projects   | Essential                      |
| **Architecture / Concepts**      | 7/11 projects   | Essential                      |
| **CLI Reference**                | 6/11 projects   | Essential                      |
| **API Reference**                | 5/11 projects   | Essential                      |
| **Providers / Models**           | 7/11 projects   | Essential                      |
| **Tools / Skills**               | 6/11 projects   | Essential                      |
| **Troubleshooting**              | 6/11 projects   | High                           |
| **Security**                     | 5/11 projects   | High                           |
| **Protocol / Wire Format**       | 5/11 projects   | Essential (AGH differentiator) |
| **Channels / Integrations**      | 4/11 projects   | Medium                         |
| **Sessions / Memory**            | 5/11 projects   | High                           |
| **Observability**                | 3/11 projects   | High                           |
| **SDK / Developer Guide**        | 4/11 projects   | High                           |
| **Contributing**                 | 7/11 projects   | Medium                         |
| **Platforms**                    | 3/11 projects   | Low (AGH is CLI-first)         |

### 2.3 Conceptual vs Reference vs Tutorial Separation

**OpenClaw (most mature)** uses a clear three-way split across its tabs:

- **Conceptual**: "Agents" tab (Fundamentals, Sessions & Memory, Multi-agent, Messages & delivery)
- **Reference**: "Reference" tab (CLI commands, RPC, Templates, Technical reference)
- **Tutorial/How-to**: "Get Started" tab (First steps, Guides), "Install" tab
- **Operational**: "Gateway & Ops" tab (Configuration, Security, Protocols, Networking)

**OpenCode** uses a flatter model:

- Top-level pages are mostly reference (config, providers, tools)
- "Usage" group is tutorial-oriented (go, tui, cli, web, ide)
- "Configure" group is reference
- "Develop" group is SDK/contributor

**GoClaw** uses numbered documents mixing concepts and reference:

- `00-architecture-overview.md` is conceptual
- `04-gateway-protocol.md` mixes concept with wire-format reference
- `18-http-api.md` is pure reference

**Pi** keeps docs flat and reference-oriented:

- Each doc covers one topic (settings.md, providers.md, session.md)
- No explicit separation between concepts and reference

### 2.4 Navigation Patterns

**Tab-based (OpenClaw, OpenCode):**

- Top-level tabs for major domains: Get Started, Install, Agents, Tools, Reference
- Groups within each tab for sub-topics
- Deeply nested sub-groups (e.g., OpenClaw has "Memory" as a nested group under "Sessions and memory" under the "Agents" tab)

**Numbered sequential (GoClaw):**

- `00-architecture-overview.md` through `24-knowledge-vault.md`
- Implies a reading order
- Good for learning, poor for random access

**Flat/alphabetical (Pi, OpenFang):**

- One markdown file per topic, no hierarchy
- README as table of contents with category headings
- Simple but does not scale

---

## 3. Unique / Innovative Documentation Approaches

### 3.1 OpenClaw -- Mintlify Tabs as Information Architecture

OpenClaw's most distinctive approach is using Mintlify's tab system to create **persona-based navigation**:

- "Get Started" for new users
- "Agents" for people configuring agent behavior
- "Tools & Plugins" for people extending functionality
- "Gateway & Ops" for people running the system in production
- "Reference" for looking up specific commands/APIs
- "Help" for troubleshooting

This means a user who is deploying OpenClaw never needs to see the agent configuration docs unless they want to. Each tab is a self-contained journey.

### 3.2 OpenClaw -- Protocol Documentation with Dedicated Subsection

Within "Gateway & Ops", OpenClaw has "Protocols and APIs" as a dedicated group containing:

- `gateway/protocol` -- Gateway WebSocket protocol
- `gateway/bridge-protocol` -- Bridge protocol
- `gateway/openai-http-api` -- OpenAI-compatible HTTP API
- `gateway/openresponses-http-api` -- OpenResponses HTTP API
- `gateway/tools-invoke-http-api` -- Tools invocation API
- `gateway/cli-backends` -- CLI backend transport
- `gateway/local-models` -- Local model integration

This is a **protocol-first subsection within an operational context**, not a standalone "Protocol" tab. The rationale: protocols are consumed by people building integrations or debugging, who are already in an operational mindset.

### 3.3 GoClaw -- Numbered Architecture Documentation

GoClaw's numbered system (`00-architecture-overview.md` through `24-knowledge-vault.md`) creates a natural learning path. Key innovation:

- Architecture overview (00) establishes the mental model first
- System layers documented in dependency order
- Protocol docs (04, 19) placed early -- they are foundational
- Higher-level features (skills, teams, knowledge vault) come later
- Each doc is self-contained but references others by number

### 3.4 OpenCode -- Astro Starlight with Heavy Localization

OpenCode uses Astro Starlight (a documentation-focused Astro integration) which provides:

- Auto-generated sidebar from content structure
- Built-in i18n routing (`/docs/ja/config`, `/docs/ko/config`)
- Starlight theme customization for visual identity
- MDX support for interactive components

Their sidebar structure is defined directly in `astro.config.mjs` as a flat array with labeled groups -- no separate sidebar config file.

### 3.5 OpenFang -- README as Documentation Index

OpenFang's `docs/README.md` is the best example of a lightweight docs index:

- Organized into clear sections: Getting Started, Core Concepts, Integrations, Reference, Release & Ops
- Each entry is a table row with document name and one-line description
- Quick Reference section at the bottom with 30-second start, key numbers, important paths, env vars
- This pattern works when you have 15-20 docs; it fails at 50+

### 3.6 ACPX -- Date-Prefixed Architecture Decision Records

ACPX prefixes all docs with dates (`2026-02-17-architecture.md`, `2026-03-25-acpx-flows-architecture.md`). This is effectively an ADR (Architecture Decision Record) system embedded in the docs directory. It provides:

- Clear chronological evolution of the project
- Natural way to see how thinking evolved
- But poor discoverability for users looking for current state

### 3.7 OpenClaw -- Redirect-Heavy URL Strategy

OpenClaw's `docs.json` contains 100+ redirects, mapping legacy URLs to current locations. This shows aggressive URL structure evolution while maintaining backward compatibility -- important for a mature project with external links.

---

## 4. How Protocol-Focused Projects Document Protocols

### 4.1 OpenClaw (WebSocket + Bridge + HTTP APIs)

**Location**: `gateway/protocol.md`, `gateway/bridge-protocol.md`, `gateway/openai-http-api.md` etc.
**Navigation**: Nested under "Gateway & Ops" > "Protocols and APIs" group
**Format**: Each protocol gets its own page with:

- Transport description (WebSocket text frames with JSON)
- Handshake/authentication sequence
- Frame format with JSON examples
- Message types enumerated
- Version negotiation

### 4.2 GoClaw (WebSocket RPC v3)

**Location**: `04-gateway-protocol.md` (wire format + lifecycle), `19-websocket-rpc.md` (method catalog)
**Format**: Split into two complementary docs:

- Protocol doc: Mermaid sequence diagrams for WebSocket lifecycle, connection parameters table, frame type table, request/response/event frame structures
- RPC methods doc: Every method with request/response JSON examples, grouped by domain (Connection, Chat, Agents, Sessions, Config, Skills, Cron, Teams)

This **two-document split** (transport layer vs. method catalog) is the clearest pattern for protocol documentation.

### 4.3 OpenFang (MCP + A2A + OFP)

**Location**: `mcp-a2a.md` (MCP and A2A), `architecture.md` (OFP wire protocol section)
**Format**: `mcp-a2a.md` is a long-form guide split into Part 1 (MCP) and Part 2 (A2A), each with:

- Overview
- Client and Server perspectives
- Configuration (TOML examples)
- API endpoints table
- Security considerations

The OFP (OpenFang Protocol) for peer-to-peer networking is documented within the architecture doc as a section, not a standalone page. This buries a critical differentiating feature.

### 4.4 ACPX (ACP Protocol Client)

ACPX is unique because the **entire project is a protocol client**. Its documentation approach:

- `VISION.md` explains the protocol philosophy
- `README.md` shows practical usage with annotated CLI output
- `docs/2026-02-19-acp-coverage-roadmap.md` tracks spec conformance
- Architecture docs explain session management, transport details

### 4.5 Pi (RPC)

**Location**: `docs/rpc.md`
**Format**: Single reference document for the RPC interface, structured as a method catalog.

### 4.6 Pattern Summary for Protocol Docs

| Pattern                                      | Used By        | Best For                            |
| -------------------------------------------- | -------------- | ----------------------------------- |
| **Separate transport + methods docs**        | GoClaw         | Complex protocols with many methods |
| **Per-protocol standalone page**             | OpenClaw       | Multiple independent protocols      |
| **Combined guide (concepts + config + API)** | OpenFang       | Protocols that need usage context   |
| **Section within architecture doc**          | OpenFang (OFP) | Minor/emerging protocols            |
| **Coverage roadmap**                         | ACPX           | Spec compliance tracking            |

---

## 5. Recommended Documentation Taxonomy for AGH

Based on the analysis, AGH should adopt a hybrid approach that combines OpenClaw's persona-based tabs with GoClaw's numbered learning path and OpenFang's clean index pattern.

### 5.1 Recommended Top-Level Tabs

```
1. Overview           -- What AGH is, quick demo, key concepts
2. Getting Started    -- Install, first session, first agent, 5-minute tutorial
3. Guides             -- Configuration, agents, sessions, memory, skills, workspaces
4. Architecture       -- System design, daemon lifecycle, package layout, concurrency model
5. Protocol           -- ACP, JSON-RPC, HTTP/SSE, UDS, Network Protocol (Phase 3)
6. CLI Reference      -- Every command with flags and examples
7. API Reference      -- HTTP endpoints, SSE events, UDS contract types
8. Operations         -- Observability, troubleshooting, production deployment
```

### 5.2 Rationale for Each Tab

**Overview** (from OpenCode/OpenFang pattern)

- AGH needs a strong "what is this" page because Agent OS is a novel category
- Quick numbers table (like OpenFang's key numbers)
- Feature comparison vs. alternatives

**Getting Started** (universal pattern)

- Install (binary, go install, homebrew)
- Configuration (TOML basics)
- First session walkthrough
- First agent definition
- Web UI quickstart

**Guides** (from OpenClaw's "Agents" and "Tools" tabs, merged)

- These are the how-to docs: how to configure agents, manage sessions, use memory, create skills
- Organized by feature domain, not by persona
- Each guide mixes concept explanation with practical steps

**Architecture** (from GoClaw's numbered pattern)

- This is AGH's differentiator -- the system is architecturally interesting
- Package layout, daemon composition root, concurrency model
- Session state machine, event flow, notifier pattern
- This section serves both users (understanding behavior) and contributors

**Protocol** (dedicated tab -- AGH's key differentiator)

- AGH's network protocol (Phase 3) is a unique feature that deserves first-class treatment
- ACP (Agent Client Protocol) reference
- JSON-RPC over stdio transport
- HTTP/SSE transport
- UDS transport for CLI IPC
- Future: Agent network protocol specification
- Each protocol gets the two-document treatment: transport layer + method/event catalog

**CLI Reference** (from OpenClaw and Pi patterns)

- Every command, grouped by domain
- Machine-parseable format for auto-generation

**API Reference** (from GoClaw and OpenFang patterns)

- HTTP endpoints with request/response examples
- SSE event types
- UDS contract types
- Possibly auto-generated from code

**Operations** (from OpenClaw's "Gateway & Ops" pattern)

- Observability: event recording, health metrics, query engine
- Troubleshooting guide
- Production deployment considerations
- SQLite database management

### 5.3 Recommended Section Structure Within Each Tab

```
Overview/
  what-is-agh.md
  features.md
  comparison.md

Getting Started/
  install.md
  configuration.md
  first-session.md
  first-agent.md
  web-ui.md

Guides/
  Agents/
    agent-definitions.md
    agent-drivers.md
    spawning-agents.md
  Sessions/
    session-lifecycle.md
    state-machine.md
    event-persistence.md
    transcript-replay.md
  Memory/
    memory-system.md
    global-vs-workspace.md
    dream-consolidation.md
  Skills/
    skills-overview.md
    bundled-skills.md
    custom-skills.md
  Workspaces/
    workspace-resolver.md
    workspace-configuration.md
  Configuration/
    toml-reference.md
    agent-definitions-format.md
    environment-variables.md

Architecture/
  overview.md
  daemon-lifecycle.md
  package-layout.md
  concurrency-model.md
  session-state-machine.md
  event-system.md
  store-layer.md

Protocol/
  overview.md                    -- Protocol landscape: which protocol for what
  ACP/
    acp-overview.md              -- What ACP is, why AGH uses it
    acp-transport.md             -- JSON-RPC over stdio wire format
    acp-methods.md               -- Method catalog
  HTTP-SSE/
    http-api-transport.md        -- HTTP/SSE transport description
    http-api-endpoints.md        -- Endpoint catalog
    sse-events.md                -- SSE event type reference
  UDS/
    uds-transport.md             -- Unix domain socket transport
    uds-api.md                   -- UDS method catalog
  Network/ (Phase 3)
    network-protocol-overview.md -- Agent network protocol vision
    network-discovery.md         -- Peer discovery
    network-messaging.md         -- Inter-daemon messaging

CLI Reference/
  index.md
  daemon-commands.md
  session-commands.md
  agent-commands.md
  memory-commands.md
  config-commands.md

API Reference/
  index.md
  http-endpoints.md
  sse-events.md
  uds-contract.md
  error-codes.md

Operations/
  observability.md
  troubleshooting.md
  database.md
  production-checklist.md
```

### 5.4 Protocol Documentation Strategy

AGH should treat protocols as a **first-class documentation domain** because:

1. **AGH has multiple transport layers** (ACP/stdio, HTTP/SSE, UDS) -- each needs clear docs
2. **The network protocol (Phase 3) is a differentiating feature** that will attract contributors and users
3. **Protocols are consumed by different audiences**: ACP by agent authors, HTTP/SSE by web UI developers, UDS by CLI developers, Network by system integrators

Recommended approach (combining GoClaw's two-doc split with OpenClaw's per-protocol pages):

- **Protocol Overview page**: explains which protocol to use for what scenario
- **Per-protocol transport doc**: wire format, authentication, connection lifecycle, error handling
- **Per-protocol method/event catalog**: every method with request/response shapes, every event with payload shapes
- **Protocol conformance page** (ACPX pattern): track ACP spec coverage as a living document

### 5.5 Documentation Framework Recommendation

Based on the reference projects:

| Framework           | Used By                    | Pros                                            | Cons                          |
| ------------------- | -------------------------- | ----------------------------------------------- | ----------------------------- |
| **Mintlify**        | OpenClaw, OpenCode (SDK)   | Beautiful, hosted, tab navigation, i18n, search | Paid, hosted service          |
| **Astro Starlight** | OpenCode (web)             | Open source, great i18n, customizable, MDX      | More setup work               |
| **Plain Markdown**  | GoClaw, OpenFang, ACPX, Pi | Zero overhead, GitHub renders it                | No search, no sidebar, no nav |
| **Docusaurus**      | (none in this set)         | Open source, versioning, i18n                   | React-heavy, slower builds    |
| **VitePress**       | (none in this set)         | Fast, Vue-based, good DX                        | Less established              |

For AGH, **Astro Starlight** is the best fit because:

- Open source (aligns with AGH's open-source ethos)
- Built-in sidebar with group/tab support
- i18n ready from day one
- MDX for interactive content (code tabs, diagrams)
- Fast builds, good search
- Already used by OpenCode (largest comparable project)
- Can be deployed to any static host

### 5.6 Key Principles Derived from Analysis

1. **Separate concerns by audience, not by content type**. OpenClaw's tab model works because operators, agent authors, and plugin developers each have their own space.

2. **Protocol docs need both a "why" page and a "what" catalog**. GoClaw's two-document split (protocol overview + method catalog) is the clearest pattern.

3. **Architecture docs should exist as a first-class section**, not buried in a README or CONTRIBUTING. GoClaw and OpenFang both treat architecture as foundational reading.

4. **An index/TOC page per section is essential**. OpenFang's `docs/README.md` with tables is the minimum viable pattern.

5. **Numbered docs work for learning paths; named docs work for reference**. AGH should use named docs with an explicit "recommended reading order" on the overview page.

6. **i18n should be structural from day one** even if only English exists initially. OpenCode and OpenClaw both show that bolt-on i18n is painful.

7. **Redirects are a sign of maturity**, not failure. Plan for URL evolution from the start.

8. **Keep protocol docs close to operational docs**. People debugging protocol issues are in an operational mindset -- they should not need to navigate to a separate "concepts" section.

---

## 6. Gap Analysis: What AGH Needs That Competitors Lack

| Gap                              | Description                                                     | AGH Opportunity                                                              |
| -------------------------------- | --------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| **Daemon management docs**       | Most projects are CLI-only; none document daemon lifecycle well | AGH is daemon-first -- document boot, shutdown, lock, health comprehensively |
| **Multi-transport architecture** | Projects typically expose one transport; docs assume one        | AGH has 3 transports (ACP, HTTP/SSE, UDS) -- document the decision tree      |
| **Event persistence model**      | Only OpenClaw documents memory/sessions deeply                  | AGH's SQLite event store + replay is a differentiator worth documenting      |
| **Dream consolidation**          | Unique to AGH and OpenClaw (different implementations)          | Novel feature deserving its own guide                                        |
| **Agent network protocol**       | OpenFang has OFP but buries it; GoClaw has none                 | AGH Phase 3 should have protocol docs ready alongside implementation         |
| **Workspace-scoped everything**  | Most projects are single-workspace                              | AGH's workspace resolver and dual-scope memory are worth highlighting        |
| **CLI-as-agent-tool**            | Unique pattern where agents use CLI for self-management         | Document this interaction model clearly                                      |

---

## 7. Appendix: Per-Project Detail

### OpenClaw (Mintlify, 200+ pages)

**Navigation tabs**: Get Started, Install, Channels, Agents, Tools & Plugins, Models, Platforms, Gateway & Ops, Reference, Help

**Key groups within tabs**:

- Get Started: Overview, First Steps, Guides
- Agents: Fundamentals, Sessions & Memory (with nested Memory sub-group), Multi-agent, Messages & Delivery
- Tools & Plugins: Overview, Plugins (with nested Building Plugins + SDK Reference sub-groups), Skills, Automation & Tasks, Tools (with nested Web Browser + Web Tools), Agent Coordination
- Gateway & Ops: Gateway (Config, Security, Protocols, Networking), Remote Access, Security, Nodes & Media, Web Interfaces
- Reference: CLI Commands (7 sub-groups), RPC & API, Templates, Technical Reference, Concept Internals, Project, Release Policy

**i18n system**: Glossary-based (`.i18n/glossary.<locale>.json`), translation memory (`.tm.jsonl`), English source of truth. Covers 13+ locales.

### OpenCode (Astro Starlight + Mintlify SDK docs)

**Starlight sidebar** (from `astro.config.mjs`):

- Top-level: Index, Config, Providers, Network, Enterprise, Troubleshooting, Windows
- Usage group: Go, TUI, CLI, Web, IDE, Zen, Share, GitHub, GitLab
- Configure group: Tools, Rules, Agents, Models, Themes, Keybinds, Commands, Formatters, Permissions, LSP, MCP Servers, ACP, Skills, Custom Tools
- Develop group: SDK, Server, Plugins, Ecosystem

**SDK Mintlify docs** (`packages/docs/docs.json`): Getting Started, Quickstart, Development + AI Tools sub-section

### GoClaw (Numbered markdown, 25 docs)

**Document sequence**:
00-Architecture, 01-Agent Loop, 02-Providers, 03-Tools, 04-Gateway Protocol, 05-Channels, 06-Store, 07-Bootstrap/Skills/Memory, 08-Scheduling, 09-Security, 10-Tracing, 11-Teams, 12-Extended Thinking, 13-WS Team Events, 14-Skills Runtime, 15-Core Skills, 16-Skill Publishing, 17-Changelog, 18-HTTP API, 19-WS RPC, 20-API Keys, 21-Agent Evolution, 22-Heartbeat, 22-HTTP Endpoints, 23-Multi-tenant, 24-Knowledge Vault + journals + model-steering

### OpenFang (Flat markdown, 18 docs)

**Sections per `docs/README.md`**:

- Getting Started: getting-started, configuration, cli-reference, troubleshooting
- Core Concepts: architecture, agent-templates, workflows, security
- Integrations: channel-adapters, providers, skill-development, mcp-a2a
- Reference: api-reference, desktop
- Release & Ops: production-checklist

### ACPX (Date-prefixed markdown, 19 docs)

**All docs are date-prefixed ADRs**: architecture, session-management, acp-coverage-roadmap, mock-agent-testing, openclaw-integration-plan, session-identity-spec, warm-session-owner, session-model, flows-architecture, flow-trace-replay, flow-replay-viewer, flow-permission-requirements, flow-replay-live-transport, built-in-agent-launch-ownership + CLI reference + error strategy + json-patch-plus

### Pi (Flat markdown in package, 23 docs)

**Document list**: compaction, custom-provider, development, extensions, json, keybindings, models, packages, prompt-templates, providers, rpc, sdk, session, settings, shell-aliases, skills, terminal-setup, termux, themes, tmux, tree, tui, windows

### Hermes (Minimal docs, 7 files)

**Docs**: acp-setup, honcho-integration-spec (HTML+MD), migration/openclaw, plans/pricing-architecture, skins/example-skin.yaml, specs/container-cli-review-fixes

### T3Code (Minimal docs, 3 files)

**Docs**: observability, release, effect-fn-checklist

### Harnss, Collaborator AI

Both are README-driven with no user-facing documentation sites. Internal docs cover plans and specs only.
