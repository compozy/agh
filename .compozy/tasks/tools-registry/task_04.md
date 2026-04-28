---
status: pending
title: Dispatch Pipeline, Hooks, Budgets, and Observability
type: backend
complexity: critical
dependencies:
  - task_03
---

# Task 04: Dispatch Pipeline, Hooks, Budgets, and Observability

## Overview

Implement the central call path for all registry-backed tools. This task adds dispatch-time validation, policy rechecks, canonical hook payloads, cancellation, result limiting, redaction metadata, telemetry seams, and normalized errors before native, extension, or MCP providers are exposed publicly.

<critical>
- ALWAYS READ `_techspec.md`, ADR-003, ADR-005, ADR-006, and ADR-007 before editing dispatch
- DO NOT let provider handles bypass schema validation, policy recheck, hooks, budgets, or redaction
- HARD CUT registry-owned hook payloads to canonical `tool_id`; do not create dual `tool_name` aliases
- TESTS REQUIRED: hook and error behavior must be deterministic across success, denial, cancellation, and provider failure
</critical>

<requirements>
1. MUST implement one registry dispatch pipeline for `native_go`, `extension_host`, and `mcp` handles.
2. MUST validate input schema and re-evaluate availability/policy immediately before invocation.
3. MUST execute pre-call, post-call, and error hooks with canonical `tool_id` payloads.
4. MUST enforce result byte budgets, redaction metadata, and stable truncation semantics.
5. MUST propagate `context.Context` cancellation through provider handles and hook execution.
6. MUST emit structured observability events without leaking tokens, approval secrets, bind nonces, or raw tool input marked sensitive.
</requirements>

## Subtasks
- [ ] 4.1 Implement `Registry.Call` and call-context inputs
- [ ] 4.2 Add schema validation, availability recheck, policy recheck, and normalized errors
- [ ] 4.3 Hard-cut registry hook payloads and matchers to canonical `tool_id`
- [ ] 4.4 Add result limiting, redaction metadata, and sensitive-field filtering
- [ ] 4.5 Add telemetry/event seams for success, denial, timeout, cancellation, and provider errors
- [ ] 4.6 Add unit and integration tests for call ordering, hooks, budgets, cancellation, and redaction

## Implementation Details

Use TechSpec "Core Interfaces", "Integration Points: Hooks", "Test Strategy", and "Safety Invariants". Keep dispatch provider-agnostic; backend-specific adapters are implemented in tasks 05, 07, and 09.

### Relevant Files
- `internal/tools/dispatch*.go` - central call path and call context
- `internal/tools/result*.go` - result budgets, truncation, and redaction metadata
- `internal/hooks/payloads.go` - registry-owned hook payload changes
- `internal/hooks/types.go` - hook type contracts affected by canonical tool IDs
- `internal/hooks/matcher.go` - matcher behavior for `tool_id`
- `internal/observe/**` - event/log integration seams if required by existing observability patterns

### Dependent Files
- `internal/tools/builtin_*.go` - task_05 native handles enter through dispatch
- `internal/extension/manager.go` - task_07 extension calls enter through dispatch
- `internal/mcp/**` - task_09 MCP calls enter through dispatch
- `internal/api/contract/` - task_11 exposes normalized call errors and result envelopes

### Related ADRs
- [ADR-003: Runtime Registry Package Boundary](adrs/adr-003-runtime-registry-package-boundary.md) - dispatch remains owned by `internal/tools`
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - dispatch revalidates approval and policy gates
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - callable session projection is rechecked at invocation
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - hook payloads use canonical IDs

### Web/Docs Impact
- `web/`: task_13 must update any tool-call displays or generated event consumers that currently assume `tool_name`/`tool_namespace`.
- `packages/site`: task_14 must document hook payload hard cut, result budgets, redaction behavior, and deterministic error classes.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: establishes the only safe executable path for extension, MCP, native, hook, and future bridge tool providers.
- Agent manageability: later CLI/HTTP/UDS invoke surfaces must use this pipeline and return its structured errors.
- Config lifecycle: consumes result budgets and approval timeout policy from task_02; does not add new config keys.

## Deliverables
- Central provider-agnostic dispatch pipeline
- Canonical `tool_id` hook payloads and matcher support for registry calls
- Result budget and redaction enforcement
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for hook order, policy recheck, and cancellation **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Invalid inputs fail before provider invocation
  - [ ] Policy and availability are rechecked after projection and before handle call
  - [ ] Pre-call hook denial prevents provider invocation and returns the expected reason
  - [ ] Post-call and post-error hooks receive canonical `tool_id` and redacted metadata
  - [ ] Context cancellation stops hook/provider execution and returns deterministic cancellation errors
  - [ ] Oversized results are truncated with metadata and never leak configured sensitive fields
- Integration tests:
  - [ ] A fake provider called through the registry observes schema, policy, hook, budget, and telemetry ordering
  - [ ] Existing hook tests are migrated off registry-owned `tool_name`/`tool_namespace` assumptions
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Every executable backend must pass through the same dispatch pipeline
- Registry hook identity is canonical `tool_id` with no dual identity path
