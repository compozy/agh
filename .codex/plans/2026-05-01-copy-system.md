# Create `COPY.md`: Agent-Readable Copy System for AGH

## Summary

- Create a root-level `COPY.md` as the verbal counterpart to `DESIGN.md`: `DESIGN.md` governs visual grammar; `COPY.md` governs product language, positioning, claims, vocabulary, voice, public copy, docs prose, release copy, and UI microcopy.
- Use `COPY.md` instead of `MESSAGING.md` because AGH has protocol-level message kinds, so "messaging" can be confused with network/protocol semantics.
- Frame the file internally as **Copy System: AGH**, not a narrow copywriting guide, so it can safely cover positioning, narrative, proof standards, docs, changelog, CLI help, package descriptions, and UI labels.

## Key Changes

- Add root `COPY.md` with sections for purpose/source hierarchy, positioning, message architecture, audience intent, voice/editorial rules, vocabulary, claim standards, surface playbooks, copy patterns, examples, agent prompt guide, and review/maintenance checklist.
- Update root `AGENTS.md` and `CLAUDE.md` with a sibling section to `Design System` that makes `COPY.md` mandatory before changing public copy, marketing copy, narrative docs, release copy, package metadata, UI microcopy, CLI help, SEO/OG metadata, or launch/blog content.
- Update `packages/site/AGENTS.md` and `packages/site/CLAUDE.md` to require `COPY.md` for landing components, blog/changelog, runtime/protocol docs, site config, OpenGraph metadata, and public-facing copy.
- Update `web/AGENTS.md` and `web/CLAUDE.md` to require `COPY.md` for labels, empty states, errors, onboarding/settings text, toasts, page headers, and runtime UI copy.
- Lightly update `DESIGN.md` section 7, Voice & Content, so it stays a visual-context summary and points to `COPY.md` as the canonical source for voice, positioning, claims, vocabulary, and copy patterns.

## Interfaces

- New agent-facing interface: root `COPY.md`.
- New instruction contract: AGENTS/CLAUDE files route copy, public language, and content work to `COPY.md`.
- No runtime API, CLI, OpenAPI, DB schema, or generated client changes.
- No public copy rewrite is required in v1 except small cross-reference/instruction edits.

## Test Plan

- Static content checks:
  - `rg -n "COPY\\.md|Copy System" AGENTS.md CLAUDE.md packages/site web DESIGN.md`
  - Confirm every added instruction points to root `COPY.md`.
  - Confirm `DESIGN.md` section 7 no longer acts as the full canonical voice source.
- Vocabulary/drift spot checks:
  - Search target public surfaces for forbidden legacy terms after copy edits: `recipe`, `workflow`, `procedure`, `playbook` when used as capability synonyms.
  - Search marketing copy for unsupported hype patterns: `revolutionary`, `game-changing`, `seamless`, `effortless`, `AI-powered`, `10x`, `cutting-edge`.
  - Search landing/blog copy for marketing-body `we`/`our`, allowing quoted text or contributor bylines only when intentional.
- Repo gate:
  - Run `make verify` before completion, per AGH policy. If the implementation is docs-only and `make verify` exposes unrelated existing failures, report the exact failing target and evidence instead of weakening checks.

## Assumptions

- `COPY.md` is the chosen name. It is short, agent-friendly, and avoids the protocol ambiguity of `MESSAGING.md`.
- The internal title will be `Copy System: AGH`, so the file clearly covers more than ad copy: positioning, claims, vocabulary, docs prose, UI microcopy, release copy, and public metadata.
- The file content will be English.
- Existing research under `.compozy/tasks/site-copy/analysis/*` is evidence, not permanent policy. `COPY.md` should distill durable rules and link to sources where useful.
- `docs/_memory/glossary.md` remains the canonical vocabulary authority. `COPY.md` should summarize high-use terms and point back to the glossary rather than duplicating it wholesale.
- `DESIGN.md` remains the visual authority. `COPY.md` becomes the verbal/product-language authority.
