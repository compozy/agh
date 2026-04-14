# Claude Code Extensibility Ecosystem -- Research Analysis for AGH

**Date:** 2026-04-11
**Scope:** MCP servers, hooks, skills/commands, plugins, Agent SDK patterns, CLAUDE.md conventions, workflow automations
**Purpose:** Identify concrete extension ideas adaptable to AGH's three-dimensional extension model (Resources, Capabilities, Actions)

---

## Overview of Findings

Claude Code has evolved from a standalone CLI agent into a full extensible platform with a maturing plugin marketplace (101 official plugins, thousands of community skills). The ecosystem is organized around five extension axes:

1. **MCP Servers** -- external tool connections via the Model Context Protocol (3,000+ integrations, 251+ vendor-verified)
2. **Hooks** -- lifecycle event callbacks (12 events: PreToolUse, PostToolUse, UserPromptSubmit, Stop, SessionStart, Notification, Setup, Elicitation, ElicitationResult, PostCompact, PermissionDenied, PostToolUseFailure)
3. **Skills & Commands** -- reusable slash-command instructions (`.claude/skills/*/SKILL.md` or `.claude/commands/*.md`)
4. **Plugins** -- bundled packages of skills + hooks + MCP servers + commands (official marketplace with `claude-plugins-official` repo)
5. **Agent SDK** -- programmatic agent building in Python/TypeScript with subagent orchestration, hooks, and tool control

The most impactful patterns for AGH are: hook-based policy enforcement, MCP-driven tool federation, skill-as-instruction files, multi-agent delegation, and classifier-based permission gating.

---

## Extension Catalog

### MCP Servers (Resources: MCP)

| Name                        | Category      | Description                                                               | AGH Mapping                                                                           |
| --------------------------- | ------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| **GitHub MCP**              | DevOps        | Full repo management: PRs, issues, code search, branches, commits via API | Resource: MCP server; Capability: agent.driver integration for PR-driven workflows    |
| **Filesystem MCP**          | Core          | Read/write/organize local files with configurable access boundaries       | Resource: MCP server (AGH already has file access; useful as a sandboxed alternative) |
| **PostgreSQL MCP**          | Database      | Natural language database queries and schema inspection                   | Resource: MCP server; Capability: memory.backend alternative                          |
| **Playwright MCP**          | Testing       | Browser automation, E2E testing, screenshot capture, UI interaction       | Resource: MCP server; Action: session-level test execution                            |
| **Memory MCP**              | Persistence   | Persistent knowledge graph across sessions                                | Capability: memory.backend; maps directly to AGH's memory layer                       |
| **Notion MCP**              | Productivity  | Read/write Notion pages, databases, blocks                                | Resource: MCP server for knowledge management                                         |
| **Figma MCP**               | Design        | Read Figma frames/components, design-to-code pipeline                     | Resource: MCP server; Capability: prompt.provider (design context)                    |
| **Brave Search MCP**        | Research      | Privacy-first web search with source citation                             | Resource: MCP server; Action: observe queries                                         |
| **Supabase MCP**            | Backend       | Database, auth, edge functions, storage integration                       | Resource: MCP server                                                                  |
| **Sequential Thinking MCP** | Reasoning     | Enhanced problem-solving via structured thinking steps                    | Capability: message.transform (reasoning augmentation)                                |
| **Sentry MCP**              | Monitoring    | Real-time error fetching, debugging, report creation                      | Resource: MCP server; Capability: observe.exporter                                    |
| **Linear MCP**              | Project Mgmt  | Create/update/query issues, sprint management                             | Resource: MCP server                                                                  |
| **Slack MCP**               | Communication | Send messages, search history, manage channels                            | Resource: MCP server; Action: notification fan-out                                    |
| **Jira MCP**                | Project Mgmt  | JQL search, status transitions, comments, ticket creation                 | Resource: MCP server                                                                  |
| **Neon MCP**                | Database      | Serverless Postgres with branching, migrations, query tuning              | Resource: MCP server; Capability: memory.backend                                      |

### Hooks (Capabilities: permission.gate, content.validate, message.transform)

