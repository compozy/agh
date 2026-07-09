# Design Spec Template

Canonical structure for `.compozy/tasks/<name>/_design-spec.md`. Fill every applicable
section; delete sections that do not apply (no stubs). Reference implementations:
`docs/design/opendesign/LOOPS-DESIGN-SPEC.md`, `.compozy/tasks/marketplace/_design-spec.md`.

## Contents

- Header block (purpose, sources, status legend)
- §1 Scope
- §2 Design foundation
- §3 Shared anatomy contracts
- §4+ Screens
- Overlays
- Deltas
- Microcopy
- Accessibility & keyboard
- Final `[VERIFY]` section

---

## Header block (required, verbatim shape)

```markdown
# <Feature> — Design Specification

> **What this document is.** An implementation-facing UI/UX specification for <feature>,
> written to be drawn against: the HTML artifacts in `docs/design/opendesign/` must
> implement these screens exactly, and any deviation must be argued back into this document.
>
> **What it is for.**
> 1. **Design phase** — sections N–M are the per-screen build brief: anatomy, data
>    contracts, state matrices, interactions.
> 2. **PRD / TechSpec review** — the final section lists every place the design surfaces
>    a control, field, default, or flow the spec must confirm. Treat it as the actionable diff.
>
> **Sources of truth (do not contradict).**
> - Decisions: `.compozy/tasks/<name>/_brief.md` + `adrs/adr-NNN.md`.
> - Design system: `DESIGN.md` + `packages/ui/src/tokens.css` + component recipes.
> - Listing pattern: `docs/design/opendesign/LISTING-STANDARD.md` (when listings are in scope).
> - Copy: `COPY.md` + `docs/_memory/glossary.md`.
>
> **Status legend:** `[locked-adr]` traced to an ADR · `[design-assert]` deliberate design
> choice consistent with the ADRs · `[VERIFY]` the PRD/TechSpec must confirm or define ·
> `[UNCONFIRMED]` not found in code, flagged.
```

## §1 Scope — surfaces

Table: `# | Surface | Route | Kind of change (New / Slimmed / Split / Moved / Delta)`.
Note the app-shell convention: artifacts mock content + in-page header only; rail and
sidebar live in the shell.

## §2 Design foundation

- **Register** (almost always Product for runtime UI; Brand only for `packages/site`).
- **Scene sentence** — one sentence of physical context that forces the design answers.
- **Dials** — `VISUAL_VARIANCE` / `MOTION_INTENSITY` / `INFORMATION_DENSITY` with values
  and one-line rationale each.
- **§2.x Accent budget `[design-assert]`** — where the single `--color-accent` target
  lives per viewport, and the explicit list of elements that must stay neutral (grids of
  repeated CTAs are the canonical overload).
- **§2.x Action vocabulary** — fixed verb table per surface/kind when verbs differ
  (verb, click behavior, in-flight label). Ban synonyms explicitly.
- **§2.x Verified contrast (measured)** — the measured pair table from the contrast run:
  pair, ratio, verdict. Failures get consequences:
  resolve-at-source, affected components, and a pointer into the final `[VERIFY]` list.
- **§2.x Motion contract `[design-assert]`** — allowed durations/easings (tokens only)
  and the explicit ban list (grid entrances, stagger, load choreography, shimmer beyond
  an opacity pulse).

## §3 Shared anatomy contracts

One subsection per shared element, each with:

- **Composition** — ASCII anatomy + which existing compound/primitive implements each slot.
- **State matrix** — table `State | Treatment` covering at minimum: default, hover,
  focus-visible, disabled (with reason), loading/in-flight, empty, error, plus domain
  states (installed, update-available, selected…). A state listed but not designed is a
  spec defect.
- **Worst-case content rule** — truncation/clamp behavior, i18n +30%, geometry stability
  across states.

Also as subsections when applicable:

- **Search contracts** (which input, scope, shortcut, debounce, URL sync).
- **URL contract** — table `URL | Surface`; state lives in search params, shareable and
  back-button-correct.
- **Loading vocabulary** — skeletons hold known geometry; spinners only inside buttons
  or inline indicators; `RouteState`/equivalent reserved for errors and unknown-shape loads.

## §4+ Screens (one section per surface)

Each screen section carries, in order:

1. **Job** — one sentence from the user's perspective. More than one job → split the screen.
2. **Layout** — ASCII block naming every zone and the owning primitive (Topbar slots,
   page head, toolbar, content grid/rows, rail widths from tokens).
3. **Behavior bullets** — interactions, deep links, param handling, per-kind deltas.
4. **State matrix** — same contract as §3, screen-level (include partial-failure states
   when the screen composes multiple sources: one source down must not blank the page).

## Overlays section

Open with the modal justification paragraph (product register bans modal-as-first-thought;
state why each overlay legitimately blocks). Per overlay: width token, ASCII anatomy,
validation/error/in-flight behavior, success path (toast copy + where focus returns),
and the a11y contract (focus trap, labelled-by, Esc, `aria-live` for async phases).
Destructive overlays state exactly what is deleted; typed confirmation only when
consequences are real.

## Deltas section

For every existing surface this feature changes: what is removed (hard cuts, no aliases —
greenfield policy), what is added, and cross-links. Include settings and sidebar deltas
with exact array/section names when known.

## Microcopy section

Rules line first (COPY.md + DESIGN.md §7): concrete nouns/verbs, no welcome copy, no
exclamation marks, no emoji, no em dashes in UI strings, sentence case except the
Eyebrow contract, verb + object labels (bare verb allowed only where the object is the
immediate context, e.g. card footers). Then a `Surface | Copy` table with the exact
strings for empties, errors, toasts, tooltips, destructive dialogs.

## Accessibility & keyboard section

WCAG 2.2 AA floor tripwires + the screen-specific contracts: tab-stop design for
composite elements (card link vs footer button), `aria-label`s that carry the object
name, ARIA patterns per widget (dialog, tabs), live-region announcements for async
results, shortcut suppression rules, reduced-motion posture. Include the
disabled-control rule: policy/state-blocked controls use `aria-disabled` (focusable,
announces reason) — a truly `disabled` control cannot fire its tooltip.

## Final section: `[VERIFY]` — actionable diff for PRD/TechSpec

Numbered list. Every item is one decision or confirmation the PRD/TechSpec owes back:
API capabilities (browse/pagination/fields), schema definitions, config keys, policy
semantics, deep-link params, and any design-system findings from the contrast run
(resolve-at-source items). This list is the spec's contract with the next pipeline step.

Close the document with a one-line footer naming the next step (build the HTML artifacts,
then run the PRD pipeline with this section as input).
