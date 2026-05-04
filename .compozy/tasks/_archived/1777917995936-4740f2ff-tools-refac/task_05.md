---
status: completed
title: Config Mutable Tool Family
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 05: Config Mutable Tool Family

## Overview

Make validated AGH config management tool-callable by default, while keeping trust-root and secret boundaries out of the normal agent tool loop. This task should reuse the shipped config writer and validation stack instead of inventing a second mutation path for tools.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, and ADR-006 before widening mutable config surfaces
- REFERENCE TECHSPEC sections "Config Lifecycle", "Old vs New Effective Behavior", and "Post-Implementation Residual Checks"
- FOCUS ON WHAT: expose validated config mutation; do not use tools to bypass existing persistence, merge, or validation rules
- MINIMIZE CODE — reuse current config writers, overlays, and validation contracts
- TESTS REQUIRED — all forbidden trust-root or secret paths must fail deterministically
</critical>

<requirements>
1. MUST expose the canonical config inspection and mutation family through built-in tools.
2. MUST allow mutation only for paths that remain tool-callable under the TechSpec trust-root and secret rules.
3. MUST require mutating approval and deterministic reason codes for forbidden or sensitive paths.
4. MUST preserve current merge, overlay, validation, and persistence behavior for the same config writes.
</requirements>

## Subtasks

- [x] 5.1 Add built-in descriptors for config inspection and mutation verbs
- [x] 5.2 Reuse the existing config writer and validation path for tool calls
- [x] 5.3 Enforce trust-root, secret, and operator-only path denial rules
- [x] 5.4 Thread approval and policy decisions through the config mutation tool family
- [x] 5.5 Add unit and integration coverage for allow/deny and transport parity

## Implementation Details

See TechSpec sections "Config Lifecycle", "Data-Model Field Rationale", and "Implementation Steps". This task should keep the authoritative config path exactly one layer deep: tool, CLI, HTTP, and UDS all converge on the same writer and validator behavior.

### Relevant Files

- `internal/tools/builtin/config.go` — shipped built-in config read/mutation entry points
- `internal/cli/config.go` — current operator config lifecycle semantics
- `internal/config/config.go` — config structs and defaults
- `internal/config/merge.go` — overlay merge behavior that tool writes must preserve
- `internal/config/persistence.go` — validated persistence path for config changes

### Dependent Files

- `internal/daemon/native_config_hook_tools.go` — daemon glue for config and hook tool families
- `internal/config/tools.go` — tool grammar and policy-facing config declarations

### Related ADRs

- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`; checked `web/src/systems/settings/adapters/settings-api.ts` and `web/src/systems/settings/*` — no new settings route is required here, but shared config semantics and generated types must not drift.
- `packages/site`: `packages/site/content/runtime/core/configuration/config-toml.mdx`, `packages/site/content/runtime/core/configuration/agent-md.mdx`, and CLI reference pages under `runtime/cli-reference/config/`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: affects tool-callable configuration of other runtime subsystems without changing extension manifests or protocol formats directly.
- Agent manageability: gives agents structured config mutation instead of raw file edits or CLI-only management for allowed paths.
- Config lifecycle: directly affects `config.toml` path mutability, validation, overlay semantics, examples, and docs; this task must keep one authoritative lifecycle.

## Deliverables

- Canonical config tool family with validated mutation paths
- Deterministic denial behavior for trust-root and secret config paths
- Approval-integrated config mutation flow
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for config mutation parity **(REQUIRED)**

## Tests

- Unit tests:
- [x] allowed config paths mutate successfully through tools and preserve current merge behavior
- [x] forbidden paths such as daemon transport roots, provider secrets, or sandbox trust-root settings fail with deterministic reason codes
- [x] tool-driven writes require approval and do not bypass validation or persistence hooks
- Integration tests:
- [x] tool-driven config get/set/unset/show/list parity matches existing operator surfaces for the same caller scope
- [x] config mutation denial reasons stay consistent across tool, CLI, HTTP, and UDS management paths
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agents can mutate validated config paths through dedicated tools without bypassing the existing config lifecycle
- Trust-root and secret boundaries remain explicit, deterministic, and auditable
