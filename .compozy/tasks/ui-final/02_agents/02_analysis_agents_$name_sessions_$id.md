# UI/UX Analysis - `Agents` :: `/agents/$name/sessions/$id`

> **Status:** draft
> **Owner subagent:** `ui-final/02_agents`
> **Date:** 2026-05-06
> **Module:** Agents (`02_agents`)
> **Route path:** `/agents/$name/sessions/$id` (TanStack Router id: `_app/agents/$name/sessions/$id`)
> **Web source:** `web/src/routes/_app/agents.$name.sessions.$id.tsx`
> **System owner:** `web/src/systems/session/`, `web/src/components/assistant-ui/`
> **Storybook story id(s):** `routes-app-stories-agents-name-sessions-id--default`, `routes-app-stories-agents-name-sessions-id--loading`, `routes-app-stories-agents-name-sessions-id--stopped`, `routes-app-stories-agents-name-sessions-id--pending-permission`, `routes-app-stories-agents-name-sessions-id--not-found-redirect`
> **Live URLs probed:** `http://localhost:3000/agents/general/sessions/<empty>` (no real sessions in daemon - storybook is the populated source); Storybook iframe `http://localhost:6006/iframe.html?id=routes-app-stories-agents-name-sessions-id--default&viewMode=story`

---

## 0. Inputs & Probes (mandatory evidence)

- **Source files read:**
  - `web/src/routes/_app/agents.$name.sessions.$id.tsx`
  - `web/src/systems/session/components/chat-header.tsx`
  - `web/src/systems/session/components/session-inspector.tsx`
  - `web/src/systems/session/components/session-resume-failure.tsx`
  - `web/src/systems/session/components/runtime-activity-notice.tsx`
  - `web/src/systems/session/components/permission-prompt.tsx`
  - `web/src/systems/session/components/tool-call-card.tsx`
  - `web/src/systems/session/components/message-markdown.tsx`
  - `web/src/systems/session/components/thinking-block.tsx`
  - `web/src/components/assistant-ui/session-thread.tsx`
  - `web/src/systems/session/index.ts` (drawer export confirmation)
  - `web/src/routes/_app/stories/-agents.$name.sessions.$id.stories.tsx`

- **Storybook stories opened:**
  - `routes-app-stories-agents-name-sessions-id--default` -> `_evidence/agents.name.sessions.id/sb_default.png`
  - `routes-app-stories-agents-name-sessions-id--loading` -> `_evidence/agents.name.sessions.id/sb_loading.png`
  - `routes-app-stories-agents-name-sessions-id--stopped` -> `_evidence/agents.name.sessions.id/sb_stopped.png` and `sb_stopped_full.png`
  - `routes-app-stories-agents-name-sessions-id--pending-permission` -> `_evidence/agents.name.sessions.id/sb_pending-permission.png`
  - `routes-app-stories-agents-name-sessions-id--not-found-redirect` -> `_evidence/agents.name.sessions.id/sb_not-found-redirect.png`

- **Live web probes (`localhost:3000`):**
  - The daemon has no real sessions, so the live SPA renders the empty-state agent detail at `/agents/general` and never hosts a populated chat. Live populated states are read from Storybook per `_README.md` rule 3.

- **Screenshots captured at multiple widths:**
  - `_evidence/agents.name.sessions.id/sb_default_320.png` - 320px viewport
  - `_evidence/agents.name.sessions.id/sb_default_768.png` - 768px
  - `_evidence/agents.name.sessions.id/sb_default_1024.png` - 1024px
  - `_evidence/agents.name.sessions.id/sb_default.png` - 1440px

- **Console / network errors observed:** none in storybook iframe.

- **Keyboard / a11y probes performed:**
  - `agent-browser snapshot -i` of stopped story enumerated: `Delete session`, `Resume session`, `Thought process`, `Copy code`, `Launch brief` (link), `Ran command…` (button, expanded=false), `Read file…` (button, expanded=false), `Session prompt` (textbox, disabled), `Clear conversation`, `Send message` (button, disabled), 5 inspector tabs.
  - All actions reachable by keyboard.

---

## 1. Route Purpose & User Journey

- **What this route does in product terms:** This is the chat surface for a single durable session belonging to a named agent. It hosts the live transcript (assistant-ui-driven), exposes session controls (delete, stop, resume) in the header, surfaces a forensic inspector (Trace / Usage / Memory / Files / Vault) on the right rail, and shows alerts / permission prompts / activity notices inline.
- **Primary user goal on this route:** Send a prompt and read the agent's response, with full ability to inspect tool calls and the underlying ledger.
- **Entry vectors:** clicking a session row on `/agents/$name`; deep link from `/session/$id` (auto-redirect); CLI permalink shared into a chat tool.
- **Exit vectors:** Delete (back to `/agents/$name`); Stop (in-place state change); sidebar nav.
- **Critical states this route MUST handle:**

