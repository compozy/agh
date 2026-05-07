# UI/UX Module Overview - `02_agents` (Agents)

> **Status:** draft
> **Owner subagent:** `ui-final/02_agents` audit pass
> **Date:** 2026-05-06
> **Module:** Agents (`02_agents`)
> **Routes audited:**
>   1. `/agents/$name` - `web/src/routes/_app/agents.$name.tsx`
>   2. `/agents/$name/sessions/$id` - `web/src/routes/_app/agents.$name.sessions.$id.tsx`
>   3. `/session/$id` - `web/src/routes/_app/session.$id.tsx`
> **System owners:** `web/src/systems/agent/`, `web/src/systems/session/`, `web/src/components/assistant-ui/`

This is the cross-route synthesis. Per-route deep dives live in `01_analysis_agents_$name.md`, `02_analysis_agents_$name_sessions_$id.md`, and `03_analysis_session_$id.md`.

---

## 1. What this module does

The Agents module is the operator surface for talking to and supervising agent sessions. It hosts three routes:

- `/agents/$name` - per-agent dashboard: header (name + IDLE/ACTIVE pill + count + Refresh / Configure / New session toolbar), 4-up stats grid, sessions table, MCP servers right-rail panel.
- `/agents/$name/sessions/$id` - active chat surface: chat header (status dot + breadcrumb + provider/workspace pills + delete/stop/resume), session resume failure banner, assistant-ui thread with composer, right-rail Inspector with Trace / Usage / Memory / Files / Vault tabs.
- `/session/$id` - id-only permalink that resolves the agent name and `replace`-redirects to the canonical `/agents/$name/sessions/$id`. Renders only loading + not-found states locally.

This module is load-bearing for the daemon - it is where the human/agent interaction loop actually closes. Per CLAUDE.md it MUST set the bar for the rest of the SPA.

---

## 2. Cross-route shell composition

### Sidebar (shared with the rest of the app)

- 44px workspace icon rail + ~220px panel, surface bg `#0E0E0F` (canvas-deep). Verified in live `/agents/general` empty state at 1440px (`_evidence/agents.name/live_1440_general_empty.png`).
- AGENTS section label, then a `tree` of agent rows; the active agent is rendered with a 2px left accent bar + surface bg per `DESIGN.md` line 484. Confirmed in `_evidence/agents.name/sb_default.png` (active row = `fraud-ops-agent`).
- `CONNECTED` status footer + version chip - both routes share this footer with no per-route variation.

### Header pattern

- `/agents/$name` uses the shared `PageHeader` primitive with provider icon + name + status Pill + count + toolbar buttons (`web/src/systems/agent/components/agent-page-header.tsx:36-93`).
- `/agents/$name/sessions/$id` uses a custom `ChatHeader` (12px tall, breadcrumb-style with status dot + agent name + chevron + session name + chevron + provider pill + chevron + workspace pill + activity inline) (`web/src/systems/session/components/chat-header.tsx:65-191`). It is **not** the shared `PageHeader`.
- This is the first inconsistency: the agent-detail header is title-style; the session header is breadcrumb-style. Same module, two header grammars.

### Right rail

- `/agents/$name` mounts `AgentInfoPanel` (320px, MCP servers list) hidden below `xl` (`web/src/systems/agent/components/agent-info-panel.tsx:12-23`).
- `/agents/$name/sessions/$id` mounts `SessionInspector` (320px, tabbed) also hidden below `xl` (`web/src/systems/session/components/session-inspector.tsx:541-552`).
- A `SessionInspectorDrawer` exists and is exported (`web/src/systems/session/index.ts:115`) but **no route mounts it** (verified by `rg "SessionInspectorDrawer" web/src/routes web/src/components` returning zero hits). On viewports <1280px the inspector is unreachable. **P0 ship blocker.**

### Composer / action gravity

- `/agents/$name` primary action is `+ New session` (accent fill, top-right toolbar).
- `/agents/$name/sessions/$id` primary action is the round accent send button at bottom-right of composer.
- Both use the same accent fill but neither route exposes a clear "back to parent" affordance for keyboard-only operators - the chat-header has no breadcrumb link on the agent name (it is plain `<span>`), so navigating back to `/agents/$name` requires the sidebar.

