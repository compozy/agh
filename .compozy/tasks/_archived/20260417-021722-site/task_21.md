---
status: completed
title: "Docs: Protocol Implementation Guide"
type: docs
complexity: medium
dependencies:
    - task_19
---

# Task 21: Docs: Protocol Implementation Guide

## Overview

Write the four protocol implementation tutorial pages that guide developers through building an AGH Network-compatible implementation from scratch. These are Diataxis **Tutorial** type — they walk implementers through concrete steps to achieve a working protocol participant, from minimal sender to full trust verification. Code examples in Go and pseudocode make the guide language-agnostic. This section bridges the gap between specification (task_19/task_20) and practice.

<critical>
- ALWAYS READ the TechSpec before starting — see "Appendix B: Documentation Taxonomy — Protocol (Implementation Guide)"
- ACTIVATE `/documentation-writer` and `/copywriting` before writing or revising any content — all pages MUST follow Diataxis Tutorial principles
- ACTIVATE `/qmd` and search markdown collections that include `.compozy/tasks/_archived/`, `.codex/ledger/`, and `.codex/plans/` — mine prior specs, techspecs, archived tasks, ledger entries, and planning notes for terminology, decisions, and edge cases that belong in the docs (reconcile with current code; archived material may be stale)
- REFERENCE TECHSPEC for site structure — do not duplicate TechSpec here
- Each MDX page MUST include proper frontmatter (`title`, `description`, optionally `icon`)
- Protocol pages live in the `protocol` collection, not `runtime`
- DEPENDS ON task_19 — these pages reference the protocol spec extensively
- Tutorials must work — every code example must be correct and runnable
- Include BOTH Go code and language-agnostic pseudocode
- USE SUBAGENTS to explore the codebase and any RFCs or contracts relevant to this section — map real packages, CLI surfaces, configuration keys, and runtime flows; do not draft from memory or uncited assumptions
- BE EXPLICIT and DELIVER REAL USER VALUE: document actual behavior (defaults, limits, errors, edge cases); prefer one verified, concrete walkthrough over generic marketing language
- USE RICH MDX STRUCTURE: clear hierarchy, tables, and scannable lists; employ Fumadocs/UI components (Steps, Tabs, Cards, Callouts) when they clarify; add Mermaid diagrams when relationships are easier to see than to read
- INCLUDE WORKED EXAMPLES whenever a command, configuration, payload, or sequence would be unclear without one — validate every snippet against the repository (CLI commands must run as written)
- **MANDATORY DOCS QA (agent-browser)** — Before marking this task complete, use the **agent-browser** skill to exercise the documentation in a real browser: run the site locally (e.g. `make site-dev` from the repo root per Makefile), open every route touched by this task, follow representative internal links, and confirm MDX renders without runtime errors, broken navigation, or blank sections; fix issues before finishing. Required for all documentation tasks.
</critical>

<requirements>
- MUST create `packages/site/content/protocol/guide/minimal-sender.mdx` — build the smallest possible AGH Network sender: construct an envelope, serialize to JSON, send via stdout/file. No transport, no trust — just a valid envelope.
- MUST create `packages/site/content/protocol/guide/nats-transport.mdx` — add NATS transport: connect to NATS, map subjects, send and receive messages, handle subscriptions, implement basic request-response
- MUST create `packages/site/content/protocol/guide/trust-verification.mdx` — add trust layer: generate Ed25519 keys, sign messages with JCS, verify incoming signatures, handle trust chain
- MUST create `packages/site/content/protocol/guide/testing.mdx` — test your implementation: conformance test suite, testing against AGH's built-in echo peer, integration testing strategies
- MUST create `packages/site/content/protocol/guide/meta.json` — sidebar ordering
- MUST include Go code examples that compile and run
- MUST include language-agnostic pseudocode alongside Go examples
- MUST build incrementally — each page builds on the previous
- MUST reference spec pages (task_19, task_20) for normative details rather than duplicating
- MUST read `internal/network/` for reference implementation patterns
- MUST read `docs/rfcs/003_agh-network-v0.md` and `docs/rfcs/004_agh-network-v1.md` for spec details
- SHOULD include a "What you'll build" section at the start of each page
- SHOULD include a "Verify it works" section at the end of each page
- SHOULD provide a complete working example repository reference
</requirements>

## Subtasks