| State | Implemented? | Evidence | Quality |
|---|---|---|---|
| First-run / empty (session has no transcript) | yes | `session-thread.tsx:258-272` `ThreadEmpty` with agent eyebrow + "Start a conversation. The assistant thread replays persisted history and continues live over the daemon stream." | weak (mentions implementation: "daemon stream") |
| Loading / skeleton | yes (poor) | `agents.$name.sessions.$id.tsx:127-133` centered `Loader2` only - no skeleton chrome (`_evidence/agents.name.sessions.id/sb_loading.png`) | weak |
| Streaming / running | yes | `SessionMessageEmpty` shows `Loader2 + "Thinking…"`; `RuntimeActivityNotice` for runtime_progress events | adequate |
| Tool-call rendering | yes | `ToolCallCard` with status pill + expand/collapse + `ExpandedToolContent` (`tool-call-card.tsx:1-122`) | strong |
| Permission prompt | yes | `PermissionPrompt` (`permission-prompt.tsx:38-124`) | drift (uses `amber-500` literal palette, not the design-token warning) |
| Stopped / read-only | yes | `chat-header.tsx:54-191` swaps Stop for Resume; composer disables prompt | weak (disabled send button still renders accent fill) |
| Resume failure | yes | `SessionResumeFailure` banner with retry / dismiss + missing-provider detail (`session-resume-failure.tsx:32-119`) | strong |
| Forensic ledger lineage (post-stop) | yes | `useSessionLedger` (`agents.$name.sessions.$id.tsx:50`); `SessionLedgerMetaPanel` + `SessionLedgerEventsPanel` | strong |
| Error (network) | partial | `useSession` errors trigger toast + redirect on "not found"; other errors fall through | weak |
| Error (permission / 403) | unverified | no dedicated path; would surface as toast | weak |
| Mobile / narrow viewport | partial | breadcrumb truncates aggressively at 768px; inspector hidden below 1280px with no drawer fallback | **broken** (`_evidence/agents.name.sessions.id/sb_default_320.png`, `sb_default_768.png`, `sb_default_1024.png`) |

---

## 2. Design Health Score (Nielsen 10)

| #  | Heuristic                              | Score | Evidence (file:line or screenshot) | Key issue |
|----|----------------------------------------|:-----:|------------------------------------|-----------|
| 1  | Visibility of system status            |   3   | `Pill.Dot` status + breadcrumb, `RuntimeActivityNotice`, `ThinkingBlock`, Inspector trace tab | No SSE-disconnect banner; if the streaming connection drops the user sees nothing. |
| 2  | Match between system and real world    |   2   | "Trace" / "Memory" / "Vault" / "Files" tabs use runtime nouns; thread empty mentions "daemon stream" | Implementation leak in copy. "claude" provider chip is lowercase mono in normal-case override - inconsistent with other mono pills. |
| 3  | User control and freedom               |   3   | Delete / Stop / Resume in header, Clear conversation in composer, Inspector tabs persist user choice | No undo for clear-conversation (modal warns but no recovery); no Esc-to-stop streaming (only the inline Stop button). |
| 4  | Consistency and standards              |   2   | uses `Pill`, `Pill.Dot`, `Empty`, `Tabs` from `@agh/ui` | `permission-prompt.tsx` uses raw `amber-500` instead of `--color-warning`; `message-markdown.tsx` uses `oneDark` Prism theme, breaking the warm-only palette. |
| 5  | Error prevention                       |   3   | Delete + Clear conversation both have confirm dialogs | Confirm dialogs lack a typed-name guard for the destructive irreversible delete. |
| 6  | Recognition rather than recall         |   2   | Inspector tabs persistent, header chips give context | Assistant message has no agent-name eyebrow; user message has no YOU label - DESIGN.md spec ignored. |
| 7  | Flexibility and efficiency of use      |   2   | Enter to submit, Shift+Enter for newline (assistant-ui default) | No `/` to focus composer, no `Esc` to abort stream, no keyboard shortcut for Resume / Stop. |
| 8  | Aesthetic and minimalist design        |   2   | flat depth, warm grays | the `oneDark` code-block theme drops the warm palette; the chat-header packs 7+ chrome elements into a 48px row that fragments into ellipses below 1024px. |
| 9  | Help users recognize / recover errors  |   3   | `SessionResumeFailure` is the strongest error UI in the SPA (named cause + retry + dismiss + meta) | Other errors fall back to `toast.error` without a route-level banner. |
| 10 | Help and documentation                 |   1   | no inline help, no "what is the Inspector?" hint, no doc link | The Inspector is a dense surface and first-timers have no anchor. |
|    | **Total**                              | **23/40** | | **Band:** ◯ adequate (20-28) |

---

## 3. AI-Slop & Anti-Pattern Verdict

