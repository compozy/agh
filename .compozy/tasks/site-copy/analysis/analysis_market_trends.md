# Agent Harness Market Trends — 2025-2026

> Research compiled 2026-05-01 for the AGH marketing-site rewrite (`packages/site/`). All quotes are verbatim from competitor landing pages or cited articles. Where a date is known it is included.

---

## Competitor positioning (one-liners with sources)

### Cognition / Devin — `cognition.ai`, `devin.ai`
- Hero (Cognition): **"cognition is an Agent Lab"** — "we're the makers of [ devin The AI software engineer ]"
- Hero (Devin): **"Devin, the AI software engineer"** / "Devin is built for engineering teams with complex, multi-repo projects."
- Lead value props (verbatim): *"Learns your codebase & picks up tribal knowledge"*, *"Multi-week, multi-repo projects"*, *"Assign a fleet of agents to migrate all repos in parallel"*, *"Investigate Datadog incidents immediately"*.
- Source: https://www.cognition.ai/, https://devin.ai/

### Cursor — `cursor.com`
- Hero: **"Built to make you extraordinarily productive, Cursor is the best way to code with AI."**
- Sub: *"Trusted every day by teams that build world-class software"*.
- Lead props: *"Agents turn ideas into code"*, *"Works autonomously, runs in parallel — Agents use their own computers to build, test, and demo features end to end"*, *"In every tool, at every step — Cursor runs in your terminal, collaborates in Slack, and reviews PRs in GitHub."*
- Source: https://www.cursor.com/

### Anthropic / Claude Code — `claude.com/product/claude-code`
- Hero: **"Claude Code: AI-powered coding assistant for developers"**
- Sub: *"Work with Claude directly in your codebase. Build, debug, and ship from your terminal, IDE, Slack, or the web. Describe what you need, and Claude handles the rest."*
- Lead props: *"Manage multiple parallel tasks, review visual diffs, preview servers, and monitor PR status"*; *"Use Claude Code where you work"* (Desktop / Terminal / IDE / Web/iOS / Slack).
- Source: https://claude.com/product/claude-code

### OpenAI Codex CLI — `github.com/openai/codex`
- Tagline: **"Lightweight coding agent that runs in your terminal"**
- README hero: *"Codex CLI is a coding agent from OpenAI that runs locally on your computer."*
- Source: https://github.com/openai/codex

### OpenHands (All-Hands AI) — `docs.openhands.dev`
- Hero: **"Welcome to OpenHands, a community focused on AI-driven development"**
- SDK hook: *"Define agents in code, then run them locally, or scale to 1000s of agents in the cloud"*.
- CLI hook: *"The easiest way to start using OpenHands. The experience will be familiar to anyone who has worked with e.g. Claude Code or Codex."*
- Source: https://docs.openhands.dev/, https://github.com/All-Hands-AI/OpenHands

### Block / Goose — `goose-docs.ai`
- Hero: **"Your native open source AI agent. Desktop app, CLI, and API — for code, workflows, and everything in between."**
- Sub: *"goose is a general-purpose AI agent that runs on your machine. Not just for code — use it for research, writing, automation, data analysis, or anything you need to get done."*
- Lead props: *"Built in Rust for performance and portability"*; *"Connect to 70+ extensions"*; *"Capture workflows as portable YAML configs. Share with your team, run in CI"*; *"Prompt injection detection, tool permission controls, sandbox mode, and an adversary reviewer"*.
- Source: https://block.github.io/goose/ → https://goose-docs.ai/

### Aider — `aider.chat`
- Hero: **"AI pair programming in your terminal"**
- Sub: *"Aider lets you pair program with LLMs to start a new project or build on your existing codebase."*
- Lead props: *"Maps your codebase"*, *"Git integration"* (auto-commits with sensible messages), *"Linting & testing"* (auto-runs tests and fixes detected issues).
- Source: https://aider.chat/

### LangGraph — `langchain.com/langgraph`
- Hero: **"Balance agent control with agency"**
- Sub: *"Design agents that reliably handle complex tasks with LangGraph, an agent runtime and low-level orchestration framework."*
- Lead props: *"Guide, moderate, and control your agent with human-in-the-loop"*, *"Persist memory for future interactions"*, *"First-class streaming for better UX design"*.
- Source: https://www.langchain.com/langgraph

