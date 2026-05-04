# ADR-007: Canonical Tool ID Format

## Status

Accepted.

## Context

AGH needs one stable tool identifier that can be used across the runtime registry, policy rules, CLI, HTTP, UDS, telemetry, hooks, audit logs, and the AGH-hosted MCP surface.

Earlier options considered dotted internal IDs such as `agh.skill.view` plus a separate MCP-safe wire alias such as `agh_skill_view`. ACP/MCP compatibility research showed this would work technically, but it creates two strong names for the same tool and increases the chance of policy, audit, telemetry, or dispatch code using the wrong identity.

The identifier also needs to remain compatible with provider and host constraints. MCP allows dots in tool names, but common model tool/function APIs are stricter and accept letters, digits, underscores, and hyphens with a 64-character limit. AGH should choose a lowest-common-denominator format for callable tool IDs rather than rely on a more permissive protocol layer.

Claude Code uses the `mcp__server__tool` convention for MCP tools. This gives an explicit namespace boundary while staying inside provider-safe characters.

## Decision

AGH will use one canonical public `ToolID` format across every surface:

```text
<segment>( "__" <segment> )*
```

Each segment must match:

```text
[a-z][a-z0-9_]*
```

Global constraints:

- maximum length: 64 characters;
- lowercase ASCII only;
- digits allowed after the first character of each segment;
- `_` allowed inside a segment;
- `__` is reserved exclusively as a segment separator;
- no dot;
- no hyphen;
- no uppercase;
- no empty segment;
- no leading or trailing `_` inside a segment if it would create an empty separator ambiguity.

Examples:

```text
agh__tool_list
agh__tool_search
agh__tool_info
agh__skill_list
agh__skill_view
agh__network_peers
agh__network_send
agh__task_list
agh__task_read
ext__linear__search
ext__linear__create_issue
mcp__github__create_issue
mcp__context7__query_docs
```

`ToolID` is the identity used by:

- registry descriptors;
- provider registration;
- policy allow/deny rules;
- toolsets;
- CLI commands;
- HTTP and UDS APIs;
- hooks;
- telemetry and audit logs;
- hosted MCP `Tool.name`;
- dispatch requests.

AGH will not use a second wire alias for the same tool in the MVP. Display titles are UI-only and do not participate in policy, authorization, conflict resolution, or dispatch.

Source/provenance remains structured metadata, not an alternate identity:

```json
{
  "id": "mcp__github__create_issue",
  "source": {
    "kind": "mcp",
    "serverName": "github",
    "rawToolName": "create_issue"
  }
}
```

AGH may show a shorter display title such as `Create Issue`, but the canonical ID remains `mcp__github__create_issue`.

## Collision Rules

Registration and session projection must fail closed:

- If two providers produce the same `ToolID`, the later registration is rejected or marked `conflicted`.
- If sanitizing an external MCP/server/extension tool name would collide with an existing `ToolID`, the candidate tool is marked `conflicted` and is not exposed to model-visible surfaces.
- AGH must not silently truncate, overwrite, or choose "last writer wins".
- Operator surfaces may show conflicted tools with reason codes and provenance.
- Session/model-visible surfaces expose only non-conflicted callable tools.

## Consequences

Positive:

- One identifier works across registry, wire, policy, telemetry, and dispatch.
- No dotted-to-wire alias mapping is needed in the MVP.
- Namespace boundaries remain visible through reserved `__`.
- The format is compatible with stricter provider tool-name constraints.
- Policy patterns stay simple, for example `agh__skill_*` and `mcp__github__*`.

Tradeoffs:

- Dotted names such as `agh.skill.view` are more visually familiar for namespace trees, but they are not provider-safe enough to use as callable IDs.
- `__` is less aesthetically clean than dots, but it avoids dual identity.
- Raw external names must be preserved in `SourceRef` for exact provenance and debugging.

## Follow-Ups

- The TechSpec must update all tool examples to this format.
- The registry validator must enforce the grammar.
- Extension and MCP adapters must sanitize external names deterministically and report conflicts.
- Policy matching must treat `__` as an identity segment separator and `_` as a normal segment character.
