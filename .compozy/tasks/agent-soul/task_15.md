---
status: completed
title: Update Runtime, Config, Extension, and CLI Documentation
type: docs
complexity: high
dependencies:
  - task_12
  - task_13
  - task_14
---

# Task 15: Update Runtime, Config, Extension, and CLI Documentation

## Overview

Update AGH documentation for authored context after implementation surfaces exist. This task documents `SOUL.md`, `HEARTBEAT.md`, session health, managed authoring, config, CLI/API/UDS, extension surfaces, SDK usage, and the explicit boundaries against task ownership and network presence.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, all ADRs, `COPY.md`, `packages/site/CLAUDE.md`, and docs memory before editing public docs.
- REFERENCE TECHSPEC for final behavior; docs must match implemented commands, routes, config, and contracts.
- FOCUS ON WHAT operators and agents need: authoring, management, status, config, boundaries, troubleshooting, and examples.
- MINIMIZE CODE in prose; show short examples only where they prove usage.
- TESTS REQUIRED for docs build, links, CLI reference generation, and runtime-doc canonical tests.
- NO WORKAROUNDS: do not document aspirational UI editors, independent heartbeat queues, or unsupported network behavior.
</critical>

<requirements>
- MUST activate `documentation-writer`, `crafting-effective-readmes`, and `copywriting` before public docs/copy edits.
- MUST read `COPY.md`, `docs/_memory/glossary.md`, and `packages/site/CLAUDE.md`.
- MUST document `SOUL.md` as optional authored identity/persona context subordinate to `AGENT.md`, capabilities, config, and runtime state.
- MUST document `HEARTBEAT.md` as optional authored wake/reentry policy subordinate to session health, scheduler, task leases, and config.
- MUST document `[agents.soul]`, `[agents.heartbeat]`, session health, CLI commands, HTTP/UDS surfaces, Host API, hooks, tools/resources, and SDK helpers.
- MUST update or generate CLI reference docs after task_12 if the repo uses generated CLI docs.
- MUST include competitor-informed rationale only where useful and avoid internal analysis leakage in public docs.
- MUST run docs/site validation commands required by the repo.
</requirements>

## Subtasks
- [x] 15.1 Update runtime authored-context docs for `SOUL.md` and `HEARTBEAT.md`.
- [x] 15.2 Update config docs for `[agents.soul]` and `[agents.heartbeat]`.
- [x] 15.3 Update session docs for health/status/inspect and wake eligibility.
- [x] 15.4 Update CLI/API/UDS docs and regenerate CLI reference if required.
- [x] 15.5 Update extension, Host API, hooks, tools/resources, SDK, bundle, and registry docs.
- [x] 15.6 Run site/docs validation and fix root-cause failures.

## Implementation Details

Docs must preserve the architectural distinction that drove the specs: Soul is authored identity, Heartbeat is authored wake policy, session health is runtime liveness, task-run heartbeat is lease ownership, and network greet is peer presence.

### Relevant Files
- `COPY.md` - public language authority.
- `packages/site/CLAUDE.md` - site-specific instructions.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` - config docs.
- `packages/site/content/runtime/core/configuration/agent-md.mdx` - authored agent-file docs.
- `packages/site/content/runtime/core/agents/` - agent context and authoring docs.
- `packages/site/content/runtime/core/sessions/index.mdx` - session health/status docs.
- `packages/site/content/runtime/core/network/protocol.mdx` - note network greet remains separate if relevant.
- `packages/site/content/runtime/core/extensions/index.mdx` - extension surface docs.
- `packages/site/content/runtime/cli-reference/` - CLI reference output.
- `packages/site/lib/runtime-tools-canonical-docs.test.ts` - docs/tool consistency tests.

### Dependent Files
- `packages/site/content/**/*.mdx` - docs pages updated by this task.
- `packages/site/lib/**/*test*` - docs validation and canonical tests.
- `openapi/agh.json` - source for API docs where applicable.
- `internal/cli/doc.go` or generated CLI metadata - source for CLI reference.
- `.compozy/tasks/agent-soul/task_16.md` - QA plan uses final docs behavior.

### Related ADRs
- [ADR-001: Optional Scoped SOUL.md Persona Artifact](adrs/adr-001.md) - docs Soul boundary.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - docs managed authoring.
- [ADR-007: HEARTBEAT.md Is Advisory Wake Policy](adrs/adr-007.md) - docs Heartbeat boundary.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - docs health distinction.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - docs manageability.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - docs config authority.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: document Host API grants, hooks, tools/resources, SDK helpers, bundles, registries, and bridge SDK implications.
- Agent manageability: document CLI/HTTP/UDS commands/routes, JSON output, deterministic errors, status discovery, and CAS.
- Config lifecycle: document new config keys, defaults, examples, validation errors, and disabled-state behavior.

### Web/Docs Impact
- Web impact: no UI editor docs; mention only truthful runtime/API/CLI surfaces and generated contract availability.
- Docs impact: this task owns all site/doc/CLI-reference changes for the feature.

## Deliverables
- Updated site docs for authored context, config, sessions, extensions, CLI/API/UDS, and SDK behavior.
- Updated/generated CLI reference documentation.
- Documentation tests, link checks, source generation, typecheck, and build evidence.
- Clear public boundary statements for Soul, Heartbeat, session health, task-run lease heartbeat, scheduler, and network greet.

## Tests
- Unit tests:
  - [ ] Site content tests pass for runtime tools/canonical docs.
  - [ ] CLI reference generation detects the new commands and excludes forbidden `agh session heartbeat`.
  - [ ] Docs examples use implemented command names, config keys, and payload fields.
  - [ ] Link and content checks pass for new pages/sections.
- Integration tests:
  - [ ] `cd packages/site && bun run source:generate` passes if required.
  - [ ] `cd packages/site && bun run typecheck` passes.
  - [ ] `cd packages/site && bun run test` passes.
  - [ ] `cd packages/site && bun run build` passes.
  - [ ] `make cli-docs` passes if CLI reference is generated.
- Test coverage target: docs validation coverage for every changed public page.
- All tests must pass.

## References
- `_techspec.md` - aggregate docs and Web/Docs Impact requirements.
- `_techspec_soul.md` - Soul docs requirements.
- `_techspec_heartbeat.md` - Heartbeat/session health docs requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw.md` - useful authored context docs contrast.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - useful Heartbeat docs contrast.
- `.resources/openclaw/docs/gateway/heartbeat.md:41-59` - docs precedent for `HEARTBEAT.md`.
- `.resources/openclaw/docs/reference/templates/HEARTBEAT.md:8-12` - template docs precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/SOUL.md:1-33` - Soul example precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/HEARTBEAT.md:1-85` - Heartbeat example precedent.

## Success Criteria
- All docs validation commands pass.
- Docs accurately describe implemented runtime behavior and boundaries.
- Agents and operators can learn how to configure, author, inspect, manage, and troubleshoot Soul, Heartbeat, and session health.
- No public docs imply unsupported UI editors, queues, leases, or network behavior.
