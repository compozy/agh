# RFC: Self-Contained Agent Definitions with Scoped Skills and Memory

- **Status:** Draft
- **Authors:** AGH Core Team
- **Created:** 2026-04-06
- **Relates to:** AGENTS.md (agents.md), AgentSkills Specification (agentskills.io), MCP (modelcontextprotocol.io), A2A (a2a-protocol.org)

---

## Abstract

The AI agent ecosystem has converged on standards for project instructions (AGENTS.md), reusable workflow instructions (AgentSkills/SKILL.md), and tool integration (MCP). What remains unaddressed is **the agent itself** — there is no standard for defining an agent's identity, capabilities, permissions, skill set, memory, and lifecycle as a single, portable unit.

Today, an agent's definition is scattered: its prompt lives in one file, its skills in a global directory, its memories in another global store, its MCP servers in a platform config. Moving an agent between machines, projects, or teams means reassembling these pieces manually.

This RFC proposes a **self-contained agent definition format** where each agent is a directory containing an AGENT.md file (YAML frontmatter + Markdown prompt), a `skills/` subdirectory with agent-specific skills, and a `memory/` subdirectory with agent-scoped persistent context. The agent directory is the unit of portability — copy the directory, and the agent works.

---

## 1. Problem Statement

### 1.1 The Missing Layer: Agent Definition

The current standards landscape covers three layers:

```
┌─────────────────────────────────────┐
│  Project Instructions               │  AGENTS.md, CLAUDE.md, .cursorrules
│  "How to work in THIS codebase"     │  (project-scoped, always loaded)
├─────────────────────────────────────┤
│  Reusable Skills                    │  SKILL.md (AgentSkills spec)
│  "How to do a specific task"        │  (portable, loaded on demand)
├─────────────────────────────────────┤
│  Tool Integration                   │  MCP servers
│  "How to connect to external tools" │  (protocol-level, client-server)
├─────────────────────────────────────┤
│  Agent Definition                   │  ??? (no standard)
│  "What IS this agent"               │
└─────────────────────────────────────┘
```

AGENTS.md tells agents about the project. SKILL.md teaches agents how to do things. MCP gives agents access to tools. But nothing defines **the agent itself**: what model it uses, what tools it has access to, what permissions it operates under, what skills are exclusive to it, what it remembers across sessions.

### 1.2 Agents Are Not Portable

Consider a team that has built a specialized "code-reviewer" agent:

- Prompt tuned over weeks of iteration
- Three custom skills for security review, performance analysis, and team coding standards
- Memory of the team's review preferences ("keep comments short", "focus on error handling")
- MCP servers for GitHub PR access and Jira ticket context

To share this agent with another team member, you'd need to:

1. Copy the prompt file
2. Copy the skills to the right global/workspace directories
3. Export and import memories
4. Document which MCP servers to configure
5. Explain the permission model

This is fragile, error-prone, and doesn't scale. The agent's identity is scattered across the filesystem.

### 1.3 No Agent-Specific Specialization

In every current implementation, skills are global or project-scoped. All agents in a workspace see the same skill pool. There's no way to say "this debugging agent should have access to log-analysis skills, but the code-review agent should not." The closest analog is Cursor's `.mdc` rules with glob-based activation, but those are file-pattern-scoped, not agent-scoped.

This matters because agents with different roles need different capabilities. A security-focused reviewer shouldn't see deployment skills. A documentation writer shouldn't see database migration skills. Specialization reduces context noise and improves task accuracy — the ETH Zurich study (February 2026) found that including irrelevant context increased inference costs by 20%+ while reducing task success.

### 1.4 Memory Is Not Agent-Scoped

Memory systems in the ecosystem are either:

- **Global** (Claude Code's MEMORY.md, Windsurf's auto-memories): all agents share the same context
- **Session-scoped** (most implementations): context dies when the session ends
- **Proprietary** (Mem0, MemOS): framework-specific, not portable

None support agent-scoped memory — context that persists across sessions, belongs to a specific agent, and travels with the agent when it moves. A debugging agent's memory of past root causes is useless to a documentation agent, and vice versa.

### 1.5 Existing Approaches and Their Gaps

| Approach                  | Agent Definition                                      | Agent Skills           | Agent Memory                 | Portability                    |
| ------------------------- | ----------------------------------------------------- | ---------------------- | ---------------------------- | ------------------------------ |
| **AGENTS.md**             | No (project instructions only)                        | No                     | No                           | Yes (universal format)         |
| **CLAUDE.md**             | No (project instructions only)                        | Via AgentSkills        | Via MEMORY.md (global)       | No (Claude-only)               |
| **Claude Code subagents** | Partial (`.claude/agents/*.md` with frontmatter)      | Global pool only       | No agent memory              | No (Claude-only)               |
| **Codex plugins**         | No (plugins are skill bundles, not agent definitions) | Via plugins            | No                           | No (Codex-only)                |
| **Cursor rules**          | No (rules are instructions, not agent definitions)    | No                     | Auto-memories (not portable) | No (Cursor-only)               |
| **A2A Agent Cards**       | Partial (capability discovery only)                   | No                     | No                           | Partial (JSON, but no runtime) |
| **This proposal**         | **Yes** (full YAML + prompt)                          | **Yes** (agent-scoped) | **Yes** (agent-scoped)       | **Yes** (directory = agent)    |

