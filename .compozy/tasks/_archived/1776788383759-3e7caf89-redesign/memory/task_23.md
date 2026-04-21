# Task Memory: task_23.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Task 23 landed — Network domain rewritten on `@agh/ui`:
- Route `/network` rewritten around `PageHeader` (title + count + `Pills` + primary action) above `SplitPane`; `WorkspacePageShell` / `PillButton` / `MetricStrip` dropped. Metric row stays above SplitPane inside a `network-shell-body` container so `network-metric-queued-msgs` + the other metric test-ids still resolve for e2e + artifact session fixtures.
- Channels + peers list panels rewritten on `SearchInput` + button list rows + `Empty` states for loading/error/no-results. Rows expose `data-state="selected"` + `aria-pressed` and preserve `network-channel-item-{id}` / `network-peer-item-{id}` / `network-channels-list-panel` / `network-peers-list-panel` / `network-channels-list-empty` / `network-peers-list-empty` / `network-channels-list-loading` / `network-peers-list-loading` / `network-channels-list-error` / `network-peers-list-error` test-ids.
- Channel detail rewritten as three `Section`s: Wire trace (`Table` of channel metadata k/v), Members (list with `StatusDot` + `MonoBadge`), Messages (per-message `KindChip` + author + `CodeBlock` payload). Loading renders the messages-section Loader2. Error/empty fall back to `Empty`. `network-channel-message-{id}` preserved; new test-ids: `network-channel-wire-trace`, `network-channel-members-list`, `network-channel-member-{peer_id}`, `network-channel-message-kind-{id}`, `network-channel-message-payload-{id}`.
- Peer detail rewritten as three `Section`s: Capabilities (`KindChip` list from `peer_card.capabilities`), Channels (`Table` single-row with join/last_seen), Message Statistics (4-column `Metric` row). `network-peer-metric-{slug}` + `network-peer-detail-panel` + "View Session" link preserved. New test-ids: `network-peer-capabilities`, `network-peer-capability-{capability}`, `network-peer-channel-{channel}`.
- Create channel dialog rewritten on `Dialog` + `Field` + `Input` + `Section` + agent list with `MonoBadge` provider chips. Submit button stays disabled while `canSubmit=false` and no-op's on click; form onSubmit also short-circuits when disabled to avoid edge-case double-firing.
- `network-empty-state` accepts both Lucide component icons and pre-rendered ReactNode via the same detection pattern as `@agh/ui` `Empty`.
- Stories updated: channels + peers list panels grew `Error` + `SearchFilter`/`RowSelect` play-fn stories; route-level stories grew `SelectChannel` (play-fn) + `DisabledSplitPaneAbsent` (play-fn) + `ChannelsError`. Loading route story kept.
- Playwright visual baselines regenerated via `bun run test:visual:update` — 248 passed. New network baselines added: `networkchannelslistpanel--error`, `networkpeerslistpanel--error`, `networkchanneldetailpanel--loading`, `networkchanneldetailpanel--no-messages`, `networkpeerdetailpanel--loading`, plus `routes-app-stories-network--channels-error`.
- Unit coverage on rewritten components: channels-list 100%, peers-list 87.5%, channel-detail 84.84%, peer-detail 92.85%, create-channel-dialog 90.9% (all ≥80%).
- Existing `src/routes/_app/-network.test.tsx` updated to assert `network-shell` (new) instead of `workspace-page-shell` (removed) and "Capabilities" / "Channels" / "Message Statistics" instead of the old "Peer Identity" Section label.

## Important Decisions

