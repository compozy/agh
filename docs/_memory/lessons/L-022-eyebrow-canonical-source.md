# L-022 — Eyebrow typography needs one canonical source

**Class:** Frontend / Design system
**Date discovered:** 2026-05-10 (DashboardCard inline-eyebrow audit while landing the redesign branch)
**Evidence sources:** Audit run on the `redesign` branch ahead of the dashboard polish; fixed in
the `eyebrow-adjusts` work — `packages/ui/src/components/custom/eyebrow.tsx`,
`packages/ui/src/tokens.css`, `DESIGN.md` §3, `.claude/skills/agh-design/SKILL.md`,
`packages/ui/src/lib/utils.ts`, and the 27-file `web/src` sweep.

## Context

While reviewing why the `DashboardCard` "ACTIVE SESSIONS / DAEMON UPTIME / QUEUE DEPTH" labels
felt drifty, an audit surfaced **124 callsites** across the monorepo applying the JetBrains-Mono
uppercase eyebrow style. ~62 used the `<Eyebrow>` primitive and ~62 inlined the same idea by
hand. At least **five different tuples** were in active use:

- `text-[10.5px]` + `tracking-[0.05em]` (the original `<Eyebrow>` body)
- `text-eyebrow` (11 px) + `tracking-mono` (0.06em) (the token-aligned majority)
- `text-badge` (10 px) + `tracking-badge` (0.08em) (legacy chip tracking applied to eyebrows)
- `text-[11px]` + `tracking-[0.06em]` hard-coded inside chat/metric/wire-card primitives
- `text-[10px]` + `tracking-[0.08em]` arbitrary tuples in dialog labels

The drift was **triplicated in the spec layer too**: `DESIGN.md` table §3 line 147 said
`Inter | 10.5 px | 0.05em UC`, `tokens.css` declared `--tracking-mono: 0.06em`, and the
`agh-design` skill brief said `letter-spacing: 0.06em`. The visible eyebrow in the UI was
JetBrains Mono, not Inter — so the spec, the token, and the implementation all disagreed at
once.

`twMerge` made it worse: with the default class group config, `text-eyebrow` and `text-(--muted)`
collapsed into the same "text-color" group, so `cn("text-eyebrow", "text-(--muted)")` silently
dropped the size. Any consumer that relied on token-aligned size + token-aligned color saw the
size erased without warning.

## Root cause

The eyebrow style had **no single source of truth**. Three artifacts held overlapping authority
(spec table / CSS token / component implementation), each authored at a different time, and the
`<Eyebrow>` API wasn't expressive enough (no `size`, no `subtle`/`strong` tone) to stop callers
from inlining their own variant. Because `cn()` quietly collapsed token-named text utilities
into the text-color group, the few callers that did try to use tokens still couldn't compose
size + color through `<Eyebrow>` cleanly. Every new feature kept reaching for arbitrary values
(`text-[10.5px]` / `tracking-[0.05em]`) because the canonical tuple was simultaneously
under-specified and contradicted between sources.

## Rule

> One eyebrow primitive, one tracking value, three sizes, one set of tones — and `cn()` knows
> the project's token names. Every uppercase JetBrains-Mono label in `web/`, `packages/site/`,
> and `packages/ui/`'s public consumer surface MUST render through `<Eyebrow>` (`@agh/ui`) using
> token classes. The canonical tracking is `--tracking-mono` (0.06em). The canonical sizes are
> `--text-eyebrow` (11 px), `--text-badge` (10 px), `--text-micro` (9 px). The canonical tone
> map covers `muted` (default), `subtle`, `strong`, plus the signal palette.

Inlining `font-mono` + `uppercase` + a `text-*` + a `tracking-*` tuple in product `<span>`,
`<p>`, or `<div>` content is forbidden. Structural primitives that must apply eyebrow typography
to a non-span element (`<dt>`, `<label>`, breadcrumb wrappers, sidebar/table headers) live
inside `@agh/ui` — consumers always reach for `<Eyebrow>`. Arbitrary values like
`text-[10.5px]` / `tracking-[0.05em]` are forbidden everywhere, including the design-system
implementation files.

## Operationalization

- `packages/ui/src/components/custom/eyebrow.tsx` is the single primitive. Adding a new size or
  tone happens there, behind a new prop variant — never inline.
- `packages/ui/src/lib/utils.ts` extends `tailwind-merge` with the project's `font-size` group
  (`text-eyebrow`, `text-badge`, `text-micro`, `text-small-body`, `text-display-2xl`, …). Any
  new `--text-*` token in `tokens.css` MUST be added to that group on the same change.
- `DESIGN.md` §3 holds the authoritative type ladder. The eyebrow row references the tokens by
  name. Drift between this row, `tokens.css`, the `agh-design` skill brief, and `<Eyebrow>` is
  treated as a code defect, not a documentation tweak.
- A repo-wide guard rejects new inlined `font-mono` + `uppercase` tuples in `web/src` and
  `packages/site` (see the redesign-branch oxlint setup).
- When auditing for drift, search both `font-mono.*uppercase` and arbitrary `text-[Npx]` /
  `tracking-[Nem]` patterns. The audit only tells the truth when both forms are scanned.

## Anti-pattern

- Inlining `font-mono text-[10.5px] uppercase tracking-[0.05em] text-(--muted)` "just for one
  span" — every callsite that did this turned into a permanent drift point.
- Updating `<Eyebrow>` to support a new visual variant by overriding its className at the
  callsite instead of adding a prop variant.
- Changing `--tracking-mono` to "match the component" instead of fixing the component to
  match the token.
- Adding a new `--text-*` token to `tokens.css` without registering it in the
  `extendTailwindMerge` config — it will silently collide with `text-color`.
- Treating `DESIGN.md` and `tokens.css` as independent specs. They are two views of one
  contract; if they disagree, both are wrong until reconciled.

## Source

- `packages/ui/src/components/custom/eyebrow.tsx` — the single primitive (post-fix).
- `packages/ui/src/components/custom/dashboard-card.tsx` — the original drift report (DashboardCard inlined the
  eyebrow tuple instead of consuming `<Eyebrow>`).
- `packages/ui/src/lib/utils.ts` — `extendTailwindMerge` registration of project font-size
  tokens.
- `packages/ui/src/tokens.css` — `--text-eyebrow`, `--text-badge`, `--text-micro`,
  `--tracking-mono`.
- `DESIGN.md` §3 ("Type Ladder", "Typography Principles") — authoritative type ladder + Eyebrow
  rule.
- `.claude/skills/agh-design/SKILL.md` — brand brief reaffirming `--tracking-mono`.
- `web/CLAUDE.md` ("Critical Rules") and `packages/site/CLAUDE.md` ("Critical Rules") — surface
  guards for the rule.
