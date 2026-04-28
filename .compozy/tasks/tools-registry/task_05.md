---
status: pending
title: Native Go Built-In Providers
type: backend
complexity: high
dependencies:
  - task_04
---

# Task 05: Native Go Built-In Providers

## Overview

Add the first executable `native_go` providers backed by existing AGH services. This task exposes the bounded MVP tool set for registry, skill, network, and task operations while preserving service ownership and avoiding duplicate business logic.

<critical>
- ALWAYS READ `_techspec.md`, ADR-004, ADR-005, and ADR-006 before adding built-in tools
- DO NOT expose task claim/release/complete/fail/run-start or skill install/remove/update in MVP
- DO NOT bypass existing skill, task, network, or registry services with parallel implementations
- TESTS REQUIRED: every built-in must prove policy flags, input validation, and real service wiring
</critical>

<requirements>
1. MUST add executable `native_go` providers for `agh__tool_list`, `agh__tool_search`, and `agh__tool_info`.
2. MUST add executable `agh__skill_list`, `agh__skill_search`, and `agh__skill_view` using existing skill registry behavior.
3. MUST add executable `agh__network_peers` and `agh__network_send` using existing network manager boundaries.
4. MUST add only the bounded task tools from ADR-004: list, read, create, child create, update, cancel, and run list.
5. MUST mark read-only, mutating, open-world, and destructive risk metadata accurately for each native tool.
6. MUST wire providers through daemon composition and central dispatch only.
</requirements>

## Subtasks
- [ ] 5.1 Add registry bootstrap tools for list, search, and info
- [ ] 5.2 Add skill catalog native tools using existing skill registry APIs
- [ ] 5.3 Add network native tools using existing network manager APIs
- [ ] 5.4 Add bounded task native tools using existing task services and child-lineage rules
- [ ] 5.5 Wire native providers in the daemon composition root
- [ ] 5.6 Add tests proving descriptors, risk flags, dispatch wiring, and excluded tools

## Implementation Details

Use TechSpec "MVP Boundary Statement", "Integration Points", and "Implementation Steps" plus ADR-004. Keep adapter code thin: native tools should translate registry inputs/outputs to existing services, not become new domain services.

### Relevant Files
- `internal/tools/builtin_*.go` - native provider adapters and descriptors
- `internal/daemon/**` - composition root wiring for providers
- `internal/skills/registry.go` - skill list/search/view behavior to reuse
- `internal/task/interfaces.go` - task service boundaries to reuse
- `internal/network/manager.go` - network operations to wrap
- `internal/store/session_lineage.go` - child-task lineage constraints

### Dependent Files
- `internal/api/contract/tools.go` - task_11 exposes native descriptors/results
- `internal/cli/tool*.go` - task_12 invokes native tools
- `web/src/systems/tools/**` - task_13 displays native tool state
- `packages/site/content/runtime/core/tools.mdx` - task_14 documents native MVP tools

### Related ADRs
- [ADR-004: MVP Native Tool Scope](adrs/adr-004-mvp-native-tool-scope.md) - defines exact native built-in tool set
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - constrains mutating/destructive calls
- [ADR-006: Tool Visibility by Surface](adrs/adr-006-tool-visibility-by-surface.md) - constrains operator/session projections

### Web/Docs Impact
- `web/`: task_13 must render native tool descriptors, callable state, and unavailable/denied reasons through the tool diagnostics surface.
- `packages/site`: task_14 must document each native MVP tool, explicitly listing excluded skill/task lifecycle operations.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: creates first-party `native_go` provider precedent for future runtime-owned tools.
- Agent manageability: native tools become manageable through registry API/CLI/UDS surfaces in tasks 11-12.
- Config lifecycle: consumes policy/toolset config from task_02; no new config keys in this task.

## Deliverables
- Executable native providers for registry, skill, network, and bounded task tools
- Daemon wiring that registers native providers through the registry
- Tests proving included and excluded tool scope
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests using real services where practical **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Native descriptors use canonical IDs, accurate backend refs, accurate source refs, and correct risk flags
  - [ ] Excluded task and skill lifecycle operations are not registered
  - [ ] Invalid input schemas for each native tool fail before service calls
  - [ ] Mutating native tools require the expected policy and approval conditions
- Integration tests:
  - [ ] Skill list/search/view tools return real skill registry results through `Registry.Call`
  - [ ] Task child creation enforces parent/child lineage constraints
  - [ ] Network send goes through the existing network manager and preserves deterministic errors
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Built-in MVP tools are executable through the registry, not descriptor-only
- Existing domain services remain the source of truth for skills, tasks, and network behavior
