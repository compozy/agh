# Competitor Analysis: OpenClaw Tool and Plugin Registry

## Overview

OpenClaw's strongest contribution is its two-phase extension model: manifest-first static discovery followed by runtime materialization. Plugin manifests can declare tool contracts, configuration, activation hints, and ownership metadata without immediately executing plugin code. Runtime tool registration then materializes concrete tools through plugin APIs or MCP adapters.

This is highly relevant to AGH because AGH already lets extension manifests publish static `resources.tools`, but lacks a runtime execution boundary and a registry that can connect static declarations to callable backends.

## Mechanisms / Patterns

OpenClaw reads plugin manifests first. Static `contracts.tools`, config schema, activation hints, ownership, and duplicate diagnostics are available before runtime code is loaded. This supports cheap discovery, trust decisions, and policy projection.

Runtime tools are registered through `api.registerTool` as either concrete tools or factories receiving a plugin tool context. This avoids global singleton state and gives each tool access to scoped runtime services.

Tool materialization is policy-filtered. Core and plugin tools are assembled, then filtered by profile/provider/global/agent/group/sandbox/subagent rules. Plugin owner metadata supports policy by plugin id or broad plugin groups.

OpenClaw treats MCP as a provider backend. Bundle/user MCP configs are connected over stdio, SSE, or streamable HTTP, tools are listed and sanitized, calls are wrapped, and per-session runtime instances have idle TTL and fingerprint invalidation.

OpenClaw also exposes a direct HTTP gateway for tool invocation. This is useful as a manageability pattern, but AGH should only expose direct invoke through strict local authorization and the same registry dispatch pipeline.

## Relevant Code Paths

- `.resources/openclaw/src/plugins/manifest.ts:250-367`: manifest contract shape.
- `.resources/openclaw/src/plugins/manifest.ts:539-583`: manifest validation and duplicate handling.
- `.resources/openclaw/src/plugins/manifest.ts:1161-1251`: manifest registry integration.
- `.resources/openclaw/src/plugins/manifest-registry.ts:303-379`: plugin manifest discovery.
- `.resources/openclaw/src/plugins/manifest-registry.ts:640-805`: precedence/diagnostics behavior.
- `.resources/openclaw/src/plugins/types.ts:2209-2353`: plugin API and tool registration types.
- `.resources/openclaw/src/plugins/tool-types.ts:8-45`: tool type definitions.
- `.resources/openclaw/src/plugins/registry.ts:421-446`: plugin registry access patterns.
- `.resources/openclaw/src/plugins/registry.ts:1464-1557`: activation/materialization path.
- `.resources/openclaw/src/plugins/tools.ts:111-239`: tool assembly from plugins.
- `.resources/openclaw/src/agents/pi-tools.ts:585-656`, `673-727`: policy-filtered agent tool projection.
- `.resources/openclaw/src/agents/tool-policy.ts:102-164`: tool policy model.
- `.resources/openclaw/src/agents/tool-policy-pipeline.ts:36-147`: policy pipeline.
- `.resources/openclaw/src/agents/pi-bundle-mcp-runtime.ts:181-575`: MCP runtime sessions and lifecycle.
- `.resources/openclaw/src/agents/pi-bundle-mcp-materialize.ts:64-174`: MCP tool materialization.
- `.resources/openclaw/docs/gateway/tools-invoke-http-api.md:11-146`: direct tool invocation gateway.

## Transferable Patterns

AGH should keep manifest-first discovery. Extension `resources.tools` should remain static and cheap to inspect. Runtime code should not be needed to list declared tools.

AGH should attach structured owner/provenance metadata to every tool: source kind, source id, namespace, extension id, MCP server id, bundle id, trust tier, and conflict state.

AGH should materialize tools through factories/adapters with a scoped context rather than globals. Built-ins receive daemon services; MCP tools receive server clients; extension tools receive a Host API or sidecar client.

AGH should support extension-level grants and expand them into explicit tool permissions. "Allow extension X" should resolve to the tool ids owned by extension X at a specific registry generation.

AGH should isolate failures. If an extension sidecar is unhealthy, its tools should become unavailable with reasons without breaking the full registry.

AGH should cache materialized context-specific views. Tool projection can be hot-path work for session starts, catalog queries, and live deltas.

AGH should expose direct invocation only through the same registry dispatch pipeline and only on local trusted surfaces such as UDS/daemon-authenticated HTTP.

## Risks / Mismatches

OpenClaw manifests are broad. AGH should keep the MVP manifest addition small and avoid a general plugin DSL inside the tool registry workstream.

Some OpenClaw discovery modes may still execute plugin code. AGH should make manifest-first discovery a hard rule for untrusted extensions.

Global tool names create collision pressure. AGH should require stable namespaced ids and optionally expose short display names.

`optional: true` style availability is too coarse for AGH. Availability should distinguish not installed, disabled, unauthorized, unhealthy, dependency missing, config missing, sandbox denied, and conflict.

Trusted bundled-only policies are insufficient for AGH's extension story. Trust tier and execution boundary should both be explicit.

## Open Questions

1. Should AGH extension tools declare a backend kind in the manifest, such as `mcp`, `host_api`, or `subprocess`?
2. Should extension-owned tools be disabled until an operator grants the extension's requested tool family?
3. Should AGH allow extension-private tools that only that extension can call?
4. How should AGH represent duplicate names: hard error, namespaced id only, or visible conflict diagnostics?

## Evidence

- `.resources/openclaw/src/plugins/manifest.ts:250-367`, `539-583`, `1161-1251`: manifest-first contracts and validation.
- `.resources/openclaw/src/plugins/manifest-registry.ts:303-379`, `640-805`: discovery and precedence diagnostics.
- `.resources/openclaw/src/plugins/types.ts:2209-2353`: runtime plugin tool registration API.
- `.resources/openclaw/src/plugins/tool-types.ts:8-45`: tool type shape.
- `.resources/openclaw/src/plugins/tools.ts:111-239`: plugin tool assembly.
- `.resources/openclaw/src/agents/pi-tools.ts:585-656`, `673-727`: agent projection.
- `.resources/openclaw/src/agents/tool-policy.ts:102-164`: policy model.
- `.resources/openclaw/src/agents/tool-policy-pipeline.ts:36-147`: policy pipeline.
- `.resources/openclaw/src/agents/pi-bundle-mcp-runtime.ts:181-575`: MCP runtime.
- `.resources/openclaw/src/agents/pi-bundle-mcp-materialize.ts:64-174`: MCP materialization.
- `.resources/openclaw/docs/gateway/tools-invoke-http-api.md:11-146`: direct invoke gateway.