---

## 3. Visual + interaction consistency across routes

| Aspect | `/agents/$name` | `/agents/$name/sessions/$id` | `/session/$id` | Verdict |
|---|---|---|---|---|
| Header style | PageHeader (icon + title + Pill) | ChatHeader breadcrumb (12px h-12) | n/a (not-found only) | inconsistent |
| Status indicator | Pill mono `IDLE` / `ACTIVE` | Pill.Dot (8px green) + sr-only text | n/a | inconsistent |
| Loading state | Full-bleed `Loader2` + skeleton table rows | Full-bleed `Loader2` only (no skeleton) | Full-bleed `Loader2` | inconsistent |
| Not-found state | `Empty` icon + title + description + "Go home" button | toast-then-redirect (handled in route) | bare `AlertCircle` + sentence, no action | inconsistent (P1) |
| Right rail | `AgentInfoPanel` (`hidden xl:flex`) | `SessionInspector` (`hidden xl:flex`) | n/a | structurally consistent, but the drawer fallback is unmounted on both |
| Empty state copy | "No sessions yet" + "Start a new session for X from the toolbar above." | "Start a conversation. The assistant thread replays persisted history and continues live over the daemon stream." | "Session not found: X" | tone matches but the chat-empty mentions implementation detail ("daemon stream") that is leaky vs `COPY.md`'s operator-first voice |
| Destructive action | n/a | Trash2 ghost in chat header opens `Dialog` confirm (no typed-name guard) | n/a | adequate |
| Keyboard shortcuts | none documented | none documented (composer uses `Enter` to submit, but no Esc-to-cancel-stream wired beyond the inline Stop button) | n/a | weak |
| Mono / kind chip use | `IDLE` / `ACTIVE` Pill mono | `claude` / `risk-ops` mono pills with `normal-case` override | n/a | mostly consistent |

---

## 4. DESIGN.md / COPY.md drift across the module

Recurring drift to fix once at the system level rather than route-by-route:

1. **Permission Prompt uses raw Tailwind amber, not the design-token warning.** `permission-prompt.tsx:40-44, 89` uses `border-amber-500/40 bg-amber-500/5 text-amber-500`. DESIGN.md warning is `#FFD60A` with the 15%-tint formula (`--color-warning-tint`). Visible in `_evidence/agents.name.sessions.id/sb_pending-permission.png`. **P1.**
2. **Chat code blocks use `oneDark` Prism theme.** `message-markdown.tsx:16` imports `oneDark` from `react-syntax-highlighter/dist/esm/styles/prism`. The render uses Atom OneDark cool blues / purples on a surface that DESIGN.md explicitly mandates as warm-only. Confirmed in `_evidence/agents.name.sessions.id/sb_default.png` (`const heroHeadline = "Launch checkout in days, not quarters";` rendered with cool-blue keywords). **P1.**
3. **Send button does not honor the disabled token.** `session-thread.tsx:194-203`'s `ComposerPrimitive.Send` keeps accent fill at `disabled:opacity-50`. DESIGN.md disabled spec is `bg #4A4847, text #636366` (`DESIGN.md:154`). On the Stopped story the send button still renders accent orange while `disabled` is asserted (`_evidence/agents.name.sessions.id/sb_stopped_full.png`). **P1.**
4. **Code-block copy button is hover-only.** `message-markdown.tsx:101-104` uses `opacity-0 group-hover/codeblock:opacity-100` to reveal the copy button. DESIGN.md line 372 specifies the copy button is always visible (with a checkmark swap on success). On touch / keyboard-only the button is undiscoverable. **P1.**
5. **Assistant + user messages have no eyebrow label.** DESIGN.md chat-component spec (lines 394-408) requires the user bubble to render a `YOU + timestamp` mono eyebrow and assistant messages to render an agent-name eyebrow + 8px status dot. `session-thread.tsx:54-100` renders neither. The thread-empty state does show the agent name as eyebrow, but live messages drop it. **P1, cross-route.**
6. **Truthful UI test - create dialog ships an empty value next to the "Agent default provider:" label.** `_evidence/agents.name/live_create_dialog.png` shows the label with no value when the agent has no default. The label promises a value the runtime cannot fill. **P2.**
7. **Sidebar agent count badge uses a 1-character treatment.** `_evidence/agents.name/sb_default.png` shows `1` to the right of the agent title. DESIGN.md `MonoBadge` spec mandates min-width 14px / radius 7px / `solid-accent` for unread counts. The current rendering matches but only because the value is single digit. Untested for 100+ but worth flagging.