### CrewAI — `crewai.com`
- Hero: **"Accelerate AI agent adoption and start delivering production value"**
- Sub: *"CrewAI makes it easy for enterprises to operate teams of AI agents that perform complex tasks autonomously, reliably and with full control."*
- Three-pillar framing: *Easy / Trusted / Scalable* — heavy enterprise vocabulary.
- Source: https://www.crewai.com/

### MetaGPT — `github.com/geekan/MetaGPT`
- Tagline: **"Assign different roles to GPTs to form a collaborative entity for complex tasks."**
- Positioning: *"The Multi-Agent Framework: First AI Software Company, Towards Natural Language Programming"*; slogan *"Code = SOP(Team)"*.
- Source: https://github.com/geekan/MetaGPT

### Microsoft AutoGen — `github.com/microsoft/autogen`
- Tagline: *"A framework for creating multi-agent AI applications that can act autonomously or work alongside humans."*
- **Status (2025-2026): in maintenance mode.** Microsoft now points new users to the **Microsoft Agent Framework** as the "enterprise-ready successor to AutoGen".
- Source: https://github.com/microsoft/autogen

### Letta — `letta.com`
- Hero: **"Letta Code — Memory-first agents that learn"**
- Sub: *"Create agents that can be taught with natural language and improve from experience."*
- Lead props: *"Persistent agents instead of stateless sessions"*, *"Always improving and learning — Letta uses background memory subagents to enhance prompts, context, and skills over time"*, *"Own your memory and port it across models"*, *"Chat from any device, run on any environment"*.
- Source: https://www.letta.com/

### MCP (Anthropic) — `modelcontextprotocol.io`
- Position: *"MCP (Model Context Protocol) is an open-source standard for connecting AI applications to external systems."*
- Hook: *"Think of MCP like a USB-C port for AI applications."*
- Source: https://modelcontextprotocol.io/

### A2A (Linux Foundation, ex-Google) — `a2a-protocol.org`
- Hero: **"Agent2Agent (A2A) Protocol"** — *"An open standard designed to enable seamless communication and collaboration between AI agents."*
- Pillars: *"Interoperability"*, *"Complex Workflows"*, *"Security & IP Protection — Agents interact without needing to share internal memory, tools, or proprietary logic, ensuring security and preserving intellectual property."*
- Source: https://a2a-protocol.org/latest/

---

## Vocabulary that's WORKING right now

These phrases consistently show up in winning landing pages, dev articles, and HN/RedMonk discourse — they signal something concrete and defensible:

| Phrase | Why it lands | Where seen |
| --- | --- | --- |
| **"runs locally" / "runs on your machine"** | Direct statement of trust and data residency, contrasts with cloud-only competitors | Codex CLI, Goose, Letta ("run on any environment"), local-first agent listicles |
| **"Agent harness"** | Has become the *industry term of art* in 2026, replacing "agent framework" | Aakash Gupta ("2025 was Agents. 2026 is Agent Harnesses."), Phil Schmid, Jonathan Fulton, awesome-harness-engineering |
| **"background agents" / "long-running agents"** | Concrete capability with measurable benefit (parallel throughput, overnight runs) | Cursor, Devin, Addy Osmani, RedMonk #1 thing devs want |
| **"Persistent memory" / "memory-first" / "stateful agents"** | Direct counter to "stateless LLM" pain | Letta, MemGPT, RedMonk #2, Sleep-time Compute paper |
| **"Plan / Worker / Judge"** (or *Planner-Worker-Judge*) | Cursor's hierarchical-agents lexicon — adopted across discourse | Cursor FastRender post-mortem |
| **"Spec-driven"** | Specs as source of truth that agents update | Kiro, RedMonk #6, Anthropic playbook |
| **"Composable"** | Letta SDK, OpenHands SDK ("composable Python library") | OpenHands README |
| **"Own your memory" / "audit-able" / "version-controllable"** | Local-first/data-sovereignty appeal | Letta, OpenClaw, profClaw |
| **"Parallel coding" / "fleet of agents"** | Devin's "fleet of agents to migrate all repos in parallel"; Simon Willison's "embracing the parallel coding agent lifestyle" | Devin, Simon Willison via Addy Osmani |
| **"Observability" / "what your agent actually did"** | Hard counter to opaque cloud agents; spawns whole ecosystem of dashboards | codeburn, tokentap, claude-code-hooks-multi-agent-observability |
| **"Skills" (reusable, version-controlled)** | Anthropic's framing has propagated; RedMonk #10 | Anthropic, RedMonk |
| **"Single binary" / "Built in Rust"** | Concrete proof of portability — Goose leans on this | Goose |
| **"Hooks"** | Claude Code 27-event hook pipeline now industry-standard; agentic governance | Claude Code, awesome-harness-engineering |
| **"Open protocol" / "open standard"** | A2A and MCP both lead with this; aligns with developer trust | A2A, MCP |
| **"Auditable" / "every decision is logged"** | Trust + governance angle | OpenClaw, Microsoft Agent Governance Toolkit |

