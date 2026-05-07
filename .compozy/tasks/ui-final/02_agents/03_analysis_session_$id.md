# UI/UX Analysis - `Agents` :: `/session/$id`

> **Status:** draft
> **Owner subagent:** `ui-final/02_agents`
> **Date:** 2026-05-06
> **Module:** Agents (`02_agents`)
> **Route path:** `/session/$id` (TanStack Router id: `_app/session/$id`)
> **Web source:** `web/src/routes/_app/session.$id.tsx`
> **System owner:** `web/src/systems/session/`
> **Storybook story id(s):** **none** (no `routes-app-stories-session-id--*` story file exists). The route is a redirect; only the in-route loading and not-found branches are reachable.
> **Live URLs probed:** `http://localhost:3000/session/sess-fake-id` (forces the not-found path)

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/session.$id.tsx` (the entire route - 56 lines)
  - `web/src/systems/session/hooks/use-sessions.ts` (verified `useSession` returns `{ data, isLoading, error }`)

- **Storybook stories opened:**
  - **Gap:** there is no `-session.$id.stories.tsx` under `web/src/routes/_app/stories/`. Confirmed by `ls web/src/routes/_app/stories/` (no `session.id` entry, only `agents.$name.sessions.$id` and others). **P2 finding** in §11.

- **Live web probes (`localhost:3000`):**
  - `/session/sess-fake-id` -> initial loading spinner -> settled to "Session not found: sess-fake-id" red AlertCircle line.
  - Real session permalink redirect path is unverifiable today because the daemon has no sessions; the route's redirect logic depends on `session?.agent_name` being set, which the storybook fixtures provide but no live data exercises.

- **Screenshots / DOM snapshots captured:**
  - `_evidence/session.id/live_1440_notfound.png` - mid-load spinner state at 1440px.
  - `_evidence/session.id/live_1440_settled.png` - settled not-found state.
  - `_evidence/session.id/dom_live_notfound.txt` - DOM during loading (only sidebar visible).
  - `_evidence/session.id/dom_live_settled.txt` - DOM after settle (red icon + sentence in `<main>`).

- **Console / network errors observed:**
  - `/api/sessions/sess-fake-id` returned 404 envelope; route consumed it correctly.
  - No client-side errors.

- **Keyboard / a11y probes performed:**
  - DOM inspection: in the not-found state the only content in `<main>` is the AlertCircle SVG (decorative) and a paragraph. **No interactive elements.** The user cannot recover with keyboard.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** A **permalink-by-id redirect**. External surfaces (automation history, task tree, CLI output) hold a session id without knowing the originating agent. This route resolves the agent name from the session payload and `replace`-redirects to the canonical `/agents/$name/sessions/$id`. It owns only loading and not-found branches.
- **Primary user goal on this route:** Be redirected to the canonical session URL within 1-2 seconds. The route is a transient.
- **Entry vectors:** automation run history (`web/src/systems/automation/components/automation-run-history.tsx` references), task tree drill-in, CLI permalinks (`agh session view <id>` may emit a URL), shared links pasted by an operator.
- **Exit vectors:** auto-redirect to `/agents/$name/sessions/$id` on success; nothing on failure.
- **Critical states this route MUST handle:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| Loading (in-flight) | yes | `session.$id.tsx:32-41` centered `Loader2` with `data-testid="session-permalink-loading"` | adequate (matches sibling loading pattern) |
| Resolved + redirecting | yes | `session.$id.tsx:22-30` `useEffect` triggers `navigate({...replace:true})` once `session?.agent_name` is present | strong (replace prevents back-button trap) |
| Not found (404) | yes | `session.$id.tsx:43-55` `AlertCircle` icon + `error?.message ?? "Session not found"` line | **weak** (no Empty chrome, no recovery action) |
| Error (network) | partial | falls into the not-found branch with the error message | weak (network errors and 404 render the same UI) |
| Permission denied (403) | unverified | would surface as the `error.message` text | weak |
| Session resolved but missing `agent_name` | edge case | `session.$id.tsx:32` condition `(session && session.agent_name)` keeps showing the loader; if `agent_name` is null the user is stuck on the spinner forever | **bug** |
| Mobile / narrow viewport | yes | layout is centered flexbox, scales to any width | adequate |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   2   | spinner during fetch; nothing during the redirect handoff | A user landing on a valid permalink sees the spinner then a flash of the canonical route - no explicit "redirecting…" cue. |
| 2  | Match between system and real world    |   2   | `Session not found: <id>` matches backend nouns | "Session not found: sess-fake-id" is technical; first-time users will not recognize what `sess-...` is. |
| 3  | User control and freedom               |   1   | no Go home / Open agents button on not-found | User cannot recover without using the sidebar. |
| 4  | Consistency and standards              |   2   | uses `Loader2` + `AlertCircle` icons consistent with sibling routes | But the not-found state ignores `Empty` (`@agh/ui`) which `/agents/$name` uses for the same purpose. Two not-found patterns in the same module. |
| 5  | Error prevention                       |   3   | `replace: true` on navigate prevents back-button loop into the dead permalink | Strong choice. |
| 6  | Recognition rather than recall         |   2   | `AlertCircle` is recognized as error | The message has no surrounding card or section header - reads as orphaned text. |
| 7  | Flexibility and efficiency of use      |   1   | no shortcuts | n/a for a transient route |
| 8  | Aesthetic and minimalist design        |   2   | flat, on-brand spinner | The not-found state has no rhythm: icon + 14px text floating in a blank canvas with no anchor. |
| 9  | Help users recognize / recover errors  |   1   | error message is the entire UI | No retry, no Go home, no "the session might have been deleted" hint. |
| 10 | Help and documentation                 |   1   | no help anchor | Contextless. |
|    | **Total**                              | **17/40** | | **Band:** ✗ poor (<20) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders | OK | none. |
| Gradient text | OK | none. |
| Glassmorphism / blur as default | OK | none. |
| Hero-metric template | OK | none. |
| Identical card grids | OK | none. |
| Modal as first thought | OK | none. |
| Em dashes in copy | OK | none. |
| Generic AI palette | OK | none observed; the AlertCircle uses `--color-danger`. |
| Category-reflex theme | OK | n/a. |
| Restated headings | OK | n/a. |
| Decorative shadows | OK | none. |
| Hardcoded `#000` / `#fff` | OK | none. |