| Hook / Pattern                    | Category      | Description                                                                                    | AGH Mapping                                |
| --------------------------------- | ------------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------ |
| **Dangerous command blocker**     | Security      | PreToolUse on Bash: block `rm -rf`, `DROP TABLE`, force-push commands                          | Capability: permission.gate                |
| **Sensitive file protector**      | Security      | PreToolUse on Edit/Write: block changes to `.env`, `package-lock.json`, `.git/`                | Capability: permission.gate                |
| **Auto-formatter**                | Quality       | PostToolUse on Edit: run Prettier/Black/gofmt after every file edit                            | Capability: content.validate (post-action) |
| **Auto-test runner**              | Quality       | PostToolUse on Edit: run test suite on modified files for instant regression feedback          | Capability: content.validate               |
| **Auto-commit agent work**        | DevOps        | PostToolUse on Edit: create micro-commits to track agent changes                               | Action: session event recording            |
| **Prompt logger**                 | Observability | UserPromptSubmit: log every prompt with timestamp to audit file                                | Capability: observe.exporter               |
| **Context injector**              | Augmentation  | UserPromptSubmit: inject project context, environment info, or relevant docs before processing | Capability: prompt.provider                |
| **Tool input modifier**           | Transform     | PreToolUse (v2.0.10+): transparently modify tool inputs (add dry-run flags, redact secrets)    | Capability: message.transform              |
| **Permission classifier**         | Gating        | Transcript classifier (Sonnet 4.6) evaluates tool calls against natural-language rules         | Capability: permission.gate (AI-based)     |
| **PermissionDenied retry**        | Recovery      | PermissionDenied hook: retry with modified parameters or defer decision                        | Capability: permission.gate                |
| **PostCompact context preserver** | Memory        | PostCompact: ensure critical context survives summarization                                    | Capability: memory.backend                 |
| **Setup initialization**          | Lifecycle     | Setup hook: run maintenance scripts, environment checks on session init                        | Action: session lifecycle                  |

### Skills & Commands (Resources: skills)

| Skill / Command                 | Category    | Description                                                                                   | AGH Mapping                                        |
| ------------------------------- | ----------- | --------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **Frontend Design (Anthropic)** | UI          | Design system + philosophy injection; bold aesthetics, typography, animations (277K installs) | Resource: skill; Capability: prompt.provider       |
| **Taste**                       | UI          | Collection improving AI frontend code quality (6.9K stars)                                    | Resource: skill                                    |
| **Apple HIG Designer**          | UI          | Interfaces following Apple Human Interface Guidelines                                         | Resource: skill; Capability: prompt.provider       |
| **Shannon (AI Pen Testing)**    | Security    | Autonomous pen testing, 96% exploit success rate, 50+ vulnerability types                     | Resource: skill; Capability: content.validate      |
| **VibeSec**                     | Security    | Secure code patterns and vulnerability prevention (496 stars)                                 | Resource: skill                                    |
| **Skill-Threat-Modeling**       | Security    | STRIDE threat modeling and security review workflows                                          | Resource: skill                                    |
| **Code Review**                 | Quality     | Structured review: security, performance, style violations                                    | Resource: skill; Capability: content.validate      |
| **Test Planner/Executor**       | Testing     | Risk-based test scenario creation (E2E, integration, unit) + execution                        | Resource: skill                                    |
| **Commit Helper**               | DevOps      | Conventional commits, co-author tags, force-push prevention                                   | Resource: skill; hook integration                  |
| **Project Bootstrap**           | Scaffolding | New project scaffolding with preferred stack, linting, CI config                              | Resource: skill                                    |
| **cc-devops-skills**            | DevOps      | Comprehensive DevOps skill set: deploy, infrastructure, monitoring                            | Resource: skill                                    |
| **Ship command**                | Workflow    | Review diff, run tests, commit, push -- all in one `/ship`                                    | Resource: skill (compound workflow)                |
| **Valyu (Research)**            | Research    | Web search + 36 specialized data sources (SEC, PubMed, FRED, etc.)                            | Resource: MCP + skill; Capability: prompt.provider |

### Plugins (Resources: bundled packages)

