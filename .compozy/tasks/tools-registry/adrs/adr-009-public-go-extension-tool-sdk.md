# ADR-009: Public Go Extension Tool SDK

## Status

Accepted

## Date

2026-04-28

## Context

The Tool Registry must support extension authors defining tools using Go functions or TypeScript. AGH already has a TypeScript extension SDK, but Go extension authoring currently relies on direct protocol implementation or internal packages such as `internal/bridgesdk`, which are not public extension-author surfaces.

Built-in daemon tools can use in-process Go functions, but third-party Go extension tools must remain out-of-process for safety and lifecycle isolation.

## Decision

The Tool Registry MVP includes a public Go subprocess extension SDK for tool providers.

The SDK exposes an authoring API equivalent to TypeScript `extension.tool(descriptor, handler)`, but implemented as a Go subprocess runtime. It handles initialize/shutdown, health checks, `tool.provider` negotiation, `tools/call` dispatch, Host API client calls, typed errors, descriptor export for runtime reconciliation, and test harness support.

In-process `native_go` remains limited to first-party/built-in tools compiled into the daemon and wired by `internal/daemon`. Third-party Go extension tools use the public Go SDK and execute as managed subprocesses.

## Alternatives Considered

### Protocol examples only

- **Description**: Ship Go examples that implement JSON-RPC manually, without a reusable SDK.
- **Pros**: Smaller implementation.
- **Cons**: Weak developer experience and inconsistent correctness across extension authors.
- **Why rejected**: The user explicitly requested Go function authoring, not raw protocol plumbing.

### Go `plugin` or shared-library handlers

- **Description**: Load Go functions into the daemon process dynamically.
- **Pros**: Natural function-call authoring.
- **Cons**: Unsafe daemon process boundary, platform friction, ABI issues, and lifecycle complexity.
- **Why rejected**: Third-party code must stay out-of-process.

### TypeScript-only extension authoring

- **Description**: Support TypeScript tool handlers first and defer Go SDK work.
- **Pros**: Reuses existing SDK.
- **Cons**: Does not satisfy Go function authoring requirements.
- **Why rejected**: Go function authoring is in MVP scope.

## Consequences

### Positive

- Go extension authors get a real function-based authoring path without compromising daemon isolation.
- TypeScript and Go SDKs share the same runtime protocol and test expectations.
- E2E tests can prove both language paths through the same registry dispatch pipeline.

### Negative

- MVP scope includes a new SDK package, scaffolding, tests, docs, and generated contract parity.
- Public SDK design must be stable enough for extension authors.

### Risks

- SDK and daemon protocol can drift. Mitigation: generate shared contract types and include cross-SDK conformance tests.
- Go SDK may accidentally import `internal/*` packages. Mitigation: place it under a public package path and test from an external-package perspective.

## Implementation Notes

- Add a public Go SDK package under `sdk/go`, mirroring `sdk/typescript`.
- Add a create-extension template for Go tool provider extensions.
- Add a Go SDK harness that can load a tool extension, mock Host API calls, and call `tools/call`.
- Do not use the SDK for daemon built-ins; built-ins use `native_go` providers.

## References

- `internal/subprocess/handshake.go`
- `internal/extension/manager.go`
- `sdk/typescript/src/extension.ts`
- `internal/bridgesdk/runtime.go`
