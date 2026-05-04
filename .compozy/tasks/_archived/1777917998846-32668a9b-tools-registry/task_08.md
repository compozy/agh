---
status: completed
title: Public Go Extension SDK
type: backend
complexity: critical
dependencies:
  - task_07
---

# Task 08: Public Go Extension SDK

## Overview

Create a public Go SDK for out-of-process extension-host tools so Go authors can define executable tools with Go functions. This task mirrors the TypeScript extension SDK protocol without importing daemon internals or running third-party handlers in-process.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-008, and ADR-009 before creating SDK APIs
- DO NOT import `internal/*` from `sdk/go`; public SDK code must build as an external consumer
- DO NOT confuse first-party `native_go` providers with third-party Go extension tools
- TESTS REQUIRED: SDK conformance must prove protocol, digest, and subprocess behavior from an external package
</critical>

<requirements>
1. MUST create `sdk/go` with public APIs for defining tools using Go functions.
2. MUST implement the extension subprocess JSON-RPC runtime compatible with task_07 protocol constants.
3. MUST expose Host API client primitives needed by Go extension tools without depending on daemon internals.
4. MUST match daemon and TypeScript SDK descriptor/digest fixtures from task_06 and task_07.
5. MUST add Go tool-provider create-extension template support.
6. MUST test the SDK from an external-package perspective.
</requirements>

## Subtasks
- [x] 8.1 Create `sdk/go` package layout, public APIs, and module/test setup
- [x] 8.2 Implement Go subprocess runtime for initialize, `provide_tools`, and `tools/call`
- [x] 8.3 Add Go function-based `Tool(...)` registration and typed result/error helpers
- [x] 8.4 Add Host API client primitives required by tool handlers
- [x] 8.5 Add digest parity and conformance fixtures shared with daemon and TypeScript SDK
- [x] 8.6 Add Go create-extension tool-provider template and scaffold tests
- [x] 8.7 Add external-package tests proving no `internal/*` imports or daemon-only dependencies

## Implementation Details

Use TechSpec "Go Extension SDK Contract" and ADR-009. The SDK should behave like a public authoring surface equivalent to `@agh/extension-sdk`, while built-in daemon tools continue to use `native_go` providers from task_05.

### Relevant Files
- `sdk/go/**` - new public Go extension SDK
- `internal/extension/protocol/host_api.go` - protocol constants and wire contracts to mirror or generate from
- `sdk/typescript/src/extension.ts` - TypeScript authoring precedent
- `sdk/create-extension/src/index.ts` - template generation support
- `sdk/create-extension/src/index.test.ts` - scaffold tests

### Dependent Files
- `sdk/go/**/fixtures` - conformance and digest fixtures
- `sdk/typescript/test-fixtures/**` - shared digest vector parity if created by task_07
- `packages/site/content/runtime/core/extensions/develop.mdx` - task_14 documents Go SDK authoring
- `openapi/agh.json` - no direct dependency unless SDK docs link generated API surfaces

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - Go extension tools execute out-of-process
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - Go descriptors must reconcile with manifests
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - defines public SDK boundary

### Web/Docs Impact
- `web/`: no direct code impact - checked generated API and systems; Go SDK state appears through generic extension tool descriptors already handled by task_13.
- `packages/site`: task_14 must add Go extension tool authoring docs, examples, and template instructions.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: adds public Go extension SDK, Go authoring API, subprocess runtime, conformance fixtures, and create-extension template.
- Agent manageability: Go extension tools become callable through registry CLI/HTTP/UDS surfaces once installed and enabled.
- Config lifecycle: consumes existing extension config and tool policy; no new config keys in this task.

## Deliverables
- Public `sdk/go` extension SDK with function-based tool registration
- Go subprocess runtime compatible with TypeScript and daemon protocol contracts
- Go tool-provider create-extension template
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration/conformance tests using external-package SDK consumers **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `Tool(...)` rejects invalid IDs, schemas, duplicate handlers, and missing metadata
  - [x] Go digest output matches daemon and TypeScript RFC 8785/JCS fixtures
  - [x] Host API client redacts or rejects sensitive values according to protocol rules
  - [x] SDK package tests do not import daemon `internal/*`
- Integration tests:
  - [x] A compiled Go extension publishes and executes a read-only tool through the registry
  - [x] A compiled Go extension publishes a mutating tool gated by policy/approval
  - [x] Go create-extension template scaffolds a buildable tool-provider extension
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Go authors can define extension-host tools with Go functions through a public SDK
- Public SDK conformance is proven without daemon internal imports