| Plugin                            | Category | Description                                                                                       | AGH Mapping                                                                  |
| --------------------------------- | -------- | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| **Security Guidance (Anthropic)** | Security | Scans file edits for vulnerabilities before execution; blocks + explains                          | Resource: hook + skill bundle; Capability: content.validate, permission.gate |
| **Local-Review**                  | Quality  | 5 parallel review agents, scores issues, only flags 80+ severity                                  | Resource: agent orchestration; Action: multi-session coordination            |
| **Superpowers**                   | Workflow | Structured lifecycle planning + skills for brainstorming, TDD, debugging, review                  | Resource: skill bundle                                                       |
| **Shipyard**                      | DevOps   | Lifecycle mgmt + IaC validation (Terraform, Ansible, Docker, K8s, CloudFormation) + auditor agent | Resource: skill + agent bundle                                               |
| **Claude-Mem**                    | Memory   | Action capture, compression, context injection via SQLite + Chroma vector search                  | Capability: memory.backend; Resource: MCP                                    |
| **Ralph Wiggum**                  | Testing  | Visual testing by driving Xcode simulator for Swift apps                                          | Resource: skill + MCP                                                        |
| **Figma Plugin**                  | Design   | Read Figma files, generate code from frames/components                                            | Resource: MCP + skill bundle                                                 |
| **Language Servers (12)**         | IDE      | Real-time code intelligence for specific programming languages                                    | Capability: prompt.provider (context enrichment)                             |
| **Feature-dev (Anthropic)**       | Workflow | Guided feature development workflow                                                               | Resource: skill                                                              |
| **Commit-commands (Anthropic)**   | DevOps   | Standardized commit workflows                                                                     | Resource: skill                                                              |

### Agent SDK Patterns (Actions: Host API)

| Pattern                        | Category      | Description                                                                              | AGH Mapping                                                          |
| ------------------------------ | ------------- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **Subagent delegation**        | Orchestration | Spawn specialized child agents with own context window and tool access                   | Action: session spawning via Host API                                |
| **Explore-Plan-Act**           | Workflow      | Sequential three-phase loop with escalating permissions                                  | Action: session state machine (maps to AGH session lifecycle)        |
| **Operator/Orchestrator**      | Coordination  | Central agent decomposes tasks, delegates to specialized sub-agents, synthesizes results | Action: multi-session coordination via Host API                      |
| **Split-and-Merge**            | Parallelism   | Multiple agents in isolated git worktrees working in parallel, merge results             | Action: parallel session management                                  |
| **Custom agents via Markdown** | Configuration | `.claude/agents/*.md` with YAML frontmatter defining name, tools, model, system prompt   | Resource: agent definition (maps directly to AGH agent config)       |
| **Research pipeline**          | Workflow      | Explore subagents gather info, then act on aggregated results                            | Action: session chaining                                             |
| **Tool allowlist/blocklist**   | Security      | `allowed_tools` / `disallowed_tools` for fine-grained tool access per agent              | Capability: permission.gate                                          |
| **Context compaction**         | Memory        | Auto-summarize when context limit approaches, preserve critical info                     | Capability: memory.backend; Action: observe (context health metrics) |

### CLAUDE.md / Configuration Patterns (Resources: skills; Capability: prompt.provider)

| Pattern                       | Category   | Description                                                                             | AGH Mapping                                                  |
| ----------------------------- | ---------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| **Hierarchical config files** | Config     | Root CLAUDE.md + subdirectory overrides; auto-loaded based on working context           | Resource: skill loading by workspace path                    |
| **Path-scoped rules**         | Config     | YAML frontmatter restricts rule activation to matching directories                      | Capability: prompt.provider (context-aware)                  |
| **Compaction instructions**   | Memory     | "When compacting, always preserve X" directives in CLAUDE.md                            | Capability: memory.backend (consolidation rules)             |
| **Auto-memory (MEMORY.md)**   | Memory     | Agent auto-detects patterns and writes own notes (v2.1.32+)                             | Capability: memory.backend (maps to AGH dream/consolidation) |
| **Hook-enforced rules**       | Governance | Critical rules as hooks (100% enforcement) vs. CLAUDE.md instructions (~70% compliance) | Capability: permission.gate vs. prompt.provider              |
| **Custom command files**      | Workflow   | `.claude/commands/*.md` becoming slash commands with shell execution                    | Resource: skill with action execution                        |

---

## Detailed Analysis of High-Impact Extensions

### 1. Hook-Based Policy Enforcement (PreToolUse)

**What it does:** Intercepts every tool call before execution. Inspects the tool name, arguments, and context. Can approve, deny (exit code 2), or modify the call. The most powerful control mechanism in Claude Code.

**Why it matters:** CLAUDE.md instructions achieve ~70% compliance. Hooks achieve 100%. For security-critical rules (no force push, no production data deletion, no secrets in commits), this gap is unacceptable.

