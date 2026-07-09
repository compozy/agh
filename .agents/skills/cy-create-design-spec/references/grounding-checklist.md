# Grounding Checklist

Everything steps 2–4 and 6 of the workflow require: authorities, the API-truth
procedure, the contrast run, the hardening pass, and the traps that recur in AGH design
work. Work top to bottom; nothing here is optional.

## Contents

- §1 Skills to activate
- §2 Authorities to read
- §3 Component-recipe truth
- §4 API-truth procedure
- §5 Contrast run
- §6 Hardening pass (product register)
- §7 Known traps

## §1 Skills to activate (before any visual decision)

- `agh-design` — token invariants, dark-only warm ramp, flat depth, accent scarcity.
- `ui-craft` — HARD-GATE (brand authorities, job sentence, state list, scene sentence,
  register), anti-slop gates, state-matrix discipline.
- `impeccable` — when available: product-register rules, absolute bans, contrast rigor.

## §2 Authorities to read (normative, in this order)

1. `DESIGN.md` (repo root) — generated token tables + component contracts + anti-patterns (§10).
2. `packages/ui/src/tokens.css` — only when DESIGN.md leaves a value ambiguous (DESIGN.md wins conflicts; flag divergence).
3. `docs/design/opendesign/LISTING-STANDARD.md` + `catalog-design-system.html` — when any listing/inventory/browse surface is in scope.
4. `COPY.md` + `docs/_memory/glossary.md` — before writing any UI string; `capability` is the canonical artifact term.
5. `PRODUCT.md` — register confirmation (runtime UI = product; `packages/site` = brand).
6. `web/CLAUDE.md` — frontend conventions when the spec names routes/hooks/systems.

## §3 Component-recipe truth

For **every** primitive the spec names (`CatalogCard`, `Pill`, `Topbar`, `Empty`,
`DetailHeader`, `PillGroup`, `Tabs`, `SearchInput`, …): read its recipe in
`packages/ui/src/components/**` and spec against the real compound slots and props —
never a remembered or imagined API. If a needed primitive has no recipe, spec the
closest existing one and record a `[VERIFY]` token/component proposal; do not invent chrome.

## §4 API-truth procedure

Goal: every data element in the spec is **verified** or **[VERIFY]** — no third state.

1. Locate the surface's routes in `internal/api/httpapi/routes.go` (UDS mirror:
   `internal/api/udsapi/routes.go`).
2. Extract exact payload fields from `web/src/generated/agh-openapi.d.ts` (grep the
   operation id) or from an existing component already consuming the endpoint.
3. Record per surface: fields that exist (name them), fields the design wants but the
   API lacks (→ `[VERIFY]` items), and behavioral capabilities (empty-query browse,
   pagination totals, search params) — capabilities are claims too.
4. Asymmetries between similar kinds are findings, not annoyances (e.g. extensions
   expose `trust{decision, registry_tier, checksum_verified, warnings[]}` + `downloads`;
   skills expose only `author/version/description` → no trust pill for skills).

## §5 Contrast run (measured, never presumed)

Helper (read-only): `.claude/skills/ui-craft/scripts/check-contrast.mjs`.

1. List every information-bearing pair the screens use: text token on its owning
   surface (including hover surfaces), button ink on button fill, signal text on
   tint/surface, meta/eyebrow text on card surfaces.
2. Run in batch mode:

   ```bash
   node .claude/skills/ui-craft/scripts/check-contrast.mjs --json pairs.json
   # pairs.json: [{"label":"subtle on canvas-soft","fg":"#76767c","bg":"#1a1918"}, …]
   ```

3. Floors: 4.5:1 body/small text · 3:1 large text and non-text indicators.
4. Record the full measured table in the spec (§2.x Verified contrast), pass and fail.
5. Failures on information-bearing pairs are **resolve-at-source** findings (retune the
   token or change the component contract, co-shipped, `make codegen` after token
   changes) — never a callsite override (DESIGN.md §2). Add to the final `[VERIFY]` list
   with the affected consumers.

## §6 Hardening pass (product register)

Run the finished draft through these gates and fold outcomes back in:

- **Loading** — skeletons hold known geometry; `Spinner` only in buttons/inline
  indicators; route-level spinner states in a matrix are defects.
- **Modals** — justify every overlay (required input, consequence diff, destructive
  intent, trust decision); ban new modals outside the justified set; exhaust inline first.
- **Worst-case content** — 40+ char names, 200+ char descriptions, i18n +30%,
  geometry stable across all states; add the rule to the shared anatomy section.
- **Focus/disabled traps** — `aria-disabled` for blocked-but-explained controls;
  two-tab-stop design for card-link + footer-button composites; `aria-label` carries the
  object name.
- **Microcopy** — no em dashes in UI strings, no AI vocabulary, verb + object labels,
  empties teach, errors name the failing operation + recovery.
- **Accent budget** — recount accent targets per viewport after every edit.
- **Motion** — confirm the ban list survived (no grid entrances, no stagger, no load
  choreography).

## §7 Known traps

- **Eyebrow sections** — `<Eyebrow>` is AGH's deliberate structural-label system; usable
  for real content groupings, but justify it in the spec so slop audits don't flag it —
  and never one per section as scaffolding.
- **Accent grids** — any repeated card CTA in accent is the canonical accent-overload
  violation; grids act in neutral, accent lives on the detail CTA and modal confirms.
- **Empty-query browse** — search-shaped endpoints often require a query; a landing that
  shows "first N" items depends on browse capability that may not exist. Verify before
  promising idle content; spec the "Search to browse" fallback otherwise.
- **Partial composition failure** — screens composing multiple sources need a
  per-source failure state; one source down must never blank the page.
- **`--color-subtle` on cards** — measured at 3.89:1 on `canvas-soft` (2026-07): fails
  AA for small information-bearing text. Re-measure; if still failing, it is a
  resolve-at-source finding, not a reason to use it anyway.
- **`--color-faint`** — decorative only; never carries information.
- **Disabled + tooltip** — a `disabled` control is keyboard-unreachable and its tooltip
  never fires; use `aria-disabled` + inline caption where the reason matters.
- **Update-available states** — need a real version-comparison signal; "Update" without
  detection semantics is plausible UI.
- **Counts and totals** — `Showing N of M`, section counts, and badges require a real M
  from the API; render only what returns.
