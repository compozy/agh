# UI/UX Analysis — `<MODULE>` :: `<ROUTE>`

> **Status:** draft  
> **Owner subagent:** `<agent-id>`  
> **Date:** `<YYYY-MM-DD>`  
> **Module:** `<module-name>` (`<module-folder>`)  
> **Route path:** `<route-path>` (TanStack Router id: `<route-id>`)  
> **Web source:** `web/src/routes/_app/<file>.tsx`  
> **System owner:** `web/src/systems/<system>/`  
> **Storybook story id(s):** `<story-id-1>`, `<story-id-2>`  
> **Live URLs probed:** `http://localhost:5173<route>` · `http://localhost:6006/?path=/story/<story-id>`

---

## 0. Inputs & Probes (mandatory evidence)

Every claim below MUST be backed by one of these evidence sources. List each one before analysis.

- **Source files read** (relative paths, not just folder names):
  - `web/src/routes/_app/<file>.tsx`
  - `web/src/systems/<system>/components/<component>.tsx`
  - …
- **Storybook stories opened** (story id + URL):
  - `<story-id>` → `http://localhost:6006/?path=/story/<story-id>`
- **Live web probes (`localhost:5173`)** — list every page/state visited via `agent-browser`:
  - `<route>` empty state
  - `<route>` loading state (forced)
  - `<route>` error state (forced)
  - `<route>` populated state (storybook only — daemon has no data)
- **Screenshots / DOM snapshots captured** (file paths under `.compozy/tasks/ui-final/<num>_<mod>/_evidence/<route>/`):
  - `<screenshot.png>` — what it shows
- **Console / network errors observed** (verbatim):
  - …
- **Keyboard / a11y probes performed** (TAB order, focus rings, screen-reader labels via DevTools):
  - …

> **Hard rule.** If a section below cites no evidence from this list, it is invalid and must be rewritten or removed. No "general impression" findings without a probe to back them.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** one paragraph in plain English. No marketing speak, no AI-slop adjectives.
- **Primary user goal on this route** (single sentence; if you cannot pick one, that is itself a finding).
- **Entry vectors** (how users arrive — sidebar, deep link, redirect, modal trigger).
- **Exit vectors** (CTAs, links, modals that take the user elsewhere).
- **Critical states this route MUST handle** — list every legitimate state and whether it currently exists:
  | State | Implemented? | Evidence | Quality |
  |---|---|---|---|
  | First-run / empty | yes/no | `<file:line>` or `<screenshot>` | strong / weak / missing |
  | Loading / skeleton | | | |
  | Partial data | | | |
  | Populated (typical) | | | |
  | Populated (dense, 100+ rows) | | | |
  | Error (network) | | | |
  | Error (permission / 403) | | | |
  | Error (not found / 404) | | | |
  | Read-only / disabled | | | |
  | Live-update (stream / SSE) | | | |
  | Mobile / narrow viewport | | | |

---

## 2. Design Health Score (Nielsen 10)

Score 0–4 per heuristic. **Be honest. Most real interfaces score 20–32 / 40.** A 4 means genuinely excellent and you can point at evidence. Refuse to inflate.

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |       |                                    |           |
| 2  | Match between system and real world    |       |                                    |           |
| 3  | User control and freedom               |       |                                    |           |
| 4  | Consistency and standards              |       |                                    |           |
| 5  | Error prevention                       |       |                                    |           |
| 6  | Recognition rather than recall         |       |                                    |           |
| 7  | Flexibility and efficiency of use      |       |                                    |           |
| 8  | Aesthetic and minimalist design        |       |                                    |           |
| 9  | Help users recognize / recover errors  |       |                                    |           |
| 10 | Help and documentation                 |       |                                    |           |
|    | **Total**                              | **/40** | | **Band:** ✗ poor (<20) / ◯ adequate (20–28) / ◐ good (29–35) / ● excellent (36–40) |

---

## 3. AI-Slop & Anti-Pattern Verdict

