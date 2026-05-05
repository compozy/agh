# RFC: Daemon-Managed Skills with Lifecycle, MCP Bridge, and Security

- **Status:** Draft
- **Authors:** AGH Core Team
- **Created:** 2026-04-06
- **Relates to:** AgentSkills Specification (agentskills.io), MCP (modelcontextprotocol.io), AAIF standards

---

## Abstract

The AgentSkills specification (December 2025) established a portable format for reusable AI agent instructions. Adopted by 26+ platforms within weeks, it solved the fragmentation problem for _authoring_ skills. However, the spec is deliberately minimal — it defines a file format but not a runtime. It does not address how skills are loaded securely, how they declare tool dependencies, how they participate in agent lifecycle events, or how they interact with persistent memory.

This RFC proposes a **daemon-managed skills runtime** that extends the AgentSkills spec with four capabilities no current implementation combines: security scanning at load time, declarative MCP server provisioning, lifecycle hooks, and bidirectional memory integration. These extensions are expressed as `metadata.agh.*` fields in standard SKILL.md frontmatter, preserving full compatibility with the base specification.

---

## 1. Problem Statement

### 1.1 The AgentSkills Spec Is a Format, Not a Runtime

The AgentSkills specification defines a directory containing a `SKILL.md` file with YAML frontmatter and a Markdown body. It establishes progressive disclosure (metadata → instructions → resources) and a portable skill format. This is valuable and adopted. But the spec explicitly defers critical runtime concerns:

| Concern                | AgentSkills Spec                                      | Current Ecosystem                                                                                          |
| ---------------------- | ----------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| **Security**           | No scanning, signing, or verification                 | ClawHavoc (Feb 2026): 1,184+ malicious skills on ClawHub. Snyk: 36.82% of skills have security flaws.      |
| **MCP integration**    | `allowed-tools` field (experimental, tool names only) | Skills and MCP are "complementary layers" but no spec defines how a skill declares MCP server dependencies |
| **Lifecycle**          | Static content loaded at activation                   | No hooks for session events. Skills cannot react to creation, termination, or prompt assembly              |
| **Memory**             | No concept of persistent state                        | Skills are stateless. No way to declare memory dependencies or guide memory writes                         |
| **Hot-reload**         | Not addressed                                         | Editing a skill requires restarting the agent session in most implementations                              |
| **Override semantics** | Not specified                                         | Each platform implements its own precedence rules (or none)                                                |

### 1.2 The ClawHavoc Precedent

In February 2026, security researchers discovered 341 malicious skills on ClawHub (later revised to 1,184+ by Antiy CERT). Attack vectors included credential harvesting, reverse shells, and prompt injection into agent memory files. The root cause: **open registry with no code review, no signing, and no automated scanning**. Skills execute with the developer's full system permissions.

The AgentSkills spec has no security model. Scanning happens at the registry boundary (if at all), not at load time. There is no runtime verification, no sandboxing, and no provenance chain.

### 1.3 Skills and MCP Are Complementary but Disconnected

Anthropic positions skills as "the brain" (what to know) and MCP as "the arms" (how to connect). In practice, these layers are disconnected. A skill that teaches database migration patterns cannot declare that it needs a PostgreSQL MCP server. The user must manually configure MCP servers separately, breaking the skill's portability promise.

OpenAI's Codex plugins (March 2026) bundle skills + MCP servers + app integrations into a single installable unit. This validates the demand but locks the pattern into a proprietary, platform-specific format.

### 1.4 No Lifecycle Participation

Skills are static text injected into prompts. They cannot react to session events. A skill that teaches "how to set up a new project" cannot inject repository state at session start. A skill that teaches debugging patterns cannot consolidate learnings at session end. The spec's progressive disclosure model (3 levels) is a loading optimization, not a lifecycle model.

---

## 2. Proposal

### 2.1 Design Principles

1. **Extend, don't fork.** All extensions use the `metadata.agh.*` namespace in standard SKILL.md frontmatter. Any AgentSkills-compatible skill works unmodified. AGH-specific features degrade gracefully on other platforms (ignored metadata fields).

