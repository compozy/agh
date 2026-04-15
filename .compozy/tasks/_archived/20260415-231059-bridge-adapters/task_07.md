---
status: completed
title: "Update web bridge management for provider config, secret slots, and DM policy"
type: frontend
complexity: high
dependencies:
  - task_06
---

# Task 07: Update web bridge management for provider config, secret slots, and DM policy

## Overview

Bring the bridge management UI in line with the new backend contract so operators can actually configure provider-scoped bridges. This task teaches the bridge screens to separate provider runtime config from delivery defaults and surface provider-declared requirements clearly.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST update the bridge create and detail flows to capture and display `provider_config` independently from `delivery_defaults`.
2. MUST surface provider-declared secret-slot requirements, DM policy, and provider-specific configuration hints in the bridge management UI.
3. MUST preserve existing bridge-management flows such as provider selection, routing policy editing, and test-delivery behavior while the form model expands.
4. SHOULD present provider-specific configuration progressively so unsupported or irrelevant fields do not pollute the generic bridge workflow.
</requirements>

## Subtasks

- [x] 7.1 Update bridge web types and API adapters for the expanded bridge-management payloads
- [x] 7.2 Redesign the create and detail panels to separate delivery defaults from provider configuration
- [x] 7.3 Surface provider secret-slot requirements, DM policy controls, and provider hints in the UI
- [x] 7.4 Add component and hook coverage for the updated bridge-management flows

## Implementation Details

Follow the TechSpec sections "Data Model Changes", "Provider Manifest", and "Impact Analysis". This task should stop at web bridge management; it should not implement provider runtimes or daemon-side runtime logic.

### Relevant Files

- `web/src/systems/bridges/types.ts` — Current bridge draft model only understands `deliveryDefaults`
- `web/src/systems/bridges/adapters/bridges-api.ts` — Web API adapter must send and receive the expanded bridge contract
- `web/src/systems/bridges/components/bridge-create-dialog.tsx` — Create flow currently has no provider-config or DM-policy fields
- `web/src/routes/_app/bridges.tsx` — Bridge route state and mutations need to accommodate the new provider-owned fields

### Dependent Files

- `web/src/systems/bridges/hooks/use-bridge-actions.ts` — Mutations later need the expanded payload shape
- `web/src/generated/agh-openapi.d.ts` — Generated schema drives the typed bridge-management contract
- `web/src/systems/bridges/components/bridge-detail-panel.tsx` — Detail view later needs to show provider requirements and current config

### Related ADRs

- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — DM policy and provider configuration are part of the approved v1 management surface

## Deliverables

- Updated bridge web models and API adapters for `provider_config` and provider metadata
- Bridge creation and detail UI that separates provider config from delivery defaults
- UI support for provider secret-slot requirements and DM policy editing or display
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for bridge management flows **(REQUIRED)**

## Tests

- Unit tests:
  - [ ] bridge draft helpers preserve `provider_config` separately from `deliveryDefaults`
  - [ ] provider selection updates provider-specific config defaults or hints without clobbering unrelated routing fields
  - [ ] DM policy controls serialize only supported policy values into the bridge mutation payload
- Integration tests:
  - [ ] creating a bridge with provider config submits the expanded payload and shows the persisted values in the UI
  - [ ] bridge detail panels render provider secret-slot requirements and DM policy without regressing existing health and routing sections
  - [ ] test-delivery flow still uses delivery defaults and does not accidentally depend on provider-config fields
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The web UI can manage provider-scoped bridge configuration without abusing `delivery_defaults`
- Operators can see the provider requirements needed to configure a bridge instance correctly