---

## Vocabulary that's BURNED OUT

These phrases are templated marketing tropes — they have been used so universally that they actively *signal slop* to a developer audience. AGH should avoid them in hero copy:

- **"AI-powered"** — used on every SaaS site, including non-AI ones. Note: even Anthropic uses *"AI-powered coding assistant"* on the Claude Code page, but they earn it because of brand. We don't.
- **"Supercharge your workflow / your team / your X"** — Copy.ai uses "supercharge" 4× in a single post; Jasper, Optiminastic, Superside all use it verbatim.
- **"Revolutionary" / "game-changing" / "next-generation"** — universal AI-startup filler.
- **"Unleash the power of"** — same template family.
- **"AI-driven"** — synonym fatigue with "AI-powered".
- **"10x your productivity"** — quantitative claim no developer believes anymore.
- **"Reliable, scalable, trusted"** (the CrewAI three-pillar pattern) — generic enterprise filler unless paired with concrete proof.
- **"The future of [X]"** — temporal vagueness.
- **"Effortlessly" / "seamlessly"** — adverbs of nothing.
- **"Build the future" / "shape what's next"** — vibe copy.
- **"Smart" / "intelligent" / "magical"** — magic = no specifics. Cursor gets away with *"magically accurate autocomplete"* because they pair it with the specific claim "predicts your next action".
- **"Production-ready"** — has become so overused (Microsoft, CrewAI, Vercel, et al.) that it now reads as defensive.
- **"Empower your developers"** — every dev-tool company circa 2018-2024.
- **"Cutting-edge" / "state-of-the-art"** — research lab leakage; on a landing page it reads as filler.
- **"Drop-in replacement"** — overused in the local-LLM space (LocalAGI, etc.); risk of sounding derivative.

**Specific patterns AGH should avoid for itself:**
- "Agent OS" as a *noun phrase in the hero*. The category is real but RedMonk, Microsoft, OpenAGI, and 4-5 listicles all already claim it. Use "agent operating system" only as a category descriptor lower on the page, not as the headline.
- Claiming **"truly autonomous"** — every framework claims this; it has lost meaning.
- "Connect anything to anything" — MCP already owns this lane.

---

## Hot themes in agent infrastructure

Top 7 themes with traction in 2025-2026, ranked by signal strength across sources:

### 1. The harness *is* the product
"The model is commodity. Claude, GPT-4, Gemini perform similarly. The harness determines whether agents succeed or fail." (Aakash Gupta, Mar 2026.) LangChain reportedly moved their score on Terminal Bench 2.0 from **52.8% → 66.5% by changing only the harness, not the model**. This is the strongest macro-narrative in the space and AGH sits squarely in the middle of it.

### 2. Background / long-running agents as a *category*
Cursor (Feb 2026), Devin, GitHub Copilot coding agent (May 2025), Codex Cloud, Anthropic Managed Agents — all pushing the same shape: queue tasks, work for hours/days, return PRs. RedMonk's #1 dev-want for 2025 IDEs is "Background Agents". Addy Osmani: *"An agent that runs for ten hours can own an entire feature, finish a migration that was on the backlog for six quarters."*

### 3. Memory-first / stateful / sleep-time compute
Letta's entire pivot from MemGPT into "Letta Code" centers on this. The *sleep-time compute* paradigm (background memory consolidation while idle) is the freshest hook in 2025-2026 — Letta blog, papers, and podcasts. Episodic vs semantic memory is *technical* discourse; "memory that compounds with use" is the *marketing* phrasing that lands.

### 4. Local-first / self-hosted / "you own the runtime"
The cost-and-data-sovereignty narrative is strong: *"There is no per-agent fee, no compute markup, and no cost tied to the number of tasks."* (Programming Insider, 2026.) Hermes Agent (Nous Research, Feb 2026), OpenClaw, profClaw, LocalAGI all lead with this. The phrase **"single binary"** is rare and notable — Goose comes closest with *"Built in Rust for performance and portability"*.

