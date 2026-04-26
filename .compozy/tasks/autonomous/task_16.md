---
status: pending
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
Document the autonomy MVP across runtime docs and CLI references so users and agents understand the new task, coordinator, channel, and spawn behavior. This co-ships public documentation with the generated contract and UI changes.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-003, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, and `packages/site` docs conventions before writing docs
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
- MUST document new hook families `coordinator.*`, `spawn.*`, and `task.run.*` where public.
- MUST link docs to existing task, sessions, network, config, and hooks pages without broad site redesign.
- MUST run site/docs verification gates and update generated CLI docs if the repository has a generator.
</requirements>

## Subtasks
- [ ] 16.1 Audit existing `packages/site/content/runtime` structure and CLI reference generation workflow.
- [ ] 16.2 Add autonomy overview docs focused on MVP concepts and manual/autonomous coexistence.
- [ ] 16.3 Add or update CLI reference pages for agent self, channel, task lease, and spawn commands.
- [ ] 16.4 Add config and hooks documentation for coordinator settings and typed hook families.
- [ ] 16.5 Add examples that use implemented commands only and avoid post-MVP promises.
- [ ] 16.6 Run site typecheck/build/link tests and any CLI docs generation check.

## Implementation Details
The docs should be operational and precise: what starts a coordinator, how leases are claimed and heartbeated, what happens when a lease expires, how spawn permissions narrow, and how users keep manual control.

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

## Deliverables
- Runtime autonomy docs for MVP concepts and operator workflows.
- CLI reference docs for new agent-facing commands.
- Config/hooks/task lifecycle docs updated for coordinator and lease behavior.
- Docs examples that match implemented commands and flags.
- Site docs tests/build/link checks passing **(REQUIRED)**.
- Contract/CLI/docs consistency verification documented in completion notes **(REQUIRED)**.

## Tests
- Unit/docs tests:
  - [ ] Every new CLI example maps to an implemented command and flag.
  - [ ] Docs state that task creation does not enqueue work or spawn a coordinator.
  - [ ] Docs describe publish/start/approval as the execution boundary.
  - [ ] Docs describe raw claim token handling without suggesting it appears in list/get outputs.
  - [ ] Docs describe coordinator provider/model/workspace override precedence accurately.
- Integration/site tests:
  - [ ] `packages/site` typecheck passes.
  - [ ] `packages/site` build passes.
  - [ ] Link/content validation passes for all new autonomy pages.
  - [ ] Generated CLI docs are current if a generation command exists.
  - [ ] Web/docs navigation exposes runtime autonomy docs without marketing redesign.
- Test coverage target: docs verification must cover all new/changed pages.
- All tests must pass.

## Success Criteria
- All tests passing.
- Docs match the implemented MVP behavior and command surface.
- Users can understand manual task creation, coordinator handoff, task leases, channels, and safe spawn from the docs.
- Post-MVP features are not documented as available.