---

## 2. Proposal

### 2.1 Agent as Directory

Each agent is a self-contained directory:

```
.agents/
  code-reviewer/
    AGENT.md                    # Agent definition (frontmatter + prompt)
    skills/                     # Agent-exclusive skills
      review-checklist/
        SKILL.md
      security-patterns/
        SKILL.md
    memory/                     # Agent-scoped persistent memory
      MEMORY.md                 # Memory index
      feedback_style.md
      project_context.md

  debugger/
    AGENT.md
    skills/
      systematic-debug/
        SKILL.md
      log-analysis/
        SKILL.md
    memory/
      MEMORY.md
      debug_patterns.md
```

Agent directories can live in:

- `~/.agh/agents/<name>/` — user-level agents (available everywhere)
- `<workspace>/.agents/<name>/` — project-level agents (shared via version control)

### 2.2 AGENT.md Format

```yaml
---
name: code-reviewer
description: Code review focused on security and quality
provider: claude
model: claude-sonnet-4-6
tools:
  - Read
  - Grep
  - Glob
permissions: plan
mcp_servers:
  - name: github
    command: npx
    args: ["@github/mcp-server"]
skills:
  inherit: true
  disabled:
    - agh-session-guide
  extra_sources:
    - ./shared-skills/
memory:
  inherit: true
  scope: agent
  auto_consolidate: true
---

You are a senior code reviewer. Your focus is:

1. Security (OWASP top 10)
2. Code quality (readability, maintainability)
3. Performance (hot paths, algorithmic complexity)

Use your internal skills to guide the review. Consult your memories
to understand team preferences and project context.
```

### 2.3 Frontmatter Schema

**Core fields** (existing AgentDef, unchanged):