### 5. MCP everywhere — but as table stakes, not differentiation
MCP support is now *expected*. Cursor, Claude Code, Goose, OpenHands, OpenCode, Letta, profClaw all support MCP. RedMonk #4. **Claiming "MCP support" is no longer a hero-level differentiator — it belongs in the feature grid.**

### 6. Observability / hooks / "what did the agent actually do"
Real pain. Reverse-engineering blog posts about Claude Code's 27-event hook pipeline are widely cited. Third-party ecosystem (codeburn, tokentap, claude-code-hooks-multi-agent-observability) exists *because* native tooling is insufficient. This is a genuinely under-served need.

### 7. Multi-agent / planner-worker-judge / fleets
Cursor FastRender (1M+ lines, 30k commits, 2000 parallel agents). Devin "fleet of agents". Microsoft Agent Mesh. CrewAI "crew of agents". This pattern is now *the* shape of high-end agent systems. AGH's network protocol fits this.

---

## A2A / Agent communication landscape

**Where the field has settled (mid-2026):**
- **MCP = agent-to-tool** ("USB-C for AI"). Anthropic-originated, governed by MCP Working Group, ubiquitous.
- **A2A = agent-to-agent** (originally Google, governance transferred to Linux Foundation in 2025; **IBM's ACP merged into A2A in August 2025**, consolidating the agent-comms standards).
- The official line from Google and the LF: *"complementary, not competing"*. The de-facto layered pattern is *A2A across agents, MCP within each agent*.

**A2A's pitch (verbatim):**
- *"Connect agents built on different platforms (LangGraph, CrewAI, Semantic Kernel, custom solutions) to create powerful, composite AI systems."*
- *"Agents interact without needing to share internal memory, tools, or proprietary logic, ensuring security and preserving intellectual property."*

**What A2A claims that MCP doesn't (and where AGH Network can position):**
1. **Peer collaboration vs client-server tool access.** MCP is host→server (the agent calls a tool). A2A is peer↔peer (agents delegate, exchange goals). This is a fundamentally different communication shape.
2. **Discovery via "Agent Cards"** — A2A defines *agent cards* (structured capability advertisements). MCP has *tool schemas*. The card = identity + capabilities + endpoints, which lets agents *find* each other; tool schemas don't.
3. **Stateful, multi-turn delegation** — A2A models long-running task delegation natively (streaming, async). MCP is fundamentally request/response RPC over JSON-RPC.
4. **Boundary preservation** — agents keep their internal memory and tools private. MCP requires you expose tools to the host.

**Tension AGH should be aware of:**
The honest skepticism in the discourse: *"In theory they can coexist, in practice I foresee a tug of war. Developers can only invest their energy into so many ecosystems."* (Koyeb, multiple analyses.) Also: *"the difference between agents and tools is vanishing — tools can be implemented with AI agents."*

**Strategic takeaway for AGH Network:**
A2A is the macro-trend AGH rides. **Lead with the *agent-to-agent* primitive, treat MCP as table stakes, and own the runtime + protocol-coship narrative**. AGH's edge isn't "yet another protocol" — it's that the runtime, network, observability, and SDK all ship together and are local-first.

---

## Memory / sleep-time / consolidation discourse

**The conversation in 2025-2026:**

- The phrase **"stateless LLM"** is now the canonical antagonist. Letta opens with: *"despite their impressive capabilities, today's language models are fundamentally stateless"*.
- **"Sleep-time compute"** is a hot, recent term (Letta, 2025). It refers to *background* memory consolidation that runs while the foreground agent is idle, producing cleaner long-term memory. It maps cleanly to AGH's "dream/consolidation" feature.
- **MemGPT-style hierarchy** ("RAM vs disk") is the dominant mental model — context window = RAM, archival/recall = disk, with the agent calling explicit tools to swap.
- **Letta V1** deprecated heartbeats and `send_message` in favor of native reasoning + assistant-message generation; this is meaningful because it shows the *first wave* of memory-agent designs is already being torn out and rewritten — exactly the "harnesses must be lightweight, ready to be ripped out" lesson.
- **What gets engagement on social/HN/Twitter:**
  - "Agents that remember you across sessions" (concrete user benefit).
  - "Memory drift" (Addy Osmani: *"Over many context windows, agents drift. The original goal gets summarized, then re-summarized, then loses fidelity."*).
  - "Self-verification failure" (*"models reliably skew positive when they grade their own work"*).
  - "Continual learning in token space" (Letta blog post).
- **Episodic vs semantic vs procedural** taxonomy is *technical* — used in papers and infra blogs, but **doesn't show up in marketing copy**. Letta's marketing reduces it to *"persistent agents that learn from experience"*. AGH should keep the taxonomy in docs and reach for plainer marketing language up top.

**Phrases that resonate (verbatim from the wild):**
- *"Memory-first agents that learn"* (Letta)
- *"Background memory subagents"* (Letta)
- *"Own your memory and port it across models"* (Letta)
- *"Continuously learn and adapt from their own experience"* (Letta)
- *"Agents that compound through use"* (Hermes / Nous Research)

---

## Audience pain points (with sources)

Real complaints from the developer audience AGH targets — quoted verbatim:

### "Sick of AI Agent Frameworks" — HN, Jan 2025 (item 42691946)
- bsenftner: *"These 'frameworks' are useless, and as you say you can do what they offer in a few hours and better."*
- lunarcave: *"Most 'agent frameworks' are just workflow builders with LLMs as a node in the workflow."*
- thiago_fm: *"Agents are all useless at the moment for me. I couldn't find a single good use to assist me in coding."*

### Reliability / "lazy model" regressions
From Anthropic's own GitHub issues (`claude-code#42796`):
> *"When thinking is shallow, the model defaults to the cheapest action available — edit without reading, stop without finishing, dodge responsibility for failures, take the simplest fix rather than the correct one."*
> Users have built **programmatic stop hooks** to force continuation. The hook's existence is itself the symptom.

### Context management is a manual chore
- Best-practice sweet spot for `CLAUDE.md` is *"100–200 lines maximum"* — meaning developers are hand-curating context to avoid drift.
- Subdirectory `CLAUDE.md` files exist *to prevent context overflow and cross-bleed*.
- *"Context window management is where agent harnesses earn their keep."* (Jonathan Fulton, Apr 2026.)

### Observability is missing in native tooling
- Whole ecosystems exist *because* native is insufficient: **codeburn**, **tokentap**, **claude-code-hooks-multi-agent-observability**.
- Quote: *"I often use background agents for observability (monitoring log / error) purposes."* — devs are *abusing* background agents to plug an observability hole.
- Academic critique: there is an *"observability-evaluation gap"* in current agent harnesses. (arxiv 2604.14228)

### Vendor lock-in fear
> *"Vendor lock will force you to choose one provider and stick with it. I'd be happy to use different models to always be on the frontier, but due to the efforts of large corporations that want to vendor-lock you to one provider [...] I can't do that."*
(zero8.dev *State of Agentic Harnesses*, March 2026.)

### Bench reality check
- DoltHub *"Coding Agents Suck Too"* (Apr 2025): *"the original author generated 30k lines of toy code and then threw it all away — suggesting we're not getting mech-suits, we're just getting cosplay."*
- HN October 2025 thread title: *"Why the push for Agentic when models can barely follow a simple instruction?"*
- 2025 DORA Report: *"AI is an amplifier, not a fix"* — orgs with poor practices see bottlenecks amplified.

### Microservices / distributed-systems blind spot
> *"Coding Agents Suck at Microservices"* — agents can't reason about side effects across multiple services without massive context windows and better architectural understanding. (Late 2025 HN trending piece.)

### Cost surprise / pricing transparency
- RedMonk #3: *"Predictable Pricing — Clear cost transparency showing token usage per prompt, session costs, and spending limits upfront to avoid surprise overages."*
- Cursor's FastRender disclosed *"trillions of tokens (millions of dollars)"* to produce 1M lines — devs notice.

### Empirical bug study (arxiv 2603.20847 — Claude Code, Codex, Gemini CLI bug analysis, ~3.8k reports)
- API errors **18.3%**, terminal problems **14%**, command failures **12.7%**.
- Tool invocation **37.6%** and command execution **25%** are the most bug-prone stages.

---

## Strategic recommendation for AGH

### 1. Lead with the *agent-to-agent network* — it is the only thing in this space genuinely under-claimed by competitors.

Every competitor leads with **"build agents"** or **"AI software engineer"** or **"the harness"**. Nobody — *not Letta, not Goose, not OpenHands, not Cursor, not Devin* — leads with **agents that talk to each other**. The closest is A2A itself (*"seamless communication and collaboration between AI agents"*) but A2A is a protocol spec, not a product. AGH owns this lane the moment it claims it.

The previously-placeholder hero — *"Your agents can finally talk to each other"* (now relocked to *"An open workplace for AI agents"*) — was **strong** because:
- It names a frustration ("finally") without being whiny.
- It implies the listener already has agents (Claude Code, Codex, Cursor users — our audience).
- It is concrete, not adverbial.
- It does not contain a single burned-out word.

We should keep this as the hero or a near variant. Avoid swapping it for "agent OS" or "agent harness" — both are crowded.

### 2. Position AGH as the *runtime that ships the protocol* — runtime moat, not protocol moat.

CLAUDE.md already encodes this: *"AGH competes on runtime, SDK, observability, DX, and integration depth — NOT the wire protocol."* The site copy must mirror that. The protocol is *open and implementable outside AGH*; the daemon, persistence, observability, autonomy kernel, and SDK are what users come for. Sub-headline candidate direction: *"A local-first agent runtime that speaks the open AGH Network protocol — so any agent, in any harness, on any machine, can collaborate."*

This neutralizes the "yet another agent framework" objection (HN's #1 complaint) by *not* being a framework — it's a runtime + protocol pair that hosts whatever harness the developer is already using (Claude Code, Codex, Gemini CLI).

### 3. Earn every claim with concrete proof — no `AI-powered`, no `supercharge`, no `production-ready`.

Every value-prop block needs a specific noun:
- Not "persistent memory" → **"SQLite-backed event log; auditable, version-controllable, portable across model providers."**
- Not "observability" → **"Every ACP message, every tool call, every hook event — queryable from CLI, web, and API."**
- Not "autonomous" → **"Long-running tasks survive restarts; consolidation runs while you sleep."**
- Not "local-first" → **"Single Go binary. Your data. Your machine. Your network."**

The wins-list (RedMonk top 10, Letta concrete language, Goose's "70+ extensions / 15+ providers" specificity) shows this is what lands.

### 4. Treat MCP as table stakes; lead with what's *next* (A2A / network / consolidation / dream).

**Mention MCP support in the feature grid, not the hero.** Every competitor supports MCP — claiming it as a differentiator now signals "we're catching up". Instead, lead with the things competitors *don't* have or claim awkwardly: agent-to-agent network, dream/sleep-time consolidation, single-binary daemon, hooks pipeline, autonomy kernel.

### 5. Lean into pain-point counter-positioning, not generic feature lists.

Each section of the site can mirror a real pain quoted in this analysis:
- "Sick of AI agent frameworks?" → **"AGH is a runtime, not a framework. Bring your own harness."**
- "Models stop without finishing?" → **"The autonomy kernel decides when work is actually done — not the model."**
- "Where did my tokens go?" → **"Every event, every cost, every artifact, in one observable timeline."**
- "Vendor lock-in?" → **"Local binary. Open protocol. Any model, any harness."**
- "Memory drift across sessions?" → **"Memory consolidation runs in the background. Agents wake up smarter."**

This pattern (state-the-pain → state-the-fix) is what Letta does well, what Cursor does well, and what generic CrewAI/AutoGen do badly.

---

## Sources

Competitor pages:
- [cognition.ai](https://www.cognition.ai/) · [devin.ai](https://devin.ai/) · [cursor.com](https://www.cursor.com/) · [claude-code](https://claude.com/product/claude-code) · [Codex CLI](https://github.com/openai/codex) · [openhands.dev](https://docs.openhands.dev/) · [OpenHands repo](https://github.com/All-Hands-AI/OpenHands) · [goose-docs.ai](https://goose-docs.ai/) · [aider.chat](https://aider.chat/) · [LangGraph](https://www.langchain.com/langgraph) · [CrewAI](https://www.crewai.com/) · [MetaGPT](https://github.com/geekan/MetaGPT) · [AutoGen](https://github.com/microsoft/autogen) · [Letta](https://www.letta.com/)

Protocol pages:
- [Model Context Protocol](https://modelcontextprotocol.io/) · [A2A Protocol](https://a2a-protocol.org/latest/)

Discourse / market analysis:
- [Aakash Gupta — 2025 was Agents, 2026 is Agent Harnesses](https://aakashgupta.medium.com/2025-was-agents-2026-is-agent-harnesses-heres-why-that-changes-everything-073e9877655e)
- [Phil Schmid — The importance of Agent Harness in 2026](https://www.philschmid.de/agent-harness-2026)
- [Jonathan Fulton — Inside the Agent Harness](https://medium.com/jonathans-musings/inside-the-agent-harness-how-codex-and-claude-code-actually-work-63593e26c176)
- [zero8.dev — State of Agentic Harnesses March 2026](https://zero8.dev/blog/state-of-agentic-harnesses-march-2026)
- [awesome-harness-engineering](https://github.com/ai-boost/awesome-harness-engineering)
- [Cursor — Long-Running Agents](https://cursor.com/blog/long-running-agents)
- [Addy Osmani — Long-running Agents](https://addyosmani.com/blog/long-running-agents/)
- [RedMonk — 10 Things Developers Want from Agentic IDEs in 2025](https://redmonk.com/kholterhoff/2025/12/22/10-things-developers-want-from-their-agentic-ides-in-2025/)
- [MindStudio — Codex vs Claude Code 2026](https://www.mindstudio.ai/blog/codex-vs-claude-code-2026)

Memory:
- [Letta — Agent Memory](https://www.letta.com/blog/agent-memory)
- [Letta — Sleep-time Compute](https://www.letta.com/blog/sleep-time-compute)
- [Letta — Continual Learning in Token Space](https://www.letta.com/blog/continual-learning)
- [Letta V1 Agent Loop](https://www.letta.com/blog/letta-v1-agent)

A2A vs MCP:
- [Auth0 — MCP vs A2A](https://auth0.com/blog/mcp-vs-a2a/)
- [Koyeb — A2A and MCP: Start of the AI Agent Protocol Wars?](https://www.koyeb.com/blog/a2a-and-mcp-start-of-the-ai-agent-protocol-wars)
- [DigitalOcean — A2A vs MCP](https://www.digitalocean.com/community/tutorials/a2a-vs-mcp-ai-agent-protocols)
- [Niklas Heidloff — Comparison of MCP, ACP, and A2A](https://heidloff.net/article/mcp-acp-a2a-agent-protocols/)

Pain points:
- [HN — Sick of AI Agent Frameworks](https://news.ycombinator.com/item?id=42691946)
- [HN — Why the push for Agentic when models can barely follow a simple instruction?](https://news.ycombinator.com/item?id=45577080)
- [HN — AI agents break rules under everyday pressure](https://news.ycombinator.com/item?id=46067995)
- [DoltHub — Coding Agents Suck Too](https://www.dolthub.com/blog/2025-04-23-coding-agents-suck-too/)
- [arxiv 2603.20847 — Engineering Pitfalls in AI Coding Tools](https://arxiv.org/html/2603.20847)
- [arxiv 2604.14228 — Dive into Claude Code](https://arxiv.org/html/2604.14228v1)
- [Claude Code Issue #42796](https://github.com/anthropics/claude-code/issues/42796)

Local-first / self-hosted:
- [Fastio — Top 10 Open Source AI Agents](https://fast.io/resources/top-10-open-source-ai-agents/)
- [Fastio — 8 Best Self-Hosted AI Agent Platforms](https://fast.io/resources/best-self-hosted-ai-agent-platforms/)
- [Programming Insider — 5 Reasons Devs Build With Local-First AI Agents in 2026](https://programminginsider.com/5-reasons-developers-are-building-with-local-first-ai-agents-in-2026/)
- [Lushbinary — Hermes Agent Developer Guide](https://lushbinary.com/blog/hermes-agent-developer-guide-setup-skills-self-improving-ai/)
- [Cloudflare — Moltworker](https://blog.cloudflare.com/moltworker-self-hosted-ai-agent/)

Cliché / overused vocabulary:
- [Copy.ai — AI for Marketing](https://www.copy.ai/blog/ai-for-marketing)
- [Jasper — AI Marketing Resource Hub](https://www.jasper.ai/blog)
- [Optiminastic — Supercharge Brand Growth with AI](https://www.optiminastic.com/blogs/Supercharge-Brand-Growth-with-AI-Driven-Marketing-Strategies-in-2025)
- [Superside — AI-Powered Agencies](https://www.superside.com/blog/ai-powered-agencies)
