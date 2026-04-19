---
status: completed
title: Startup prompt section registry and network startup overlay
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Startup prompt section registry and network startup overlay

## Overview

Evolve the existing assembler/provider chain into an explicit startup section system driven by resolved harness policy. This task also turns the `agh-network` startup behavior into a first-class, documented overlay instead of an inline special case in session startup.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_01.md` before starting
- REFERENCE TECHSPEC sections "Workstream 2: Startup Prompt Architecture" and "Explicit Handling of Network Startup Behavior"
- FOCUS ON "WHAT" - select startup prompt sections deterministically before final concatenation; do not move volatile turn-time context into startup assembly
- MINIMIZE CODE - extend the current assembler/provider path instead of creating a parallel prompt builder stack
- TESTS REQUIRED - section order, section budgets, and network overlay behavior all need direct coverage
- GREENFIELD: `agh-network` nao pode continuar como excecao implita em `manager_start.go`
</critical>

<requirements>
- MUST keep startup prompt assembly on the existing assembler/provider path
- MUST add explicit section descriptors with order, eligibility, and budget metadata
- MUST add a daemon-owned section selector that evaluates against resolved harness policy
- MUST model `agh-network` as an explicit startup overlay or startup section rather than an undocumented inline append
- SHOULD preserve current provider integration points where they already map cleanly to stable startup context
</requirements>

## Subtasks
- [x] 2.1 Introduce startup section descriptors and a selector that evaluates against resolved harness policy
- [x] 2.2 Extend the composed assembler to gather and order only eligible sections
- [x] 2.3 Move the `agh-network` startup behavior into the section-selection model
- [x] 2.4 Add budget and omission behavior for startup sections
- [x] 2.5 Add coverage for section ordering, network overlay selection, and startup prompt assembly

## Implementation Details

See TechSpec "Workstream 2: Startup Prompt Architecture" and ADR-002. This task should leave startup prompting with one explicit selection path that is stable, testable, and auditable before the first agent prompt is ever sent.

### Relevant Files
- `internal/daemon/composed_assembler.go` - current startup assembly seam that must evolve into section-aware composition
- `internal/daemon/boot.go` - boot wiring currently registers prepend/append providers and is the right place to install selector-aware composition
- `internal/session/manager_start.go` - currently injects the network startup overlay inline and must stop owning that special case
- `internal/session/manager_network_skill.go` - current bundled network skill helper that must be rehomed intentionally
- `internal/session/prompt_provider.go` - existing prompt-provider contract that the section registry must continue to honor
- `internal/daemon/section_selector.go` - new daemon-owned selector introduced by this task

### Dependent Files
- `internal/daemon/composed_assembler_test.go` - needs direct coverage for section ordering and omission behavior
- `internal/session/manager_test.go` - startup prompt tests will need to assert network startup composition through the new path
- `internal/daemon/daemon_network_collaboration_integration_test.go` - natural integration lane for the network startup overlay behavior
- `internal/daemon/daemon.go` - later prompt and runtime wiring will assume startup policy is now section-driven

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - Provides the policy inputs that section selection must consume
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - Keeps this work on the existing assembler/provider chain

### External References
- `.resources/claude-code/constants/prompts.ts` - useful registry-style reference for named prompt blocks and addenda
- `.resources/claude-code/main.tsx` - shows real startup wiring of system-prompt addenda and initial messages
- `.resources/openclaw/src/agents/system-prompt.ts` - demonstrates explicit section layering around a base system prompt
- `.resources/openclaw/src/agents/pi-embedded-runner/run/attempt-bootstrap-routing.ts` - good analog for converting startup routing decisions into overlay content
- `.resources/hermes/gateway/builtin_hooks/boot_md.py` - concrete precedent for startup boot overlays sourced from a dedicated bootstrap artifact
- `.resources/openfang/crates/openfang-runtime/src/prompt_builder.rs` - useful reference for section ordering and stable prompt layering

## Deliverables
- Startup section descriptor and selector model added to the daemon prompt path
- `agh-network` startup overlay rehomed into explicit section-selection behavior **(REQUIRED)**
- Section ordering and budget behavior encoded in tests **(REQUIRED)**
- No remaining undocumented inline startup overlay behavior in session startup **(REQUIRED)**
- Regression coverage for duplicate, omitted, and over-budget startup sections plus explicit network startup gating **(REQUIRED)**
- Unit and integration tests with >=80% coverage for modified startup prompt assembly files **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Section selector includes only providers whose predicates match the resolved startup policy for one concrete session context
  - [x] Section ordering stays deterministic when prepend, base prompt, and append providers all contribute eligible sections
  - [x] Empty sections and over-budget sections are omitted or trimmed according to the declared startup-section policy
  - [x] `agh-network` startup content is selected only through the new section model and never via inline append logic
  - [x] Repeated section registration or selector re-entry does not duplicate network or bootstrap sections in the final prompt
- Integration tests:
  - [x] Starting a channel-bound session produces startup prompt content that includes the explicit network overlay in the expected position
  - [x] Starting a non-network session omits the network overlay entirely while preserving other startup sections such as memory or skills
  - [x] Session resume follows the same section-selection path and does not reintroduce the old inline `agh-network` append behavior
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Startup prompt composition is driven by explicit section selection instead of global prepend/append only
- The `agh-network` startup behavior is now first-class, testable, and discoverable in the architecture
