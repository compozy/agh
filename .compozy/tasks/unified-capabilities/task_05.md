---
status: pending
title: Rewrite RFC 003 and Runtime Capability Guide
type: docs
complexity: medium
dependencies:
  - task_04
---

# Task 05: Rewrite RFC 003 and Runtime Capability Guide

## Overview

Rewrite the core repository docs so the unified capability model becomes the canonical explanation of network and runtime behavior. This task removes the old `capability + recipe` narrative from the RFC and runtime guide, replacing it with one concept that matches the implemented backend surfaces.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and completed backend tasks before starting (`_prd.md` is absent for this feature)
- REFERENCE TECHSPEC sections "System Architecture", "Data Models", "Technical Considerations", and "Architecture Decision Records"
- KEEP DOCS ALIGNED TO IMPLEMENTED SURFACES - do not document speculative protocol shapes or deferred cleanup
- REMOVE THE SPLIT MODEL COMPLETELY - explanatory bridges are acceptable, but the steady-state docs must teach one concept
- TESTS REQUIRED - doc changes must be internally consistent and traceable to the approved ADRs and backend behavior
- GREENFIELD: prefer crisp replacement over additive notes that preserve obsolete concepts in the main narrative
</critical>

<requirements>
- MUST rewrite `docs/rfcs/003_agh-network-v0.md` so `capability` is the only surviving authored and transferred artifact
- MUST update `docs/agents/capabilities.md` so local authoring, digesting, requirements, and transfer semantics reflect the unified model
- MUST describe the discovery split clearly: brief in `greet`, rich in `whois`, transfer in `kind:"capability"`
- MUST remove or explicitly supersede protocol/runtime language that still presents `recipe` as a first-class concept
- MUST keep terminology aligned with the accepted ADRs and the backend/API behavior finalized in task_04
- SHOULD add one worked example that shows authored capabilities, discovery, and transfer as one connected flow
</requirements>

## Subtasks
- [ ] 5.1 Rewrite RFC 003 sections that currently describe recipe as a separate wire/runtime artifact
- [ ] 5.2 Rewrite the runtime capability guide for the unified authored model
- [ ] 5.3 Add a worked example covering authored capability, brief/rich discovery, and transfer semantics
- [ ] 5.4 Cross-check terminology and invariants against ADR-001 through ADR-003 and task_04 outputs

## Implementation Details

See TechSpec "System Architecture", "Technical Considerations", and "Architecture Decision Records". This task should make the repository docs match the new mental model directly instead of relying on readers to infer the unification from code or ADRs.

### Relevant Files
- `docs/rfcs/003_agh-network-v0.md` - canonical protocol RFC that still needs the split model removed
- `docs/agents/capabilities.md` - runtime/operator-facing guide for capability authoring and behavior
- `.compozy/tasks/unified-capabilities/_techspec.md` - approved technical source for the new steady-state design
- `.compozy/tasks/unified-capabilities/adrs/adr-001.md` - concept unification decision
- `.compozy/tasks/unified-capabilities/adrs/adr-002.md` - authoring and schema decision
- `.compozy/tasks/unified-capabilities/adrs/adr-003.md` - transfer-kind and lifecycle decision

### Dependent Files
- `packages/site/content/protocol/*.mdx` - site protocol docs will derive terminology and examples from the rewritten RFC
- `packages/site/content/runtime/core/agents/capabilities.mdx` - site runtime docs will inherit the updated authored model
- `.compozy/tasks/unified-capabilities/task_07.md` - protocol-reference site updates depend on the final RFC wording
- `.compozy/tasks/unified-capabilities/task_08.md` - site runtime capability docs depend on the updated runtime guide

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - defines the steady-state concept that docs must teach
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - defines the authored schema and digesting model
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - defines the protocol/lifecycle semantics the RFC must document

## Deliverables
- Rewritten RFC 003 sections describing unified capabilities and transfer semantics
- Updated runtime capability guide aligned with the finalized backend/API behavior
- One worked example connecting authored capability catalogs to discovery and transfer **(REQUIRED)**
- Documentation cross-check against ADRs and task_04 outputs **(REQUIRED)**
- Documentation quality checks with no contradictory recipe-first explanations left in the main docs **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] RFC 003 describes `kind:"capability"` as the transfer artifact and no longer treats recipe as first-class
  - [ ] The runtime capability guide documents `version`, runtime-computed `digest`, and `requirements` consistently
  - [ ] Discovery responsibilities are described consistently across `greet`, `whois`, and transfer sections
- Integration tests:
  - [ ] The rewritten docs remain consistent with ADR-001 through ADR-003 and the finalized backend/API behavior from task_04
  - [ ] The worked example uses one concept end to end and does not reintroduce a second artifact type through examples or terminology
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Repository docs teach one coherent capability model instead of a capability/recipe split
- Task_07 and task_08 can build site-facing docs from a stable, rewritten source narrative
