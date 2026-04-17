---
status: completed
title: "Docs: Protocol Overview & Specification v0"
type: docs
complexity: high
dependencies:
    - task_03
---

# Task 19: Docs: Protocol Overview & Specification v0

## Overview

Write the eight protocol documentation pages — the single most important documentation section for AGH. The protocol (AGH Network) is AGH's key differentiator: an open wire format enabling agent-to-agent communication across runtimes. This section includes an accessible overview (Diataxis **Explanation**) and seven detailed specification pages (Diataxis **Reference**). Content is adapted from RFC 003 with an editorial pass for external audience readability. Rich examples, Mermaid diagrams, and JSON envelope samples are essential.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Protocol"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — overview MUST follow Diataxis Explanation, spec pages MUST follow Diataxis Reference
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- THIS IS THE MOST IMPORTANT DOC SECTION — AGH's key differentiator. Quality bar is highest here.
- Content is adapted from `docs/rfcs/003_agh-network-v0.md` — rewrite for external audience, do not copy-paste
- Spec pages MUST use normative language (MUST/SHOULD/MAY per RFC 2119) for protocol requirements
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Protocol pages live in the `protocol` collection, not `runtime`
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/protocol/overview.mdx` — what AGH Network is, why it exists, protocol vs runtime distinction, landscape positioning, "MCP connects agents to tools, AGH connects agents to agents"
- MUST create `packages/site/content/protocol/envelope.mdx` — envelope wire format, all fields (version, kind, id, source, target, body, etc.), JSON schema, annotated example
- MUST create `packages/site/content/protocol/message-kinds.mdx` — all 7 message kinds with their semantics, required/optional fields, and JSON examples for each
- MUST create `packages/site/content/protocol/interactions.mdx` — interaction lifecycle (request→response, fire-and-forget, streaming), state diagrams, timeout semantics
- MUST create `packages/site/content/protocol/peer-discovery.mdx` — peer discovery mechanism, announce/discover messages, peer identity, capability advertisement
- MUST create `packages/site/content/protocol/recipes.mdx` — recipe message kind, multi-step coordination patterns, recipe lifecycle
- MUST create `packages/site/content/protocol/delivery.mdx` — delivery guarantees, ordering, at-most-once/at-least-once semantics, transport independence
- MUST create `packages/site/content/protocol/examples.mdx` — complete end-to-end examples: two agents coordinating on a task, agent discovery flow, recipe execution
- MUST create `packages/site/content/protocol/meta.json` — sidebar ordering
- MUST read `docs/rfcs/003_agh-network-v0.md` thoroughly and adapt (not copy) for external audience
- MUST read `internal/network/` for implementation details and accuracy
- MUST include Mermaid sequence diagrams for interaction lifecycle
- MUST include annotated JSON examples for every message kind
- MUST use RFC 2119 normative language (MUST/SHOULD/MAY) in specification pages
- SHOULD include a Mermaid diagram showing the protocol stack layers
- SHOULD include a "Protocol at a Glance" summary table in the overview
</requirements>

## Subtasks

- [ ] 19.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 19.2 Read `docs/rfcs/003_agh-network-v0.md` thoroughly — this is the primary source
- [ ] 19.3 Read `internal/network/` — implementation of protocol types, envelope, message kinds
- [ ] 19.4 Write `overview.mdx` — accessible protocol introduction for implementers
- [ ] 19.5 Write `envelope.mdx` — wire format specification with JSON schema
- [ ] 19.6 Write `message-kinds.mdx` — all 7 message kinds with semantics and examples
- [ ] 19.7 Write `interactions.mdx` — interaction lifecycle with Mermaid state diagrams
- [ ] 19.8 Write `peer-discovery.mdx` — discovery mechanism specification
- [ ] 19.9 Write `recipes.mdx` — recipe coordination pattern specification
- [ ] 19.10 Write `delivery.mdx` — delivery guarantees specification
- [ ] 19.11 Write `examples.mdx` — end-to-end protocol usage examples
- [ ] 19.12 Create `meta.json` for sidebar ordering: overview → envelope → message-kinds → interactions → peer-discovery → recipes → delivery → examples
- [ ] 19.13 Verify all pages build without errors

## Implementation Details

See TechSpec sections: "Appendix B: Documentation Taxonomy — Protocol", "Appendix A: Landing Page Copy — Key Copy Lines".

The protocol documentation is adapted from RFC 003 but rewritten for an external audience of implementers who want to build AGH Network compatible software. The overview page is accessible and motivational ("why should I care?"). The specification pages are precise and normative. Every message kind gets a complete JSON example. Interaction lifecycles are illustrated with Mermaid diagrams. The examples page ties everything together with realistic end-to-end scenarios.

Key editorial differences from the RFC:

- RFCs use internal shorthand; docs spell out concepts
- RFCs assume context; docs provide it
- RFCs are design documents; docs are implementation guides
- Docs add visual aids (diagrams, annotated JSON) the RFC may lack

### Relevant Files

- `docs/rfcs/003_agh-network-v0.md` — PRIMARY SOURCE: AGH Network protocol v0 specification
- `internal/network/envelope.go` — Envelope type definition
- `internal/network/message.go` — Message kind types
- `internal/network/interaction.go` — Interaction lifecycle
- `internal/network/peer.go` — Peer discovery types
- `internal/network/recipe.go` — Recipe types
- `internal/network/transport.go` — Transport abstraction
- `docs/ideas/market-pair/gap-analysis.md` — Competitive landscape for overview positioning

### Dependent Files

- `packages/site/content/protocol/` — Output directory (protocol collection, not runtime)
- `packages/site/lib/source.ts` — Protocol docs loader must serve these pages
- Trust Profile v1 (task_20) — Extends the protocol with trust layer
- Implementation Guide (task_21) — Uses spec as reference

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Protocol is a separate collection with distinct sidebar and URL prefix

## Deliverables

- `packages/site/content/protocol/overview.mdx` — Protocol overview (Explanation)
- `packages/site/content/protocol/envelope.mdx` — Envelope specification (Reference)
- `packages/site/content/protocol/message-kinds.mdx` — Message kinds specification (Reference)
- `packages/site/content/protocol/interactions.mdx` — Interaction lifecycle specification (Reference)
- `packages/site/content/protocol/peer-discovery.mdx` — Peer discovery specification (Reference)
- `packages/site/content/protocol/recipes.mdx` — Recipe specification (Reference)
- `packages/site/content/protocol/delivery.mdx` — Delivery guarantees specification (Reference)
- `packages/site/content/protocol/examples.mdx` — End-to-end examples
- `packages/site/content/protocol/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All eight pages render at their expected URLs under `/protocol/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Overview accurately describes the protocol's purpose and positioning
  - [ ] Envelope specification covers all wire format fields with JSON schema
  - [ ] All 7 message kinds are documented with annotated JSON examples
  - [ ] Interaction lifecycle includes Mermaid sequence diagrams
  - [ ] Peer discovery mechanism is fully specified
  - [ ] Recipe coordination pattern is fully specified
  - [ ] Delivery guarantees are clearly stated
  - [ ] End-to-end examples are realistic and complete
  - [ ] Normative language (MUST/SHOULD/MAY) is used consistently in spec pages
  - [ ] No broken internal links

## Success Criteria

- All eight MDX pages build and render correctly
- Content follows Diataxis Explanation (overview) + Reference (spec) principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Protocol specification is accurate to RFC 003 and `internal/network/` implementation
- Every message kind has a complete annotated JSON example
- Mermaid diagrams are clear and accurate
- Content is rewritten for external audience (not copy-pasted from RFC)
- Normative language is used correctly per RFC 2119
- This section stands alone as a complete protocol specification
- Zero build warnings from Fumadocs
