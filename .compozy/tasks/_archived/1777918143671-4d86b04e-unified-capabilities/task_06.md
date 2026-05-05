---
status: completed
title: Update `web/` Network UX and Typed Client for Unified Capabilities
type: frontend
complexity: high
dependencies:
  - task_04
---

# Task 06: Update `web/` Network UX and Typed Client for Unified Capabilities

## Overview

Update the network surface in `web/` so the frontend consumes and presents the unified capability model without recipe-era terminology or payload assumptions. This task covers typed client updates, network-system adapters and hooks, route-level page wiring, mocks, and UI regressions around peer capability details.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, task_04 outputs, and `web/AGENTS.md` before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "API Endpoints", "Integration Points", and "Testing Approach"
- FOLLOW WEB PROJECT RULES - `make web-lint` and `make web-typecheck` must pass for this task
- KEEP THE FRONTEND AS A CLIENT OF THE BACKEND CONTRACT - do not invent parallel frontend-only capability semantics
- TESTS REQUIRED - typed client, hooks, routes, mocks, and visible peer-detail behavior all need coverage
- GREENFIELD: remove stale recipe wording and dead mapping code instead of layering translation helpers on top
</critical>

<requirements>
- MUST update generated or hand-authored frontend contract types to match the unified backend capability payloads from task_04
- MUST update the `web/src/systems/network` adapter, hooks, and formatting logic so peer and channel surfaces understand the new capability model
- MUST update network route/page behavior and peer detail UI to present unified capabilities consistently
- MUST refresh mocks, fixtures, and tests so frontend development and regression coverage use the new payload shape
- MUST keep the frontend discovery UX aligned with the backend split between brief discovery, rich details, and transferred capabilities where surfaced
- SHOULD improve clarity in peer-detail rendering where the old model caused confusion between capabilities and recipes
</requirements>

## Subtasks
- [x] 6.1 Update frontend network types and API bindings for unified capabilities
- [x] 6.2 Rewrite `systems/network` adapters, hooks, and formatters around the new payload shape
- [x] 6.3 Update route/page view-model hooks and peer-detail UI to render unified capability data clearly
- [x] 6.4 Refresh mocks, fixtures, and regression tests for the new contract
- [x] 6.5 Run the required web verification gates for lint, typecheck, and targeted tests

## Implementation Details

See TechSpec "API Endpoints", "Integration Points", and "Testing Approach". This task should treat the backend/API contract from task_04 as authoritative and update the frontend system layer cleanly rather than adding compatibility mappers for now-obsolete recipe-aware fields.

### Relevant Files
- `web/src/generated/agh-openapi.d.ts` - frontend contract typing that must align with the unified backend payload
- `web/src/systems/network/adapters/network-api.ts` - network API surface consumed by hooks and routes
- `web/src/systems/network/types.ts` - domain types for peer and capability rendering
- `web/src/systems/network/lib/network-formatters.ts` - display-layer formatting that may still encode split-model assumptions
- `web/src/systems/network/hooks/use-network.ts` - query hooks consuming network payloads
- `web/src/hooks/routes/use-network-page.ts` - route-level view-model logic for the main network screen
- `web/src/routes/_app/network.tsx` - primary network route that surfaces peer and capability details

### Dependent Files
- `web/src/systems/network/hooks/use-network-actions.ts` - any action flows referencing network payload shapes may need alignment
- `web/src/systems/network/components/network-peer-detail-panel.tsx` - peer detail panel will present the unified capability model
- `web/src/systems/network/components/network-peers-list-panel.tsx` - peer summary UI may reflect brief capability changes
- `web/src/systems/network/mocks/fixtures.ts` - mock payloads need the new canonical capability shape
- `web/src/systems/network/mocks/handlers.ts` - mock handlers must serve the updated contract
- `web/src/routes/_app/-network.test.tsx` - route regression coverage must move to unified capability expectations
- `web/src/hooks/routes/use-network-page.test.tsx` - view-model regression coverage for the new payload shape

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - frontend should stop reflecting two overlapping network concepts
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - determines which structured fields the UI can rely on
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - constrains how transfer-related capability details may appear in the UI

## Deliverables
- Updated web contract types and network system code for unified capabilities
- Peer-detail and route-level UI rendering aligned to the backend/API contract from task_04
- Refreshed mocks and tests covering the new payload shape **(REQUIRED)**
- Passing `make web-lint` and `make web-typecheck` for the changed frontend surface **(REQUIRED)**
- Targeted frontend regression tests for routes, hooks, and capability rendering **(REQUIRED)**
- Test coverage >=80% for the touched frontend packages/modules **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Network adapters map the unified backend payload into frontend types without recipe-era fallbacks
  - [ ] Route-level view-model hooks expose peer capabilities consistently for brief and detailed peer views
  - [ ] Formatters render the structured capability fields expected by the updated UI
  - [ ] Mocks and fixtures remain contract-correct after the payload shape change
- Integration tests:
  - [ ] The main network route renders unified capability summaries and details correctly from the updated client contract
  - [ ] Peer-detail UI no longer references recipe terminology or absent recipe fields
  - [ ] `make web-lint` and `make web-typecheck` pass on the updated frontend surface
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The `web/` network UX consumes one unified capability contract end to end
- Frontend peer views no longer leak the old capability/recipe split into the operator experience
