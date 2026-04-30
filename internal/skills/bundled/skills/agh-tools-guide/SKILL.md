---
name: agh-tools-guide
description: Use AGH-native tool discovery and invocation correctly. Teaches agents to start with agh__tool_search, read descriptors with agh__tool_info, invoke dedicated AGH tools before shelling out to agh commands, and keep operator-only lifecycle, OAuth, trust-root, and raw-secret flows on management surfaces.
version: "1.0.0"
---

# AGH Tools Guide

Use this guide when deciding how to discover or call AGH runtime capabilities.

## Operating model

AGH exposes runtime capabilities through a policy-filtered tool registry. The registry is the default agent path for AGH-internal operations because tool calls are structured, policy-aware, observable, and easier for the daemon to redact and audit.

Every agent receives the discovery toolsets `agh__bootstrap` and `agh__catalog` by default unless the effective runtime policy denies them.

## Discovery loop

Use this sequence for AGH-native work:

1. Search: call `agh__tool_search` with the runtime domain or operation you need.
2. Inspect: call `agh__tool_info` for the selected ToolID before the first invocation.
3. Invoke: call the dedicated tool with the descriptor's input schema.
4. Diagnose: when a tool is missing or denied, read the descriptor diagnostics and reason codes before choosing another surface.

For skills, use the same tool-first path:

1. Search skills with `agh__skill_search` when you do not know the exact skill name.
2. Load the full skill with `agh__skill_view`.
3. Read a referenced resource through `agh__skill_view` when the descriptor or skill text points to one.

## Tool-first convention

Prefer AGH-native tools over shelling out to equivalent `agh ...` commands when a dedicated tool exists and is callable in the current policy scope.

Examples:

- Use `agh__tool_search` and `agh__tool_info` to discover registry capabilities.
- Use `agh__skill_view` to load a bundled or workspace skill.
- Use `agh__task_list`, `agh__task_read`, or `agh__task_create` for task metadata and authoring.
- Use `agh__network_peers` and `agh__network_send` for supported network coordination when those tools are visible.

Shell commands remain useful for ordinary repository work and for AGH management flows that are intentionally outside the normal tool-call loop.

## Management-surface exceptions

Keep these on operator management surfaces such as CLI, HTTP, or UDS unless the user explicitly asks for a structured tool path and the registry exposes one:

- daemon lifecycle, destructive runtime repair, socket, host, port, sandbox, and provider bootstrap settings
- creating, stopping, force-stopping, or mutating arbitrary sessions outside your own scoped authority
- MCP OAuth login/logout and browser-based auth flows
- trust-root changes, raw secrets, OAuth credentials, provider API-key bindings, PKCE material, and MCP auth config secrets
- cross-session terminal-state mutation

Use status or read-only tools to inspect these domains when available, but do not invent a tool invocation for an operation the registry does not expose.

## Failure handling

If discovery returns no result, vary the query once with the domain name and once with the action name. If the tool exists but is denied, treat the reason codes as authoritative and choose a narrower action or an operator surface. Do not retry the same denied call unchanged.