| Anti-pattern                                    | Verdict | Evidence |
|-------------------------------------------------|:------:|----------|
| Side-stripe borders | OK | none decoratively used. |
| Gradient text | OK | none observed. |
| Glassmorphism / blur as default | borderline | chat header uses `bg-[color:var(--color-surface-panel)]/90 backdrop-blur` (`chat-header.tsx:70`); DESIGN.md allows blur only on the sticky site header, but a chat header is a reasonable secondary case. Document the exception or drop the blur. |
| Hero-metric template | OK | the Inspector Usage tab renders Metric cards but as a 2-col mini-grid, not a hero. |
| Identical card grids | OK | tool-call cards differ from inspector cards. |
| Modal as first thought | borderline | Delete and Clear conversation both go through `Dialog` confirms. Inline confirm-button-with-typing-guard would be more recoverable. |
| Em dashes in copy | n/a | em dashes appear only as data placeholders. |
| Generic AI palette | borderline | the chat code blocks render in `oneDark` (cool blues / purples), introducing a non-warm palette into an otherwise warm-only system. |
| Category-reflex theme | OK | not a generic chat UI; restrained warm orange. |
| Restated headings / intros | OK | none. |
| Decorative shadows | borderline | `backdrop-blur` on header is the only depth artifact. |
| Hardcoded `#000` / `#fff` | OK | assistant-ui composer uses `text-white` on send button (`session-thread.tsx:198`); white on accent is permitted by DESIGN.md (only place pure white is allowed). |

**Summary verdict:** No, a stranger would not say "AI made this", but the chat code blocks and the amber permission prompt **break the warm-only register** and a sharper eye would notice. Borderline on AI slop because of the palette deviation, not the structure.

---

## 4. Cognitive Load & Information Architecture

- **Visible options at the primary decision point** (composer): 3 (Send, Stop while running, Clear conversation). Plus 5 Inspector tabs (Trace/Usage), 3 (Memory/Files/Vault). The composer surface is clean.
- **Eight-item cognitive load checklist:**
  1. Are >4 options visible at once? **Borderline** - the chat header alone exposes 7 chrome elements (status dot, agent name, chevron, session name, chevron, provider pill, chevron, workspace pill, activity inline) plus 2 action buttons.
  2. Are labels self-evident without docs? **Fail** - "Trace", "Vault", "Memory", "Files" are tab labels with no description; "Forensic" eyebrow above the lineage panel is unexplained.
  3. Is the primary action visually dominant? **Pass** - round accent send button is the most visually weighty element.
  4. Is information progressively disclosed? **Pass** - thinking block is collapsible, tool-call cards are collapsible, ledger events list is capped at 20.
  5. Do related elements group via proximity? **Pass** - top inspector group is Trace/Usage; bottom is Memory/Files/Vault.
  6. Is hierarchy clear via scale/weight contrast? **Pass** - large user bubble vs small mono eyebrows.
  7. Is body line length within 65-75ch? **Pass** - thread body is `max-w-3xl` (~768px / ~75ch).
  8. Is whitespace varied (rhythm)? **Pass** - 12px between messages, 16-24px around chunks.

  Failure count: 2 -> moderate.

- **IA observations:**
  - The Inspector right rail is a **two-row tablist** (Trace/Usage on top, Memory/Files/Vault on bottom) sharing a 320px column. This is a clever density move but doubles the cognitive surface area; first-timers will not realize the bottom row exists until they scroll.
  - The header workspace pill (`risk-ops`) is redundant with the sidebar workspace selector at the very top-left. Consider dropping the workspace pill from the header.
  - The activity inline (`SessionActivityInline`) renders inside the breadcrumb on `md+` and is hidden below. On wide screens the breadcrumb already has 7 elements; adding an 8th makes the row feel rope-strung.

---

## 5. Visual & Layout Audit (DESIGN.md compliance)

- **Color tokens:** Mostly via `var(--color-*)`. Drift:
  - `permission-prompt.tsx:40-44, 89` uses `border-amber-500/40 bg-amber-500/5 text-amber-500` Tailwind literals instead of `--color-warning` / `--color-warning-tint`. (`_evidence/agents.name.sessions.id/sb_pending-permission.png`)
  - `message-markdown.tsx:16, 84` imports and applies `oneDark` Prism theme to code blocks. The keywords render in cool purples / blues. Visible in the populated story (`_evidence/agents.name.sessions.id/sb_default.png` - the `const heroHeadline = ...` block).
  - `runtime-activity-notice.tsx:91-95` uses `border-[color:var(--color-warning)]/35 bg-[color:var(--color-warning-tint)]` correctly. So permission-prompt drift is local, not systemic.

- **Type scale:** Inter for messages, JetBrains Mono for chips and timestamps, no serif. Compliant.

- **Radii / spacing:** user bubble uses `rounded-xl` (12px), tool-call card uses `rounded-[var(--radius-md)]` (8px) - matches DESIGN.md sub-spec for tool-call card. Compliant.

- **Elevation:** flat. The chat-header `backdrop-blur` is the only blur on the route. Document or drop.

- **Signal palette discipline:**
  - Status dot tones map correctly (`active=success`, `starting=warning+pulse`, `stopping=warning`, `stopped=neutral`) (`chat-header.tsx:36-41`). Compliant.
  - Trace status -> Pill tone map (`session-inspector.tsx:125-130`) uses success/warning/danger/accent. Compliant.
  - Permission prompt's amber violates the discipline (see above).

