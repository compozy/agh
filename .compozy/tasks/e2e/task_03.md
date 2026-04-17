---
status: completed
title: Composition-root runtime network collaboration scenarios
type: test
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Composition-root runtime network collaboration scenarios

## Overview

Add the first real composition-root runtime E2E scenarios for AGH network collaboration. This task proves that channels, direct replies, peer discovery, and RFC-visible network exchanges work through a real daemon boot with deterministic ACP agents and domain-specific assertion surfaces.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add composition-root runtime E2E scenarios under `internal/daemon` for direct reply lifecycle and whois/recipe exchange using the shared runtime harness and expanded mock agents.
2. MUST enable and exercise the real embedded network runtime, including channel membership, peer visibility, and message delivery through public product surfaces.
3. MUST assert network correctness through domain-specific outputs such as channel messages, network audit snapshots, API projections, CLI visibility, and selected transcript/event checks where applicable.
4. MUST cover RFC-visible correlation fields such as `message_id`, `interaction_id`, `reply_to`, `receipt`, and `trace` in the direct reply lifecycle scenario.
5. SHOULD keep low-level protocol truth in the runtime lane and avoid pushing these semantics into browser automation tasks.
</requirements>

## Subtasks
- [x] 3.1 Add network-enabled runtime fixture setup to the shared daemon harness.
- [x] 3.2 Implement the direct reply lifecycle scenario in `internal/daemon`.
- [x] 3.3 Implement the whois and recipe exchange scenario in `internal/daemon`.
- [x] 3.4 Add domain-specific artifact capture and assertions for network messages, audit state, API projections, and CLI visibility.
- [x] 3.5 Add focused regression coverage for correlation and duplicate-prone message paths.

## Implementation Details

See TechSpec sections "Runtime Data Flow", "PR-Required Runtime E2E", and "Technical Considerations". This task should stay at the daemon composition root because sessions, network runtime, and persisted projections are wired together there.

### Relevant Files
- `internal/daemon/daemon_integration_test.go` — primary home for composition-root runtime scenarios.
- `internal/network/delivery_integration_test.go` — existing network-delivery patterns worth reusing for prompt-turn expectations.
- `internal/network/router_integration_test.go` — existing correlation and routing behaviors relevant to RFC-visible assertions.
- `internal/network/tasks.go` — network-to-task integration surface that informs later orchestration scenarios.
- `internal/api/core/network.go` — network projection behavior that runtime assertions should read through public surfaces.
- `docs/rfcs/003_agh-network-v0.md` — protocol semantics that the runtime scenarios must reflect.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — must expose network-enabled daemon setup and network/public-surface clients.
- `internal/testutil/acpmock/testdata/` — requires network-aware fixture scenarios for direct reply and discovery flows.
- `internal/api/httpapi/httpapi_integration_test.go` — later transport parity scenarios will read the same network state.
- `internal/api/udsapi/udsapi_integration_test.go` — later UDS parity scenarios will read the same network state.
- `web/e2e/network.spec.ts` — later browser network flow depends on this runtime scenario for system truth.

### Related ADRs
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Network collaboration proof belongs at the daemon composition root.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Network assertions must use network domain outputs, not transcript-only goldens.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — These network scenarios are PR-required runtime coverage.

## Deliverables
- Composition-root daemon runtime E2E for direct reply lifecycle
- Composition-root daemon runtime E2E for whois and recipe exchange
- Network artifact capture and assertion helpers integrated with the shared harness
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for network collaboration and RFC-visible correlation behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Network artifact helper writes stable channel-message and audit snapshots for a scenario run
  - [x] Scenario assertion helpers verify correlation fields without relying on transcript-only goldens
  - [x] Network-enabled harness seeding turns on the embedded runtime only when requested
- Integration tests:
  - [x] Direct reply lifecycle produces channel messages and audit state with matching correlation metadata
  - [x] Whois and recipe exchange surfaces peer discovery and persisted channel history through public APIs
  - [x] CLI-visible network state matches the same runtime scenario observed by the daemon assertions
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/daemon` contains real runtime E2E scenarios for network collaboration
- Network assertions use domain-specific product surfaces rather than transcript-only verification
- Later browser network tests can rely on this runtime lane as the protocol source of truth
