# Memory Ledger — packages/site token drift repair

- **Goal (incl. success criteria):**
  Restore `packages/site` design after the redesign migration was applied to `packages/ui`/`web/` but skipped site. Hard rename every `--color-*`/`--color-text-*` reference to the flat new contract per `.compozy/tasks/redesign/_techspec.md` §"Token Contract". Formalize "Site Profile" in `DESIGN.md` and `.agents/skills/agh-design/SKILL.md`. Local `packages/site` must render identically to `https://agh.network`. `make verify` green.

- **Constraints/Assumptions:**
  - Greenfield zero-tolerance: no compat aliases / no dual ramp.
  - Token mapping is the techspec table (lines 199–298), not invented.
  - `--color-canvas-deep` → `--rail` (closest deepest tier; bento overlays fade-to-black).
  - `--color-text-label` → `--muted` (per techspec, replaced per call-site; site only uses it in `hero.tsx:39` inside `<Eyebrow>`).
  - Per CLAUDE.md commit style: one bundled `refactor:` commit, `make verify` before commit.
  - User chose: full Site Profile docs, dev-server visual QA via `/impeccable`.

- **Key decisions:**
  - Single deterministic sed batch ordered longest-first to avoid prefix collisions.
  - `eyebrow-badge` utility for `kind-chip.tsx:26`; drop `tracking-badge` (uses `--tracking-mono` 0.06em).
  - DESIGN.md §13 "Site Profile" extension — not a top-level rewrite (per user choice).

- **State:**
  - **Done:** Plan approved at `/Users/pedronauck/.claude/plans/depois-do-commit-730ba4b40343-async-cookie.md`. Three Explore agents mapped scope (~480 token references in 65+ files). Token mapping confirmed from techspec.
  - **Now:** Step 1 — rename tokens in `packages/site/app/global.css`.
  - **Next:** Step 2 mass rename → Step 3 eyebrow audit → Step 4 DESIGN.md → Step 5 agh-design → Step 6 gates → Step 7 visual QA.

- **Open questions (UNCONFIRMED if needed):**
  - None blocking. Visual QA may surface per-component tone overrides that need adjustment in iteration.

- **Working set (files/ids/commands):**
  - `packages/site/app/global.css` (token rename)
  - `packages/site/components/landing/hero.tsx`, `bento-section.tsx`, `network-protocol-visual.tsx`, `supported-agents.tsx`, `extensibility-section.tsx`, `primitives/feature-card.tsx`
  - `packages/site/components/blog/kind-chip.tsx` and other blog components
  - `packages/site/components/site/site-footer.tsx`, `home-header.tsx`, `docs-header.tsx`
  - `packages/site/components/docs/doc-page-masthead.tsx`, `mdx-blocks.tsx`, `mermaid.tsx`
  - `packages/site/app/changelog/page.tsx`, `app/blog/**/page.tsx`, `app/error.tsx`, `app/not-found.tsx`
  - `DESIGN.md`, `.agents/skills/agh-design/SKILL.md`
  - Reference: `.compozy/tasks/redesign/_techspec.md:199–298`
  - Verify: `bunx turbo run typecheck/test/build --filter=./packages/site`, `make bun-lint`, `make bun-typecheck`, `make bun-test`, `make verify`