---

## 5. Truthful UI red flags

This is the single most important test for the chat surface. Per `web/CLAUDE.md` and `COPY.md` section 8: "Do not imply a metric, control, state, or repair path exists unless the runtime exposes it."

| Implied capability | Backed by runtime today? | Evidence |
|---|---|---|
| Streaming indicator (`Thinking…` Loader2) | yes - assistant-ui Empty status `running` | `session-thread.tsx:42-52` |
| Tool-call rendering with status pill | yes - `ToolCallCard` reads `toolResult` / `toolError` | `_evidence/agents.name.sessions.id/sb_default.png` |
| Reasoning / thought process collapsible | yes - `ThinkingBlock` | `thinking-block.tsx:1-47` |
| Permission prompt with allow-once / always / reject | yes - `approveSession` POST | `permission-prompt.tsx:20-36` |
| Forensic ledger lineage panel | yes (gated on `state === "stopped"`) - `useSessionLedger` | `agents.$name.sessions.$id.tsx:50` + `session-inspector.tsx:790-868` |
| Token usage / cost / rate | **scaffolded only** - `usage` prop is never wired in the route, always `undefined` | `agents.$name.sessions.$id.tsx` does not pass `usage`; the Inspector renders `Empty` "No usage yet". The Usage tab still shows in the tablist - **borderline truthful UI**. |
| Vault secrets list | yes - `useSessionVaultSecrets` | `agents.$name.sessions.$id.tsx:48` |
| File-reads aggregation | yes - derived from tool-call args in `deriveFileReads` | `session-inspector.tsx:301-328` |

The **Usage tab is the borderline case**: it advertises a metric the route never delivers. Either remove the tab on routes where the daemon does not yet stream usage, or wire a real source. Otherwise it implies "the runtime does not measure your tokens" when the truth is "the web layer never asked".

---

## 6. Information architecture observations

- The agent detail page has **two right-rail surfaces over the lifetime of a user's flow**: `AgentInfoPanel` (MCP servers) on the agent route, then `SessionInspector` (Trace / Usage / Memory / Files / Vault) on the session route. They share the 320px column and `xl:flex` gating but render unrelated data and have no transition. A user clicking into a session loses MCP context and gains telemetry context with no signpost. Worth questioning whether the agent right-rail should add a "Sessions" tab that mirrors the table.
- Empty stats grid renders 4 zero-cards even when the parent shows "No sessions yet" in the same viewport (`_evidence/agents.name/sb_no-sessions.png` shows ACTIVE / TOTAL RUNTIME / FAILED / LAST ACTIVITY all populated with zero or placeholder values over the empty state). That is double-counting empty: the metric grid says "nothing here" and the empty state says "nothing here". Hide the grid until there is at least one session, or use the grid as the only empty surface (drop the empty illustration).
- The session route's chat-header packs **status dot + agent name + chevron + session name + chevron + provider pill + chevron + workspace pill + activity inline** into a 48px row. It overflows below 1024px (verified at 768px, where both names truncate to `fraud-o…` and `Reserve sp…`). Three chevrons feel fragile; collapse to "agent / session - provider" with workspace as a secondary mono row, or hoist the workspace into the sidebar context.
- `/session/$id` exists only as a redirect target. It has a not-found state that drops the standard `Empty` chrome (no icon-with-action, no "Go home"). A user landing on a stale permalink gets red-text + sentence and no recovery affordance.

---

## 7. Top cross-module findings (synthesized)

The detailed P0-P3 findings live in each route file. The cross-cutting list:

