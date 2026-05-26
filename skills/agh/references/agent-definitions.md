# Agent Definitions

## Contents

- Files and precedence
- Minimal AGENT.md
- Fields
- Tool grants
- Providers and MCP
- Setup workflow
- Provider aliases and settings apply

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

Do not add `agh__bootstrap` or `agh__catalog` only for discovery. AGH adds those default discovery toolsets unless policy denies them.

Keep frontmatter grants narrow and intentional. Add extra tools only when the agent needs those runtime capabilities.

## Managed Bundled Agents

AGH ensures two managed agent definitions exist on first boot and during `agh install`:

- `general` — the default public general-purpose agent (`defaults.agent`). It is the agent operators see in public agent lists and the workspace sidebar unless a workspace-local `general` overrides it.
- `onboarding` — a reserved internal first-run setup agent. It stays hidden from public agent lists, workspace detail payloads, and `agh agent list/info`, but the onboarding wizard can still start sessions with `agent_name: "onboarding"`. It interviews the operator in the web onboarding wizard and provisions channels and agents through exact tool grants: `agh__workspace_list`, `agh__workspace_describe`, `agh__network_channels`, `agh__network_channel_create`, and `agh__agent_create`. It runs with `approve-all` over only those tools.

Both are recreated only when missing; operator edits are preserved.
Public authoring surfaces reject attempts to create an agent named `onboarding`.

## Providers And MCP

Built-in provider names include claude, codex, gemini, opencode, copilot, cursor, kiro, and pi. Provider config can supply launch command, default model, API key environment, and provider-level MCP servers.

Per-agent MCP servers belong in AGENT.md or an agent-local mcp.json sidecar. mcp.json replaces same-name frontmatter servers. Use provider-level MCP when every agent for that provider needs the server; use agent-level MCP when one agent needs it.

## Setup Workflow

1. Set common defaults in $AGH_HOME/config.toml.
2. Create $AGH_HOME/agents/<name>/AGENT.md or workspace-local equivalent.
3. Keep frontmatter small and put behavior in the Markdown body.
4. Add only the toolsets and MCP servers the agent actually needs.
5. Reconcile desired config with runtime truth after config edits, using `agh config reload -o json` when the daemon is running.
6. Validate with AGH CLI/API rather than guessing from file shape.

If AGH rejects the agent, inspect missing name, invalid permissions, empty prompt body, malformed mcp_servers, or a directory/name mismatch first.

## Provider Aliases And Settings Apply

Provider aliases are small built-in conveniences, not user-configured compatibility keys. `claude-code` resolves to the canonical `claude` provider; aliases such as `ai-gateway`, `vercel`, `kimi`, `glm`, `x.ai`, `grok`, `open-code`, and `qwen` resolve before launch. Config files must still reference canonical provider IDs, and the removed `providers.<id>.aliases` key is rejected.

Settings writes are governed by the config apply lifecycle. After changing provider defaults, MCP sidecars, sandboxes, hooks, or skills config, inspect `lifecycle`, `applied`, `next_action`, `active_generation`, and `apply_record_id` in the command response or `agh config apply-history -o json`. New-session or restart-required changes are not active for already-running sessions unless the lifecycle says they are.