Apply the `impeccable` shared design laws verbatim. For each, mark `OK` or list specific violations with `file:line` evidence.

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders (`border-l/r > 1px`) used decoratively |  | |
| Gradient text (`background-clip: text` + gradient)        |  | |
| Glassmorphism / blur as default                            |  | |
| Hero-metric template (big number + label + sparkline)      |  | |
| Identical card grids                                       |  | |
| Modal as first thought (modal where inline would do)       |  | |
| Em dashes in copy                                          |  | |
| Generic AI palette (default Tailwind blues, neon-on-black) |  | |
| Category-reflex theme (e.g. "observability ⇒ dark blue")   |  | |
| Restated headings / intros that repeat the title           |  | |
| Decorative shadows / heavy elevation (DESIGN.md = flat)    |  | |
| Hardcoded `#000` / `#fff` instead of tinted neutrals       |  | |

**Summary verdict:** If a stranger said "AI made this," would you believe them immediately? `yes / borderline / no` — explain in one sentence.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point** — count them. If >4, flag.
- **Eight-item cognitive load checklist** (run it; report failure count, 0–1 = low, 2–3 = moderate, 4+ = critical):
  1. Are >4 options visible at once? `pass / fail` — evidence
  2. Are labels self-evident without docs? `pass / fail`
  3. Is the primary action visually dominant? `pass / fail`
  4. Is information progressively disclosed (advanced hidden until needed)? `pass / fail`
  5. Do related elements group via proximity / shared container, not just colour? `pass / fail`
  6. Is hierarchy clear via scale/weight contrast (≥1.25 ratio)? `pass / fail`
  7. Is body line length within 65–75ch? `pass / fail`
  8. Is whitespace varied (rhythm) instead of uniform padding? `pass / fail`
- **Information architecture observations** (groupings, ordering, naming, redundancy):

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

`DESIGN.md` is authoritative — never invent values. Cite the rule, then the violation.

- **Color tokens:** are all colors pulled from `DESIGN.md` tokens? Any `#hex` literals or stray Tailwind defaults? Evidence:
- **Type scale:** Inter / JetBrains Mono / (Playfair Display only on site-home / NuixyberNext only for wordmark). Any other fonts? Evidence:
- **Radii / spacing:** match the system scale? Any one-off values?
- **Elevation:** flat depth model — any drop-shadows, glassmorphism, neumorphism creeping in? Evidence:
- **Signal palette discipline:** `#E8572A` action / `#30D158` success / `#FF453A` danger / `#FFD60A` warning / `#BF5AF2` info — used only for those meanings? Any decorative use?
- **Grid / rhythm:** is spacing varied or monotone?
- **Density:** comfortable / cramped / sparse — pick one and back it.

---

## 6. Interaction & Behaviour Audit

- **Primary actions** — are they unambiguous and visually dominant?
- **Destructive actions** — confirmation flow? `confirm typing` for irreversibles?
- **Forms** — inline validation? error placement? required-field affordance? autofocus first invalid field on submit?
- **Tables / lists** — sort, filter, pagination, virtualization, keyboard nav (↑↓, Enter, Esc)?
- **Selection model** — single, multi, range? Bulk-action toolbar?
- **Modals / drawers** — focus trap, ESC closes, click-outside policy, restore focus on close?
- **Live updates / streaming** — is there a visible "connected / reconnecting / stale" indicator?
- **Optimistic vs pessimistic updates** — which is used and is it obvious to the user?
- **Hover / focus / active states** — every interactive element has all three?
- **Loading patterns** — skeleton vs spinner vs progress; debounce on fast loads to avoid flash?

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** can every interactive element be reached + activated by keyboard alone? List failures.
- **Focus rings:** visible, contrast ≥3:1, never `outline: none` without replacement. Evidence (`file:line`):
- **TAB order:** logical (top-to-bottom, left-to-right)? Trap or skip?
- **ARIA roles / labels:** every `button`, `link`, icon-only control labelled? Live regions for async updates?
- **Color contrast:** body ≥4.5:1, large ≥3:1 — list any pair below threshold with the actual ratio measured.
- **Motion:** any auto-playing motion, parallax, or essential animation that ignores `prefers-reduced-motion`?
- **Text scaling:** does layout survive 200% zoom and 16px → 24px font scaling without overflow?
- **Forms:** every input has a programmatically-associated label, not just placeholder?

---

## 8. Empty / Loading / Error States

For each, pick `excellent / adequate / weak / missing` and back it.

