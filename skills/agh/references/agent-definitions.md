# Agent Definitions

## Contents

- Files and precedence
- Minimal AGENT.md
- Fields
- Tool grants
- Providers and MCP
- Setup workflow

## Files And Precedence

AGH agent definitions live in AGENT.md files with YAML frontmatter and a Markdown prompt body. Global agents live under $AGH_HOME/agents/<name>/AGENT.md; workspace agents live under <workspace>/.agh/agents/<name>/AGENT.md.

Runtime configuration starts from $AGH_HOME/config.toml, then workspace configuration can overlay it with <workspace>/.agh/config.toml. Agent-local skills and MCP sidecars are resolved after the effective agent definition is chosen.

## Minimal AGENT.md

    ---
    name: general
    provider: claude
    model: claude-sonnet-4-6
    permissions: approve-all
    ---
    You are a reliable software engineering agent.

The prompt body is required. AGH rejects an agent definition with no prompt.

## Fields

- name is required and must match the directory name for filesystem-loaded agents.
- provider, model, and command can be omitted when provider defaults supply them.
- tools grants exact ToolIDs or namespace-prefix wildcard patterns.
- toolsets grants named ToolsetIDs such as agh\_\_catalog.
- deny_tools narrows grants.
- permissions must be one of deny-all, approve-reads, or approve-all.
- category_path is display-only hierarchy and must be an array.
- mcp_servers declares per-agent MCP servers.

Do not use categories or slash strings for hierarchy. They are not runtime semantics.

## Tool Grants

Do not add agh**bootstrap or agh**catalog only for discovery. AGH adds those default discovery toolsets unless policy denies them.

Keep frontmatter grants narrow and intentional. Add extra tools only when the agent needs those runtime capabilities.

## Providers And MCP

Built-in provider names include claude, codex, gemini, opencode, copilot, cursor, kiro, and pi. Provider config can supply launch command, default model, API key environment, and provider-level MCP servers.

Per-agent MCP servers belong in AGENT.md or an agent-local mcp.json sidecar. mcp.json replaces same-name frontmatter servers. Use provider-level MCP when every agent for that provider needs the server; use agent-level MCP when one agent needs it.

## Setup Workflow

1. Set common defaults in $AGH_HOME/config.toml.
2. Create $AGH_HOME/agents/<name>/AGENT.md or workspace-local equivalent.
3. Keep frontmatter small and put behavior in the Markdown body.
4. Add only the toolsets and MCP servers the agent actually needs.
5. Validate with AGH CLI/API rather than guessing from file shape.

If AGH rejects the agent, inspect missing name, invalid permissions, empty prompt body, malformed mcp_servers, or a directory/name mismatch first.
