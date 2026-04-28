---
status: pending
title: Extension Runtime Protocol and TypeScript SDK Tools
type: backend
complexity: critical
dependencies:
  - task_04
  - task_06
---

# Task 07: Extension Runtime Protocol and TypeScript SDK Tools

## Overview

Add executable extension-host tool support through the existing subprocess extension model and TypeScript SDK. This task introduces `tool.provider`, runtime descriptor reconciliation, `provide_tools`, `tools/call`, and `extension.tool(...)` so TypeScript extensions define real handlers rather than descriptors only.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-008, and the existing extension subprocess code before editing protocol behavior
- DO NOT run third-party extension handlers in-process inside the daemon
- DO NOT add a parallel TypeScript runtime; extend the existing `Extension.handle(...)` and transport patterns
- TESTS REQUIRED: descriptor reconciliation and `tools/call` must be proven through real subprocess integration
</critical>

<requirements>
1. MUST add `tool.provider`, `provide_tools`, and `tools/call` protocol constants and request/response structs.
2. MUST require initialized extensions that declare `tool.provider` to implement all required service methods.
3. MUST reconcile runtime-provided descriptors with manifest-authoritative descriptors before marking tools executable.
4. MUST route extension tool invocation through the existing subprocess manager and central registry dispatch.
5. MUST add TypeScript SDK `extension.tool(...)` with typed handler registration, descriptor export, digest generation, and typed errors.
6. MUST add TypeScript tool-provider create-extension template and scaffold tests.
</requirements>

## Subtasks
- [ ] 7.1 Add extension protocol constants, structs, and initialization method coverage
- [ ] 7.2 Add manager-side `provide_tools` reconciliation and `tools/call` invocation
- [ ] 7.3 Add TypeScript SDK `extension.tool(...)` handler registration and descriptor export
- [ ] 7.4 Add TypeScript digest parity tests using task_06 fixtures
- [ ] 7.5 Add TypeScript create-extension tool-provider template and scaffold tests
- [ ] 7.6 Add subprocess integration tests for missing methods, schema mismatch, handler mismatch, handler errors, and cancellation

## Implementation Details

Use TechSpec "Extension Runtime Contract", "Core Interfaces", and "Implementation Steps" 7-9. Existing extension subprocess handshake and `Process.Call` behavior should be reused; do not create a separate protocol loop.

### Relevant Files
- `internal/extension/protocol/host_api.go` - protocol constants and method contracts
- `internal/subprocess/handshake.go` - required method validation during initialize
- `internal/subprocess/process.go` - existing JSON-RPC call path to reuse
- `internal/extension/manager.go` - manager-side runtime reconciliation and invocation
- `sdk/typescript/src/extension.ts` - SDK registration API
- `sdk/typescript/src/types.ts` - SDK type contracts
- `sdk/create-extension/src/index.ts` - template selection and generation

### Dependent Files
- `sdk/typescript/src/generated/contracts.ts` - generated/shared contracts if protocol types are code-generated
- `sdk/typescript/src/integration.test.ts` - subprocess integration coverage
- `sdk/create-extension/src/index.test.ts` - scaffold coverage
- `internal/api/contract/tools.go` - task_11 exposes extension availability and invocation results

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - executable extension tools remain out-of-process
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - defines runtime reconciliation
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - Go SDK must align with protocol contracts introduced here

### Web/Docs Impact
- `web/`: task_13 must display executable/unavailable extension-host tools and mismatch reasons from generated API types.
- `packages/site`: task_14 must document TypeScript `extension.tool(...)`, manifest/runtime reconciliation, error handling, and template usage.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: adds executable TypeScript extension tools, extension protocol methods, SDK APIs, and create-extension template.
- Agent manageability: extension-host tools become callable through CLI/HTTP/UDS in tasks 11-12 with deterministic errors.
- Config lifecycle: consumes extension enablement and tool policy from task_02; no additional config keys in this task.

## Deliverables
- Extension protocol support for `tool.provider`, `provide_tools`, and `tools/call`
- Runtime descriptor reconciliation and executable extension handles
- TypeScript SDK `extension.tool(...)` API and tests
- TypeScript tool-provider create-extension template
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests with real subprocess extension fixtures **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Initialize rejects `tool.provider` extensions missing `provide_tools` or `tools/call`
  - [ ] Runtime descriptor digest, risk, handler, and schema mismatches surface deterministic availability reasons
  - [ ] TypeScript `extension.tool(...)` registers descriptors and handlers with digest parity
  - [ ] SDK errors redact tool input fields marked sensitive
- Integration tests:
  - [ ] A TypeScript extension publishes a read-only executable tool and succeeds through `Registry.Call`
  - [ ] A TypeScript extension publishes a mutating tool and is gated by policy/approval
  - [ ] Cancellation propagates from registry dispatch through subprocess `tools/call`
  - [ ] `bun test` covers SDK and create-extension template behavior
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- TypeScript extensions can define executable tools with functions through the existing extension runtime
- Descriptor-only extension tools are not treated as callable