- **Empty (first-run):** does it explain *what this is*, *why it's empty*, and *what the next step is*? Or is it a generic "No data" string?
- **Loading:** skeleton matches final layout? Spinner only as last resort? Debounce <200 ms loads?
- **Error:** specific, actionable, never "Something went wrong"? Includes retry + support path?
- **Permission denied:** distinct from generic error?
- **Stale / disconnected:** is there a banner when SSE/WebSocket drops?

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms** (`docs/_memory/glossary.md` is authoritative). Any forbidden terms? (`recipe`/`workflow`/`procedure`/`playbook` for capability, `AGENTS.md` vs `AGENT.md`, etc.)
- **Tone:** matches `COPY.md`? List any drift (marketing fluff, hype, vague claims).
- **Em dashes** — flag every `—` and every `--`.
- **Restated headings** — flag any subhead that repeats the page title.
- **Sentence case vs Title Case** — consistent?
- **Truthful UI test** — does any label, metric, or control imply functionality the daemon does not actually expose? (Truthful UI > plausible UI.) List violations.

---

## 10. Performance & Responsiveness

- **Initial render** — any obvious waterfalls or render-blocking? (Run Lighthouse / Performance tab if browser allows.)
- **Re-render hot spots** — list any components that re-render on every keystroke / interval tick (look for missing memoization).
- **List virtualization** — present where row count can exceed ~100? Evidence.
- **Bundle red flags** — any obvious oversized imports (charting lib pulled into a route that just shows a label)?
- **Responsive behaviour** — does the layout survive 320 / 768 / 1024 / 1440 / 1920 widths? Any overflow / clipping / unreachable controls?
- **Mobile interactions** — any hover-only affordances?

---

## 11. Storybook Coverage

- **Stories present:** list each story id with link.
- **States covered in Storybook:** which from the table in §1?
- **Gaps:** which states (error, dense, mobile…) are NOT covered? — actionable list.
- **Story drift:** does the story render the same component the route renders? (If Storybook renders a stale prop API, flag it.)

---

## 12. Findings — Prioritised

Use these tags strictly:
- **P0 = ship-blocker** (broken, embarrassing, or violates a documented invariant).
- **P1 = high-value polish** (clear win, no architectural cost).
- **P2 = worthwhile but optional** before release.
- **P3 = nice-to-have / parking lot.**

For every finding: `What`, `Why it matters`, `Fix (concrete)`, `Suggested impeccable command`, `Effort (S/M/L)`, `Evidence (file:line or screenshot)`.

### P0 — Ship Blockers
1. **[P0] What:** …
   - **Why:** …
   - **Fix:** …
   - **Cmd:** `/impeccable <command> <target>`
   - **Effort:** S/M/L
   - **Evidence:** `<file:line>`

### P1 — High-Value Polish
…

### P2 — Worthwhile
…

### P3 — Parking Lot
…

---

## 13. Persona Red Flags

Pick 2–3 personas relevant to this route. Walk through the primary action; list specific failures by element name.

- **Operator (returning power user, keyboard-first):** failures observed…
- **First-timer (onboarding, no mental model yet):** failures observed…
- **Agent (yes — agents read this UI via screenshots / DOM scrapes):** does the DOM expose stable selectors, semantic roles, and predictable text? Anything that breaks programmatic reading?

---

## 14. Cross-Module Consistency Notes

Anything on this route that diverges from siblings (e.g. button placement, table density, error toast position, header pattern). Cross-reference the sibling: `<module>/<route>` — `<difference>`.

---

## 15. Open Questions

Provocative questions the design lead should answer before fixing:
- What if the primary action were inline rather than in a header?
- Is this route doing two jobs (list + edit) and would splitting clarify it?
- …

---

## 16. Recommended Action Plan

Ordered list of `/impeccable` commands the implementing agent should run, scoped to this route. Each item carries enough context for the command to do its job.

1. `/impeccable <command> <target>` — `<what to fix, with specifics>`
2. `/impeccable <command> <target>` — …
3. …
4. End with `/impeccable polish <target>` if any fixes were recommended.

---

## 17. Sign-off Checklist

- [ ] Every claim cites a `file:line` or screenshot in `_evidence/<route>/`.
- [ ] No section is left as `<TODO>` or empty.
- [ ] Nielsen scores total is consistent with the band claimed.
- [ ] Findings are tagged P0–P3 with effort and command.
- [ ] No hallucinated routes, components, or props (everything cross-referenced to source or storybook).
- [ ] No em dashes in this report.
- [ ] Report length: aim for thorough but not padded — the value is in evidence + concreteness, not word count.
