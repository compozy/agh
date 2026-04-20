# Task Memory: task_21.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite Session composer onto `@agh/ui` primitives + DESIGN.md §4 "Chat Input" chrome. Preserve Enter / Shift+Enter / disabled / 200px auto-grow / whitespace-ignore behavior. Promote `onSend` from a bare `(text: string)` call to a structured payload with `skillId` / `channel` / `attachments`. Add per-session draft persistence through the session store.

## Important Decisions

- **`onSend(payload: MessageComposerPayload)`** shape: `{ text, skillId?, channel?, attachments? }`. Route `session.$id.tsx` now passes `page.handleSend` which narrows to `chat.sendMessage(payload.text)` — `skillId`/`channel`/`attachments` are surfaced for future daemon wiring but not yet sent to the prompt endpoint.
- **Draft persistence lives on the session store**, not a dedicated store. Added `drafts: Record<sessionId, { text, skillId?, channel? }>` + `setDraft` / `clearDraft` actions. `setActiveSession` explicitly preserves `drafts` across switches so unsent text survives route navigations; `clearSession` (logout) wipes them.
- **Picker trigger pattern**: Base UI `ComboboxPrimitive.Trigger` rendered as a pill-style `<button>` directly — the `@agh/ui` wrapper `ComboboxTrigger` always appends a ChevronDown, which is designed for inline ComboboxInput trigger chevrons and doesn't fit a standalone pill chip. Same reasoning for `ComboboxPrimitive.Input` rendered inside `ComboboxContent` so the search input sits inside the popup rather than the default inline input-group chrome.
- **Textarea chrome lives on the container**: the wrapped `@agh/ui` `Textarea` gets overriding classes (`border-none bg-transparent shadow-none outline-none focus-visible:border-transparent focus-visible:ring-0`) so the outer `<div data-testid="composer-container">` carries the 12px radius, surface fill, divider border, and `focus-within:border-[color:var(--color-accent)]` — matching DESIGN.md §4.
- **Composer extracted `useMessageComposer` hook** — required to clear the `compozy-react(max-component-complexity)` oxlint rule (behavior score >7). The hook owns text/skill/channel/attachment state + handlers; the component owns only layout + pickers + attachments chips.
- **Skills + Channels sourced at the page hook** (`useSessionPage`) via `useSkills(workspaceId)` + `useNetworkChannels({ enabled })`. Composer stays pure/presentational and hides the skill/channel pill when the list is empty. Non-`enabled` skills are filtered out.
- **Stories for open-picker states** are `play-fn`-tagged interaction tests (not visual baselines) because the workflow memory gate skips `play-fn` stories from `web-visual`. Visual baselines cover `Empty` / `Typing` / `Disabled`; interaction stories cover attach-flow / skill-flow / channel-flow / focus-border / keyboard-send.
- **Kept `data-testid="composer-container"`, `composer-textarea`, `composer-send-button`** as-is so the existing test contract + chat-view integration tests continue to pass.

## Learnings

- `@agh/ui` `Textarea` is a thin styled `<textarea>`; it is NOT self-contained chrome. When using it inside a custom container, you have to *override* its own border + background + rounded so the container's chrome shows through. Setting `dark:bg-transparent` is necessary because the primitive defaults to `dark:bg-input/30`.
- `<Popover>`'s controlled `open` / `onOpenChange` must be threaded explicitly when a downstream handler needs to close the popup on pick (attach flow). Uncontrolled mode stays open after click.
- `base-ui` `Combobox` with `items={string[]}` + `value={""}` is a clean way to get single-select string IDs without authoring `itemToStringLabel` / `itemToStringValue`. Empty-string value = "no selection".
- oxlint's `compozy-react(max-component-complexity)` is real and fires on components with >7 hooks/handlers. Extract to a hook in the same folder when it fires.
- The session page test (`-session.$id.test.tsx`) renders the real `useSessionPage` hook, so adding `useSkills` + `useNetworkChannels` inside that hook required adding module-level `vi.mock` calls for `@/systems/skill` and `@/systems/network` in the test file. Without them, the test renderer blows up with "No QueryClient set".

## Files / Surfaces

- `web/src/systems/session/components/message-composer.tsx` — full rewrite.
- `web/src/systems/session/hooks/use-message-composer.ts` — new hook (state + handlers).
- `web/src/systems/session/stores/session-store.ts` — added `drafts` / `setDraft` / `clearDraft`.
- `web/src/systems/session/index.ts` — re-exports composer + draft types.
- `web/src/hooks/routes/use-session-page.ts` — wires skills + channels, narrows `onSend` payload to text.
- `web/src/routes/_app/session.$id.tsx` — passes `sessionId` + `skills` + `channels` + new `handleSend`.
- `web/src/routes/_app/-session.$id.test.tsx` — added `useSkills` + `useNetworkChannels` mocks.
- `web/src/systems/session/components/message-composer.test.tsx` — 15 specs, 93% coverage on composer, 98% on hook.
- `web/src/systems/session/stores/session-store.test.ts` + `session-store-switch.test.ts` — new draft tests + switch-preservation test.
- `web/src/systems/session/components/stories/message-composer.stories.tsx` — 7 stories: 3 visual + 4 play-fn + 1 send-shortcut + 1 focus-border.
- `web/tests/visual/__snapshots__/systems-session-messagecomposer--{empty,typing,disabled}-chromium-darwin.png` — new darwin baselines (stale `--default` deleted).

## Errors / Corrections

- First composer iteration triggered `max-component-complexity: score 17`. Fix: extract `useMessageComposer` hook.
- First `use-session-page` wire used `skillsData?.skills` but `listSkills` returns the unwrapped array (`SkillPayload[]`). Fix: `skillsData ?? []`.

## Ready for Next Run

- Task 22 (Session inspector panel): may want to source `channels` via `useNetworkChannels` too (already available via the `useSessionPage` wiring — can be passed through).
- When the daemon prompt endpoint grows to accept `skillId` / `channel` / `attachments`, wire them through `useSessionChat.sendMessage` — today `useSessionPage.handleSend` drops those payload fields.
- The `/design-system` showcase does NOT yet feature the new composer; if a future task adds a Composer card it should use `MessageComposer` with `skills`/`channels` fixtures.
