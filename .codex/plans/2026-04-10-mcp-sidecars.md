# MCP Top-Level + `mcp.json` Sidecars

## Summary

- Add top-level `mcp_servers` to `config.toml`, merged global `~/.agh/config.toml` then workspace `.agh/config.toml`.
- Support `mcp.json` in four containers: `~/.agh/`, `<workspace>/.agh/`, `<agentDir>/`, and `<skillDir>/`.
- Keep existing sources: `providers.<name>.mcp_servers`, `AGENT.md` `mcp_servers`, and `SKILL.md` `metadata.agh.mcp_servers`.
- Final runtime order: resolved top-level config MCP servers, then `providers.<name>.mcp_servers`, then agent MCP servers, then active skill MCP servers.

## Key Changes

- `config.toml` accepts `[[mcp_servers]]` at the root level as a session-wide baseline.
- `mcp.json` accepts `mcpServers` and alias `mcp_servers`, both as a map keyed by server name.
- Same-scope sources coexist:
  - global: `config.toml` + `~/.agh/mcp.json`
  - workspace: `.agh/config.toml` + `.agh/mcp.json`
  - agent: `AGENT.md` + `<agentDir>/mcp.json`
  - skill: `SKILL.md` metadata + `<skillDir>/mcp.json`
- Within the same scope, `mcp.json` has higher precedence and replaces the whole server object on same-name collision.
- Across scopes, preserve the current runtime merge order so provider, agent, and skill behavior remains predictable.

## Implementation Changes

- Add one shared MCP JSON loader/normalizer with:
  - support for `mcpServers` and `mcp_servers`
  - path-aware validation errors
  - normalization into canonical runtime MCP server values
- Extend `internal/config` to load global/workspace `mcp.json` alongside `config.toml` and expose top-level `Config.MCPServers`.
- Extend agent loading to auto-load `<agentDir>/mcp.json` and merge it with `AGENT.md`.
- Extend skill loading to auto-load `<skillDir>/mcp.json` and merge it with `metadata.agh.mcp_servers`.
- Update workspace and skills snapshot/invalidation logic so changes to all relevant `mcp.json` files invalidate caches correctly.

## Test Plan

- Loader tests for `mcpServers`, `mcp_servers`, invalid JSON, missing commands, and normalization.
- Config merge tests for global/workspace `config.toml` + `mcp.json`, including same-scope precedence and cross-scope overrides.
- Agent tests for `AGENT.md` + sidecar coexistence and same-name replacement by `mcp.json`.
- Skill tests for `SKILL.md` + sidecar coexistence and cache invalidation when only `mcp.json` changes.
- Session/runtime tests for `config -> provider -> agent -> skill` merge order.
- Final verification: `make verify`.

## Assumptions

- `mcp.json` is read-only support in this task; no writer/editor commands are added.
- Existing formats are not deprecated.
- JSON shape is a map keyed by server name, not an array mirror of TOML/frontmatter.