- **Grid / rhythm:** thread is `max-w-3xl` centered; inspector is fixed 320px; chat header h-12. Layout is rigid - no fluid responsive collapse.

- **Density:** comfortable on wide; cramped at 768px (chat header). Sparse / lonely at 320px (no inspector path).

---

## 6. Interaction & Behaviour Audit

- **Primary actions:** Send button is unambiguous (round accent), Stop button replaces it during run.
- **Destructive actions:** Delete uses confirm `Dialog` (`chat-header.tsx:193-238`); Clear conversation uses confirm Dialog (`session-thread.tsx:209-253`). Neither requires typed-name confirmation. Acceptable for alpha but worth flagging for irreversible delete.
- **Forms:** the composer is a `ComposerPrimitive.Root` -> `ComposerPrimitive.Input` -> `ComposerPrimitive.Send`. `submitMode="enter"` is correct for chat, Shift+Enter inserts a newline (assistant-ui default).
- **Tables / lists:** Inspector trace list is `<ol>` with rows; no virtualization needed for 6 default events; total event count is shown but the "View all" handler (`onViewAllTrace`) is **not wired by this route** - it would need an external history modal which the route does not mount. So when total > 6 the View all link still appears but does nothing meaningful unless a parent passes `onViewAllTrace`. Truthful UI risk.
- **Selection model:** none on the thread; tool-call cards are individually expand-collapse.
- **Modals / drawers:** Delete + Clear conversation use `Dialog`. The right-rail `Sheet`-based `SessionInspectorDrawer` exists in code but is **never mounted by this route** (`grep "SessionInspectorDrawer" web/src/routes` -> empty). On viewports below `xl` (<1280px) the inspector is unreachable. **P0.**
- **Live updates / streaming:** running state is reflected by `useSessionComposerState`. `RuntimeActivityNotice` shows runtime progress / warning messages mid-stream. There is no "reconnecting" / "stream stale" banner.
- **Optimistic vs pessimistic updates:** Send is pessimistic (waits for assistant response); Resume / Stop / Delete are pessimistic.
- **Hover / focus / active states:** tool-call card has explicit focus-visible ring (`tool-call-card.tsx:74-78`). Composer focuses the textarea on click but there is no auto-focus on route mount; first-time users have to click into the composer.
- **Loading patterns:** session route loading is a centered `Loader2` only - no skeleton chrome resembling the chat layout (`agents.$name.sessions.$id.tsx:127-133`). Bad first impression.

---

## 7. Accessibility (WCAG 2.2 AA targets)

- **Keyboard reachability:** all interactive elements reachable per the storybook DOM snapshot. Tab order: header buttons -> thinking-block trigger -> assistant message tool-call buttons -> composer textarea -> Clear conversation -> Send -> Inspector tabs.
- **Focus rings:** tool-call card has explicit `focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]/40` (`tool-call-card.tsx:74-77`) - on-brand. Composer textarea has `focus-visible:ring-0` because focus is owned by the parent ComposerPrimitive.Root border swap. Acceptable.
- **TAB order:** logical.
- **ARIA roles / labels:**
  - Header buttons: `aria-label="Delete session"`, `aria-label="Stop session"`, `aria-label="Resume session"` (`chat-header.tsx:147, 164, 181`). Compliant.
  - Resume failure: `aria-live="assertive"` + `role="alert"` (`session-resume-failure.tsx:34-41`). Compliant.
  - Runtime activity notice: `role={isWarning ? "alert" : "status"}` (`runtime-activity-notice.tsx:86`). Compliant.
  - Inspector tabs have `aria-label="Trace and usage"` and `aria-label="Memory, files, and vault"` (`session-inspector.tsx:411, 456`). Compliant.
  - Composer input: `aria-label="Session prompt"` (`session-thread.tsx:144`). Compliant.
  - **Drift:** the disabled send button still says `aria-label="Send message"` while disabled - acceptable; the `Stop` icon-button on the resume bar does not announce the busy state (`isResuming` toggles the icon but no `aria-busy`).

- **Color contrast:**
  - Assistant text `--color-text-primary` `#E5E5E7` on `--color-canvas` `#141312` ~12:1 - pass.
  - User bubble bg `--color-surface-panel` `#181716` with `--color-divider` `#3C3A39` border on canvas - subtle but readable at 1440px.
  - Tertiary text in tool-call summary `#636366` on surface `#1E1C1B` ~4.7:1 - borderline.
  - **Permission prompt amber-500 on amber-500/5 background** - amber-500 is `#F59E0B`; 5% bg ~ `#1A1611`. Contrast ~ 9:1 is fine **but** it is the wrong color per DESIGN.md.

- **Motion:** `Loader2` spin, `Pill.Dot pulse`, `animate-spin` on busy buttons. The DESIGN.md says reduced-motion is respected globally; this needs verification but the pattern is consistent with the rest of the SPA.

- **Text scaling:** at 200% zoom the chat header further compresses; the breadcrumb is already at risk at 100% / 768px.