**AGH mapping:** This maps directly to AGH's `permission.gate` capability. AGH should implement a PreToolUse hook system where:

- Hooks are registered per-agent or per-workspace
- Each hook receives the tool call as structured input (tool name, arguments, session context)
- Hooks return allow/deny/modify decisions
- Hooks can be shell scripts, Go plugins, or HTTP endpoints
- Multiple hooks chain with configurable precedence

**Key insight from Claude Code:** The three-tier handler system (Command hooks for simple checks, Prompt hooks for semantic evaluation, Agent hooks for deep analysis) is a powerful graduated model. AGH could adopt this with shell-based hooks for speed and agent-based hooks for complex policy decisions.

### 2. MCP Server Federation

**What it does:** Connects Claude Code to 3,000+ external tools via a standardized protocol. Each MCP server is a subprocess exposing tools, resources, and prompts over JSON-RPC. Claude Code discovers tools on-demand via Tool Search (lazy loading), reducing context consumption by ~95%.

**Why it matters:** No single agent can have all tools built in. MCP makes the tool surface area effectively infinite while keeping the runtime lean.

**AGH mapping:** AGH already supports MCP as a resource type. Key lessons from Claude Code's implementation:

- **Lazy tool discovery** is essential at scale (10+ servers). AGH should implement tool search / on-demand schema loading rather than dumping all tool definitions into agent context.
- **Three scope levels** (user/local/project) map to AGH's global/workspace scoping. Add a `.mcp.json` project-level config for team-shared MCP servers.
- **Skills + MCP composition**: Claude Code skills can orchestrate MCP tools into workflows. AGH's skill system should support MCP tool references in skill definitions.

### 3. Skills as Instruction Files

**What it does:** A `SKILL.md` file with YAML frontmatter (name, description, trigger conditions) + markdown body (instructions Claude follows). No compilation, no build step. Skills load on-demand via slash commands or auto-detection based on task context.

**Why it matters:** This is the lowest-friction extension mechanism. Anyone who can write markdown can create a skill. It democratizes agent customization.

**AGH mapping:** AGH's skill system should adopt this pattern:

- Skills are markdown files with frontmatter metadata
- Stored in `~/.agh/skills/` (global) or `.agh/skills/` (workspace)
- Auto-discovered and lazy-loaded based on task context
- Can reference other skills, MCP tools, and hooks
- Budget-capped (1% of context window, ~8K chars fallback) to prevent context bloat
- Keep skills under 500 words / 2K tokens for optimal performance

### 4. Multi-Agent Orchestration (Subagents)

**What it does:** The operator pattern decomposes complex tasks and delegates to specialized sub-agents, each with their own context window, tool access, and instructions. Sub-agents can run in parallel in isolated git worktrees.

**Why it matters:** Single-agent context windows are finite. Complex tasks (refactor + test + review + deploy) benefit from specialized agents that don't pollute each other's context.

**AGH mapping:** This is core to AGH's architecture. Key patterns to adopt:

- **Custom agents via markdown** (`.claude/agents/*.md`): AGH already has agent definitions in TOML config. Extend to support workspace-level agent definitions in markdown for quick customization.
- **Split-and-merge in worktrees**: AGH should support spawning sessions in isolated worktrees with automatic branch management and merge coordination.
- **Explore-Plan-Act lifecycle**: Map to AGH's session state machine. Three phases with escalating tool permissions.

### 5. Classifier-Based Permission Gating

**What it does:** A fast AI classifier (running on a smaller model) evaluates each tool call against natural-language rules before execution. Two-stage: fast single-token filter, then chain-of-thought only if flagged. Rules are written in prose, not regex.

**Why it matters:** Traditional permission systems use regex or glob patterns. Prose rules ("don't modify infrastructure files unless the user explicitly asked for infrastructure changes") capture intent that patterns cannot.

**AGH mapping:** This is a sophisticated `permission.gate` capability:

- Use a smaller/faster model as a classifier for tool call evaluation
- Rules defined in natural language in config
- Two-stage evaluation for performance (fast filter + deep reasoning)
- Configurable per-agent and per-workspace
- Precedence: deny rules > allow exceptions > explicit user intent

### 6. Plugin Marketplace Model

**What it does:** Plugins bundle skills + hooks + MCP servers + commands into installable packages. Official marketplace (`claude-plugins-official`) with 101 plugins, plus community marketplaces. Install via `/plugin install name@registry`.

