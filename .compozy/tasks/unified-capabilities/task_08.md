---
status: pending
title: Update `packages/site` Runtime Capability Docs
type: docs
complexity: medium
dependencies:
  - task_05
---

# Task 08: Update `packages/site` Runtime Capability Docs

## Overview

Update the runtime-facing site docs so capability authoring, agent definitions, and runtime overview pages describe the unified capability model consistently. This task focuses on the runtime information architecture in `packages/site/content/runtime/**`, separate from the protocol reference updates handled in task_07.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the rewritten runtime docs from task_05 before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "System Architecture", "Data Models", and "Technical Considerations"
- KEEP RUNTIME DOCS DISTINCT FROM PROTOCOL DOCS - this task should teach authoring and runtime behavior, not duplicate the full protocol reference
- ALIGN SITE RUNTIME WORDING WITH `docs/agents/capabilities.md` - the site must not fork its own capability story
- TESTS REQUIRED - runtime pages and metadata must remain internally consistent after the rewrite
- GREENFIELD: replace stale explanations directly instead of stacking warning callouts on obsolete copy
</critical>

<requirements>
- MUST update runtime capability pages in `packages/site/content/runtime/**` so they describe the unified authored model
- MUST reflect the approved schema decisions: current local layouts remain, `version` is optional, `digest` is runtime-computed, and `requirements` targets `capability.id`
- MUST update runtime overview or agent-definition pages that currently imply capabilities and recipes are separate concepts
- MUST keep site runtime navigation metadata coherent after page rewrites
- MUST align examples and explanatory copy with `docs/agents/capabilities.md` and the finalized backend behavior
- SHOULD keep operator-facing explanations focused on authoring, discovery visibility, and runtime expectations rather than raw protocol detail
</requirements>

## Subtasks
- [ ] 8.1 Rewrite runtime capability authoring pages for the unified model
- [ ] 8.2 Update agent-definition and runtime overview pages that reference capability behavior
- [ ] 8.3 Align examples, metadata, and navigation with the rewritten runtime capability guide
- [ ] 8.4 Cross-check site runtime wording against task_05 outputs and accepted ADRs

## Implementation Details

See TechSpec "System Architecture", "Data Models", and task_05 outputs. This task should make the runtime section of the site a clean operator-facing mirror of the repository capability guide, while leaving deep protocol detail to task_07.

### Relevant Files
- `packages/site/content/runtime/core/agents/capabilities.mdx` - primary runtime capability page that must reflect the unified authored model
- `packages/site/content/runtime/core/agents/definitions.mdx` - agent-definition docs that may reference capability-sidecar behavior
- `packages/site/content/runtime/core/configuration/agent-md.mdx` - configuration guide that may need updated capability catalog references
- `packages/site/content/runtime/core/overview/what-is-agh.mdx` - runtime overview page that may still describe old concepts or flows
- `packages/site/content/runtime/core/agents/meta.json` - runtime agents-section navigation metadata
- `docs/agents/capabilities.md` - rewritten repository guide that the site runtime copy must mirror

### Dependent Files
- `packages/site/content/runtime/core/overview/architecture.mdx` - architecture page may need wording updates if it mentions network/runtime capability concepts
- `packages/site/content/runtime/core/skills/overview.mdx` - may need contrast wording to avoid capability-vs-skill confusion
- `packages/site/content/runtime/core/meta.json` - runtime section metadata may need updates after page rewrites
- `packages/site/content/runtime/meta.json` - top-level runtime nav may need ordering or labeling adjustments

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - runtime docs should describe one surviving concept
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - governs authoring layout, field model, and digest/version semantics

## Deliverables
- Updated runtime site docs for capability authoring and behavior
- Runtime examples aligned with the finalized unified schema **(REQUIRED)**
- Navigation and metadata updates for any touched runtime pages **(REQUIRED)**
- Consistency checks against `docs/agents/capabilities.md` and the ADRs **(REQUIRED)**
- Documentation quality checks with no conflicting capability/recipe explanations left in runtime-facing site docs **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Runtime capability pages describe the current local layouts and derived `digest` behavior correctly
  - [ ] Runtime docs explain `version` and `requirements` consistently with the unified schema
  - [ ] Agent-definition and overview pages no longer imply recipe is a separate authored/runtime primitive
  - [ ] Runtime metadata files remain valid after any page rewrites or nav changes
- Integration tests:
  - [ ] The runtime site section reads consistently with `docs/agents/capabilities.md` and task_05 outputs
  - [ ] Capability-vs-skill explanations remain clear without reintroducing the old capability/recipe split
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The runtime section of `packages/site` teaches one unified capability model for authors and operators
- Site runtime docs stay aligned with repository docs instead of drifting into a separate explanation