**Summary verdict:** No, a stranger would not say "AI made this" - because there is barely any UI to evaluate. The route renders a spinner or a one-line error.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point:** 0 on success (redirect happens). 0 on failure (no recovery action). **Critical fail.**
- **Eight-item cognitive load checklist:**
  1. Are >4 options visible at once? **Pass** - 0 options; transient route.
  2. Are labels self-evident without docs? **Fail** - the error displays a raw `sess-...` id.
  3. Is the primary action visually dominant? **Fail** - no primary action exists in the not-found branch.
  4. Is information progressively disclosed? **n/a** - nothing to disclose.
  5. Do related elements group via proximity? **n/a** - nothing groups.
  6. Is hierarchy clear via scale/weight contrast? **n/a** - one icon + one paragraph.
  7. Is body line length within 65-75ch? **Pass** - one line.
  8. Is whitespace varied (rhythm)? **Fail** - the content sits in the dead-center of the canvas with no anchor.

  Failure count: 3 -> moderate. (For a screen this small, even 3 fails is a strong signal.)

- **IA observations:** the not-found state is **structurally inconsistent** with the sibling `/agents/$name` not-found state, which uses the standard `Empty` component (icon + title + description + action). On the same module, two routes render the same conceptual state (resource not found) with two different shells.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** uses `var(--color-text-tertiary)` for the spinner and the not-found paragraph; uses `var(--color-danger)` for the AlertCircle (`session.$id.tsx:38, 47, 51`). Compliant.
- **Type scale:** Inter, default body weight, 13px small body. No bolds, no serif. Compliant.
- **Radii / spacing:** none (flexbox centering only).
- **Elevation:** none.
- **Signal palette discipline:** danger is correctly applied to the not-found icon. Compliant.
- **Grid / rhythm:** the entire route is `flex flex-1 items-center justify-center` - centered single block, zero rhythm. Acceptable for a transient surface but undersized for the not-found terminal state.
- **Density:** ultra-sparse. The not-found state occupies ~3% of the available canvas.

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** none.
- **Destructive actions:** none.
- **Forms:** none.
- **Tables / lists:** none.
- **Selection model:** n/a.
- **Modals / drawers:** none.
- **Live updates / streaming:** none.
- **Optimistic vs pessimistic updates:** the `replace: true` redirect on success is essentially synchronous once the session payload arrives; pessimistic against the API.
- **Hover / focus / active states:** no interactive elements, so no states.
- **Loading patterns:** centered `Loader2` with `data-testid="session-permalink-loading"`. Matches the agent-detail loading pattern (also a centered Loader2). At least it is consistent within the module's loading branches (and equally weak in both).

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** **fail** - the not-found state has zero interactive elements. A keyboard-only user cannot recover from a stale permalink.
- **Focus rings:** n/a.
- **TAB order:** n/a (no tab targets in `<main>`).
- **ARIA roles / labels:** the AlertCircle has no aria-label and is not wrapped in a `role="alert"` region. A screen reader will read the paragraph but the error context is implicit.
- **Color contrast:** danger `#FF453A` on canvas `#141312` ~5.8:1 - pass for icon. Tertiary text `#636366` for the paragraph - ~4.7:1 - borderline.
- **Motion:** Loader2 spin via `animate-spin`. Reduced-motion is handled globally per DESIGN.md.
- **Text scaling:** survives 200% zoom (one line, one icon).
- **Forms:** none.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** **n/a** - this route is never empty in the user-facing sense; either it redirects or it errors.
- **Loading:** **adequate** - centered Loader2 is consistent with sibling routes.
- **Error:** **weak**. AlertCircle + bare sentence with no action. No retry, no Go home, no "the session might have been deleted by the operator or expired" hint. Missing recovery path violates Nielsen heuristic 9.
- **Permission denied:** **missing as a distinct state**. Would render as the same not-found surface.
- **Stale / disconnected:** **n/a**.

