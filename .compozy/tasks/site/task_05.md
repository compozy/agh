---
status: completed
title: "Build landing page"
type: frontend
complexity: high
dependencies: [task_03]
---

# Task 05: Build landing page

## Overview

Implement the custom landing page at `packages/site/app/page.tsx` with 7 section components that introduce AGH's dual identity: a local agent runtime and an open network protocol. The landing page is a standalone Next.js page — NOT rendered inside the Fumadocs docs layout — and leads with the protocol as AGH's key differentiator. All copy, section structure, and visual decisions follow TechSpec "Appendix A: Landing Page Copy" and "Implementation Design > Data Models > Landing page sections". The page applies the DESIGN.md dark operator aesthetic via `@agh/ui` tokens.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
- **IMPECCABLE (non-blocking, Compozy-safe)** — Read and apply the **impeccable** skill (`/impeccable` — already in Claude Code; no installs). Use `.impeccable.md` when present. **Never** run `/impeccable teach` unattended (interactive; blocks `compozy start`). Treat this task + TechSpec Appendix A + DESIGN.md as the full brief — implement in one pass the same way **`/impeccable craft`** would, without slash flows that expect a live interview. After implementation, you **may** run **`/polish`** on the landing files for a non-interactive pass (alignment, spacing, typography). Obey typography, OKLCH discipline, spatial rules, motion constraints, UX writing, **AI Slop Test**, and **absolute_bans**. Appendix A copy and named AGH tokens stay authoritative.
- **Frontend copy & layout** — Apply **frontend-design** and **copywriting** (skill names / slash commands as exposed by the harness — no installs); do not run interactive discovery — PRD, TechSpec Appendix A, and DESIGN.md are the source of truth.
</critical>

<requirements>
- MUST implement `packages/site/app/page.tsx` as the landing page route, bypassing Fumadocs layout (it uses the root `app/layout.tsx` directly, not the `DocsLayout`)
- MUST create 7 section components in `packages/site/components/landing/`:
  1. `hero.tsx` — "Your agents can finally talk to each other." headline, subheadline from TechSpec Appendix A (Hero Option A), dual CTA buttons ("Read the Protocol Spec" linking to `/protocol`, "Get Started" linking to `/runtime`)
  2. `two-pillars.tsx` — Side-by-side presentation of Runtime vs Protocol as AGH's two products, each with brief description and link to its doc collection
  3. `how-it-works.tsx` — 3-step flow: install AGH, start daemon, create first session. Uses code snippets with JetBrains Mono
  4. `runtime-features.tsx` — 8 feature cards with short titles and benefit descriptions (sessions, memory, skills, workspaces, automation, bridges, hooks, extensions)
  5. `protocol-section.tsx` — 7 message kinds, interaction lifecycle diagram, emphasis on the open wire format
  6. `architecture.tsx` — Architecture diagram showing AGH's component relationships (text/SVG, not an image)
  7. `comparison.tsx` — "AGH vs typical agent harness" comparison table highlighting AGH's differentiators
  8. `final-cta.tsx` — Closing CTA block with dual buttons matching the hero CTAs
- MUST include a `packages/site/components/landing/index.ts` barrel exporting all section components
- MUST use approved copy from TechSpec Appendix A — Hero headline: "Your agents can finally talk to each other.", Subheadline: "AGH is an agent runtime with a built-in network protocol..."
- MUST use key copy lines from TechSpec Appendix A throughout sections where contextually appropriate:
  - "Single binary. No sidecars. No external services."
  - "Orchestrates real agent CLIs, not API wrappers."
  - "Copy the directory, and the agent works."
  - "MCP connects agents to tools. AGH connects agents to agents."
  - "Keep your runtime, map to AGH envelopes, implement the smallest core first."
