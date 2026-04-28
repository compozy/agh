---
status: pending
title: Extension Manifest Tool Metadata and Reconciliation
type: backend
complexity: high
dependencies:
  - task_03
---

# Task 06: Extension Manifest Tool Metadata and Reconciliation

## Overview

Make extension tool descriptors manifest-authoritative while preparing them for executable runtime reconciliation. This task extends extension metadata, cold resource publication, schema digest fixtures, and availability reasons without invoking extension handlers yet.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-008, and ADR-009 before changing extension tool metadata
- DO NOT revive descriptor-only executable behavior; descriptors become callable only after runtime reconciliation
- DO NOT let extensions claim reserved `agh__*` IDs or omit handler binding metadata
- TESTS REQUIRED: manifest, digest, mismatch, disabled, and lifecycle behavior must be covered
</critical>

<requirements>
1. MUST extend `extension.toml` `resources.tools` with canonical ID, backend metadata, handler binding, risk flags, result budgets, schemas, and toolset metadata.
2. MUST keep manifest descriptors authoritative while allowing runtime descriptors to prove they match.
3. MUST add RFC 8785/JCS-compatible schema digest fixtures shared by daemon, TypeScript SDK, and Go SDK tasks.
4. MUST publish cold tool resources without marking them executable until runtime reconciliation succeeds.
5. MUST surface mismatch, missing handler, disabled extension, inactive extension, and reserved namespace reason codes.
6. MUST update extension manifest validation and resource publication tests.
</requirements>

## Subtasks
- [ ] 6.1 Extend extension manifest tool metadata and validation
- [ ] 6.2 Split cold resource publication from executable descriptor readiness
- [ ] 6.3 Add schema digest fixture files and daemon digest tests
- [ ] 6.4 Add reconciliation reason codes for manifest/runtime mismatch states
- [ ] 6.5 Reject reserved namespaces and invalid handler bindings
- [ ] 6.6 Add extension lifecycle tests for enabled, disabled, removed, and unhealthy extensions

## Implementation Details

Use TechSpec "Extension Runtime Contract", "Data Models", and ADR-008. Keep this task manifest-side only; the subprocess protocol and SDK handlers are task_07 and task_08.

### Relevant Files
- `internal/extension/manifest.go` - extension tool metadata fields and validation
- `internal/extension/resource_publication.go` - cold resource publication behavior
- `internal/extension/capability*.go` - extension capability metadata if required
- `internal/tools/descriptor*.go` - descriptor/digest contracts consumed by extensions
- `internal/extension/*_test.go` - manifest and resource publication coverage

### Dependent Files
- `internal/extension/protocol/host_api.go` - task_07 adds runtime tool provider protocol
- `sdk/typescript/src/types.ts` - task_07 mirrors manifest/runtime descriptors
- `sdk/go/**` - task_08 mirrors descriptor and digest rules
- `sdk/create-extension/src/index.ts` - task_07/task_08 add templates

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - separates first-party native tools from out-of-process extension tools
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - defines manifest/runtime reconciliation
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - constrains Go SDK compatibility with manifest contracts

### Web/Docs Impact
- `web/`: task_13 must display extension tool availability, disabled/unhealthy states, and mismatch reasons through generated tool diagnostics.
- `packages/site`: task_14 must update extension authoring docs to explain manifest-authoritative descriptors and executable reconciliation.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: changes extension manifests, tool resources, schema digest contracts, and extension registry behavior.
- Agent manageability: exposes readiness/reason data later through CLI/HTTP/UDS in tasks 11-12.
- Config lifecycle: consumes extension enable/disable lifecycle; no new top-level config keys beyond task_02.

## Deliverables
- Extended extension tool manifest schema and validation
- Cold-resource to executable-descriptor reconciliation metadata
- Shared schema digest fixtures for daemon/SDK parity
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for extension resource publication lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Valid manifest tool entries produce canonical descriptors and digest metadata
  - [ ] Reserved `agh__*`, invalid ToolID, missing handler, invalid risk, and non-object schema entries fail validation
  - [ ] Runtime mismatch placeholders surface deterministic availability reasons
  - [ ] Disabled and unhealthy extensions remain operator-visible but session-hidden
- Integration tests:
  - [ ] Existing extension resource publication still works for non-tool resources
  - [ ] Tool resources publish cold metadata without creating callable handles before task_07 runtime reconciliation
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Extension tools are manifest-authoritative but not descriptor-only callable
- Digest fixtures are ready for TypeScript and Go SDK parity tests