- **Forms:** the create dialog (separate route action) is not on this surface. The composer is a single textarea with proper `aria-label`. OK.

---

## 8. Empty / Loading / Error States

- **Empty (first-run):** **adequate**. `ThreadEmpty` (`session-thread.tsx:258-272`) renders agent name as eyebrow + "Start a conversation. The assistant thread replays persisted history and continues live over the daemon stream." - operator-first but mentions implementation ("daemon stream") in violation of `COPY.md` voice.
- **Loading:** **weak**. Centered `Loader2` only (`_evidence/agents.name.sessions.id/sb_loading.png`). No header skeleton, no composer skeleton, no Inspector skeleton.
- **Error (session not found):** the route `useEffect` toasts and redirects to `/agents/$name`; user sees a flash of the loading spinner then the agent route. Acceptable but the toast `Session not found` is a low-information error.
- **Error (other API errors):** falls through; no dedicated banner.
- **Permission denied:** unverified.
- **Stale / disconnected:** **missing**. SSE drop has no banner. The composer continues to accept input.

---

## 9. Microcopy & UX Writing (COPY.md compliance)

- **Glossary terms:** uses `session`, `agent`, `transcript`, `tool call`, `permission`, `vault`, `ledger`, `provider`, `workspace`. All canonical. No `recipe` / `workflow` / `procedure` / `playbook`.
- **Tone:**
  - "Start a conversation. The assistant thread replays persisted history and continues live over the daemon stream." - **mentions implementation** ("daemon stream"). Per `COPY.md` section 8 "do not paraphrase implementation internals before the value is clear". Should be e.g. "Send a message to start. Past messages will load when this session resumes."
  - "Permission Required" inside the permission prompt is sentence-case with caps - consistent with chip labels.
  - "Loading session ledger…" uses `…` (ellipsis char) - acceptable, follows mono eyebrow conventions in DESIGN.md.
  - "The forensic ledger materializes once the session stops. Lineage and ledger event metadata appear here after that." - operator-first, mechanism-first, explains the constraint. **Strong** copy.
  - "The session ended without recorded events; nothing was journaled for this run." - dry, operator-first. Strong.
- **Em dashes:** "Trace rows appear as the agent sends prompts, runs tools, and receives responses." - no em dash. Compliant.
- **Restated headings:** none.
- **Sentence case vs Title Case:** `Permission Required` (Title Case) inside `permission-prompt.tsx:46` - inconsistent with other DesignSystem chip labels (DESIGN.md says sentence case). The other dialog titles (`Delete session`, `Clear conversation`) use sentence case correctly.
- **Truthful UI test:**
  - Trace tab shows runtime-truthful events.
  - Usage tab is **scaffolded**: the route does not pass `usage` to `SessionInspector`; the panel renders `Empty` "No usage yet" but the tab still appears in the tablist. Borderline truthful UI - the tab implies a metric that the route never delivers.
  - Memory tab requires `state === "stopped"` to fetch the ledger (`agents.$name.sessions.$id.tsx:50`). Until then the panel renders the "No session ledger yet" Empty - **truthful**.
  - Vault tab fetches secrets via `useSessionVaultSecrets(sessionId)` - truthful.

---

## 10. Performance & Responsiveness

- **Initial render:** TanStack Router code-split per route. Heavy deps (`react-syntax-highlighter`, `assistant-ui`) load with this route. Should validate via Lighthouse but no obvious smell.
- **Re-render hot spots:** `MessageMarkdown` is `memo`'d with custom comparator on `content`. `ToolCallCard` is `memo`'d with comparator on tool input/result/error. `ThinkingBlock` is `memo`'d. Good.
- **List virtualization:** thread relies on assistant-ui's `ThreadPrimitive.Messages` - depends on the underlying runtime to virtualize; with 200+ messages this needs validation.
- **Bundle red flags:** `react-syntax-highlighter` with PrismLight + 11 languages is heavy. Acceptable for a chat surface but a PrismAsync registration audit is worth doing.
- **Responsive behaviour:**
  - 1440px: complete - sidebar + chat + 320px Inspector.
  - 1280px+: Inspector visible.
  - 1024px: **Inspector hidden, no drawer trigger**. Trace/Usage/Memory/Files/Vault all unreachable.
  - 768px: same Inspector loss; chat header truncates `fraud-o…` and `Reserve sp…`. (`_evidence/agents.name.sessions.id/sb_default_768.png`)
  - 320px: chat header reduced to status dot + chevron + claude pill + 2 buttons; agent and session names completely clipped; Inspector unreachable. (`_evidence/agents.name.sessions.id/sb_default_320.png`)
- **Mobile interactions:** the code-block CopyButton uses hover-only reveal (`message-markdown.tsx:101-104` `opacity-0 group-hover/codeblock:opacity-100`) - touch-only users cannot reveal it because hover does not trigger.

---

## 11. Storybook Coverage

