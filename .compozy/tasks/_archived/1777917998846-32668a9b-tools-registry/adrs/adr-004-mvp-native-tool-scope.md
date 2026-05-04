# ADR-004: MVP Tool Scope

## Status

Accepted

## Context

The Tool Registry MVP must prove more than metadata listing. It must demonstrate that AGH-owned tools can be discovered, authorized, invoked through the hosted MCP surface, called through CLI/HTTP/UDS contracts, routed through one dispatch pipeline, and observed.

The smallest read-only bootstrap toolset would prove catalog mechanics, but it would not prove meaningful autonomy. AGH's product direction requires agents to manage coordination and task flows, so network and task tools should be represented in the first implementation slice.

At the same time, skill installation is a separate supply-chain surface. It requires stricter install policy, trust tiers, approval flows, and scanner decisions. It should not be bundled into the first registry execution proof unless the supply-chain work is explicitly scoped.

## Decision

The MVP tool scope includes four executable groups:

1. Built-in `native_go` tools for AGH catalog, skill, network, and bounded task operations.
2. Installed extension-host tools implemented through TypeScript or Go subprocess SDKs.
3. Remote/local MCP tools discovered from existing MCP config/resource sources and called through daemon-owned MCP clients.
4. The AGH-hosted MCP exposure proxy that presents the effective session projection for all callable groups.

The built-in `native_go` scope includes:

- `agh__tool_list`
- `agh__tool_search`
- `agh__tool_info`
- `agh__skill_list`
- `agh__skill_search`
- `agh__skill_view`
- `agh__network_peers`
- `agh__network_send`
- `agh__task_list`
- `agh__task_read`
- `agh__task_create`
- `agh__task_child_create`
- `agh__task_update`
- `agh__task_cancel`
- `agh__task_run_list`

Claim/release/complete/fail/run-start task operations are excluded from this MVP because they cross claim-token, lease, spawn, and session lifecycle authority. They require a separate task execution TechSpec.

Skill install/remove/update tools are not included in the MVP native tool scope unless a later decision explicitly adds the required supply-chain and approval work.

The MVP must also include executable proof fixtures:

- a TypeScript extension defining at least one read-only tool and one mutating tool through `extension.tool(...)`;
- a Go subprocess extension defining equivalent tools through the public Go extension SDK;
- an MCP test server with read-only and mutating tools, auth status coverage, and remote call-through.

## Consequences

The MVP must include both read-only and mutating tools. The registry must model risk, read-only/destructive/open-world flags, permission checks, and policy gates from the first implementation.

Native, extension-host, and MCP tools must use the same registry dispatch path as catalog tools. They must not call around policy, availability, hooks, result budgeting, auth redaction, or telemetry.

The hosted MCP server must expose only the tool subset authorized for the session. Agent-visible discovery must not advertise network/task tools to sessions that lack the required permission atoms.

QA must include real scenario coverage for:

- listing and searching tools,
- viewing a skill body through `agh__skill_view`,
- listing peers,
- sending a network message through `agh__network_send` with permission enforcement,
- creating/updating or otherwise exercising the bounded task tool set,
- invoking a TypeScript extension-host tool through CLI/HTTP/UDS/hosted MCP,
- invoking a Go SDK extension-host tool through CLI/HTTP/UDS/hosted MCP,
- invoking a remote MCP-backed tool through CLI/HTTP/UDS/hosted MCP,
- proving unauthorized sessions cannot see or call mutating/destructive tools.

## Rejected Alternatives

### Read-only bootstrap only

This would be safer and simpler, but it would leave the registry unproven for AGH's coordination and autonomy use cases.

### Bootstrap plus skill install

This would improve agent self-service, but it introduces supply-chain risk that belongs in a dedicated policy/scanning/install decision.

### Foundation only

This would create the architecture without proving the agent-first experience that motivated the Tool Registry work.

## Evidence

- `.compozy/tasks/tools-registry/analysis/synthesis.md`: recommends a small bootstrap set and identifies network/task tools as later groups.
- `.compozy/tasks/autonomous/analysis/analysis_skills_tools_registry.md`: prior gaps identify network and task tools as strategically important agent-callable surfaces.
- `.compozy/tasks/hermes/analysis/analysis_tools_security.md`: mutating and open-world tools require stronger permission and security gates before broad exposure.
