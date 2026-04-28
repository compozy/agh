# Competitor Analysis: Claude Code Tool System

## Overview

Claude Code does not expose a single mutable "ToolRegistry" class in the generic plugin sense. Its effective registry is a layered system:

1. a strongly typed `Tool` contract,
2. a static built-in tool list,
3. request-time tool pool assembly,
4. MCP adaptation into the same contract,
5. deferred discovery through Tool Search,
6. centralized dispatch and permission handling.

For AGH, the main lesson is not Claude Code's static `tools.ts` shape. The useful pattern is the separation between canonical definition, contribution adapters, context-specific assembly, permission decision, dispatch, result mapping, and dynamic discovery.

## Mechanisms / Patterns

The `Tool` interface carries model identity, schemas, runtime behavior, permission hooks, safety metadata, rendering hooks, output mapping, and dynamic discovery flags. `buildTool` applies defaults for omitted fields.

Built-in tools are imported statically and exposed through `getAllBaseTools()`. `getTools(permissionContext)` filters this base list by mode, deny rules, special/internal tool rules, REPL visibility, and per-tool availability. `assembleToolPool()` then merges built-ins with MCP tools, sorts for prompt-cache stability, and deduplicates by name with built-ins winning.

MCP tools are adapted into the same `Tool` contract. Claude Code calls MCP `tools/list`, maps schemas and annotations, preserves server/tool provenance in `mcpInfo`, and refreshes on `tools/list_changed`.

Deferred discovery is a key pattern. Deferred tools are indexed by name/search hints and hidden from the initial API request. The model can call `ToolSearchTool`, which returns `tool_reference` blocks that cause the full schemas to be included later. Claude Code also has fallback delta/message paths when provider-native dynamic discovery is unavailable.

Permissions are evaluated in an ordered policy pipeline. Deny rules can hide tools from visibility and block dispatch. Runtime permission checks consider explicit allow/deny/ask rules, tool-specific checks, user interaction requirements, safety classification, headless behavior, mode bypasses, and hooks.

Dispatch is centralized. `runToolUse` resolves the tool, validates schema input, runs optional validation, prepares observable input, executes pre-tool hooks, resolves permission, calls the handler, maps results, and handles result-size/storage behavior. `StreamingToolExecutor` uses `isConcurrencySafe` to parallelize safe tools while serializing unsafe ones.

Plugins contribute tool-like behavior primarily through MCP servers, skills, commands, agents, hooks, and settings rather than native in-process `Tool` objects. This is a useful extension boundary for AGH because it keeps third-party tools behind a protocol/process adapter.

## Relevant Code Paths

- `.resources/claude-code/Tool.ts:362-520`: canonical `Tool` fields including schemas, `call`, availability, read-only/destructive/concurrency metadata, user-interaction requirements, deferred-loading flags, MCP metadata, validation, and permissions.
- `.resources/claude-code/Tool.ts:701-783`: `ToolDef`, defaults, and `buildTool`.
- `.resources/claude-code/tools.ts:161-193`: built-in tool source of truth.
- `.resources/claude-code/tools.ts:262-326`: runtime filtering by deny rules, special tools, REPL visibility, and `isEnabled`.
- `.resources/claude-code/tools.ts:345-389`: built-in + MCP tool pool assembly and deduplication.
- `.resources/claude-code/utils/api.ts:123-234`: conversion to API tool schemas and deferred-loading fields.
- `.resources/claude-code/utils/toolSearch.ts:155-197`: Tool Search modes.
- `.resources/claude-code/utils/toolSearch.ts:270-385`: provider/model capability checks for Tool Search.
- `.resources/claude-code/utils/toolSearch.ts:525-646`: discovered deferred tool extraction and delta computation.
- `.resources/claude-code/tools/ToolSearchTool/ToolSearchTool.ts:167-471`: search scoring and `tool_reference` outputs.
- `.resources/claude-code/services/api/claude.ts:1120-1339`: request-time Tool Search enablement and fallback injection.
- `.resources/claude-code/services/tools/toolExecution.ts:340-1297`: central dispatch.
- `.resources/claude-code/services/tools/streamingToolExecutor.ts:35-391`: concurrency-safe scheduling.
- `.resources/claude-code/utils/permissions/permissions.ts:236-362`, `1067-1312`: rule matching and ordered permission engine.
- `.resources/claude-code/services/mcp/client.ts:1738-2010`, `2160-2195`, `3020-3075`: MCP tools/list ingestion, adaptation, refresh, and call execution.
- `.resources/claude-code/services/mcp/useManageMCPConnections.ts:600-690`: `tools/list_changed` handling.
- `.resources/claude-code/utils/plugins/mcpPluginIntegration.ts:100-634`: plugin MCP server extraction, scoping, env resolution, and contribution adapter.
- `.resources/claude-code/types/plugin.ts:14-67`: plugin shape.
- `.resources/claude-code/skills/loadSkillsDir.ts:185-335`: skill frontmatter, `allowed-tools`, and user-invocable visibility.
- `.resources/claude-code/tools/AgentTool/runAgent.ts:440-690`: agent-specific permission scoping and merged tool pools.

