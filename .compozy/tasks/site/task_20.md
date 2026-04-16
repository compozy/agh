---
status: pending
title: "Docs: Trust Profile v1, Transport Bindings & Conformance"
type: docs
complexity: medium
dependencies:
  - task_19
---

# Task 20: Docs: Trust Profile v1, Transport Bindings & Conformance

## Overview

Write the four protocol documentation pages covering the trust layer (Ed25519+JCS signatures), NATS transport binding, and conformance levels. These pages extend the base protocol spec (task_19) with security, transport, and compliance dimensions. All pages are Diataxis **Reference** type — precise, normative, structured for implementers. Content is adapted from RFC 004.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Protocol (Trust Profile v1, Transport Bindings, Conformance)"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis Reference principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Content is adapted from `docs/rfcs/004_agh-network-v1.md` — rewrite for external audience
- MUST use RFC 2119 normative language (MUST/SHOULD/MAY) for protocol requirements
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Protocol pages live in the `protocol` collection, not `runtime`
- DEPENDS ON task_19 — these pages extend and reference the base protocol spec
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/protocol/ed25519-jcs.mdx` — Ed25519 trust profile: key generation, JSON Canonicalization Scheme (JCS), signature envelope fields, signing algorithm, worked example with actual bytes
- MUST create `packages/site/content/protocol/verification.mdx` — signature verification flow: verification algorithm, key resolution, trust chain, handling unsigned messages, error semantics
- MUST create `packages/site/content/protocol/nats.mdx` — NATS transport binding: subject mapping, connection configuration, JetStream for persistence, NATS-specific delivery semantics, subject hierarchy design
- MUST create `packages/site/content/protocol/conformance.mdx` — conformance levels (Core, Trust, Full), required capabilities per level, conformance test suite overview, self-certification
- MUST create or update `packages/site/content/protocol/meta.json` — update sidebar ordering to include new pages after base spec pages
- MUST read `docs/rfcs/004_agh-network-v1.md` thoroughly and adapt for external audience
- MUST read `internal/network/` for trust and transport implementation details
- MUST include a worked Ed25519 signing example with step-by-step byte operations
- MUST include NATS subject hierarchy diagram or table
- MUST include conformance level comparison table
- SHOULD include code snippets (Go + pseudocode) for signing and verification
- SHOULD reference the base protocol spec pages (task_19) for envelope and message kind details
</requirements>

## Subtasks

- [ ] 20.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 20.2 Read `docs/rfcs/004_agh-network-v1.md` thoroughly — trust profile, transport, conformance
- [ ] 20.3 Read `internal/network/` — trust implementation, NATS transport, signature types
- [ ] 20.4 Write `ed25519-jcs.mdx` — Ed25519 trust profile specification with worked example
- [ ] 20.5 Write `verification.mdx` — signature verification flow specification
- [ ] 20.6 Write `nats.mdx` — NATS transport binding specification
- [ ] 20.7 Write `conformance.mdx` — conformance levels and test suite
- [ ] 20.8 Update `meta.json` to include new pages in correct sidebar position
- [ ] 20.9 Verify all pages build without errors

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Protocol (Trust Profile v1, Transport Bindings, Conformance)".

The trust profile adds cryptographic identity and message integrity to the base protocol. Ed25519 keys sign messages using JCS (JSON Canonicalization Scheme) for deterministic serialization. The NATS transport binding maps AGH Network messages to NATS subjects and provides JetStream-backed persistence. Conformance levels define what a compliant implementation must support.

### Relevant Files

- `docs/rfcs/004_agh-network-v1.md` — PRIMARY SOURCE: Trust profile, NATS binding, conformance
- `internal/network/trust.go` — Trust types, Ed25519 operations (if exists)
- `internal/network/sign.go` — Signing implementation (if exists)
- `internal/network/verify.go` — Verification implementation (if exists)
- `internal/network/nats.go` — NATS transport binding (if exists)
- `internal/network/transport.go` — Transport abstraction
- `internal/network/conformance.go` — Conformance definitions (if exists)

### Dependent Files

- `packages/site/content/protocol/` — Output directory (protocol collection)
- Protocol Overview & Spec v0 (task_19) — Base spec that these pages extend
- Implementation Guide (task_21) — References trust and NATS as implementation targets

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Protocol is a separate collection

## Deliverables

- `packages/site/content/protocol/ed25519-jcs.mdx` — Ed25519 trust profile specification
- `packages/site/content/protocol/verification.mdx` — Verification flow specification
- `packages/site/content/protocol/nats.mdx` — NATS transport binding specification
- `packages/site/content/protocol/conformance.mdx` — Conformance levels reference
- Updated `packages/site/content/protocol/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/protocol/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Ed25519 page includes worked signing example with byte-level detail
  - [ ] Verification page specifies complete verification algorithm
  - [ ] NATS page includes subject hierarchy and connection configuration
  - [ ] Conformance page includes level comparison table with required capabilities
  - [ ] Normative language (MUST/SHOULD/MAY) is used correctly
  - [ ] Cross-references to base protocol spec (task_19) pages are correct
  - [ ] No broken internal links

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Reference principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Trust profile specification is accurate to RFC 004 and implementation
- Worked Ed25519 example is correct and reproducible
- NATS transport binding is fully specified
- Conformance levels are clearly defined with required capabilities
- Pages integrate cleanly with base protocol spec (task_19) in sidebar
- Zero build warnings from Fumadocs