| Field         | Type        | Required | Description                                          |
| ------------- | ----------- | -------- | ---------------------------------------------------- |
| `name`        | string      | Yes      | Agent identifier (lowercase, alphanumeric + hyphens) |
| `description` | string      | No       | One-line description for discovery and selection     |
| `provider`    | string      | Yes      | AI provider (claude, openai, gemini, etc.)           |
| `model`       | string      | No       | Model identifier (defaults to provider's default)    |
| `command`     | string      | No       | Custom command to spawn the agent subprocess         |
| `tools`       | []string    | No       | Tool allowlist                                       |
| `permissions` | string      | No       | Permission mode (plan, auto, default)                |
| `mcp_servers` | []MCPServer | No       | MCP server declarations                              |

**New fields — Skills configuration:**

| Field                  | Type     | Default | Description                                               |
| ---------------------- | -------- | ------- | --------------------------------------------------------- |
| `skills.inherit`       | bool     | true    | Whether to inherit skills from global and workspace pools |
| `skills.disabled`      | []string | []      | Named skills from inherited pools to exclude              |
| `skills.extra_sources` | []string | []      | Additional directories to scan for skills                 |

**New fields — Memory configuration:**

| Field                     | Type   | Default | Description                                                             |
| ------------------------- | ------ | ------- | ----------------------------------------------------------------------- |
| `memory.inherit`          | bool   | true    | Whether to inherit memories from global and workspace stores            |
| `memory.scope`            | string | "agent" | Default write scope for new memories: `agent`, `workspace`, or `global` |
| `memory.auto_consolidate` | bool   | true    | Automatically consolidate agent memories at session end                 |

### 2.4 Skills Resolution Hierarchy

When the daemon assembles the prompt for a session with a specific agent, skills are resolved in five layers:

```
1. Bundled skills (go:embed)                         — base, immutable
2. Global skills (~/.agh/skills/ + ~/.agents/skills/) — user-level
3. Workspace skills (.agents/skills/ + .agh/skills/)  — project-level
4. Extra sources (skills.extra_sources paths)         — agent-declared
5. Agent skills (.agents/<name>/skills/)              — agent-specific, highest precedence
```

**Override rules:**

- Same-name collision: highest precedence wins (agent skill overrides global skill)
- `skills.disabled` removes named skills from inherited layers before merge
- `skills.inherit: false` skips layers 1-3, uses only agent-specific and extra sources
- Override audit trail logs all shadows for debugging

**Example:** Agent `code-reviewer` has `skills.disabled: [agh-session-guide]` and a local `review-checklist` skill. The effective skill set is: all global/workspace skills minus `agh-session-guide`, plus the agent's `review-checklist` (which would override any global skill with the same name).

### 2.5 Memory Resolution Hierarchy

```
1. Global memory (~/.agh/memory/)                 — user-wide context
2. Workspace memory (.agh/memory/)                — project context
3. Agent memory (.agents/<name>/memory/)           — agent-specific, highest precedence
```

**Merge rules:**

- All levels are loaded and concatenated in the prompt (most specific last)
- Name conflict: agent memory shadows workspace/global memory with the same filename
- `memory.inherit: false` skips layers 1-2, loads only agent-scoped memories
- Memory writes default to the scope declared in `memory.scope`

### 2.6 Agent Memory Lifecycle

**Writes.** When the agent is instructed to save a memory:

1. Agent generates memory content (via tool call or embedded prompt instruction)
2. Daemon receives write request with target `scope` (agent | workspace | global)
3. Default scope comes from `memory.scope` in AGENT.md
4. File written to `.agents/<name>/memory/<file>.md` (for agent scope)
5. Index file `.agents/<name>/memory/MEMORY.md` updated automatically

**Automatic Consolidation.** If `memory.auto_consolidate: true`:

1. On session end, daemon analyzes accumulated agent memories
2. Identifies redundancies, outdated information, contradictions
3. Generates consolidated version (merge duplicates, remove stale entries)
4. Updates memory files and index
5. Records consolidation event in observability for audit

**Why auto-consolidation?** Without it, agent memories grow unboundedly. The ETH Zurich study showed that context file bloat increases inference costs by 20%+ while degrading task success. A dedicated consolidation pass keeps the memory store lean and relevant.

### 2.7 Portability

The agent directory is the atomic unit of portability:

```bash
# Copy the complete agent (definition + skills + memory)
cp -r .agents/code-reviewer/ /other/project/.agents/

# Share via version control
git add .agents/code-reviewer/
git commit -m "feat: add code-reviewer agent"

# Export/import (future marketplace integration)
agh agent export code-reviewer > code-reviewer.tar.gz
agh agent import code-reviewer.tar.gz
```

No external dependencies to chase. No global state to replicate. The directory contains everything the agent needs to function. Skills inside the agent directory follow the standard AgentSkills SKILL.md format — they work in any AgentSkills-compatible platform if extracted.

### 2.8 CLI

```bash
# Agent management
agh agent list                            # List available agents (user + workspace)
agh agent info <name>                     # Show AGENT.md + skills + memories
agh agent create <name>                   # Scaffold .agents/<name>/ with full structure

# Agent-scoped skills
agh agent skills <name>                   # List effective (merged) skills
agh agent skills <name> --local-only      # List only agent-internal skills

# Agent-scoped memory
agh agent memory <name>                   # List agent memories
agh agent memory <name> --consolidate     # Force manual consolidation
```

---

## 3. Comparison with Existing Approaches

### 3.1 vs. AGENTS.md

AGENTS.md defines project instructions — "how to work in this codebase." It's universally adopted (60,000+ projects) and tool-agnostic. But it describes the _project_, not the _agent_. Two agents working in the same project read the same AGENTS.md, even if they have completely different roles.

This proposal is complementary: AGENTS.md provides project context that all agents inherit. AGENT.md defines the agent itself — its capabilities, specialization, and state.

### 3.2 vs. Claude Code Subagents

Claude Code supports custom subagents in `.claude/agents/*.md` with YAML frontmatter (tools, model, permissions). This is the closest existing precedent. However:

- Subagent skills come from the global pool — no agent-scoped skills
- No agent-scoped memory — all subagents share the same MEMORY.md
- No skill inheritance control (can't disable specific skills for specific agents)
- No memory consolidation
- Claude-specific — won't work with Codex, Gemini CLI, or other agents

This proposal generalizes the Claude Code subagent pattern with agent-scoped resources and a provider-agnostic format.

### 3.3 vs. A2A Agent Cards

The A2A protocol defines Agent Cards — JSON documents describing an agent's capabilities for discovery. Agent Cards are designed for inter-agent communication ("what can you do?"), not for agent configuration ("how should you behave?"). They have no concept of skills, memory, or prompt content.

This proposal addresses a different layer: A2A Agent Cards could be _generated from_ an AGENT.md definition, providing the discovery metadata while the AGENT.md provides the runtime configuration.

### 3.4 vs. Codex Plugins

Codex plugins bundle skills + MCP servers + app integrations into installable units. But plugins are _capability bundles_, not agent definitions. A plugin says "here are tools for database work." An AGENT.md says "here is a database specialist agent with these tools, these skills, these memories, and this prompt."

---

## 4. Architecture Integration

### 4.1 Changes to Existing Components

| Component                               | Change                                                                                   |
| --------------------------------------- | ---------------------------------------------------------------------------------------- |
| `internal/config/agent.go`              | Extend `AgentDef` with `SkillsConfig` and `MemoryConfig` structs                         |
| `internal/skills/registry.go`           | Add `ForAgent()` method — resolves merged skill set for a specific agent                 |
| `internal/memory/assembler.go`          | Add `ForAgent()` method — assembles memory context with agent scoping                    |
| `internal/daemon/composed_assembler.go` | Use `ForAgent()` instead of `ForWorkspace()` when agent has scoped resources             |
| `internal/cli/agent.go`                 | New commands: `agent list`, `agent info`, `agent create`, `agent skills`, `agent memory` |
| `internal/daemon/daemon.go`             | Boot sequence loads agent directories and registers file watchers                        |

**No new packages.** All changes are extensions to existing packages, maintaining the project's flat architecture. The `ForAgent()` methods follow the established `ForWorkspace()` pattern.

### 4.2 Registry ForAgent

```go
func (r *Registry) ForAgent(
    ctx context.Context,
    workspace string,
    agentDef *config.AgentDef,
) ([]*Skill, error) {
    // 1. If inherit=true: collect global skills (bundled + user + marketplace)
    // 2. If inherit=true: collect workspace skills
    // 3. Collect extra_sources skills
    // 4. Collect .agents/<name>/skills/
    // 5. Apply agentDef.Skills.Disabled (remove named skills)
    // 6. Resolve overrides by precedence (agent > extra > workspace > global)
    // 7. Return final merged list
}
```

### 4.3 Memory ForAgent

```go
func (a *Assembler) ForAgent(
    ctx context.Context,
    workspace string,
    agentName string,
    inherit bool,
) (string, error) {
    // 1. If inherit=true: load global memory index
    // 2. If inherit=true: load workspace memory index
    // 3. Load .agents/<name>/memory/MEMORY.md
    // 4. Apply shadow rules (agent > workspace > global)
    // 5. Return concatenated context
}
```

---

## 5. Full Example

### Directory Structure

```
.agents/code-reviewer/
  AGENT.md
  skills/
    review-checklist/
      SKILL.md
    security-patterns/
      SKILL.md
  memory/
    MEMORY.md
    feedback_prefer_short.md
    project_auth_context.md
```

### skills/review-checklist/SKILL.md

Standard AgentSkills format — works in any compatible platform:

```yaml
---
name: review-checklist
description: Team's standard code review checklist. Use on every PR review.
version: 1.0.0
---

## Review Checklist

- [ ] Error handling: all errors handled with wrapped context
- [ ] Tests: >=80% coverage for new code
- [ ] Security: no SQL injection, XSS, command injection
- [ ] Performance: no N+1 queries, no unnecessary loops
- [ ] Concurrency: correct mutexes, no race conditions
```

### memory/feedback_prefer_short.md

Standard memory format with frontmatter:

```yaml
---
name: prefer-short-comments
description: Team prefers short, direct review comments
type: feedback
---
Review comments should be short and direct (1-2 lines).
Don't explain the problem in detail — just point it out and suggest a fix.

**Why:** Team reported that verbose reviews get ignored.
**How to apply:** In every review comment, limit to 2 lines max.
```

---

## 6. Open Questions

1. **Format convergence.** Should AGENT.md align more closely with AGENTS.md (pure Markdown, no frontmatter) for universal compatibility? Or is structured frontmatter essential for machine-parseable agent configuration? Current position: structured frontmatter is necessary — the fields (provider, model, tools, permissions) are inherently structured data, not prose.

2. **Cross-platform agent portability.** If another platform adopts the AGENT.md format, how should provider-specific fields (e.g., Claude's `permissions: plan`) be handled? Namespace them (`claude.permissions`)? Ignore unknown fields? Use a capabilities model instead of provider-specific fields?

3. **Agent identity across projects.** When the same agent directory is copied to multiple projects, should memories from different projects be merged, kept separate, or explicitly namespaced? Current position: memories are per-directory, so copies diverge naturally.

4. **Memory consolidation strategy.** Auto-consolidation requires heuristics to identify redundancy and staleness. Should the daemon use a simple rule-based approach (dedup, age-based expiry) or delegate to the agent's LLM for semantic consolidation? The latter is more accurate but has cost implications.

5. **Skill inheritance depth.** If agent A declares `extra_sources: ["../shared-skills/"]` and another agent B has a skill with the same name, what's the precedence? Current position: agent-local always wins over extra sources, which win over workspace/global.

6. **Standardization path.** Should this format be proposed as an extension to AGENTS.md under AAIF governance? Or as a standalone spec? The AGENTS.md convention is intentionally minimal — adding structured agent definitions might conflict with its "plain Markdown" philosophy.