## Transferable Patterns

AGH should define a first-class tool definition contract that attaches identity, schema, provenance, visibility, availability, permission requirements, dispatch handler, output policy, and observability metadata.

AGH should treat tool pool assembly as separate from registration. Registration collects contributions; assembly produces a context-specific view for a workspace, user, agent, session, provider, mode, runtime health, and permission policy.

AGH should model visibility explicitly instead of scattering "hidden" flags. Candidate tiers include internal-only, daemon-manageable, CLI/HTTP-visible, agent-visible, model-visible, deferred-discoverable, user-command-only, and extension-private.

AGH should support provider-neutral search and deferred schema loading even if Anthropic-specific `tool_reference` blocks are not portable. A registry search index and `agh__tool_search` are useful independently.

AGH should keep permissions as an ordered pipeline. Discovery-time filtering improves UX, but dispatch must revalidate deny/ask/allow rules, tool-specific requirements, session permissions, workspace permissions, and hooks.

AGH should map MCP annotations into local metadata at adapter boundaries: read-only, destructive, open-world, title, schema, search hints, and provenance.

AGH should carry `IsConcurrencySafe` or equivalent execution metadata and enforce it centrally.

## Risks / Mismatches

Claude Code makes LLM API calls directly and can use provider-specific dynamic tool features. AGH usually manages external ACP-compatible runtimes, so it cannot assume direct control of model API request payloads.

Claude Code's built-in registry is a static import list. AGH needs an extensible daemon registry with contribution adapters, not a monolithic static list.

`buildTool` defaults some permission behavior in a way that is acceptable for controlled built-ins but too permissive for third-party extension tools. AGH should default untrusted external tools to disabled, deny, or ask until policy grants them.

Claude Code uses naming conventions such as MCP prefixes for some behavior. AGH should use structured provenance and namespaces instead.

Claude Code's plugin tools primarily flow through MCP. This is a good MVP boundary, but AGH may also want a native extension Host API for richer lifecycle and local-resource management.

## Open Questions

1. Should AGH expose native registry tools to ACP runtimes as hosted MCP tools, direct ACP tools, or CLI/UDS-callable commands?
2. Should AGH persist deferred-tool discovery state per session, or keep search stateless and recompute on each call?
3. What should happen when an ACP runtime already has a tool with the same short name as an AGH-native tool?
4. Should extension/plugin tool contributions be MCP-only for MVP?

## Evidence

- `.resources/claude-code/Tool.ts:362-520`, `701-783`: canonical tool shape and defaults.
- `.resources/claude-code/tools.ts:161-389`: base registry, filtering, MCP merge, sorting, dedupe.
- `.resources/claude-code/utils/api.ts:123-234`: schema projection and deferred-loading fields.
- `.resources/claude-code/utils/toolSearch.ts:155-646`: dynamic discovery logic.
- `.resources/claude-code/tools/ToolSearchTool/ToolSearchTool.ts:167-471`: search scoring and references.
- `.resources/claude-code/services/tools/toolExecution.ts:340-1297`: central dispatch and result mapping.
- `.resources/claude-code/services/tools/streamingToolExecutor.ts:35-391`: concurrency-safe scheduling.
- `.resources/claude-code/utils/permissions/permissions.ts:236-362`, `1067-1312`: permission rule engine.
- `.resources/claude-code/services/mcp/client.ts:1738-2010`, `2160-2195`, `3020-3075`: MCP adapter.
- `.resources/claude-code/utils/plugins/mcpPluginIntegration.ts:100-634`: plugin-to-MCP contribution adapter.