- **Stories present:**
  - `routes-app-stories-agents-name-sessions-id--default` (`-agents.$name.sessions.$id.stories.tsx:30-33`)
  - `routes-app-stories-agents-name-sessions-id--loading` (`:38-51`)
  - `routes-app-stories-agents-name-sessions-id--stopped` (`:56-59`)
  - `routes-app-stories-agents-name-sessions-id--pending-permission` (`:64-76`)
  - `routes-app-stories-agents-name-sessions-id--not-found-redirect` (`:81-93`)

- **States covered:** populated, loading, stopped, pending-permission, not-found-redirect.

- **Gaps:**
  - **No streaming-in-progress story** (assistant message currently `running` with `Thinking…` indicator).
  - **No tool-call error story** (tool failure surfaces a danger pill - validate the styling).
  - **No SessionResumeFailure story at this route level** (the system has its own component story but the route never demonstrates it integrated).
  - **No mobile/tablet viewport parameter** - given the responsive breakage, this is the highest-leverage missing story.
  - **No long-conversation story (50+ messages)** to validate scroll behavior + tool-call density.
  - **No Inspector populated state** - the default story shows the Memory tab as empty even though the session is `active`. There is no story demonstrating the Memory tab populated, the Files tab populated with several reads, or the Vault tab with secrets.
  - **No code-block-heavy assistant message** to validate the `oneDark` theme drift on the populated path.

- **Story drift:** the storybook does not exercise `usage` either - confirming the live route does not wire it. Worth either removing the Usage tab or shipping an `InspectorUsage` mock in the default story.

---

## 12. Findings - Prioritised

### P0 - Ship Blockers

1. **[P0] `SessionInspectorDrawer` is exported but never mounted on this route.**
   - **Why:** below 1280px the entire inspector (Trace, Usage, Memory, Files, Vault) is unreachable. Truthful UI regression: the rail's testimony exists only for desktop users. This is a feature-bearing surface advertised everywhere in the design (DESIGN.md, RFCs) and silently absent on the default operator viewport (laptop).
   - **Fix:** mount `SessionInspectorDrawer` alongside `SessionInspector` in `agents.$name.sessions.$id.tsx`, with the trigger placed in `ChatHeader` for narrow viewports. Add `routes-app-stories-agents-name-sessions-id--mobile` story.
   - **Cmd:** `/impeccable craft session-inspector-drawer-mount`
   - **Effort:** M
   - **Evidence:** `web/src/systems/session/index.ts:115` (export), `rg "SessionInspectorDrawer" web/src/routes` (no hits), `_evidence/agents.name.sessions.id/sb_default_1024.png` (no inspector / no trigger)

2. **[P0] Chat header truncates agent and session names at 768px and clips them entirely at 320px.**
   - **Why:** users on a phone or tablet cannot read who they are chatting with. The breadcrumb has 4 chevrons, 2 names, 2 mono pills, and an activity inline competing for the same row.
   - **Fix:** at <1024px collapse to "<status dot> <session name>"; show the agent name + workspace as a secondary line; hide chevrons; tuck provider pill into the secondary line.
   - **Cmd:** `/impeccable adapt chat-header`
   - **Effort:** M
   - **Evidence:** `_evidence/agents.name.sessions.id/sb_default_768.png`, `_evidence/agents.name.sessions.id/sb_default_320.png`

3. **[P0] Permission prompt uses Tailwind `amber-500` instead of the design-token warning palette.**
   - **Why:** the highest-stakes interactive surface in the chat (allow/reject decisions for tool execution) ships in the wrong color, breaking warm-only register and creating drift from `RuntimeActivityNotice` which DOES use the proper token. This is alpha-blocking because permission decisions are the trust surface.
   - **Fix:** replace `border-amber-500/40 bg-amber-500/5 text-amber-500` with `border-[color:var(--color-warning)]/40 bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]`. Update sentence case for "Permission required".
   - **Cmd:** `/impeccable colorize permission-prompt`
   - **Effort:** S
   - **Evidence:** `web/src/systems/session/components/permission-prompt.tsx:40, 44, 89`; `_evidence/agents.name.sessions.id/sb_pending-permission.png`

### P1 - High-Value Polish

4. **[P1] Code blocks render in `oneDark` Prism theme (cool blues/purples).**
   - **Why:** the chat is the most-rendered surface in AGH; shipping it in a non-warm theme directly contradicts DESIGN.md and visually shouts "AI made this".
   - **Fix:** swap `oneDark` for a custom theme using `--color-text-primary` (default), `--color-accent` (keywords), `--color-text-secondary` (comments), `--color-text-tertiary` (punctuation). Save the theme in `packages/ui/src/syntax/agh-prism-theme.ts`.
   - **Cmd:** `/impeccable colorize message-markdown-prism`
   - **Effort:** M
   - **Evidence:** `web/src/systems/session/components/message-markdown.tsx:16, 84-94`; `_evidence/agents.name.sessions.id/sb_default.png`