### Edge case: `agent_name` is missing on a resolved session

- **Bug:** `session.$id.tsx:32` reads `if (isLoading || (session && session.agent_name))` which keeps the spinner mounted whenever `session` resolves but `agent_name` is null. The route never reaches the not-found branch in that case. Practically `agent_name` should always be set if the session record exists, but the route trusts the field and offers no escape if the contract is violated. Possible **infinite spinner** for any future session payload missing `agent_name`. **P2 finding.**

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** uses `session` (canonical). Compliant.
- **Tone:** the error sentence "Session not found: <id>" is operator-direct but bare. `COPY.md` formula for empty/error: `<What failed>. <Why, if known>. <Next safe action>.` This route only delivers `<What failed>`.
- **Em dashes:** none in the route copy. Compliant.
- **Restated headings:** none.
- **Sentence case vs Title Case:** the error message is sentence case. Compliant.
- **Truthful UI test:** the route does not advertise any control or capability beyond the error message. No drift.

---

## 10. Performance & Responsiveness

- **Initial render:** 56-line route, lazily code-split by TanStack Router.
- **Re-render hot spots:** `useEffect` on `[session?.agent_name, id, navigate]` only fires when `agent_name` becomes available.
- **List virtualization:** n/a.
- **Bundle red flags:** none - this route imports only `lucide-react`, `@tanstack/react-router`, and the session adapter.
- **Responsive behaviour:** flexbox-centered, scales to any width. Acceptable.
- **Mobile interactions:** no hover-only affordances (no affordances at all).

---

## 11. Storybook Coverage

- **Stories present:** **none.** There is no `-session.$id.stories.tsx`.
- **States covered in Storybook:** none.
- **Gaps:**
  - **No story for the loading branch** (`useSession` in flight).
  - **No story for the redirect resolution** (would need to assert the navigate call).
  - **No story for the not-found branch** (the live route is the only way to see it).
  - **No story for the missing-`agent_name` edge case** (the spinner-forever bug).
- **Story drift:** n/a (no story exists).

**Action:** create `-session.$id.stories.tsx` with at least Loading and NotFound stories. Optional: a `RedirectResolved` story that mocks the session payload and asserts the navigation effect ran (using a route-canvas decorator).

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

(none for this route - it is a transient by design and the bugs are P1)

### P1 - High-Value Polish

1. **[P1] Not-found state has no recovery action.**
   - **Why:** users landing on a stale permalink have no way to recover except using the sidebar. Violates Nielsen heuristic 9 and breaks consistency with `/agents/$name`'s not-found Empty.
   - **Fix:** swap the bare AlertCircle + paragraph for the standard `Empty` component, with `icon={AlertCircle}`, `title="Session not found"`, `description="The session may have been deleted, or the link is from a different workspace."`, and an `action={<button onClick={goHome}>Go home</button>}` plus a secondary button to `/agents`.
   - **Cmd:** `/impeccable harden session-permalink-not-found`
   - **Effort:** S
   - **Evidence:** `web/src/routes/_app/session.$id.tsx:43-55`; `_evidence/session.id/live_1440_settled.png`

2. **[P1] Not-found state shows a raw `sess-...` id.**
   - **Why:** first-time users do not recognize the `sess-` prefix; the surface is unfriendly.
   - **Fix:** keep the id as a mono pill below the title (so it can be copied for support), but the headline reads "This session no longer exists" or "Session not available".
   - **Cmd:** `/impeccable clarify session-permalink-not-found`
   - **Effort:** XS
   - **Evidence:** `_evidence/session.id/live_1440_settled.png`