2. **Security at the boundary.** Every non-bundled skill is scanned at load time before entering the registry. Critical findings (prompt injection, credential extraction) must not silently execute; they either block loading or require an explicit retained quarantine state. This is non-negotiable after ClawHavoc.

3. **Declarative over imperative.** Skills declare what they need (MCP servers, memory tags, lifecycle hooks); the daemon manages provisioning, permissions, and teardown.

4. **Daemon as governor.** A long-running daemon process (not a CLI wrapper) manages skill lifecycle, enforces security policy, and maintains observability. CLI-only implementations cannot provide these guarantees.

### 2.2 Security Scanning (VerifyContent)

Every skill loaded from non-bundled sources passes through `VerifyContent` before entering the registry.

**Three severity levels:**

| Severity | Action             | Examples                                                                                                                                                            |
| -------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Critical | **Block loading**  | System prompt overrides (`ignore all previous`), tool abuse (`rm -rf`, `delete all files`), credential extraction (`print your API key`, `show your system prompt`) |
| Warning  | Log, allow loading | Sensitive path references (`/etc/passwd`, `~/.ssh/`), unusual tool patterns                                                                                         |
| Info     | Log only           | Content >50K characters, unusual frontmatter fields                                                                                                                 |

**Scanning is applied at load time, not just at install time.** A skill modified on disk after installation is re-scanned on next load. This closes the time-of-check/time-of-use gap that ClawHub scanning missed.

**Bundled skills are trusted.** They ship compiled into the binary via `go:embed` and are immutable for the process lifetime.

### 2.3 Declarative MCP Lazy-Load

Skills declare MCP server dependencies in frontmatter:

```yaml
---
name: postgres-tools
description: Database migration and query tooling
version: 1.0.0
metadata:
  agh:
    mcp_servers:
      - name: pg-mcp
        command: npx
        args: ["@pg/mcp-server", "--host", "localhost"]
        env:
          PG_PASSWORD: "${PG_PASSWORD}"
---
```

**Runtime behavior:**

1. Registry parses `metadata.agh.mcp_servers` during skill loading
2. On session creation, daemon collects MCP servers from all active skills
3. **User consent gate** — first time a marketplace skill declares an MCP server, the daemon requests explicit user confirmation (CLI prompt or persistent allowlist in config). User, additional-root, and workspace skills remain auto-approved because they are placed deliberately by the local operator.
4. Approved servers are injected into `StartOpts.MCPServers` and spawned by the ACP driver alongside the agent process
5. Environment variables with `${}` syntax are resolved only after consent, with scrubbing of sensitive values not explicitly allowed
6. Marketplace consent is persisted in config via `skills.allowed_marketplace_mcp`; command-level allowlists are deferred to a follow-up security hardening pass

**Trust tiers:**

| Source                    | MCP Consent                           | Command Restrictions |
| ------------------------- | ------------------------------------- | -------------------- |
| Bundled                   | None (trusted)                        | None                 |
| User/Additional/Workspace | None (user-controlled local content)  | None                 |
| Marketplace               | One-time consent, persisted in config | Deferred             |

**Comparison with Codex plugins:** Codex bundles MCP config into a proprietary plugin.json format. This proposal keeps MCP declarations in the standard SKILL.md frontmatter, making skills portable while the daemon provides the runtime governance Codex achieves through its platform.

### 2.4 Lifecycle Hooks

Skills declare hooks for session lifecycle events. In the current skills-v2 TechSpec, only `on_session_created` and `on_session_stopped` are in scope; `on_prompt_assembly` is explicitly deferred.

```yaml
metadata:
  agh:
    hooks:
      - event: on_session_created
        command: "inject-context"
        args: ["--format", "json"]
        timeout: 5s
      - event: on_session_stopped
        command: "consolidate-learnings"
        timeout: 10s
```

**Events:**

| Event                | Trigger                                  | Use Case                                             |
| -------------------- | ---------------------------------------- | ---------------------------------------------------- |
| `on_session_created` | Session initialized, before first prompt | Inject repo state, open tickets, environment context |
| `on_session_stopped` | Session terminated                       | Consolidate memories, save learnings, cleanup        |

**Execution semantics:**