- MUST apply DESIGN.md tokens throughout: canvas bg (#121212), surface cards (#1C1C1E), elevated elements (#2C2C2E), accent CTAs (#E8572A with hover #D14E25), text hierarchy (primary #E5E5E7, secondary #8E8E93, tertiary #636366)
- MUST use Inter for all readable content and JetBrains Mono for code snippets, metadata labels, and structural elements (uppercase with letter-spacing per DESIGN.md)
- MUST follow flat depth model: no box-shadows, no gradients, no glassmorphism — depth via background lightness stepping only (canvas -> surface -> elevated)
- MUST use divider color (#3A3A3C) for section separators and card borders where needed
- MUST be responsive: desktop-first layout optimized for 1440px, mobile-friendly with proper stacking and readable text at narrow viewports
- MUST include the shared navigation bar (created in task_03) linking to `/`, `/runtime`, and `/protocol`
- MUST import base components (Button, Badge, Card) from `@agh/ui` where applicable
- MUST NOT use any external images or CDN resources — all visual elements are CSS/SVG/text
- MUST NOT use shadows, gradients, blur effects, or decorative textures per DESIGN.md "Don'ts"
</requirements>

## Subtasks

- [x] 5.1 Create `packages/site/components/landing/` directory
- [x] 5.2 Implement `hero.tsx` with approved headline, subheadline, and dual CTA
- [x] 5.3 Implement `two-pillars.tsx` with Runtime vs Protocol split
- [x] 5.4 Implement `how-it-works.tsx` with 3-step install flow and code snippets
- [x] 5.5 Implement `runtime-features.tsx` with 8 feature cards
- [x] 5.6 Implement `protocol-section.tsx` with 7 message kinds and interaction lifecycle
- [x] 5.7 Implement `architecture.tsx` with text/SVG architecture diagram
- [x] 5.8 Implement `comparison.tsx` with AGH vs typical harness table
- [x] 5.9 Implement `final-cta.tsx` with closing dual CTA
- [x] 5.10 Create `packages/site/components/landing/index.ts` barrel export
- [x] 5.11 Update `packages/site/app/page.tsx` to compose all sections
- [x] 5.12 Apply DESIGN.md tokens: colors, fonts, spacing, flat depth
- [x] 5.13 Implement responsive layout: desktop-first, mobile stacking
- [x] 5.14 Verify all section links work (/runtime, /protocol)
- [x] 5.15 Verify `make site-build` succeeds with landing page

## Implementation Details

See TechSpec sections: "Implementation Design > Data Models > Landing page sections", "Appendix A: Landing Page Copy".

The landing page is a custom Next.js page that does NOT use the Fumadocs `DocsLayout`. It renders inside the root `app/layout.tsx` (which provides the dark theme, fonts, and shared nav) but has its own full-width layout without sidebar or TOC.

**Section flow**: Hero (full viewport height or near it) -> Two Pillars -> How It Works -> Runtime Features -> Protocol Section -> Architecture -> Comparison -> Final CTA. Each section is a self-contained React component. Sections alternate between canvas (#121212) and surface (#1C1C1E) backgrounds for visual rhythm.

**Visual structure per DESIGN.md**:

- Section containers: max-width constrained (e.g., 1200px), centered, generous vertical padding (64-96px)
- Cards inside sections: surface bg (#1C1C1E) on canvas, or elevated bg (#2C2C2E) on surface, radius 12px, padding 16-20px
- CTAs: accent fill (#E8572A), white text, 8px radius, 36px height — use Button from `@agh/ui`
- Secondary CTAs: border-only (1px solid #3A3A3C), primary text
- Feature cards: surface bg, 12px radius, JetBrains Mono label + Inter description
- Code snippets: elevated bg (#2C2C2E), JetBrains Mono, 1px solid #3A3A3C border
- Comparison table: alternating row backgrounds per DESIGN.md "Metadata Table" pattern

**Responsive behavior**: At narrow viewports, side-by-side layouts stack vertically, feature card grids collapse from 4-col to 2-col to 1-col, section padding reduces. No horizontal scroll.

### Relevant Files

- `packages/site/app/page.tsx` — Landing page route (replace placeholder from task_03)
- `packages/site/app/layout.tsx` — Root layout with shared nav (task_03 deliverable)
- `packages/ui/src/components/button.tsx` — Button component for CTAs (task_01 deliverable)
- `packages/ui/src/components/badge.tsx` — Badge component for feature tags (task_01 deliverable)
- `packages/ui/src/components/card.tsx` — Card component for feature cards (task_01 deliverable)
- `packages/ui/src/tokens.css` — DESIGN.md CSS tokens (task_01 deliverable)
- `DESIGN.md` — Canonical design reference (color palette at section 2, typography at section 3, component stylings at section 4, layout principles at section 5, flat depth model at section 6, do's and don'ts at section 7)

### Dependent Files

- None — this is a leaf task. The landing page links to /runtime and /protocol docs but does not generate or modify content.

### Related ADRs

- [ADR-001: Fumadocs as Documentation Framework](adrs/adr-001.md) — Landing page is a custom Next.js page, not a Fumadocs docs page
- [ADR-003: Two-Collection Content Architecture](adrs/adr-003.md) — Landing page links to both /runtime and /protocol collections

## Deliverables

- `packages/site/components/landing/hero.tsx`
- `packages/site/components/landing/two-pillars.tsx`
- `packages/site/components/landing/how-it-works.tsx`
- `packages/site/components/landing/runtime-features.tsx`
- `packages/site/components/landing/protocol-section.tsx`
- `packages/site/components/landing/architecture.tsx`
- `packages/site/components/landing/comparison.tsx`
- `packages/site/components/landing/final-cta.tsx`
- `packages/site/components/landing/index.ts`
- Updated `packages/site/app/page.tsx` composing all sections
- Snapshot tests for each section component
- Build passing: `make site-build`

## Tests

- Build verification:
  - [ ] `make site-build` completes without errors
  - [ ] Static export produces `out/index.html` with landing page content
- Component tests (snapshot):
  - [ ] `hero.tsx` renders headline "Your agents can finally talk to each other." and both CTA buttons
  - [ ] `two-pillars.tsx` renders Runtime and Protocol pillars with links
  - [ ] `how-it-works.tsx` renders 3 steps with code snippets
  - [ ] `runtime-features.tsx` renders 8 feature cards
  - [ ] `protocol-section.tsx` renders 7 message kinds
  - [ ] `architecture.tsx` renders architecture diagram
  - [ ] `comparison.tsx` renders comparison table with at least 5 rows
  - [ ] `final-cta.tsx` renders dual CTA buttons
- Visual compliance:
  - [ ] No box-shadows in any section component
  - [ ] No gradients in any section component
  - [ ] Background colors use only DESIGN.md tokens (#121212, #1C1C1E, #2C2C2E)
  - [ ] Accent color is #E8572A on CTA buttons
  - [ ] Font families are Inter (readable content) and JetBrains Mono (code/metadata)
- Responsive:
  - [ ] Page renders without horizontal scroll at 375px viewport width
  - [ ] Feature card grid stacks at narrow viewports
- Links:
  - [ ] Hero CTA "Read the Protocol Spec" links to `/protocol`
  - [ ] Hero CTA "Get Started" links to `/runtime`
  - [ ] All internal links use Next.js `<Link>` for client-side navigation
- Test coverage target: snapshot coverage for all 8 section components

## Success Criteria

- `make site-build` succeeds
- Landing page renders all 7 sections in correct order
- Approved copy from TechSpec Appendix A is used verbatim for hero headline and subheadline
- DESIGN.md flat depth aesthetic applied: no shadows, no gradients, correct color tokens
- Responsive layout works at desktop (1440px) and mobile (375px) viewports
- All internal links navigate correctly to /runtime and /protocol
- Snapshot tests pass for all section components
