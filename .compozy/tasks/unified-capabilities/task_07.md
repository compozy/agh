---
status: pending
title: Update `packages/site` Protocol Reference and Examples
type: docs
complexity: high
dependencies:
  - task_05
---

# Task 07: Update `packages/site` Protocol Reference and Examples

## Overview

Update the protocol-facing site docs under `packages/site` so the public reference aligns with the new unified capability protocol. This task covers protocol pages, navigation metadata, message-kind references, and examples that still teach or imply `recipe` as a separate artifact.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the rewritten RFC from task_05 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "System Architecture", "Technical Considerations", and "Architecture Decision Records"
- KEEP `packages/site` PROTOCOL DOCS ALIGNED WITH THE FINAL RFC - do not let site pages drift into alternative terminology
- REMOVE OR REPLACE `recipes.mdx` CLEANLY - do not keep it as a first-class concept page in the steady-state navigation
- TESTS REQUIRED - protocol pages, examples, and nav metadata must remain internally consistent after the rewrite
- GREENFIELD: prefer deleting obsolete pages over preserving them as prominent reference material
</critical>

<requirements>
- MUST update protocol-reference pages in `packages/site/content/protocol/` to describe unified capabilities instead of a capability/recipe split
- MUST remove, replace, or explicitly supersede `packages/site/content/protocol/recipes.mdx` in the public protocol navigation
- MUST update message-kind, discovery, and example pages so `kind:"capability"` is the transferred artifact described to readers
- MUST keep protocol metadata and navigation files consistent after page renames or removals
- MUST align all protocol examples with the rewritten RFC 003 from task_05
- SHOULD preserve clear migration of concepts for readers by explaining the steady-state model without teaching recipe as a live primitive
</requirements>

## Subtasks
- [ ] 7.1 Rewrite protocol reference pages that still describe recipe as a first-class protocol concept
- [ ] 7.2 Replace protocol examples and message-kind tables with unified capability language and payloads
- [ ] 7.3 Remove or repurpose `recipes.mdx` and update site navigation metadata accordingly
- [ ] 7.4 Cross-check protocol-site wording against RFC 003 and the accepted ADRs

## Implementation Details

See TechSpec "System Architecture", "Technical Considerations", and task_05 outputs. This task should make the site protocol reference read like the public-facing version of the rewritten RFC, with no second conceptual layer lingering in navigation or examples.

### Relevant Files
- `packages/site/content/protocol/message-kinds.mdx` - protocol kind reference that must switch from recipe to capability
- `packages/site/content/protocol/capability-discovery.mdx` - discovery page that must explain the unified model clearly
- `packages/site/content/protocol/examples.mdx` - examples page that must remove recipe-first flows
- `packages/site/content/protocol/recipes.mdx` - obsolete or superseded page in the new steady state
- `packages/site/content/protocol/index.mdx` - protocol landing page that should reflect the unified narrative
- `packages/site/content/protocol/meta.json` - navigation and ordering metadata for the protocol section
- `docs/rfcs/003_agh-network-v0.md` - rewritten RFC source for the protocol narrative and examples

### Dependent Files
- `packages/site/content/protocol/envelope.mdx` - envelope reference may need updated artifact-kind examples
- `packages/site/content/protocol/delivery.mdx` - delivery semantics may need updated capability-transfer wording
- `packages/site/content/protocol/interactions.mdx` - interaction docs may need the new transfer kind terminology
- `packages/site/content/protocol/overview.mdx` - overview page must remain consistent with the updated message-kind and discovery docs
- `packages/site/content/protocol/guide/testing.mdx` - guide examples may depend on renamed protocol kinds

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - defines the single concept the protocol reference must present
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - governs protocol wording for transfer and interactions

## Deliverables
- Updated `packages/site` protocol reference pages for the unified capability model
- Removal, replacement, or explicit supersession of `recipes.mdx` in the public protocol section
- Updated protocol examples and message-kind references using `kind:"capability"` **(REQUIRED)**
- Navigation metadata updates that keep the site coherent after the protocol rewrite **(REQUIRED)**
- Documentation consistency checks against RFC 003 and the ADRs **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `message-kinds.mdx` documents `kind:"capability"` and no longer treats recipe as a first-class kind
  - [ ] `capability-discovery.mdx` explains brief discovery, rich discovery, and transfer without conceptual duplication
  - [ ] Protocol examples reflect the updated RFC terminology and payload expectations
  - [ ] `meta.json` and related nav metadata remain valid after any page removal or rename
- Integration tests:
  - [ ] The protocol section reads consistently end to end without pages contradicting the unified model
  - [ ] Any retained references to recipe are clearly historical or superseded rather than normative
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The public protocol reference teaches the unified capability model with no first-class recipe page left in steady-state docs
- Site readers can understand transfer, discovery, and examples without encountering the old split