- **Wire trace = channel metadata Table**, not message trace. Rendered as key/value rows covering `channel`, `last message`, `messages`, `peers`, `local peers`, `remote peers`, `sessions`. Keeps the Wire trace Section distinct from Messages Section below it. The protocol-kinds oracle (agh-network/v0) is surfaced via the per-message `KindChip` in the Messages section.
- **Message kind derivation** uses the optional `intent` field on `NetworkChannelMessage` (OpenAPI `intent?: string`), validated against the seven-kind set (`greet | whois | say | direct | recipe | receipt | trace`). Anything unknown falls back to `say`.
- **Peer status tone** derived inline from `local`/`last_seen` without touching `network-formatters` — the legacy `getPeerPresenceTone` helper returns raw tailwind class strings that conflict with `StatusDot.tone`. Left `getPeerPresenceTone` untouched to keep the formatters module stable.
- **CreateChannelDialog submit guard.** The form's onSubmit calls `onSubmit` only when `canSubmit && !isSubmitting`; mirrors the button-disabled state so `userEvent.click` on a button that's disabled but technically in the DOM cannot double-fire.
- **Route test mocked `WorkspacePageShell` stays in the test file** even though the route no longer uses it — other mocks in the same `@/systems/workspace` factory (`useActiveWorkspace`, `useWorkspace`) are still needed. The unused shell mock is harmless.
- **CodeBlock per message payload** rendered with `copyable={false}` + `showPrompt={false}` since the content is chat text, not a shell command.

## Learnings

- TanStack's `RouterProvider` renders children asynchronously — minimal router harnesses for testing `Link` components do not flush to the DOM in a synchronous `render()`. The repo-wide idiom is `vi.mock("@tanstack/react-router", () => ({ Link: ({ children, ...rest }) => <a>{children}</a> }))`. Always prefer the stub over spinning up a full router for component tests.
- `PageHeader.icon` accepts a **component reference** (e.g. `NetworkIcon`) or a **render function** that yields the icon; both render inside the 24px icon well. Inline functions are fine but lose the Lucide-forwarded `size` prop.
- `SplitPane` auto-hides the list at `< narrowBreakpoint` (default 768px) and drives detail ↔ empty via AnimatePresence on its own. The host only supplies slots + optional `onDetailClose`; no manual width management.

## Files / Surfaces

- `web/src/routes/_app/network.tsx` — rewritten (PageHeader + Pills + Metric strip + SplitPane).
- `web/src/systems/network/components/network-channels-list-panel.tsx` — rewritten.
- `web/src/systems/network/components/network-peers-list-panel.tsx` — rewritten.
- `web/src/systems/network/components/network-channel-detail-panel.tsx` — rewritten.
- `web/src/systems/network/components/network-peer-detail-panel.tsx` — rewritten.
- `web/src/systems/network/components/network-create-channel-dialog.tsx` — rewritten.
- `web/src/systems/network/components/network-empty-state.tsx` — rewritten (icon accepts IconComponent | ReactNode).
- `web/src/systems/network/components/{network-channels-list-panel,network-peers-list-panel,network-channel-detail-panel,network-peer-detail-panel,network-create-channel-dialog}.test.tsx` — new unit tests (20 specs).
- `web/src/systems/network/components/stories/*.stories.tsx` — five story files updated with new states + play-fn interaction tests.
- `web/src/routes/_app/stories/-network.stories.tsx` — added `SelectChannel`, `DisabledSplitPaneAbsent`, `ChannelsError` stories.
- `web/src/routes/_app/-network.test.tsx` — 3 specs retargeted to new shell + peer-detail Section labels.
- `web/tests/visual/__snapshots__/` — 18 network-related darwin baselines (re-generated).

## Errors / Corrections

- Initial `NetworkChannelDetailPanel` + `NetworkPeerDetailPanel` tests rendered with a fully-mounted `RouterProvider`. The `Link` components inside the panels registered asynchronously and the panel never appeared in the DOM before `getByTestId` ran. Switched to the `vi.mock("@tanstack/react-router", …)` pattern used elsewhere in the repo.
- The existing `src/routes/_app/-network.test.tsx` expected `workspace-page-shell` + "Peer Identity". Updated to `network-shell` + "Capabilities"/"Channels"/"Message Statistics" to match the new peer detail structure.

## Ready for Next Run

- Phase 5 continues with task 24 (Automation domain) next. Same pattern applies: PageHeader + SplitPane + Section/Metric/KindChip/StatusDot over `useAutomationPage`.
- Network domain is greenfield-clean: `rg "@/components/(ui|design-system)/" web/src/systems/network/ web/src/routes/_app/network.tsx` returns zero matches.
