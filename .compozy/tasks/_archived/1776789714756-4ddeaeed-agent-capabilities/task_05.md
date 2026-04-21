---
status: completed
title: Runtime Authoring Documentation for Capabilities
type: docs
complexity: low
dependencies:
  - task_03
  - task_04
---

# Task 05: Runtime Authoring Documentation for Capabilities

## Overview

Document how AGH authors declare capabilities locally and how those declarations project into brief and rich discovery on the network. This task turns the runtime behavior from tasks 01-04 into a clear operator-facing guide so authors know which files are valid, which layouts are invalid, and how discovery behaves when a catalog is absent.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, RFC 001, RFC 003, and tasks 01-04 before writing docs (`_prd.md` is absent for this feature)
- FOCUS ON RUNTIME AUTHORING - explain the local catalog model and its projection boundaries without re-litigating the design debate
- KEEP RUNTIME VS RFC BOUNDARIES CLEAR - file layouts belong to runtime docs; wire fields belong to network docs
- INCLUDE INVALID EXAMPLES - mixed-mode and mixed-format failures must be documented, not just the happy path
- TESTS REQUIRED - docs quality still needs explicit verification steps for consistency and completeness
</critical>

<requirements>
- MUST document all supported local authoring shapes: `capabilities.toml`, `capabilities.json`, `capabilities/*.toml`, and `capabilities/*.json`
- MUST document the invalid layouts explicitly: file plus directory, both single-file formats together, and mixed TOML/JSON in one `capabilities/` directory
- MUST document required fields (`id`, `summary`, `outcome`), optional fields, basename-without-extension matching, and the no-catalog behavior
- MUST explain the projection split between `peer_card.capabilities` / `agh.capabilities_brief` and explicit rich `whois` discovery via `agh.capability_catalog`
- MUST include at least one valid example for single-file mode and one valid example for directory mode
- SHOULD add cross-links or short clarifications in RFC 001 where agent-directory portability now includes capability sidecars alongside `AGENT.md`
</requirements>

## Subtasks
- [x] 5.1 Write the runtime-facing capability authoring guide with supported layouts and field definitions
- [x] 5.2 Add valid and invalid catalog examples that match the implemented loader behavior exactly
- [x] 5.3 Cross-link the runtime guide with RFC 001 and RFC 003 so the runtime/network boundary is explicit
- [x] 5.4 Verify wording, examples, and field names against the shipped runtime behavior from tasks 01-04

## Implementation Details

See TechSpec "Data Models", "Projection rules", and "Technical Considerations", plus RFC 001 for the self-contained agent-directory story and RFC 003 for wire discovery semantics. The most useful deliverable is a runtime-facing guide that agent authors can follow without reading the whole TechSpec.

### Relevant Files
- `.compozy/tasks/agent-capabilities/_techspec.md` - authoritative runtime behavior, validation rules, and projection semantics
- `docs/rfcs/001_agent-md-with-skills-memory.md` - agent-directory portability RFC that should acknowledge capability sidecars
- `docs/rfcs/003_agh-network-v0.md` - wire-level brief and rich capability discovery contract

### Dependent Files
- `docs/agents/capabilities.md` - proposed runtime-facing authoring guide for local capability catalogs
- `docs/rfcs/001_agent-md-with-skills-memory.md` - may need small clarifications or cross-links about capability sidecars as part of a self-contained agent directory
- `.compozy/tasks/agent-capabilities/task_01.md` - loader and validation rules documented here must stay consistent with the implementation task
- `.compozy/tasks/agent-capabilities/task_04.md` - rich discovery docs must match the explicit `whois` behavior defined there

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - document explicit local catalogs as the source of truth
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) - document file/directory and format rules clearly
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - document the required/optional capability fields and semantics

## Deliverables
- Runtime-facing capability authoring guide under `docs/agents/capabilities.md`
- Valid and invalid local catalog examples aligned with the shipped loader behavior
- Cross-links or clarifications in existing RFC docs where needed to keep runtime and network boundaries explicit
- Documentation review checklist proving field names, examples, and wire keys match the implementation **(REQUIRED)**
- Documentation consistency checks against tasks 01-04 and RFC 003 **(REQUIRED)**

## Tests
- Unit tests:
  - [x] The authoring guide lists all four supported local catalog shapes and explicitly lists the invalid mixed-mode and mixed-format layouts
  - [x] The guide marks `id`, `summary`, and `outcome` as required and `context_needed`, `artifacts_expected`, `execution_outline`, `constraints`, and `examples` as optional
  - [x] The guide explains basename-without-extension matching in directory mode and the no-catalog behavior clearly
  - [x] The guide includes one valid single-file example and one valid directory-mode example that match the runtime schema exactly
- Integration tests:
  - [x] Every wire key named in the docs exactly matches the implemented RFC strings: `agh.capabilities_brief`, `agh.include`, `agh.capability_ids`, and `agh.capability_catalog`
  - [x] The local layout examples remain consistent with the loader and validation behavior defined in tasks 01-04
  - [x] RFC 001, RFC 003, and the new runtime guide do not contradict one another about where local authoring ends and wire projection begins
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agent authors have one clear runtime-facing document that explains how to declare capabilities locally
- Runtime and network documentation describe the same feature without mixing local loader rules into the RFC