3. **[P1] Route module has no Storybook coverage.**
   - **Why:** without stories the loading + not-found branches have no enforced visual contract; refactors silently regress.
   - **Fix:** add `-session.$id.stories.tsx` with `Loading`, `NotFound`, and (optionally) `RedirectResolved` stories.
   - **Cmd:** `/impeccable craft session-permalink-stories`
   - **Effort:** S
   - **Evidence:** `ls web/src/routes/_app/stories/` (no session-permalink entry)

### P2 - Worthwhile

4. **[P2] Resolved-but-missing-`agent_name` keeps the spinner forever.**
   - **Why:** the route trusts a contract field and has no escape if a future payload omits it. Defensive programming is OK here because the route's only job is redirection.
   - **Fix:** if `session && !session.agent_name`, fall through to a "session is malformed" error state (or an "open agents index" fallback).
   - **Cmd:** `/impeccable harden session-permalink-edge-cases`
   - **Effort:** XS
   - **Evidence:** `web/src/routes/_app/session.$id.tsx:32-41`

5. **[P2] Loading branch is a single Loader2 with no "Resolving session…" cue.**
   - **Why:** transient routes benefit from naming what is happening. A user who expects to see a chat will see a spinner and may think the page hangs.
   - **Fix:** add a small mono caption below the spinner: `Resolving session…` (mono 10px tracking 0.06em uppercase, `--color-text-tertiary`).
   - **Cmd:** `/impeccable clarify session-permalink-loading`
   - **Effort:** XS
   - **Evidence:** `web/src/routes/_app/session.$id.tsx:32-41`

### P3 - Parking Lot

6. **[P3] Not-found AlertCircle has no `role="alert"` / `aria-live`.**
   - **Cmd:** `/impeccable polish session-permalink-a11y`
   - **Effort:** XS
   - **Evidence:** `web/src/routes/_app/session.$id.tsx:43-55`

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):** lands on stale permalink -> spinner -> red text. No keyboard recovery. Will hit `Cmd+L` to retype the URL or click the sidebar. Survivable but unpleasant.
- **First-timer (onboarding, no mental model yet):** sees `sess-fake-id` and red text. Has no anchor to "what is a session". Likely closes the tab.
- **Agent (yes - agents read this UI via screenshots / DOM scrapes):** the DOM exposes `data-testid="session-permalink-loading"` and `data-testid="session-permalink-not-found"`. Stable. Score: **good** for agent reads, weak for human reads.

---

## 14. Cross-Module Consistency Notes

- **Inconsistent with `/agents/$name` not-found state.** The agent route uses the `Empty` shell (icon + title + description + action) for its 404; this route uses a bare flexbox-centered AlertCircle + paragraph. Same module, two patterns. **Address at the module level: standardize on `Empty` for all not-found surfaces.**
- The loading pattern (centered Loader2) matches `/agents/$name` and `/agents/$name/sessions/$id` - consistent (and consistently weak).

---

## 15. Open Questions

- Should this route exist at all, or could the agent name be encoded in the permalink (e.g., `/sessions/$workspaceSlug/$id`)? Removing the resolution step would eliminate the redirect flash entirely.
- If a session is deleted but its id is referenced by automation history, should the daemon respond with the original agent name and the not-found state route to `/agents/$name` with a "this session was deleted" toast? That keeps deep-links useful.
- Is the redirect intentionally `replace: true` to prevent the back-button trap? If so, document it. If not, evaluate whether the back button should land on the previous page or the agent route.

---

## 16. Recommended Action Plan

1. `/impeccable harden session-permalink-not-found` - swap the bare AlertCircle + paragraph for the `Empty` shell with title + description + Go home / Open agents actions.
2. `/impeccable clarify session-permalink-not-found` - rewrite the message to be operator-friendly; keep the raw id as a mono pill underneath.
3. `/impeccable craft session-permalink-stories` - add Loading and NotFound storybook coverage at the canonical `routes-app-stories-session-id` namespace.
4. `/impeccable harden session-permalink-edge-cases` - guard the missing-`agent_name` infinite spinner.
5. `/impeccable clarify session-permalink-loading` - add "Resolving session…" caption below the spinner.
6. `/impeccable polish session-permalink-a11y` - add aria-live on the error region.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/session.id/`.
- [x] No section is left as `<TODO>` or empty.
- [x] Nielsen scores total (17/40) is consistent with the band claimed (✗ poor).
- [x] Findings are tagged P0-P3 with effort and command.
- [x] No hallucinated routes, components, or props (every claim cross-referenced to source).
- [x] No em dashes in this report.
- [x] Report length is thorough but not padded.