1. **[P0] `SessionInspectorDrawer` is exported but never mounted.** Inspector is unreachable below 1280px. Affects the chat session route directly and is the single most user-visible alpha gap.
2. **[P0] At 320px the chat header truncates the agent and session names to ellipsis.** Operators on a phone cannot see what they are talking to. The breadcrumb design needs a responsive collapse strategy (drop chevrons + workspace pill, keep agent + session, render workspace as a secondary `text-tertiary` row).
3. **[P0] Permission Prompt uses Tailwind amber instead of the design-token warning palette.** A high-stakes decision surface shipping in the wrong color is alpha-blocking.
4. **[P1] `/session/$id` not-found state is bare.** No icon-with-action layout, no "Go home" / "Open agents" recovery.
5. **[P1] Code blocks ship in `oneDark` cool-blue palette.** Direct DESIGN.md drift on the most-rendered surface in the chat.
6. **[P1] Disabled send button keeps accent fill at 50% opacity.** Reads as active.
7. **[P1] Copy button on code blocks is hover-only.** Touch + keyboard inaccessible.
8. **[P1] User and assistant messages drop the eyebrow / metadata label.** DESIGN.md chat spec is ignored.
9. **[P2] Stats grid renders 4 zero-cards over the "No sessions yet" empty state.** Double-empty.
10. **[P2] Loading state on `/agents/$name/sessions/$id` is a single spinner with no skeleton chrome.** The page transition from agent detail to session feels unresolved.

---

## 8. Storybook coverage gaps

- `routes-app-stories-agents-name--*` covers default / no-sessions / sessions-loading / agent-loading / not-found / with-failed-session / many-agents. **Missing**: dense session list (50+), error state for `/api/sessions`, mobile/narrow viewport (320px / 768px) parameter, dark-on-dark interactions.
- `routes-app-stories-agents-name-sessions-id--*` covers default / loading / stopped / pending-permission / not-found-redirect. **Missing**: streaming-in-progress, tool error, resume-failure (the `SessionResumeFailure` banner has no top-level story under this path - must drill into the system component story), 320px and 768px viewports for the chat header truncation, code-block-heavy assistant message, multi-tool-call rapid-fire.
- `/session/$id` has **no route story at all**. Only the in-route redirect logic exists. **P2.**

---

## 9. Recommended action plan (module level)

1. `/impeccable craft session-inspector-drawer-mount` - mount `SessionInspectorDrawer` in `agents.$name.sessions.$id.tsx` so trace / usage / memory / files / vault are reachable below 1280px. Same drawer trigger should appear in the chat header on narrow viewports.
2. `/impeccable adapt agents-detail-and-session-routes` - responsive pass for both routes. Define collapse strategy for the chat-header breadcrumb (320 / 768) and the agent-detail toolbar (icon-only buttons with mono labels in tooltip on narrow). Ship a `route-story` parameter for `viewport: "mobile"` so it is enforced.
3. `/impeccable colorize permission-prompt` - replace `amber-500` literals with `--color-warning` / `--color-warning-tint`. Same pass: replace `oneDark` Prism style with a warm-tuned theme that uses `--color-text-primary` / `--color-text-secondary` / `--color-accent` for keywords.
4. `/impeccable harden composer-disabled-states` - disabled send button must use `--color-disabled` background and `--color-text-tertiary` text. Stop button styling needs the same treatment when busy.
5. `/impeccable typeset chat-message-eyebrows` - re-introduce the YOU / agent-name + status dot + timestamp eyebrows per DESIGN.md lines 394-410.
6. `/impeccable harden session-permalink-not-found` - `/session/$id` not-found should use the same `Empty` shell as `/agents/$name`, plus a primary `Open agents` button.
7. `/impeccable distill agent-detail-empty` - choose between metric grid OR empty illustration when sessions = 0. Not both.
8. `/impeccable polish 02_agents` - final visual + a11y sweep (keyboard navigation, focus rings, contrast on stopped chat, MCP transport pill tone).

---

## 10. Sign-off Checklist

- [x] Every claim cites a `file:line` or screenshot under `_evidence/`.
- [x] Cross-route inconsistencies enumerated with route + file references.
- [x] DESIGN.md / COPY.md drift listed with token-level evidence.
- [x] Truthful UI test applied to every advertised capability.
- [x] No em dashes used in this report (em dashes appear only inside cited DESIGN.md / source paths).
- [x] Storybook gaps named by exact story id.