- [ ] 21.1 Activate `/documentation-writer` and `/copywriting` for Diataxis compliance and reader-facing polish
- [ ] 21.2 Read `docs/rfcs/003_agh-network-v0.md` and `docs/rfcs/004_agh-network-v1.md` for specification details
- [ ] 21.3 Read `internal/network/` for reference implementation patterns and code examples
- [ ] 21.4 Write `minimal-sender.mdx` — envelope construction tutorial with Go + pseudocode
- [ ] 21.5 Write `nats-transport.mdx` — NATS transport implementation tutorial
- [ ] 21.6 Write `trust-verification.mdx` — Ed25519 trust implementation tutorial
- [ ] 21.7 Write `testing.mdx` — conformance testing tutorial
- [ ] 21.8 Create `meta.json` for sidebar ordering: minimal-sender → nats-transport → trust-verification → testing
- [ ] 21.9 Verify all pages build without errors
- [ ] 21.10 Verify Go code examples compile (spot-check)

## Implementation Details

See TechSpec section: "Appendix B: Documentation Taxonomy — Protocol (Implementation Guide)".

The implementation guide follows a progressive disclosure pattern:

1. **Minimal sender**: Construct and serialize a valid envelope — proves the implementer understands the wire format
2. **NATS transport**: Connect to a real transport — enables actual message exchange
3. **Trust verification**: Add cryptographic identity — enables secure communication
4. **Testing**: Validate conformance — proves the implementation is correct

Each page starts with "What you'll build", walks through incremental code additions, and ends with "Verify it works". Go code is the primary language (matching AGH's implementation), but pseudocode accompanies each example for implementers using other languages.

### Relevant Files

- `docs/rfcs/003_agh-network-v0.md` — Protocol v0 specification (envelope, messages, interactions)
- `docs/rfcs/004_agh-network-v1.md` — Trust profile and NATS binding specification
- `internal/network/envelope.go` — Reference envelope implementation
- `internal/network/message.go` — Reference message kind implementation
- `internal/network/transport.go` — Reference transport abstraction
- `internal/network/nats.go` — Reference NATS transport (if exists)
- `internal/network/trust.go` — Reference trust implementation (if exists)
- `internal/network/sign.go` — Reference signing implementation (if exists)

### Dependent Files

- `packages/site/content/protocol/guide/` — Output directory (protocol collection)
- Protocol Spec v0 (task_19) — Spec pages referenced throughout
- Trust Profile v1 (task_20) — Trust and NATS spec referenced in later pages

### Related ADRs

- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Protocol is a separate collection

## Deliverables

- `packages/site/content/protocol/guide/minimal-sender.mdx` — Minimal sender tutorial
- `packages/site/content/protocol/guide/nats-transport.mdx` — NATS transport tutorial
- `packages/site/content/protocol/guide/trust-verification.mdx` — Trust verification tutorial
- `packages/site/content/protocol/guide/testing.mdx` — Conformance testing tutorial
- `packages/site/content/protocol/guide/meta.json` — Sidebar ordering

## Tests

- Build verification:
  - [ ] `turbo run build --filter=packages/site` completes without errors
  - [ ] All four pages render at their expected URLs under `/protocol/guide/`
- Content verification:
  - [ ] Each page has valid frontmatter with `title` and `description`
  - [ ] Each page includes "What you'll build" and "Verify it works" sections
  - [ ] Go code examples are syntactically correct (spot-check compilation)
  - [ ] Pseudocode accompanies every Go example
  - [ ] Pages build incrementally on each other
  - [ ] References to spec pages (task_19, task_20) are correct and not broken
  - [ ] No broken internal links
- Code verification:
  - [ ] Minimal sender example produces a valid envelope JSON
  - [ ] NATS transport example shows correct subject mapping
  - [ ] Trust verification example uses correct Ed25519+JCS algorithm

## Success Criteria

- All four MDX pages build and render correctly
- Content follows Diataxis Tutorial principles (verified by `/documentation-writer` + `/copywriting` workflow)
- Each page is completable as a standalone learning exercise
- Go code examples are correct and could compile
- Pseudocode is clear and language-agnostic
- Progressive complexity works: each page builds on the previous
- References to spec pages are accurate and helpful
- The guide enables an implementer to build a conformant AGH Network participant
- Zero build warnings from Fumadocs
