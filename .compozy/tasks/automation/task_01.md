---
status: completed
title: Introduce automation config and core domain types
type: backend
complexity: high
dependencies: []
---

# Task 01: Introduce automation config and core domain types

## Overview

Add the configuration surface and domain types that every later automation task depends on. This task establishes the authoritative in-process model for jobs, triggers, runs, scope, schedule specs, and activation envelopes so persistence, runtime, and transport layers can build against one contract.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add an additive `AutomationConfig` surface to `internal/config` that can parse global automation settings plus TOML-defined jobs and triggers from the hardened TechSpec.
2. MUST define the foundational automation domain types in `internal/automation/` including `Job`, `Trigger`, `Run`, `ScheduleSpec`, `RetryConfig`, `FireLimitConfig`, `AutomationScope`, and `ActivationEnvelope`.
3. MUST validate scope and workspace invariants, schedule mode invariants, source ownership fields, webhook-only fields, and strict template semantics early enough that later layers do not need to re-invent shape validation.
4. SHOULD keep these types transport-agnostic so HTTP, UDS, CLI, and Host API layers map onto them instead of forking parallel models.
</requirements>

## Subtasks
- [x] 1.1 Add `AutomationConfig` and nested TOML structs to `internal/config`
- [x] 1.2 Create the foundational `internal/automation/` package with the core domain structs and enums
- [x] 1.3 Add validation helpers for scope, schedule mode, webhook fields, retry policy, and fire-limit policy
- [x] 1.4 Add template-parse validation entry points for trigger prompts using the strict activation-envelope model
- [x] 1.5 Cover config loading and type validation with table-driven tests

## Implementation Details

Establish the base model described in the TechSpec sections "Core Interfaces", "Data Models", and "Technical Considerations". This task should not implement runtime behavior yet; it should only create the additive config and type layer that later tasks will reuse.

### Relevant Files
- `internal/config/config.go` — Owns the merged daemon config surface and will need the new `AutomationConfig`
- `internal/config/config_test.go` — Existing config coverage should be extended for automation parsing and validation
- `internal/config/merge.go` — Config overlays must continue to merge cleanly once automation is added
- `internal/daemon/boot.go` — Consumes `aghconfig.Config`, so the new automation section must fit the current boot load path

### Dependent Files
- `internal/store/globaldb/global_db.go` — Later persistence work will store the domain types introduced here
- `internal/api/contract/` — Transport DTOs in later tasks should map to these structs rather than inventing copies
- `internal/cli/client.go` — CLI transport methods will eventually serialize requests based on these definitions

### Related ADRs
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Establishes the shared package and split between schedules and triggers
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Constrains retry and fire-limit fields that the types must expose

## Deliverables
- Additive automation config parsing and validation in `internal/config`
- Foundational `internal/automation/` domain types and validation helpers
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for config loading and validation **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Parsing TOML with one global job and one workspace trigger populates the expected `scope`, `workspace`, and source fields
  - [x] Validation rejects `scope = "global"` definitions that also provide a workspace binding
  - [x] Validation rejects `scope = "workspace"` definitions that omit a workspace binding
  - [x] Validation rejects webhook-only fields on non-webhook triggers and missing webhook fields on webhook triggers
  - [x] Validation rejects unsupported schedule modes, malformed retry settings, and malformed fire-limit windows
- Integration tests:
  - [x] Loading a global home config plus a workspace overlay preserves automation defaults and applies workspace-specific automation entries without breaking unrelated config sections
  - [x] Trigger prompt validation fails fast for a template that references a missing activation-envelope field under `missingkey=error`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Automation config can be loaded through the standard AGH config path
- `internal/automation/` contains the shared domain types later tasks can compile against