**Why it matters:** Individual skills and hooks are useful but fragmented. Plugins provide complete, tested workflows. The marketplace model enables distribution and discovery.

**AGH mapping:** AGH should plan for a plugin/extension registry:

- Extensions bundle: agent definitions, skills, hooks, MCP server configs
- Registry format: Git repos with standardized manifest files
- Install via CLI: `agh plugin install name@registry`
- Scope control: user-level vs. workspace-level installation
- Enterprise: managed registries with approval workflows

### 7. Auto-Memory and Dream Consolidation

**What it does:** Claude Code v2.1.32 auto-generates MEMORY.md by observing user patterns, preferences, and project conventions. This is separate from CLAUDE.md (human-written project docs). The Claude-Mem plugin adds SQLite + Chroma vector search for hybrid memory retrieval.

**Why it matters:** Memory that builds itself from observation is more complete and current than manually maintained docs. Vector search enables semantic retrieval of relevant context.

**AGH mapping:** This maps directly to AGH's memory and dream consolidation layers:

- Auto-memory: AGH's observe layer already captures events. The consolidation/dream system should synthesize these into persistent memory entries.
- Dual-scope: global memory (user preferences) + workspace memory (project conventions) -- AGH already has this.
- Hybrid retrieval: keyword + vector search over consolidated memories.
- Compaction rules: configurable instructions for what to preserve during context compaction.

---

## Key Takeaways for AGH Extension Ideas

### High-Priority Extensions to Build

1. **Hook pipeline with PreToolUse/PostToolUse** -- The single highest-impact extension mechanism. Three handler tiers (command/prompt/agent) provide graduated complexity. Essential for permission.gate and content.validate capabilities.

2. **Lazy MCP tool discovery** -- As AGH connects to more MCP servers, eager tool loading will bloat agent context. Implement on-demand tool search and schema fetching.

3. **Skill files with auto-discovery** -- Markdown-based skill definitions with YAML frontmatter. Lowest friction for users. Budget-capped context injection.

4. **Permission classifier** -- AI-based tool call evaluation using natural-language rules. More expressive than regex patterns. Essential for autonomous agent operation.

5. **Plugin bundling format** -- Define a standard for packaging skills + hooks + MCP configs + agent definitions as installable extensions.

### Medium-Priority Extensions

6. **Subagent orchestration with worktree isolation** -- Spawn parallel agents in isolated git worktrees. Operator pattern for complex multi-phase tasks.

7. **Auto-memory from observation** -- Agent-generated memory entries from event stream analysis, distinct from human-configured project docs.

8. **Hierarchical config with path scoping** -- Config files that activate only when the agent works in matching directories.

9. **PostCompact hooks** -- Ensure critical context survives memory consolidation.

10. **CI/CD integration actions** -- GitHub Actions / GitLab CI integration for automated code review, security audit, release notes.

### Design Principles Learned

- **Deterministic enforcement via hooks, not instructions.** Instructions are probabilistic (~70%). Hooks are deterministic (100%). Use hooks for must-enforce rules, instructions for should-follow guidance.
- **Lazy loading is essential at scale.** Claude Code's Tool Search pattern (95% context reduction) is critical when connecting 10+ MCP servers.
- **Prose rules beat regex for intent.** Permission rules written as natural language capture nuance that glob patterns cannot.
- **Skills should be small.** Under 500 words / 2K tokens. Focused on one workflow. Include examples for better accuracy.
- **Three scope levels** (user/workspace/project-shared) cover all organizational needs.
- **Plugins are the distribution unit.** Individual skills/hooks are building blocks; plugins are the installable product.

### AGH Extension Model Mapping Summary

| Claude Code Concept        | AGH Dimension | AGH Component                            |
| -------------------------- | ------------- | ---------------------------------------- |
| MCP Server                 | Resource      | MCP (already supported)                  |
| Skill / Command            | Resource      | Skills (already supported)               |
| Hook (PreToolUse)          | Capability    | permission.gate, content.validate        |
| Hook (PostToolUse)         | Capability    | content.validate, observe.exporter       |
| Hook (UserPromptSubmit)    | Capability    | prompt.provider, message.transform       |
| Hook (tool input modifier) | Capability    | message.transform                        |
| Permission classifier      | Capability    | permission.gate (AI-based)               |
| Auto-memory / MEMORY.md    | Capability    | memory.backend (dream consolidation)     |
| Agent definition (.md)     | Resource      | Agent (extend TOML config with markdown) |
| Plugin bundle              | Resource      | New: composite extension package         |
| Subagent delegation        | Action        | Host API: session spawning               |
| Operator pattern           | Action        | Host API: multi-session coordination     |
| Split-and-merge            | Action        | Host API: parallel session management    |
| Context compaction         | Action        | Observe: context health metrics + memory |
| Tool Search                | Action        | Host API: lazy MCP tool discovery        |

