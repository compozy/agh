---
status: completed
title: Runtime Autonomy Docs And CLI References
type: docs
complexity: medium
dependencies:
  - task_06
  - task_09
  - task_13
  - task_14
  - task_15
---

# Task 16: Runtime Autonomy Docs And CLI References

## Overview
Document the autonomy MVP across runtime docs and CLI references so users and agents understand the new task, coordinator, coordination channel, and spawn behavior. This co-ships public documentation with the generated contract and UI changes.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-003, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, ADR-012, and `packages/site` docs conventions before writing docs
- DOCUMENT MVP BEHAVIOR ONLY - do not promise post-MVP network evolution, broad memory, dashboards, or eval/replay
- CLI REFERENCES MUST MATCH IMPLEMENTED COMMANDS AND FLAGS EXACTLY
- DOCS MUST PRESERVE MANUAL OPERATOR CONTROL and explain that task creation does not start execution
- TESTS REQUIRED - docs links, site build/typecheck, examples, and CLI reference consistency must be verified
- NO WORKAROUNDS - do not paper over incomplete behavior with vague future-tense docs
</critical>

<requirements>
- MUST add runtime autonomy docs under `packages/site/content/runtime/core/autonomy/` or the existing equivalent docs structure.
- MUST update CLI reference docs for `agh me`, `agh ch`, `agh task next|heartbeat|complete|fail|release`, and `agh spawn`.
- MUST document coordinator config, workspace override precedence, task creation/start boundary, claim/lease semantics, and safe spawn constraints.
- MUST document task-run coordination channels, message kinds, channel correlation metadata, and the boundary that channels are conversation only, not task ownership/status authority.
- MUST document new hook families `coordinator.*`, `spawn.*`, and `task.run.*` where public.
- MUST link docs to existing task, sessions, network, config, and hooks pages without broad site redesign.
- MUST run site/docs verification gates and update generated CLI docs if the repository has a generator.
</requirements>

## Subtasks
- [x] 16.1 Audit existing `packages/site/content/runtime` structure and CLI reference generation workflow.
- [x] 16.2 Add autonomy overview docs focused on MVP concepts, manual/autonomous coexistence, and task-run coordination channels.
- [x] 16.3 Add or update CLI reference pages for agent self, channel coordination, task lease, and spawn commands.
- [x] 16.4 Add config and hooks documentation for coordinator settings and typed hook families.
- [x] 16.5 Add examples that use implemented commands only and avoid post-MVP promises.
- [x] 16.6 Run site typecheck/build/link tests and any CLI docs generation check.

## Implementation Details
The docs should be operational and precise: what starts a coordinator, when a coordination channel is created, how agents should use channel messages, how leases are claimed and heartbeated, what happens when a lease expires, how spawn permissions narrow, and how users keep manual control.

Do not create marketing copy or redesign docs navigation beyond the minimum needed to place the new content.

### Relevant Files
- `packages/site/content/runtime/` - runtime docs root.
- `packages/site/content/runtime/core/` - core runtime concept docs.
- `packages/site/content/runtime/cli-reference/` - CLI reference docs or generated output.
- `packages/site/content/runtime/core/configuration/` - config docs for coordinator settings.
- `packages/site/content/runtime/core/hooks/` - typed hooks docs if present.
- `packages/site/package.json` - site verification commands.
- `internal/cli/*` - source of truth for command names and flags.
- `internal/api/contract/*` - source of truth for public DTO semantics.
- `.resources/multica/apps/docs/` - reference for docs organization around runtime features.
- `.resources/paperclip/doc/CLI.md` - reference for CLI behavior documentation.
- `.resources/hermes/README.md` - reference for concise operator-facing agent runtime docs.

### Dependent Files
- `.compozy/tasks/autonomous/qa/test-plans/` - task_17 maps docs coverage.
- `.compozy/tasks/autonomous/qa/verification-report.md` - task_18 records final docs verification evidence.

### Related ADRs
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - CLI-first public surface.
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - lease docs.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - coordinator docs.
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - spawn docs.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - hook docs.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual control docs.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - co-ship requirement.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - coordination channel docs.

## Deliverables
- Runtime autonomy docs for MVP concepts and operator workflows.
- Runtime docs explaining task-run coordination channels and message kinds.
- CLI reference docs for new agent-facing commands.
- Config/hooks/task lifecycle docs updated for coordinator and lease behavior.
- Docs examples that match implemented commands and flags.
- Site docs tests/build/link checks passing **(REQUIRED)**.
- Contract/CLI/docs consistency verification documented in completion notes **(REQUIRED)**.

## Tests
- Unit/docs tests:
  - [x] Every new CLI example maps to an implemented command and flag.
  - [x] Docs state that task creation does not enqueue work or spawn a coordinator.
  - [x] Docs state that coordination channels are bound at run enqueue for coordinated runs, not at task creation.
  - [x] Docs describe publish/start/approval as the execution boundary.
  - [x] Docs describe raw claim token handling without suggesting it appears in list/get outputs.
  - [x] Docs describe that channel messages do not own task status and must not contain raw claim tokens.
  - [x] Docs describe coordinator provider/model/workspace override precedence accurately.
- Integration/site tests:
  - [x] `packages/site` typecheck passes.
  - [x] `packages/site` build passes.
  - [x] Link/content validation passes for all new autonomy pages.
  - [x] Generated CLI docs are current if a generation command exists.
  - [x] Web/docs navigation exposes runtime autonomy docs without marketing redesign.
- Test coverage target: docs verification must cover all new/changed pages.
- All tests must pass.

## Success Criteria
- All tests passing.
- Docs match the implemented MVP behavior and command surface.
- Users can understand manual task creation, coordinator handoff, task leases, channels, and safe spawn from the docs.
- Users can understand when to use `agh ch` versus `agh task` during coordinated execution.
- Post-MVP features are not documented as available.

## Completion Notes

- Added runtime autonomy docs under `packages/site/content/runtime/core/autonomy/` and linked them from runtime core navigation.
- Regenerated CLI reference docs with `make cli-docs`; command examples come from Cobra source for `agh me`, `agh ch`, `agh task next|heartbeat|complete|fail|release`, and `agh spawn`.
- Added `packages/site/lib/runtime-autonomy-docs.test.ts` to verify CLI page presence, exact flag coverage, execution boundary docs, claim-token boundaries, channel authority, and coordinator precedence.
- Verification passed: `make cli-docs`; `cd packages/site && bun run source:generate`; `cd packages/site && bun run typecheck`; `cd packages/site && bun run test`; `cd packages/site && bun run build`; `env -u FORCE_COLOR make verify`.
