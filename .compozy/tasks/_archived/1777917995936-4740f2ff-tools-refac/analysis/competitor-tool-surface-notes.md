# Competitor Tool Surface Notes

## Purpose

Reference notes for `tools-refac` task generation and implementation. These are
 the external code examples the TechSpec relies on for agent guidance,
 discovery, policy projection, and MCP auth management posture.

## Claude Code

### Tool guidance and startup prompt

- `.resources/claude-code/constants/prompts.ts`
  - Startup prompt includes an explicit "Using your tools" guidance block.
- `.resources/claude-code/tools/ToolSearchTool/ToolSearchTool.ts`
  - Dedicated tool-search flow for deferred tool discovery.

### Discovery and runtime permission split

- `.resources/claude-code/tools.ts`
  - Tool exposure filtered before the model sees the schema set.
- `.resources/claude-code/services/api/claude.ts`
  - Final tool schema assembly for the request path.
- `.resources/claude-code/services/tools/toolExecution.ts`
  - Runtime tool execution still revalidates input and permission checks.
- `.resources/claude-code/utils/permissions/permissions.ts`
  - Effective permission evaluation runs at call time.

### MCP auth / management posture

- `.resources/claude-code/commands/mcp/mcp.tsx`
- `.resources/claude-code/services/mcp/auth.ts`
- `.resources/claude-code/components/mcp/MCPReconnect.tsx`
  - Re-auth is routed to a management surface (`/mcp`), not a normal
    model-callable tool.

## Hermes

### Tool guidance and discovery

- `.resources/hermes/agent/prompt_builder.py`
  - Startup prompt includes explicit runtime/tool guidance.
- `.resources/hermes/tools/registry.py`
  - Tool definitions are filtered by enabled toolsets and runtime checks.
- `.resources/hermes/website/docs/reference/slash-commands.md`
  - Management surfaces remain distinct from normal tools.

### MCP auth / management posture

- `.resources/hermes/website/docs/reference/mcp-config-reference.md`
  - OAuth is configured declaratively and completed through browser flow on
    first connect.
- `.resources/hermes/hermes_cli/commands.py`
  - Reload and management remain command surfaces rather than agent-callable
    login/logout tools.

## OpenClaw

### Tool guidance and startup prompt

- `.resources/openclaw/src/agents/system-prompt.ts`
  - Startup prompt includes a dedicated tooling block that teaches tool usage.
- `.resources/openclaw/src/agents/tool-policy-pipeline.ts`
  - Tool inventory is filtered before exposure.

### MCP auth / management posture

- `.resources/openclaw/docs/cli/mcp.md`
  - MCP definitions and credentials are managed through explicit CLI commands,
    not through a normal runtime tool family.

## Cross-Repo Findings Used By `tools-refac`

1. Internal runtime capabilities are tool-first, not shell-first.
2. Startup guidance explicitly teaches the agent that tools exist and how to
   use them.
3. Discovery-time filtering improves UX, but runtime execution revalidates
   policy.
4. MCP auth login/logout stays on a management surface even when auth status is
   visible to the runtime.