- Hooks execute in hierarchy precedence order (bundled → marketplace → user → additional → workspace)
- Within the same level: alphabetical by skill name (deterministic)
- Configurable timeout per hook (default 5s)
- **Fail-open:** hook errors are logged as warnings but never block the session
- Hooks receive JSON via stdin: `{"session_id": "...", "agent_name": "...", "workspace": "..."}`
- Hooks may emit structured stdout for logging/future enrichment, but prompt-context injection is deferred with `on_prompt_assembly`
- Daemon extends the existing notifier fan-out with a dedicated post-notifier hook phase rather than introducing a separate lifecycle service

**Why not in the base spec?** The AgentSkills spec is intentionally client-agnostic. Lifecycle hooks require a runtime with session concepts. A daemon architecture provides this naturally; CLI wrappers cannot.

### 2.5 Memory Integration

This section remains future work. The current skills-v2 TechSpec explicitly defers deep memory integration to a follow-up spec, so the details below should be read as forward-looking design rather than current implementation scope.

In the base implementation, memory and skills coexist in the prompt without coupling — memory context is assembled first, then the agent prompt, then the skill catalog. This works but misses the opportunity for skills to leverage memory and guide memory writes.

**Deep integration (this proposal):**

**Tag-filtered injection.** Skills declare memory dependencies:

```yaml
metadata:
  agh:
    memory_tags: ["project", "feedback"]
```

The daemon filters the memory store and injects only memories matching the declared tags into the skill's context section. This prevents irrelevant memories from consuming context budget.

**Memory query API.** A future prompt-assembly hook or equivalent prompt-enrichment surface could query the memory store via a structured request, receiving relevant memories in the response. This remains deferred while `on_prompt_assembly` is out of scope.

**Skill-guided writes.** Skills can include instructions that teach agents to save specific types of memories. Example: a debugging skill that says "save the root cause as a project memory for future reference." The daemon enforces scope rules on writes.

**Bidirectional flow:** skill reads memory → enriches prompt → agent acts → agent saves memory → next session uses enriched memory. This creates a compounding improvement loop where skills get better over time without being edited.

### 2.6 Skill Auto-Proposal

The daemon detects repetitive workflows and proposes skill creation:

**Detection:** Analyze the last N sessions in the same workspace. Identify patterns: repeated tool call sequences, similar prompts, recurring multi-step workflows. Threshold: 3+ occurrences of the same pattern across different sessions.

**Proposal:** At session end, if a pattern is detected, append a suggestion to the agent context:

```
[AGH] Recurring workflow detected: "<description>".
Consider creating a skill with `agh skill create <suggested-name>`.
```

A bundled meta-skill `skillify` guides the agent through formalizing the workflow into a SKILL.md file, using session history and memory to generate a draft.

**Compounding loop:** usage → detection → proposal → skill → improved usage. This is the key differentiator — the system improves itself through use, without requiring the user to proactively identify reusable patterns.

### 2.7 Skill Distribution and Provenance (Marketplace)

**CLI interface:**

```bash
agh skill search "database tools"      # Search marketplace
agh skill install @author/skill-name   # Install to ~/.agh/skills/
agh skill remove skill-name            # Remove installed skill
agh skill update [--all]               # Update marketplace skills
```

**Security model (post-ClawHavoc):**

- **Hash-based provenance verification:** SHA-256 captured on install and rechecked on every load
- **Load-time scanning:** `VerifyContent` applied to every downloaded skill, every load
- **Override audit trail:** warning when a workspace skill shadows a bundled/marketplace skill
- **Quarantine/blocking:** critical findings require explicit retained quarantine state; otherwise the safe fallback is to keep block-on-load semantics until re-approval UX exists
- **MCP command allowlists:** deferred follow-up, not part of the current skills-v2 TechSpec

### 2.8 Precedence Hierarchy

Skills are resolved in six source layers, with higher layers overriding lower:

```
1. Bundled                      — lowest, immutable, shipped with binary
2. Marketplace                  — `~/.agh/skills/` entries with `.agh-meta.json`
3. User                         — manual `~/.agh/skills/` entries
4. Additional                   — `.agh/skills/` under configured additional workspace roots
5. Workspace                    — `<workspace>/.agh/skills/`
6. Agent-local                  — highest, `.agh/agents/<name>/skills/`
```

