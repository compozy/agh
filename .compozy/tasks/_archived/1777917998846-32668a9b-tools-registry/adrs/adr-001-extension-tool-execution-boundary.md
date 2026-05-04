# ADR-001: Extension Tool Execution Boundary

## Status

Accepted

## Date

2026-04-28

## Context

AGH's Tool Registry must let operators and agents create real tools through the same extension model AGH already uses for hooks, memory backends, bridge adapters, and subprocess-hosted capabilities.

The previous interpretation of this ADR kept extension tools descriptor-only in the MVP. That is insufficient: it lets AGH list extension tools but does not let a TypeScript or Go extension author define a callable tool. It also conflicts with the current extension runtime, which already supports subprocess JSON-RPC, capability negotiation, Host API grants, health checks, service methods, and TypeScript `Extension.handle(...)` handlers.

The registry still must preserve daemon safety. Third-party extension code must not run in-process inside the daemon. Cold resource records must not persist function pointers or dynamic handler state.

## Decision

The Tool Registry MVP supports three executable backend classes:

- `native_go`: in-process Go function tools compiled into the daemon and registered by first-party/built-in providers at the daemon composition root.
- `extension_host`: out-of-process tools implemented by installed extensions through the existing subprocess JSON-RPC extension runtime.
- `mcp`: remote or local MCP tools called through daemon-owned MCP client adapters that consume existing MCP config and `internal/mcp/auth` redacted credential state.

Extension-host tools are first-class executable tools in the MVP. An extension declares `capabilities.provides = ["tool.provider"]`, publishes manifest-authoritative `resources.tools` descriptors, and implements the negotiated `tools/call` service method. The daemon only dispatches to an extension tool when the extension is enabled, active, healthy, negotiated `tool.provider`, implements `tools/call`, and passes registry policy, source grants, availability, hooks, and session lineage checks.

Third-party extension code never runs in the daemon process. Third-party Go tools use a subprocess Go SDK, not Go `plugin`, cgo-loaded shared libraries, reflection injection, or manifest function pointers.

## Alternatives Considered

### Descriptor-only extension tools

- **Description**: Extension manifests publish tool metadata, but calls return unavailable until a later TechSpec implements backend dispatch.
- **Pros**: Smaller MVP and easier security review.
- **Cons**: Fails the product requirement that extensions can create tools. Leaves TypeScript and Go extension authors with no executable tool path.
- **Why rejected**: The user explicitly rejected this as too weak, and subagent/codebase research confirmed existing extension runtime can support executable subprocess handlers.

### In-process third-party handlers

- **Description**: Let trusted extensions register Go or JavaScript handlers directly in the daemon process.
- **Pros**: Low call latency and a simple function-call programming model.
- **Cons**: Expands daemon compromise risk, creates a plugin ABI, complicates lifecycle isolation, and pressures AGH toward unsafe `plugin`/reflection behavior.
- **Why rejected**: Built-ins can use `native_go`; third-party extension code must cross a process/protocol boundary.

### TypeScript-only extension tools

- **Description**: Support executable TypeScript extension tools first; defer Go subprocess authoring.
- **Pros**: Reuses the existing TypeScript SDK quickly.
- **Cons**: Does not satisfy the requirement that extensions define tools using Go functions or TypeScript.
- **Why rejected**: The MVP must include a public Go subprocess SDK for tool providers.

## Consequences

### Positive

- Extension tools become real executable capabilities in the MVP.
- Built-in Go tools and third-party extension tools share one registry, policy path, hosted MCP exposure path, hook path, telemetry path, and result-budget path.
- The design reuses AGH's existing extension runtime instead of inventing a descriptor-only side channel.
- Remote MCP tools become governed by AGH policy and observability instead of living in provider-private tool universes.

### Negative

- MVP scope grows: registry execution, extension protocol, TypeScript SDK, Go SDK, MCP client call-through, redaction, and E2E coverage must ship together.
- Remote MCP call-through adds auth, transport, timeout, and result-normalization complexity.
- Extension descriptor/runtime reconciliation needs strict validation to avoid mismatch bugs.

### Risks

- Misclassified mutating tools could bypass approval expectations. Mitigation: descriptor validation, source grants, policy matrix tests, and dispatch-time revalidation.
- Extension tools could leak secrets through results or errors. Mitigation: central redaction and result limiting before crossing CLI, HTTP, UDS, MCP, SSE, logs, or events.
- MCP OAuth tokens could leak through registry views. Mitigation: consume only redacted `internal/mcp/auth` status and never copy token material into descriptors, resources, events, or responses.

## Implementation Notes

- `internal/tools` owns `native_go`, `extension_host`, and `mcp` backend contracts.
- `internal/daemon` wires first-party Go function providers and injects extension/MCP adapters.
- `internal/extension/protocol` adds `CapabilityProvideToolProvider = "tool.provider"` and `ExtensionServiceMethodToolsCall = "tools/call"`.
- `@agh/extension-sdk` adds `extension.tool(descriptor, handler)`.
- A new public Go subprocess SDK adds the equivalent Go helper for defining tools with Go functions.
- Hosted MCP remains the session exposure transport, but every call enters `internal/tools.Registry.Call`.

## References

- `.compozy/tasks/tools-registry/analysis/analysis_agh_current_state.md`
- `.compozy/tasks/tools-registry/analysis/synthesis.md`
- `internal/extension/manager.go`
- `internal/extension/protocol/host_api.go`
- `sdk/typescript/src/extension.ts`
- `internal/subprocess/handshake.go`
