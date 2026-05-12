import type { Meta, StoryObj } from "@storybook/react-vite";
import { NetworkIcon } from "lucide-react";
import { useEffect } from "react";

import { Button, Pill, useTopbarSlot } from "@agh/ui";

import { TopbarShell } from "../topbar-shell";

const meta: Meta<typeof TopbarShell> = {
  title: "components/TopbarShell",
  component: TopbarShell,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shell-level topbar host. Mounts a single `<Topbar>` for the entire `_app` outlet, hosts a `<TopbarSlotProvider>` so any descendant route can push tabs/search/actions via `useTopbarSlot`, subscribes to router `onResolved` to clear the slot on navigation and move focus to the topbar h1 for keyboard handoff. The Storybook stories simulate the route-context + slot-push pattern outside a real router.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-full bg-background border border-line">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Shell with placeholder route content. The router-derived topbar will fall back
 * to "Untitled" because Storybook does not provide a TanStack Router context.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <main className="px-6 py-5 text-[13px] text-muted">
        Outlet content. The topbar overhead reads `topbar` route context via TanStack Router; in
        Storybook the route falls back to "Untitled".
      </main>
    </TopbarShell>
  ),
};

function SlotPusher() {
  useTopbarSlot({
    title: "Tasks",
    count: 12,
    actions: <Button size="sm">New task</Button>,
  });
  // Re-push every render so the slot reflects the latest pill/button — see
  // useTopbarSlot bail-out in `packages/ui/src/components/custom/topbar.tsx`.
  useEffect(() => undefined, []);
  return null;
}

/**
 * Slot push: the descendant calls `useTopbarSlot` with title/count/actions,
 * exercising the shell's slot-bail-out on re-render.
 */
export const WithSlotPush: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <SlotPusher />
      <main className="px-6 py-5 flex items-center gap-3 text-[13px] text-muted">
        <Pill tone="accent">Live</Pill>
        Outlet content with topbar slot push.
      </main>
    </TopbarShell>
  ),
};

function ChannelSlotPusher() {
  useTopbarSlot({
    title: "Network",
    count: 5,
    search: (
      <input
        type="search"
        placeholder="Search channels..."
        className="h-7 w-48 rounded border border-line bg-canvas-soft px-2 text-[13px] text-fg outline-none placeholder:text-subtle focus:border-line-strong"
      />
    ),
    actions: <Button size="sm">New channel</Button>,
  });
  return null;
}

/**
 * Slot with search + actions trailing slot — mimics network/tasks routes.
 */
export const WithSearchAndActions: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <ChannelSlotPusher />
      <main className="px-6 py-5 flex items-center gap-3 text-[13px] text-muted">
        <NetworkIcon className="size-4 text-subtle" />
        Channel list outlet
      </main>
    </TopbarShell>
  ),
};
