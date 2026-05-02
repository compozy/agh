---
status: completed
title: Add Extension Host API, Hooks, Tools, Resources, and SDK Support
type: backend
complexity: high
dependencies:
  - task_10
  - task_11
---

# Task 13: Add Extension Host API, Hooks, Tools, Resources, and SDK Support

## Overview

Integrate Soul, Heartbeat, session health, and wake audit/status with AGH's extensibility surfaces. This task makes the feature available to extensions, hooks, tools/resources, SDKs, and future bundles without letting any extension bypass the managed authoring or runtime authority boundaries.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, all ADRs, and current extension/tool/resource architecture before editing.
- REFERENCE TECHSPEC for Host API actions, grants, hook payloads, tools/resources, SDK names, and no-bypass rules.
- FOCUS ON WHAT must be extensible: controlled reads/writes, status, hooks, resources, tools, SDK contracts, and permissions.
- MINIMIZE CODE in task notes; reuse contracts and services from tasks 08-11.
- TESTS REQUIRED for Host API grants, write denial, action routing, hook payloads, tool/resource output, and SDK typing.
- NO WORKAROUNDS: extensions must not write `SOUL.md` or `HEARTBEAT.md` directly or claim tasks through wake policy.
</critical>

<requirements>
- MUST activate `agh-code-guidelines`, `golang-pro`, and `agh-contract-codegen-coship` if generated SDK contracts change.
- MUST activate `agh-test-conventions`, `testing-anti-patterns`, and `typescript-advanced` before tests/SDK edits.
- MUST add Host API methods and permission grants for Soul read/manage, Heartbeat read/manage/status, session health read, and wake audit read as specified.
- MUST expose hook payload fields for validated Soul snapshots, Heartbeat policy resolution, wake decisions, and session health changes where the TechSpec requires hooks.
- MUST add built-in tools/resources for agent-operable read/status surfaces and managed mutations only where explicitly approved.
- MUST update TypeScript SDK helpers and generated contract consumers for extensions.
- MUST ensure all mutation paths call managed services and enforce grants.
- MUST document bundle/registry/bridge SDK implications in code comments or docs stubs for task_15.
</requirements>

## Subtasks
- [x] 13.1 Add Host API methods, grants, and permission checks for Soul, Heartbeat, session health, and wake audit.
- [x] 13.2 Add hook payloads/events for snapshot resolution, authoring mutation, wake decision, and session health changes as specified.
- [x] 13.3 Add or update built-in tools/resources for read/status and approved managed mutation operations.
- [x] 13.4 Update TypeScript SDK host API helpers and generated contract usage.
- [x] 13.5 Add extension, hook, tool/resource, and SDK tests.
- [x] 13.6 Verify extensions cannot bypass managed authoring, session health, scheduler, or task claim authorities.

## Implementation Details

Use the existing extension protocol and host API patterns. This feature is a good SD-011 test: every capability must be extensible and agent-manageable, but extension access must remain governed by grants and managed services.

### Relevant Files
- `internal/extension/protocol/host_api.go` - Host API protocol methods and action names.
- `internal/extension/contract/host_api.go` - host API contract types.
- `internal/extension/host_api.go` - host API implementation.
- `internal/extension/manifest.go` - grants/permissions if manifest schema changes.
- `internal/hooks/events.go` - hook event names.
- `internal/hooks/payloads.go` - hook payload structures.
- `internal/tools/builtin/` - built-in tools for agent-operable runtime access.
- `internal/resources/` - resource providers for read/status views.
- `sdk/typescript/src/host-api.ts` - SDK Host API helper surface.
- `sdk/typescript/src/generated/contracts.ts` - generated contract dependency.

### Dependent Files
- `internal/extension/*_test.go` - Host API action and grant tests.
- `internal/hooks/*_test.go` - hook payload serialization and dispatch tests.
- `internal/tools/**/*_test.go` - built-in tool behavior tests.
- `internal/resources/**/*_test.go` - resource output and redaction tests.
- `sdk/typescript/src/**/*test*` - TypeScript SDK typing and helper tests.
- `.compozy/tasks/agent-soul/task_15.md` - docs extension and SDK behavior.

### Related ADRs
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - extension read model obligations.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - managed mutation authority.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - Host API and status obligations.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - config-bound extension visibility.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: this task implements Host API, hooks, tools/resources, SDK, bundle/registry readiness, and bridge SDK implications.
- Agent manageability: extensions and agents can inspect/manage only through granted, managed services with structured errors.
- Config lifecycle: extension outputs must include effective config/digest where specified and respect disabled states.

### Web/Docs Impact
- Web impact: SDK/generated type impacts are coordinated with task_14; no runtime UI editor required.
- Docs impact: task_15 must document Host API grants, hook payloads, tool/resource names, SDK methods, and bundle/registry posture.

## Deliverables
- Host API methods and grants for Soul, Heartbeat, session health, and wake audit/status.
- Hook event/payload support for approved lifecycle points.
- Built-in tools/resources for managed agent-operable surfaces.
- TypeScript SDK helper updates and contract usage.
- Tests proving grants, redaction, no-bypass behavior, and SDK type safety.

## Tests
- Unit tests:
  - [x] Host API reads return redacted Soul and Heartbeat read models with required grants.
  - [x] Host API managed writes require action grants and body-level `expected_digest`.
  - [x] Extension attempts to bypass services or write files directly are denied or impossible through exposed API.
  - [x] Hook payloads contain stable digest, status, reason, and provenance fields without raw secrets.
  - [x] Tools/resources return the same redacted status data as HTTP/UDS routes.
  - [x] TypeScript SDK helpers typecheck against generated contracts.
- Integration tests:
  - [x] A test extension can inspect Soul, Heartbeat, session health, and wake audit through granted Host API methods.
  - [x] A test extension without grants receives deterministic denied errors.
  - [x] Built-in tools/resources operate through managed services and match route semantics.
- Test coverage target: >=80%.
- All tests must pass.

## Validation Evidence

- `go test ./internal/hooks ./internal/extension/contract ./internal/extension/protocol ./internal/extension ./internal/session ./internal/daemon ./internal/tools/builtin ./sdk/go -count=1`
- `bun test sdk/typescript/src/host-api.test.ts sdk/typescript/src/authored-context-contracts.test.ts`
- `bun run --cwd sdk/typescript typecheck`
- `make codegen-check`
- `make lint`
- `make verify` (passed; Bun lint/typecheck/test/build, Go fmt/lint/test/build, package boundaries; 7706 Go tests)

## References
- `_techspec.md` - SD-011 extensibility matrix and shared surface requirements.
- `_techspec_soul.md` - Soul Host API and extension requirements.
- `_techspec_heartbeat.md` - Heartbeat Host API, hooks, and status requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - gateway/protocol precedent.
- `.resources/paperclip/packages/mcp-server/src/tools.ts:224-608` - tool surface precedent.
- `.resources/openclaw/src/gateway/protocol/schema/agent.ts:131-213` - typed agent protocol precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- The feature is extensible through AGH's official surfaces, not internal Go calls or direct file writes.
- Extension access preserves managed authoring, scheduler, session health, and task claim boundaries.
