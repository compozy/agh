---
status: completed
title: Add Soul Config and Resolver Foundation
type: backend
complexity: high
dependencies: []
---

# Task 01: Add Soul Config and Resolver Foundation

## Overview

Create the backend foundation for optional `SOUL.md` files as authored agent identity context. This task defines config, parsing, validation, diagnostics, digesting, and projection limits before any runtime surface consumes the resolved profile.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, and ADR-001 through ADR-011 before implementation.
- REFERENCE TECHSPEC for interface details; keep this task focused on the resolver foundation.
- FOCUS ON WHAT must exist: config, parser, validation, digest, diagnostics, and tests.
- MINIMIZE CODE in planning artifacts; do not invent runtime prompt behavior in this task.
- TESTS REQUIRED for valid files, invalid authority claims, config precedence, redaction, and deterministic digests.
- NO WORKAROUNDS: invalid `SOUL.md` must fail closed according to the approved specs.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing Go tests.
- MUST add `[agents.soul]` config defaults, validation, overlay/merge behavior, and docs-facing examples without hand-editing `go.mod`.
- MUST parse `SOUL.md` as Markdown with optional strict frontmatter and narrative body.
- MUST reject fields owned by `AGENT.md`, capabilities, runtime state, task ownership, session liveness, network presence, or config.
- MUST produce deterministic `soul_digest`, compact projection data, full read-model fields, and sanitized diagnostics.
- MUST enforce size limits, frontmatter key allowlists, body truncation rules, and closed errors from the TechSpec.
</requirements>

## Subtasks
- [x] 1.1 Add `[agents.soul]` config structs, defaults, validation, merge behavior, and config tests.
- [x] 1.2 Implement the `SOUL.md` parser and strict frontmatter validator.
- [x] 1.3 Add deterministic digest, normalized profile, compact projection, and full read-model structures.
- [x] 1.4 Add closed diagnostic errors with redaction and source-location metadata where available.
- [x] 1.5 Add unit tests for valid, invalid, oversized, missing, and authority-conflicting files.
- [x] 1.6 Confirm no session, task, heartbeat, or network behavior changes in this task.

## Implementation Details

Keep the resolver isolated so later tasks can call it from agent load, authoring, session spawn, and task claim paths without file-system side effects. Prefer a small package boundary such as `internal/soul` if it matches the existing package layout, and reuse existing frontmatter and diagnostics helpers where possible.

### Relevant Files
- `internal/config/config.go` - add and validate `[agents.soul]` config.
- `internal/config/agent.go` - connect agent-level defaults if the existing config model requires it.
- `internal/frontmatter/frontmatter.go` - reuse or extend strict Markdown frontmatter parsing.
- `internal/diagnostics/` - emit closed, redacted validation diagnostics.
- `internal/soul/` - likely destination for resolver, profile, digest, and validation code.
- `internal/agent/` - inspect existing `AGENT.md` loading boundaries before integrating in later tasks.

### Dependent Files
- `internal/config/*_test.go` - config defaults, validation, and overlays.
- `internal/frontmatter/*_test.go` - frontmatter parsing edge cases if helpers change.
- `internal/soul/*_test.go` - resolver, digest, projection, and diagnostics coverage.
- `.compozy/tasks/agent-soul/task_02.md` - persists resolver snapshots.
- `.compozy/tasks/agent-soul/task_03.md` - uses resolver validation for managed writes.
- `.compozy/tasks/agent-soul/task_04.md` - consumes resolved soul profiles in sessions and tasks.

### Related ADRs
- [ADR-001: Optional Scoped SOUL.md Persona Artifact](adrs/adr-001.md) - defines the authored identity boundary.
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - defines compact and full read-model outputs.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - requires validation to back all write surfaces.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: provide resolver structures that Host API, hooks, tools/resources, SDKs, and future bundles can consume later; do not expose extension mutation yet.
- Agent manageability: no CLI/HTTP/UDS verbs in this task; checked surfaces are deferred to tasks 10-13.
- Config lifecycle: add `[agents.soul]` keys, defaults, validation errors, and tests; docs updates are deferred to task_15.

### Web/Docs Impact
- Web impact: no UI or generated type change in this task; later contract/codegen tasks expose the read models.
- Docs impact: config examples and behavior must be recorded for task_15, but site content is not updated here.

## Deliverables
- `[agents.soul]` config model with defaults and validation.
- `SOUL.md` parser, validator, digest, compact projection, and full profile structures.
- Redacted diagnostic errors for invalid or unsupported authored soul content.
- Unit tests and integration-level resolver tests with >=80% package coverage.
- Explicit note in completion evidence that no runtime prompt/session/task behavior changed yet.

## Tests
- Unit tests:
  - [x] Valid Markdown-only `SOUL.md` resolves to a deterministic profile and digest.
  - [x] Valid strict frontmatter plus body resolves with expected allowlisted fields.
  - [x] Unsupported authority fields fail closed with redacted diagnostic messages.
  - [x] Oversized body/frontmatter inputs fail according to config limits.
  - [x] Missing optional `SOUL.md` returns an enabled-but-empty or disabled result exactly as the TechSpec specifies.
  - [x] Config overrides update limits and defaults deterministically.
- Integration tests:
  - [x] Agent fixture loading can call the resolver without mutating sessions, task runs, or network state.
  - [x] Diagnostics preserve file path and safe location metadata without leaking disallowed content.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec.md` - aggregate sequencing and shared boundaries.
- `_techspec_soul.md` - normative Soul behavior.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw.md` - OpenClaw `SOUL.md` prompt-context model.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes.md` - Hermes prompt snapshot and system-prompt findings.
- `.compozy/tasks/agent-soul/analysis/analysis_paperclip.md` - Paperclip companion instruction-file findings.
- `.resources/openclaw/src/agents/bootstrap-files.ts:194-288` - authored agent files precedent.
- `.resources/openclaw/src/agents/system-prompt.ts:96-126` - bounded prompt inclusion precedent.
- `.resources/hermes/agent/prompt_builder.py:1028-1054` - soul-like prompt loading precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/SOUL.md:1-33` - authored persona example.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- `SOUL.md` is represented as a validated authored identity artifact, not runtime liveness or task state.
- Later tasks can consume the resolver without duplicating parsing, validation, or digest logic.