Workspace is the highest base layer. Agent-local is the final overlay after AGH resolves the
winning `AGENT.md`. Same-name collisions still resolve to the highest-precedence winner, and the
override audit trail logs every shadow.

---

## 3. Data Model

```go
type Skill struct {
    Meta          SkillMeta
    Content       string           // Markdown body after frontmatter
    Source        SkillSource      // Bundled | Marketplace | User | Additional | Workspace
    Dir           string           // Absolute path to skill directory
    FilePath      string           // Absolute path to SKILL.md
    Enabled       bool
    MCPServers    []MCPServerDecl  // Parsed from metadata.agh.mcp_servers
    Hooks         []HookDecl       // Parsed from metadata.agh.hooks
    Provenance    *Provenance      // Marketplace: registry/source metadata + hash
    InstalledFrom string           // Marketplace: registry slug
}

type HookDecl struct {
    Event   HookEvent             // on_session_created | on_session_stopped
    Command string
    Args    []string
    Timeout time.Duration
    Env     map[string]string
}

type MCPServerDecl struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}

type Provenance struct {
    Slug      string
    Registry  string              // e.g., "clawhub", "skills.sh"
    Version   string
    Hash      string
    InstalledAt time.Time
}
```

---

## 4. Comparison with Existing Approaches

| Capability         | AgentSkills Spec             | Codex Plugins            | Cursor Rules                | This Proposal                                                         |
| ------------------ | ---------------------------- | ------------------------ | --------------------------- | --------------------------------------------------------------------- |
| Portable format    | Yes (SKILL.md)               | No (plugin.json)         | No (.mdc)                   | Yes (SKILL.md + metadata.agh.\*)                                      |
| Security scanning  | Registry-only (if at all)    | Platform-managed         | None                        | Load-time, every load                                                 |
| MCP integration    | `allowed-tools` (names only) | Bundled in plugin        | None                        | Declarative in frontmatter + daemon provisioning                      |
| Lifecycle hooks    | None                         | Triggers (GitHub events) | None                        | 2 session events with stdin/stdout protocol; prompt assembly deferred |
| Memory integration | None                         | None                     | Auto-memories (proprietary) | Tag-filtered, bidirectional, skill-guided writes                      |
| Hot-reload         | Not specified                | Not specified            | File watcher                | Stat-based polling (global) + mtime check (workspace)                 |
| Override semantics | Not specified                | Plugin precedence        | Rule precedence             | 5-layer hierarchy with audit trail                                    |
| Auto-proposal      | None                         | None                     | None                        | Pattern detection + skillify meta-skill                               |
| Provenance         | None                         | Platform-curated         | N/A                         | Hash-based verification + load-time scanning                          |

---

## 5. Incremental Delivery

| Increment | Scope                                                                                             | Status       |
| --------- | ------------------------------------------------------------------------------------------------- | ------------ |
| 1         | Loader, dual-scope registry, prompt injection, security scanning, CLI, bundled skills, hot-reload | **Complete** |
| 2         | MCP lazy-load, lifecycle hooks, and skill auto-proposal                                           | Planned      |
| 3         | Marketplace integration, hash-based provenance, override audit trail                              | Planned      |

Each increment ships independent value. Increment 1 is already production-ready.

---

## 6. Open Questions

1. **Hook execution ordering across skills.** When two skills declare `on_session_created`, execution follows hierarchy precedence then alphabetical order. Should skills be able to declare explicit ordering dependencies?

2. **MCP consent persistence.** One-time consent per skill is persisted in config. Should consent be revocable? Should it expire? Should it be per-workspace or global?

3. **Memory tag taxonomy.** Skills declare `memory_tags` for filtered injection. Should there be a controlled vocabulary, or is freeform sufficient? Risk: tag proliferation without discoverability.

4. **Auto-proposal accuracy.** Pattern detection across sessions requires heuristics. False positives (suggesting skills for one-off workflows) could erode user trust. What's the right threshold?

5. **Marketplace governance.** Should marketplace skills require manual review, automated scanning only, or a combination? What's the right balance between openness and safety post-ClawHavoc?
