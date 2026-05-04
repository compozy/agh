---
status: completed
title: Web Session Creation Dialog and Resume Failure UX
type: frontend
complexity: high
dependencies:
  - task_04
  - task_05
---

# Task 06: Web Session Creation Dialog and Resume Failure UX

## Overview

Replace the current direct quick-create flow with an explicit session-creation dialog that always lets the operator confirm or override the provider for that conversation. This task also adds a dedicated resume failure state so the SPA explains persisted-provider mismatches inline instead of collapsing everything into a generic toast.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and tasks 04-05 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "API Endpoints", "Testing Approach", "Known Risks", and "Build Order"
- THE CREATE FLOW MUST ALWAYS OPEN A DIALOG - do not preserve the old sidebar quick-create bypass
- USE GENERATED CONTRACTS - the web client should consume backend `provider` and workspace `providers` fields through generated types, not handwritten drift-prone copies
- RESUME FAILURE NEEDS A FIRST-CLASS UI STATE - not just a transient toast or console error
- PRESERVE AGH VISUAL LANGUAGE - keep the flow aligned with existing `web/` interaction patterns and use the same dialog conventions already present in the app
</critical>

<requirements>
- MUST route every new-session entrypoint through a dialog-driven flow
- MUST prefill the dialog with the chosen agent, active workspace, and the agent's default provider when available
- MUST render a provider picker from `WorkspaceDetailPayload.providers`
- MUST submit the selected provider through the session create mutation
- MUST show the effective provider in client-side session state where the existing UI exposes session metadata
- MUST render a dedicated inline resume failure state when the persisted provider is unavailable, including the session id and missing provider when returned by the backend
- SHOULD keep the post-dialog submit path as close to one-click as possible once the fields are prefilled
</requirements>

## Subtasks
- [x] 6.1 Replace the sidebar and route quick-create path with a session-creation dialog flow
- [x] 6.2 Prefill agent, workspace, and default provider in the dialog state
- [x] 6.3 Bind the provider picker to workspace detail provider options
- [x] 6.4 Thread provider through session create hooks, adapters, and client types
- [x] 6.5 Add a dedicated resume failure state and cover the flow with route/component tests

## Implementation Details

See TechSpec "API Endpoints", "Testing Approach", and ADR-004. The UX goal is explicitness without friction: opening the dialog is mandatory, but the form should already be filled enough that the common path remains fast.

### Relevant Files
- `web/src/hooks/routes/use-app-layout.ts` - current sidebar new-session action wiring
- `web/src/components/app-sidebar.tsx` - create-session entrypoint UI that currently bypasses a dialog
- `web/src/systems/session/hooks/use-session-actions.ts` - session create/resume mutation hooks
- `web/src/systems/session/adapters/session-api.ts` - request/response mapping for session create/read
- `web/src/systems/session/types.ts` - client-facing session types carrying provider
- `web/src/hooks/routes/use-session-page.ts` - resume flow and failure handling
- `web/src/systems/workspace/types.ts` - workspace detail types that will now include providers
- `web/src/systems/workspace/adapters/workspace-api.ts` - workspace detail mapping
- `web/src/systems/workspace/hooks/use-workspaces.ts` - data source for provider options in the dialog
- `web/src/systems/network/components/network-create-channel-dialog.tsx` - reference dialog pattern already used in the app
- `web/src/routes/_app/-index.test.tsx` - natural place for create-flow route tests
- `web/src/routes/_app/-session.$id.test.tsx` - natural place for resume failure UX tests

### Dependent Files
- `web/src/generated/agh-openapi.d.ts` - generated contract source updated by task_04
- `.compozy/tasks/session-driver-override/task_07.md` - QA planning must map manual and regression cases from this UI flow
- `.compozy/tasks/session-driver-override/task_08.md` - QA execution must prove dialog and resume-failure behavior end to end

### Related ADRs
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - resume failure UX must surface persisted-provider errors clearly
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) - defines the dialog-based creation flow

## Deliverables
- Dialog-based web session creation flow for every create entrypoint
- Provider picker bound to workspace-visible provider options
- Session create mutations and client types updated to send/read provider
- Dedicated inline resume failure UX for unavailable persisted providers **(REQUIRED)**
- Route/component coverage for dialog open, prefill, provider selection, and resume failure **(REQUIRED)**
- Updated web tests and type safety on the new generated contract fields **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Every create-session entrypoint opens the dialog instead of creating immediately
  - [x] The dialog preselects the chosen agent, active workspace, and default provider
  - [x] The provider picker renders workspace-visible providers from the workspace detail payload
  - [x] Submitting the dialog sends the selected provider in the session create mutation
  - [x] Resume failure renders a dedicated inline state instead of only a toast
- Integration tests:
  - [x] Generated workspace/session types are consumed without handwritten contract drift
  - [x] Route tests prove the dialog flow works from the sidebar and any other create entrypoints
  - [x] Route tests prove persisted-provider resume failures remain actionable in the UI
  - [x] Web typecheck and targeted test suites pass after the create-flow and resume UX changes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The web app always creates sessions through an explicit provider-aware dialog
- Resume failures caused by missing persisted providers are understandable and actionable in the UI
