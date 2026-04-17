---
status: completed
title: ACP mock driver and multi-agent fixtures
type: test
complexity: high
dependencies:
  - task_01
---

# Task 02: ACP mock driver and multi-agent fixtures

## Overview

Expand the deterministic Go ACP mock layer so it can drive realistic agentic runtime scenarios instead of only narrow session transcript flows. This task keeps the mock boundary limited to deterministic ACP behavior while adding AGH-specific fixture primitives for multiple agents, tool permissions, network turns, exact prompt-metadata matching, and environment-aware expectations.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend `internal/testutil/acpmock/` to support multiple named agents and fixture-driven behaviors needed by runtime E2E scenarios.
2. MUST keep the daemon launch path unchanged by rendering temporary agent definitions that still resolve through normal AGH config and provider rules.
3. MUST support fixture primitives for permission requests, tool calls, inbound network-origin prompt turns, bridge response content, and environment command expectations.
4. MUST route fixture turns through exact matcher inputs (`turn_source`, `user_text`, structured network metadata) rather than rendered prompt substring heuristics.
5. SHOULD produce fixture outputs and diagnostics that plug into the artifact model introduced by `task_01`, including the matched turn selector and received prompt metadata.
</requirements>

## Subtasks
- [x] 2.1 Expand the mock ACP fixture schema to cover multi-agent and cross-domain scenario primitives.
- [x] 2.2 Implement temporary agent-definition registration that maps each fixture-driven mock agent onto the real daemon startup path.
- [x] 2.3 Add deterministic support for tool/permission events, network turns, and environment expectations.
- [x] 2.4 Add fixture diagnostics and golden coverage for expected streaming and event sequences.
- [x] 2.5 Add focused tests proving the mock layer remains narrow and launch-compatible.

## Implementation Details

See TechSpec sections "Component Overview", "Core Interfaces", "Data Models", and "Technical Considerations". The goal is not to simulate the full system inside the mock driver; the goal is to make the daemon believe it is talking to normal ACP agents while the rest of the runtime remains real.

### Relevant Files
- `internal/config/agent.go` — temporary agent definitions must remain valid against live agent config rules.
- `internal/config/provider.go` — temporary agent definitions must preserve provider-based validation and resolution.
- `internal/session/manager_start.go` — real session start path that the mock ACP driver must still flow through.
- `internal/acp/client.go` — event behavior and prompt-driving expectations that the mock subprocess needs to satisfy.
- `internal/testutil/testutil.go` — may provide generic test helpers shared by the fixture layer.
- `.compozy/tasks/e2e/adrs/adr-006.md` — decision source for the shipped Go mock driver and temp-agent registration strategy.

### Dependent Files
- `internal/testutil/acpmock/fixture.go` — expanded fixture schema and parsing helpers.
- `internal/testutil/acpmock/cmd/acpmock-driver/` — test-only Go driver command used by temporary agent definitions.
- `internal/testutil/acpmock/testdata/` — fixture scenarios and goldens for multi-agent and tool-aware flows.
- `internal/testutil/e2e/runtime_harness.go` — consumes mock agent registration once fixtures are expanded.
- `internal/daemon/daemon_integration_test.go` — first major consumer of the expanded mock-agent fixtures.

### Related ADRs
- [ADR-006: Keep ACP Mock Implemented in Go](adrs/adr-006.md) — This task is the shipped mock-agent strategy.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Fixture outputs must align with domain-specific assertions rather than transcript-only goldens.

## Deliverables
- Expanded `internal/testutil/acpmock/` fixture schema and helper surface
- Real temporary-agent registration compatible with the daemon runtime harness
- Deterministic mock support for permissions, tool calls, network turns, and environment expectations
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for mock-agent launch compatibility and multi-agent behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Multi-agent fixture parsing maps named agents and scenario primitives into stable internal specs
  - [x] Temporary agent-definition rendering preserves provider and command fields required by live config validation
  - [x] Deterministic streaming output remains stable when permission and tool events are included in the fixture
- Integration tests:
  - [x] A real daemon session launches a fixture-backed mock ACP agent through the normal agent-definition path
  - [x] Two fixture-backed mock agents can participate in one runtime scenario without cross-contaminating state
  - [x] Tool-permission and environment-expectation fixture events surface through the live ACP/session path
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Multi-agent fixtures exist and are consumable by runtime E2E scenarios
- Mock ACP agents still launch through the real daemon startup path with no new production seam
- Fixture routing is driven by exact prompt metadata instead of rendered-prompt substring matching
- Fixture behavior covers the scenario primitives required by later runtime and browser tasks
