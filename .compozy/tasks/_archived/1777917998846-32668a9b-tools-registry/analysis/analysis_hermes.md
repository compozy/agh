# Competitor Analysis: Hermes Tool Registry

## Overview

Hermes has the clearest direct precedent for the "ToolRegistry" name. Its tool system centers on a single Python registry that collects tool definitions, schemas, handlers, availability checks, toolset membership, MCP adapters, plugin contributions, and dispatch metadata.

The core pattern is powerful but too global for AGH to copy literally. The transferable part is the product shape: every tool contribution normalizes into one registry contract, the model-visible tool list is filtered by availability and toolset policy, and tool calls flow through one dispatch path.

## Mechanisms / Patterns

Hermes tools self-register at import time. A tool module imports the singleton registry and calls `registry.register(...)` with a name, toolset, JSON schema, handler, optional `check_fn`, env requirements, display metadata, async flag, and result budget.

The registry provides:

- `register`: add a tool definition and reject most non-MCP collisions.
- `get`: resolve a tool entry.
- `dispatch`: call the registered handler.
- `get_available_tools`: project the model-visible list after toolset and availability filters.

Toolsets are recursive bundles. Named toolsets can compose other toolsets, and broad aliases such as `all` / `*` expand across registered tools. This is the most relevant pattern for AGH's agent-role tool policy.

Availability is attached to the tool definition through `check_fn` and `requires_env`. Hermes filters unavailable tools before presenting definitions to the model. This is a critical reliability property, but Hermes does not consistently treat it as a dispatch-time security boundary.

MCP tools are adapted into the same registry. Hermes discovers MCP tools, normalizes schemas, prefixes/organizes names, refreshes on MCP `tools/list_changed`, and registers each remote tool as a local registry entry. Dynamic MCP refresh is tested.

Plugins can contribute tools, but the exact dispatch path can bypass parts of the normal `handle_function_call` pipeline. AGH should avoid this split by making the registry dispatch path mandatory for every executable AGH-owned tool.

## Relevant Code Paths

- `.resources/hermes/tools/registry.py:1-14`: registry purpose and top-level contract.
- `.resources/hermes/tools/registry.py:23-64`: `ToolEntry` shape.
- `.resources/hermes/tools/registry.py:176-228`: singleton registry operations.
- `.resources/hermes/tools/registry.py:260-327`: availability filtering and definitions.
- `.resources/hermes/tools/registry.py:352-433`: dispatch and result handling.
- `.resources/hermes/model_tools.py:141-153`: built-ins are imported to trigger registration.
- `.resources/hermes/model_tools.py:209-370`: tool definition projection for model calls.
- `.resources/hermes/model_tools.py:389-528`: function-call handling.
- `.resources/hermes/model_tools.py:529-705`: result transformation and tool-call lifecycle.
- `.resources/hermes/toolsets.py:483-692`: recursive toolset composition.
- `.resources/hermes/hermes_cli/tools_config.py:681-849`: operator configuration for tools.
- `.resources/hermes/hermes_cli/plugins.py:210-380`: plugin load and metadata paths.
- `.resources/hermes/hermes_cli/plugins.py:518-646`: plugin tool dispatch path.
- `.resources/hermes/tools/mcp_tool.py:860-1038`: MCP discovery and schema adaptation.
- `.resources/hermes/tools/mcp_tool.py:1058-1296`: MCP tool registration details.
- `.resources/hermes/tools/mcp_tool.py:1850-2108`: MCP refresh and runtime paths.
- `.resources/hermes/tools/mcp_tool.py:2508-2770`: dynamic discovery integration.
- `.resources/hermes/tests/tools/test_mcp_dynamic_discovery.py:1-160`: tests for MCP dynamic tool refresh.
- `.resources/hermes/tools/process_registry.py:1-21`, `465-690`: process registry and scoped runtime management.

## Transferable Patterns

AGH should build one registry contract that all tool sources normalize into: built-in Go tools, extension manifest tools, extension sidecar tools, MCP tools, and future dynamic tools.

AGH should separate toolset policy from tool definitions. Toolsets should be named bundles resolved recursively at list/dispatch time, not hardcoded into each provider.

AGH should fail closed during discovery when required env vars, binaries, MCP servers, or extension sidecars are missing. The discovery surface should explain why a tool is unavailable to operators, while the agent-visible surface should omit or mark tools according to policy.

AGH should route MCP tools through the same registry dispatch path as native tools. MCP is an adapter, not a separate tool universe.

AGH should treat dynamic refresh as a first-class event. Hermes' MCP `tools/list_changed` path is a useful precedent for a registry generation counter and catalog delta notification.

AGH should include result budgets and transformation at the registry boundary. Large outputs should be persisted or summarized consistently rather than left to individual handlers.

## Risks / Mismatches

Hermes relies heavily on a process-wide singleton and import-time registration. AGH should prefer explicit composition-root registration because daemon boot already wires skills, extensions, resource stores, hooks, API services, and session managers.

Hermes uses permissive in-process plugin execution. AGH should not load third-party executable handlers into the daemon process for MVP. Out-of-process MCP or extension sidecar execution is a better fit for AGH's security and observability model.

Hermes availability filtering is strong for model-visible definitions, but AGH must also recheck availability at dispatch. Discovery filtering alone is not a security boundary.

Hermes has some name-prefix and collision behavior around MCP tools. AGH should use structured namespaces and provenance instead of deriving security meaning from string prefixes.

Hermes' plugin dispatch split is a warning. If AGH has CLI, HTTP, UDS, session, MCP, and extension entry points, all of them must call the same `Registry.Call` pipeline.

## Open Questions

1. Should AGH expose unavailable tools to operators with reasons while hiding them from model-visible surfaces?
2. Should AGH support recursive toolsets as resources, config fields, or both?
3. Should MCP `tools/list_changed` cause live session deltas, or only refresh the next model-visible catalog query?
4. What conflict policy should AGH use when multiple providers contribute the same short tool name?

## Evidence

- Hermes registry: `.resources/hermes/tools/registry.py:23-64`, `176-228`, `260-327`, `352-433`.
- Hermes request pipeline: `.resources/hermes/model_tools.py:141-705`.
- Hermes toolsets: `.resources/hermes/toolsets.py:483-692`.
- Hermes MCP adapter: `.resources/hermes/tools/mcp_tool.py:860-1296`, `1850-2108`, `2508-2770`.
- Hermes plugin loader/dispatch: `.resources/hermes/hermes_cli/plugins.py:210-380`, `518-646`.
- Hermes dynamic discovery tests: `.resources/hermes/tests/tools/test_mcp_dynamic_discovery.py:1-160`.
- Hermes process registry: `.resources/hermes/tools/process_registry.py:1-21`, `465-690`.