---

## Sources

- [Hooks reference - Claude Code Docs](https://code.claude.com/docs/en/hooks)
- [Extend Claude with skills - Claude Code Docs](https://code.claude.com/docs/en/skills)
- [Connect Claude Code to tools via MCP - Claude Code Docs](https://code.claude.com/docs/en/mcp)
- [Configure permissions - Claude Code Docs](https://code.claude.com/docs/en/permissions)
- [Agent SDK overview - Claude Code Docs](https://code.claude.com/docs/en/agent-sdk/overview)
- [Best Practices for Claude Code - Claude Code Docs](https://code.claude.com/docs/en/best-practices)
- [Discover and install prebuilt plugins through marketplaces - Claude Code Docs](https://code.claude.com/docs/en/discover-plugins)
- [Claude Code auto mode - Anthropic](https://www.anthropic.com/engineering/claude-code-auto-mode)
- [Using CLAUDE.MD files - Claude Blog](https://claude.com/blog/using-claude-md-files)
- [Building agents with the Claude Agent SDK - Claude Blog](https://claude.com/blog/building-agents-with-the-claude-agent-sdk)
- [awesome-claude-code - GitHub (hesreallyhim)](https://github.com/hesreallyhim/awesome-claude-code)
- [awesome-mcp-servers - GitHub (wong2)](https://github.com/wong2/awesome-mcp-servers)
- [awesome-claude-code-toolkit - GitHub (rohitg00)](https://github.com/rohitg00/awesome-claude-code-toolkit)
- [awesome-agent-skills - GitHub (VoltAgent)](https://github.com/VoltAgent/awesome-agent-skills)
- [claude-plugins-official - GitHub (anthropics)](https://github.com/anthropics/claude-plugins-official)
- [claude-code-hooks-mastery - GitHub (disler)](https://github.com/disler/claude-code-hooks-mastery)
- [Claude Code Hooks Reference: All 12 Events - Pixelmojo](https://www.pixelmojo.io/blogs/claude-code-hooks-production-quality-ci-cd-patterns)
- [Claude Code hooks: A practical guide - eesel AI](https://www.eesel.ai/blog/hooks-in-claude-code)
- [Claude Code Hooks: A Practical Guide - DataCamp](https://www.datacamp.com/tutorial/claude-code-hooks)
- [Claude Code Hook Examples - Steve Kinney](https://stevekinney.com/courses/ai-development/claude-code-hook-examples)
- [CLAUDE.md best practices - DEV Community](https://dev.to/cleverhoods/claudemd-best-practices-from-basic-to-adaptive-9lm)
- [Claude Code Skills vs MCP Servers - DEV Community](https://dev.to/williamwangai/claude-code-skills-vs-mcp-servers-what-to-use-how-to-install-and-the-best-ones-in-2026-548k)
- [Best Claude Code Skills & Plugins 2026 - DEV Community](https://dev.to/raxxostudios/best-claude-code-skills-plugins-2026-guide-4ak4)
- [10 Must-Have Skills for Claude 2026 - Medium](https://medium.com/@unicodeveloper/10-must-have-skills-for-claude-and-any-coding-agent-in-2026-b5451b013051)
- [Claude Code 2.0.13 Plugin Marketplace - Medium](https://alirezarezvani.medium.com/claude-code-2-0-13-be2c0a723856)
- [The Complete Guide to Building Agents with the Claude Agent SDK - Nader Dabit](https://nader.substack.com/p/the-complete-guide-to-building-agents)
- [Top 10 MCP Servers for Claude Code - Apidog](https://apidog.com/blog/top-10-mcp-servers-for-claude-code/)
- [10 Must-Have MCP Servers for Claude Code - Medium](https://roobia.medium.com/the-10-must-have-mcp-servers-for-claude-code-2025-developer-edition-43dc3c15c887)
- [Piebald-AI/claude-code-system-prompts - GitHub](https://github.com/Piebald-AI/claude-code-system-prompts)