5. **[P1] Disabled send button keeps accent fill at 50% opacity.**
   - **Why:** `disabled:opacity-50` on an accent fill still reads as live action; conflicts with DESIGN.md disabled spec (`bg #4A4847, text #636366`).
   - **Fix:** add `disabled:bg-[color:var(--color-disabled)] disabled:text-[color:var(--color-text-tertiary)]` to the `ComposerPrimitive.Send` className.
   - **Cmd:** `/impeccable harden composer-disabled-state`
   - **Effort:** S
   - **Evidence:** `web/src/components/assistant-ui/session-thread.tsx:194-203`; `_evidence/agents.name.sessions.id/sb_stopped_full.png`

6. **[P1] Code-block copy button is hover-only.**
   - **Why:** touch and keyboard-only users cannot discover or trigger copy.
   - **Fix:** keep the button visible at low opacity (e.g., `opacity-60`) until hover/focus brings it to full opacity.
   - **Cmd:** `/impeccable adapt copy-button-discovery`
   - **Effort:** XS
   - **Evidence:** `web/src/systems/session/components/message-markdown.tsx:99-105`

7. **[P1] User and assistant messages drop the eyebrow / metadata label.**
   - **Why:** DESIGN.md chat-component spec (lines 394-408) requires user bubble to render `YOU + timestamp` mono eyebrow above the bubble and assistant messages to render an agent-name eyebrow + 8px status dot. Currently neither is present.
   - **Fix:** add `<p>` mono eyebrow before each `MessagePrimitive.Root`, populated with `YOU` + `MessagePrimitive.IfRole` or with the agent's name + status dot. Pull timestamp from `message.createdAt`.
   - **Cmd:** `/impeccable typeset chat-message-eyebrows`
   - **Effort:** M
   - **Evidence:** `web/src/components/assistant-ui/session-thread.tsx:54-100`

8. **[P1] Loading state is a single spinner on a blank canvas.**
   - **Why:** transitioning from agent detail to chat shows a 1-2s blank flash; first impression of a slow / broken page.
   - **Fix:** render skeleton chat header + skeleton thread + skeleton composer chrome during `isLoading`, matching the populated layout.
   - **Cmd:** `/impeccable harden session-route-skeleton`
   - **Effort:** M
   - **Evidence:** `web/src/routes/_app/agents.$name.sessions.$id.tsx:127-133`; `_evidence/agents.name.sessions.id/sb_loading.png`

9. **[P1] Empty thread copy mentions implementation: "daemon stream".**
   - **Why:** `COPY.md` section 8 "Web UI: avoid implementation internals before the value is clear".
   - **Fix:** rewrite to "Send a message to start. This session resumes the agent from its last persisted state."
   - **Cmd:** `/impeccable clarify thread-empty`
   - **Effort:** XS
   - **Evidence:** `web/src/components/assistant-ui/session-thread.tsx:265-269`

10. **[P1] Usage tab is scaffolded only - the route never wires `usage`.**
    - **Why:** the tab implies a metric the runtime delivers; truthful UI test fails.
    - **Fix:** either wire a real `usage` source from the session adapter or hide the Usage tab until usage data is available.
    - **Cmd:** `/impeccable harden session-inspector-usage`
    - **Effort:** M (depends on backend availability) or S (if hidden)
    - **Evidence:** `web/src/routes/_app/agents.$name.sessions.$id.tsx` does not pass `usage`; `web/src/systems/session/components/session-inspector.tsx:519` accepts it but defaults to `undefined`.

### P2 - Worthwhile

11. **[P2] No SSE-disconnect / stream-stale banner.**
    - **Why:** if the stream drops the user types into a void.
    - **Fix:** banner over the composer when the runtime activity hasn't ticked in 30s during a `running` state.
    - **Cmd:** `/impeccable harden session-stream-disconnect`
    - **Effort:** M
    - **Evidence:** no consumer of disconnect state in `agents.$name.sessions.$id.tsx`

12. **[P2] No keyboard shortcut to focus composer / abort stream / open inspector.**
    - **Why:** power users will reach for `/`, `Esc`, `Cmd+I`. None exist.
    - **Fix:** ship `/` to focus, `Esc` to cancel during run, `Cmd+I` to toggle inspector drawer.
    - **Cmd:** `/impeccable craft session-keyboard-shortcuts`
    - **Effort:** M
    - **Evidence:** none in `agents.$name.sessions.$id.tsx`

13. **[P2] "View all" link in Inspector trace appears unconditionally when total > 6 but `onViewAllTrace` is not wired.**
    - **Why:** the link rendered with no destination is a dead end. It currently does not appear because the route does not pass `totalTraceEvents`. But if a future PR wires `messages.length > 6`, the link will dead-end.
    - **Fix:** wire `onViewAllTrace` to a Sheet that surfaces the full trace event history with virtualization, OR remove the link until a route passes a handler.
    - **Cmd:** `/impeccable craft session-trace-history-sheet`
    - **Effort:** L
    - **Evidence:** `web/src/systems/session/components/session-inspector.tsx:594-606`

14. **[P2] `Permission Required` is Title Case in a sentence-case system.**
    - **Cmd:** `/impeccable clarify permission-prompt`
    - **Effort:** XS
    - **Evidence:** `web/src/systems/session/components/permission-prompt.tsx:46`

