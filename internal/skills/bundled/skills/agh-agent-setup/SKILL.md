---
name: agh-agent-setup
description: Set up AGH agent definitions, provider defaults, permissions, and MCP server entries correctly.
version: "1.0.0"
---

# AGH Agent Setup

Use this guide when you need to create or review an AGH agent definition.

## Where agent configuration lives

AGH resolves runtime settings from two places:

- `~/.agh/config.toml` for global defaults, permissions, and provider configuration
- `~/.agh/agents/<agent-name>/AGENT.md` for each agent definition

Workspace config can overlay parts of the global config at `.agh/config.toml`, and both the global `.agh/` directory and each agent directory can also contain an optional `mcp.json` sidecar for MCP server declarations.

## Minimal AGENT.md structure

AGH agent definitions are parsed from `AGENT.md` files with YAML frontmatter followed by the prompt body.

```yaml
---
name: general
provider: claude
model: claude-sonnet-4-20250514
tools: [read, glob, grep, write, bash]
permissions: approve-all
---
You are a reliable software engineering agent.
```

The prompt body after the frontmatter is required. AGH will reject an agent definition with no prompt.

## Required and optional fields

- `name`: required and must match the agent directory name when loaded from `~/.agh/agents/<name>/AGENT.md`
- `provider`: optional if you already configured a default provider
- `model`: optional if the provider config already defines a default model
- `command`: optional when the provider already defines how to launch the ACP adapter
- `tools`: optional; AGH defaults to `["*"]` when omitted
- `permissions`: optional; AGH falls back to the global permissions mode when omitted
- `mcp_servers`: optional per-agent MCP server list
- `mcp.json`: optional sidecar file in the same agent directory when you want MCP declarations outside frontmatter

## Permission modes

AGH validates permission modes strictly. Use one of:

- `deny-all`
- `approve-reads`
- `approve-all`

If you leave `permissions` out of `AGENT.md`, AGH uses the global `[permissions]` setting from `~/.agh/config.toml`.

## Providers and MCP servers

AGH ships built-in provider names such as `claude`, `codex`, `gemini`, `opencode`, `copilot`, `cursor`, `kiro`, and `pi`. A provider can contribute the launch command, default model, API key environment variable, and provider-level MCP servers.

You can attach MCP servers in `AGENT.md` like this:

```yaml
mcp_servers:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
```

AGH merges provider-level and agent-level MCP servers by name. Use the agent definition when the server belongs to one agent, and the provider config when it should apply to every agent for that provider.

You can also attach agent-local MCP servers with `<agent-dir>/mcp.json`:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  }
}
```

If both `AGENT.md` and `mcp.json` declare MCP servers, AGH keeps both and lets `mcp.json` replace same-name entries from the frontmatter.

## Practical setup workflow

1. Set defaults in `~/.agh/config.toml` for the provider and permission mode you use most often.
2. Create `~/.agh/agents/<name>/AGENT.md`.
3. Keep the frontmatter small and put the actual behavior instructions in the markdown body.
4. Add `mcp_servers` only when the agent really needs them.
5. Reuse provider defaults instead of copying the same `command` and `model` into every agent file.

If AGH rejects an agent definition, check the frontmatter first: missing `name`, invalid `permissions`, empty prompt body, or malformed `mcp_servers` entries are the fastest failure points.
