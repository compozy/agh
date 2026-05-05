# ADR-005: ACP Approval Policy Integration

## Status

Accepted

## Context

AGH already has a system-level ACP tool approval policy exposed in settings and enforced by the ACP tool host:

- `deny-all`
- `approve-reads`
- `approve-all`

The Tool Registry will add per-tool metadata and policy, including read-only, destructive, open-world, source, toolset, session permission atoms, and extension/MCP grants. If this registry policy is designed as a parallel approval system, AGH can produce contradictory states such as:

- system policy says `deny-all`, but a tool-level policy says allow;
- system policy says `approve-reads`, but a mutating tool claims read-only;
- system policy says `approve-all`, but a session/agent intentionally narrows permissions;
- hosted MCP exposes a tool the ACP host would later block.

The registry policy must integrate with the existing ACP policy rather than bypass it.

## Decision

The ACP `permissions.mode` policy is the system approval ceiling for session-visible tool execution.

Tool Registry policy operates below that ceiling as a more granular filter. It can narrow, classify, require approval, or deny a tool, but it cannot silently grant more authority than the effective system/session ACP policy allows.

The effective decision order is:

1. Resolve the system/session ACP approval mode.
2. Resolve agent/session lineage tool permission atoms.
3. Resolve registry visibility and allow/deny/toolset policy.
4. Resolve source/risk defaults for built-in, extension, MCP, and dynamic tools.
5. Resolve tool descriptor risk flags: read-only, destructive, open-world, requires interaction.
6. Run availability checks.
7. Run pre-call hooks.
8. Dispatch only if the combined decision is allowed or explicitly approved.

`approve-all` is permissive but not a bypass of explicit denies. It removes automatic ACP prompting for allowed tools, but registry deny rules, unavailable state, session lineage restrictions, hooks, and source grants still apply.

`approve-reads` auto-approves only tools classified as read-only by the registry and allowed by session policy. Extension and MCP read-only tools also require an explicit trusted source or per-tool/toolset/source/agent grant because their read-only claim is externally authored. Mutating, destructive, open-world, network-send, task-write, extension-host write, and MCP write tools require explicit allow by `ToolID`, toolset, or source policy plus approval when the effective approval mode requires it.

`deny-all` denies by default. The registry may still list operator-visible tools with reasons, but session-visible execution requires explicit approval/allowance through the existing ACP approval path or an equivalent local approval surface.

Agent frontmatter and session lineage can lower permissions relative to system policy. They cannot raise permissions above the system approval ceiling.

The MVP permits mutating, open-world, and destructive extension-host and MCP tools, but only when all gates pass:

1. The descriptor classifies `read_only`, `destructive`, `open_world`, and `requires_interaction` correctly.
2. The source tier is allowed for the effective workspace/session.
3. The concrete `ToolID` or expanded toolset is allowed by registry/session policy.
4. ACP `permissions.mode` does not deny the call.
5. The approval bridge succeeds within the configured timeout when approval is required.
6. The backend is available, healthy, authorized, and non-conflicted.
7. Hooks do not deny or narrow the call.
8. Dispatch revalidates all gates immediately before execution.

## Consequences

The registry must compute an `EffectiveToolDecision` instead of storing a single boolean. The decision should include:

- system approval mode,
- session/agent policy result,
- registry policy result,
- source/risk default result,
- availability result,
- hook result,
- final visibility decision,
- final execution decision,
- user/operator-facing reason codes.

The hosted AGH MCP server must expose only tools allowed by the effective visibility decision for that session. Dispatch must still revalidate the effective execution decision.

Tool descriptors must classify read-only vs mutating accurately. A mutating, destructive, or open-world tool mislabeled as read-only is a correctness and security bug.

The web settings UI text remains true: `approve-all` auto-approves tool calls, but agents and registry policy can lower permissions. The TechSpec should clarify that "auto-approved" does not mean "all registered tools are visible and executable regardless of registry policy."

## Rejected Alternatives

### Registry policy bypasses ACP policy

This would create inconsistent behavior between ACP-native tools, AGH-hosted MCP tools, and CLI/UDS calls. It would also make the existing settings UI misleading.

### Registry policy replaces ACP policy

This would require redesigning existing ACP permission handling and settings before the Tool Registry can ship. The MVP should integrate with the existing model and extend it.

### Tool-level policy alone controls execution

Per-tool policy is necessary but insufficient. Session lineage, system approval mode, hooks, availability, and source grants all affect whether a call is safe and authorized.

Daemon-mediated approval waits are bounded by `[tools.policy].approval_timeout_seconds`. Timeout and caller/proxy disconnect fail closed with deterministic reason codes; no registry tool call may wait indefinitely on an operator or ACP permission response.

## Evidence

- `internal/acp/permission.go:75-132`: ACP policy defaults, path validation, and decisions for `approve-all`, `approve-reads`, and `deny-all`.
- `internal/acp/tool_host.go:64-83`: local tool host is constructed with an ACP permission mode.
- `web/src/routes/_app/settings/general.tsx:307-315`: settings UI describes the three tool approval policies.
- `.compozy/tasks/tools-registry/analysis/analysis_claude-code.md`: permission should be an ordered pipeline rather than a tool-local boolean.
- `.compozy/tasks/tools-registry/analysis/analysis_goclaw.md`: runtime grants should be rechecked at execution time.
- `.compozy/tasks/tools-registry/analysis/synthesis.md`: dispatch must recheck availability and authorization and use one central pipeline.
