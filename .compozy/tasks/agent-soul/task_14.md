---
status: completed
title: Regenerate Web and SDK Contract Consumers
type: frontend
complexity: high
dependencies:
  - task_10
  - task_11
  - task_13
---

# Task 14: Regenerate Web and SDK Contract Consumers

## Overview

Update frontend and SDK contract consumers after Soul, Heartbeat, session health, and extension contracts exist. This task does not add a UI editor; it keeps generated types, API adapters, fixtures, and tests coherent so future UI work can build on truthful runtime contracts.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, every ADR, `web/CLAUDE.md`, and package instructions before editing frontend or SDK files.
- REFERENCE TECHSPEC for MVP UI boundary: generated consumers and docs yes, UI editor no.
- FOCUS ON WHAT must stay correct: generated types, adapters, tests, SDK helpers, and no fake UI.
- MINIMIZE CODE in task notes; do not invent visual surfaces beyond existing contract consumers.
- TESTS REQUIRED for TypeScript typecheck, adapter contracts, generated schema drift, and SDK helpers.
- NO WORKAROUNDS: do not render controls or metrics not backed by runtime routes.
</critical>

<requirements>
- MUST activate frontend/web-required skills from `web/CLAUDE.md` before editing `web/` or `packages/ui`.
- MUST activate `typescript-advanced`, `testing-anti-patterns`, and `vitest` before TypeScript tests.
- MUST consume generated `web/src/generated/agh-openapi.d.ts` from task_10 without hand editing generated files.
- MUST update web API adapters/types/tests only where existing systems consume agent/session/settings contracts.
- MUST update TypeScript SDK helpers/tests affected by task_13.
- MUST avoid adding a Soul/Heartbeat editor UI in MVP.
- MUST run `make bun-typecheck`, `make bun-test`, and any narrower affected workspace tests.
</requirements>

## Subtasks
- [x] 14.1 Inspect generated contract changes and identify affected web and SDK consumers.
- [x] 14.2 Update web type adapters, fixtures, and tests for Soul, Heartbeat, and session health contracts.
- [x] 14.3 Update SDK helper tests and TypeScript exports affected by Host API changes.
- [x] 14.4 Add assertions that no MVP UI editor or fake status control was introduced.
- [x] 14.5 Run Bun typecheck/test lanes and fix root-cause contract issues.
- [x] 14.6 Record any docs follow-up needed for task_15.

## Implementation Details

Keep this task truthful and contract-oriented. If existing web systems need to display or type session/agent status, update them to understand the new fields; otherwise keep the change limited to generated contracts and tests.

### Relevant Files
- `web/CLAUDE.md` - required web instructions.
- `web/src/generated/agh-openapi.d.ts` - generated contract input.
- `web/src/systems/session/types.ts` - likely session health type consumer.
- `web/src/systems/settings/types.ts` - likely config/status type consumer.
- `web/src/systems/agent/` - likely agent read-model test/fixture consumer.
- `web/src/lib/settings-api-contract.test.ts` - existing contract test precedent.
- `sdk/typescript/src/generated/contracts.ts` - generated SDK contract input.
- `sdk/typescript/src/host-api.ts` - SDK helper surface from task_13.

### Dependent Files
- `web/src/**/*test*` - affected adapter/type/fixture tests.
- `sdk/typescript/src/**/*test*` - SDK helper and generated type tests.
- `.compozy/tasks/agent-soul/task_15.md` - docs updates use final contract names.
- `.compozy/tasks/agent-soul/task_17.md` - QA execution typechecks generated consumers.

### Related ADRs
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - web contract visibility.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - generated session health/status surface.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - config/status field visibility.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: SDK generated types and helpers must align with Host API and extension surfaces from task_13.
- Agent manageability: frontend/API adapters must not hide CLI/HTTP/UDS fields needed by agents; no UI editor is introduced.
- Config lifecycle: generated consumers must preserve effective config/digest/status fields for future settings surfaces.

### Web/Docs Impact
- Web impact: update generated types, adapters, fixtures, and tests only; no visible editor/control unless an existing truthful status surface requires it.
- Docs impact: task_15 must use final generated names and SDK helper names from this task.

## Deliverables
- Updated generated TypeScript contract consumers in `web/` and SDK packages.
- Adapter/fixture/tests aligned to Soul, Heartbeat, session health, and Host API contracts.
- Evidence that no unsupported MVP UI editor or fake control was introduced.
- Passing Bun typecheck and test lanes.

## Tests
- Unit tests:
  - [ ] Web contract tests compile and assert required Soul/Heartbeat/session health fields.
  - [ ] Web adapters handle missing optional authored files and disabled config states.
  - [ ] SDK helper tests typecheck Host API methods, grants, and response payloads.
  - [ ] Tests prove generated files were not hand edited outside codegen.
- Integration tests:
  - [ ] `make bun-typecheck` passes across all Bun workspaces.
  - [ ] `make bun-test` passes across all configured Vitest projects.
  - [ ] `make codegen-check` remains clean after consumer updates.
- Test coverage target: >=80% where packages report coverage.
- All tests must pass.

## References
- `_techspec.md` - aggregate Web/Docs Impact and MVP UI boundary.
- `_techspec_soul.md` - Soul generated contract expectations.
- `_techspec_heartbeat.md` - Heartbeat/session health generated contract expectations.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - typed protocol precedent.
- `.resources/openclaw/src/gateway/protocol/schema/agent.ts:131-213` - typed agent schema precedent.

## Success Criteria
- All tests passing.
- Typecheck passes across affected Bun workspaces.
- Generated web and SDK consumers reflect the runtime contracts without inventing unsupported UI behavior.
- Future UI work can build on truthful typed surfaces.
