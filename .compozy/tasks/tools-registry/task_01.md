---
status: completed
title: Core Tool Contracts and Canonical ToolID
type: backend
complexity: high
dependencies: []
---

# Task 01: Core Tool Contracts and Canonical ToolID

## Overview

Establish the canonical runtime contracts for the Tool Registry before any executable backend is added. This task replaces the current cold metadata model with final-shape identifiers, descriptors, backend references, source references, result envelopes, availability states, and deterministic error reasons.

<critical>
- ALWAYS READ `_techspec.md` and every ADR in `adrs/` before starting; this feature has no `_prd.md`
- REFERENCE the TechSpec for interfaces and invariants instead of copying them into task-local designs
- PRESERVE the greenfield hard cut: no dotted IDs, no aliases, no legacy descriptor-only execution model
- TESTS REQUIRED: every validator and error reason introduced here must have focused coverage
</critical>

<requirements>
1. MUST define canonical `ToolID` validation using lower-snake segments separated by reserved `__`.
2. MUST enforce the TechSpec maximum length and fail with `id_too_long` instead of truncating or hash-suffixing.
3. MUST model descriptors, backend refs, source refs, availability, results, errors, risk flags, and provider/handle interfaces in `internal/tools`.
4. MUST preserve JSON-object input schema validation and result-size metadata without leaking backend secrets.
5. MUST keep `internal/tools` independent from daemon, API, CLI, extension, MCP, session, task, skill, and network packages.
6. MUST include compile-time interface assertions for exported providers/handles that cross package boundaries.
</requirements>

## Subtasks
- [x] 1.1 Replace the current `Tool{Name,...}` shape with final registry contracts in `internal/tools`
- [x] 1.2 Add `ToolID`, namespace, descriptor, backend, source, availability, result, and error validators
- [x] 1.3 Add provider and handle interfaces for `native_go`, `extension_host`, and `mcp` backends
- [x] 1.4 Preserve tool-resource schema validation while separating cold resources from executable descriptors
- [x] 1.5 Add deterministic reason constants used by policy, availability, dispatch, and public surfaces
- [x] 1.6 Add package-boundary-safe tests and compile-time assertions

## Implementation Details

Use the TechSpec sections "Core Interfaces", "Data Models", "Architectural Boundaries", and "Safety Invariants" as the source of truth. The existing `internal/tools` package is intentionally small and should become the stable dependency that other packages adapt to, not a package that imports those domains.

### Relevant Files
- `internal/tools/tool.go` - current cold metadata model to replace or reshape
- `internal/tools/resource.go` - existing tool resource schema validation to preserve
- `internal/tools/*_test.go` - new unit tests for IDs, descriptors, schemas, and errors
- `magefile.go` - package boundary checks if new internal packages or allowed imports change

### Dependent Files
- `internal/extension/resource_publication.go` - later tasks consume the cold-resource/executable-descriptor split
- `internal/mcp/auth/types.go` - later tasks map auth status into availability reasons without exposing tokens
- `internal/hooks/payloads.go` - later tasks use canonical `tool_id`
- `internal/api/contract/` - later tasks expose these contracts through HTTP/UDS DTOs

### Related ADRs
- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - defines executable backend classes
- [ADR-003: Runtime Registry Package Boundary](adrs/adr-003-runtime-registry-package-boundary.md) - constrains `internal/tools` dependencies
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - defines public ToolID grammar

### Web/Docs Impact
- `web/`: none - checked surfaces: `web/src/generated/agh-openapi.d.ts`, `web/src/systems/**`; reason: this task creates backend-internal contracts only and does not expose HTTP/UDS DTOs yet.
- `packages/site`: future docs in task_14 must describe canonical ToolID and backend kinds; no site file changes in this task unless implementation adds public docs early.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: establishes the registry contracts consumed by extensions, hooks, MCP adapters, tool resources, and SDKs.
- Agent manageability: none directly - checked CLI/HTTP/UDS surfaces; public operations are introduced in task_11 and task_12.
- Config lifecycle: none directly - checked `internal/config`; config keys are introduced in task_02.

## Deliverables
- Final `internal/tools` contract types and validators
- Deterministic error and availability reason constants
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration-style package-boundary tests or `make boundaries` evidence **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Valid IDs such as `agh__skill_view` and `mcp__github__create_issue` are accepted
  - [x] Dotted, hyphenated, uppercase, empty-segment, reserved-conflict, and over-length IDs are rejected with stable reasons
  - [x] Descriptors require object input schemas and valid backend/source refs
  - [x] Result envelopes preserve content metadata while enforcing redaction/truncation fields
  - [x] Provider and handle interfaces reject nil or incomplete implementations where applicable
- Integration tests:
  - [x] `go test ./internal/tools -race` passes
  - [x] `make boundaries` proves `internal/tools` has no forbidden domain imports
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/tools` can represent native, extension-host, and MCP executable tools without importing their implementation packages
- Invalid public ToolIDs fail closed with deterministic reason codes

## Verification Evidence

- `go test ./internal/tools -race -count=1` passed.
- `go test ./internal/tools -coverprofile=/tmp/tools.cover -count=1` reported 93.6% statement coverage.
- `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/tools/{tool_test.go,resource_test.go,boundary_test.go}` passed.
- `make boundaries` passed with `OK: all package boundaries respected`.
- `make lint` passed with `0 issues`.
- `make verify` passed: web format/lint/typecheck/tests/build, Go lint, race-enabled Go tests, Go build, and package boundaries.