### P3 - Parking Lot

15. **[P3] Workspace pill in chat header duplicates the sidebar workspace context.**
    - **Cmd:** `/impeccable distill chat-header-meta`
    - **Effort:** XS
    - **Evidence:** `_evidence/agents.name.sessions.id/sb_default.png`

16. **[P3] No type-to-confirm guard for the irreversible Delete session action.**
    - **Cmd:** `/impeccable harden session-delete-confirm`
    - **Effort:** S
    - **Evidence:** `web/src/systems/session/components/chat-header.tsx:193-238`

17. **[P3] `chat-header.tsx` uses `backdrop-blur` outside the documented exception list.**
    - **Cmd:** `/impeccable polish chat-header-blur`
    - **Effort:** XS
    - **Evidence:** `web/src/systems/session/components/chat-header.tsx:70`

---

## 13. Persona Red Flags

- **Operator (returning power user, keyboard-first):** no `/` to focus composer, no `Esc` to abort, no `Cmd+K` to switch sessions. Disabled send button still draws orange attention. Cannot toggle Inspector via keyboard. Will fall back to mouse and complain.
- **First-timer (onboarding, no mental model yet):** "Trace", "Usage", "Memory", "Vault", "Files" tabs are jargon labels; the empty Memory tab tells the user the lineage materializes "once the session stops" but does not explain why - the first-time user will think the session is broken. The composer placeholder "Send a message…" is fine but does not hint at multi-line / Shift+Enter.
- **Agent (yes - agents read this UI via screenshots / DOM scrapes):** strong `data-testid` coverage. ARIA roles + labels are clean (`status` / `alert` for runtime notices, `aria-live="assertive"` for resume failure, named tablists). Score: **good** for agent reads. One drift: `Pill` with `tone="accent"` and `normal-case` for the provider chip strips the mono uppercase convention - a scraper looking for `claude` may match it but a scraper expecting `CLAUDE` will not.

---

## 14. Cross-Module Consistency Notes

- The chat-header `backdrop-blur` is unique to this route. Other routes' headers (PageHeader on `/agents/$name`) do not blur. Choose one rule for the runtime surface: blur or not.
- The Inspector right-rail is a 320px column shared with `AgentInfoPanel` on the parent route. Both hide below `xl` and only the Inspector ships a `*Drawer` companion, but neither the parent route nor this route mounts it. **Module-wide P0 to mount drawers.**
- The `Empty` component is used consistently across loading/error states, but the loading branches on this route do NOT use Empty - they use bare `Loader2`. Inconsistent with sibling routes.

---

## 15. Open Questions

- Should the Inspector default-open the Trace tab or the Memory tab? Currently top is Trace; bottom is Memory. The user choice does not persist across sessions - should it?
- Does the chat surface need a "session metadata" overlay that exposes provider details, model, reasoning effort, workspace, started-at, claim_token_hash? Today these are split across the header pills, Inspector, and Resume failure detail.
- Should code blocks in messages have a "Run" affordance for `bash` blocks (mirroring the `agh` CLI's prompt)? Today they only have Copy.
- If the daemon does not yet stream usage telemetry, should we remove the Usage tab entirely or keep it as a placeholder? `COPY.md`'s truthful UI rule favors removal.

---

## 16. Recommended Action Plan

1. `/impeccable craft session-inspector-drawer-mount` - mount `SessionInspectorDrawer` on this route, with the drawer trigger in the chat header below `xl`. Add a mobile story.
2. `/impeccable adapt chat-header` - responsive collapse for breadcrumb at 768/320.
3. `/impeccable colorize permission-prompt` - replace `amber-500` literals with `--color-warning` / `--color-warning-tint`.
4. `/impeccable colorize message-markdown-prism` - replace `oneDark` with a warm-tuned Prism theme keyed off design tokens.
5. `/impeccable harden composer-disabled-state` - disabled send must use `--color-disabled` background.
6. `/impeccable adapt copy-button-discovery` - keep code-block copy button at low-opacity baseline so it survives touch/keyboard.
7. `/impeccable typeset chat-message-eyebrows` - re-introduce YOU / agent-name + status dot + timestamp eyebrows per DESIGN.md.
8. `/impeccable harden session-route-skeleton` - skeleton chat header + thread + composer for the loading branch.
9. `/impeccable clarify thread-empty` - remove "daemon stream" implementation leak.
10. `/impeccable harden session-inspector-usage` - hide the Usage tab if no `usage` source; or wire the source.
11. `/impeccable polish 02_agents-session` - final pass: focus rings, motion guards, header blur decision, type-to-confirm for Delete.

---

## 17. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot in `_evidence/agents.name.sessions.id/`.
- [x] No section is left as `<TODO>` or empty.
- [x] Nielsen scores total (23/40) is consistent with the band claimed (◯ adequate).
- [x] Findings are tagged P0-P3 with effort and command.
- [x] No hallucinated routes, components, or props (every claim cross-referenced to source or storybook story).
- [x] No em dashes in this report.
- [x] Report length is thorough but not padded.
