import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";

import { Button, PageHead, Pill, RouteNav, useTopbarSlot } from "@agh/ui";

import { TopbarShell } from "@/components/topbar-shell";

const meta: Meta<typeof TopbarShell> = {
  title: "systems/runtime/components/TopbarShell",
  component: TopbarShell,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shell-level topbar host. Mounts the route's stable H1 for the entire `_app` outlet, hosts a `<TopbarSlotProvider>` so descendants can push title overrides, routeNav, actions, and overflow, and restores focus to the Topbar H1 after path navigation.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-full border border-line bg-background">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Shell with placeholder route content. Without a TanStack Router match chain,
 * the breadcrumb falls back to an empty trail.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <main className="px-6 py-5 text-[13px] text-muted">
        Outlet content. Route crumbs come from per-route `beforeLoad` context; Storybook has no
        match chain.
      </main>
    </TopbarShell>
  ),
};

function SlotPusher() {
  useTopbarSlot({
    crumb: "Tasks",
    routeNav: (
      <RouteNav aria-label="Tasks views">
        <RouteNav.Link aria-current="page" href="#list">
          List
        </RouteNav.Link>
        <RouteNav.Link href="#kanban">Kanban</RouteNav.Link>
      </RouteNav>
    ),
    actions: <Button size="sm">New task</Button>,
  });
  useEffect(() => undefined, []);
  return null;
}

/**
 * Slot push: descendant pushes crumb/routeNav/actions (route chrome contract).
 */
export const WithSlotPush: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <SlotPusher />
      <main className="px-6 py-5 flex flex-col gap-3 text-[13px] text-muted">
        <PageHead count={12} title="Tasks" variant="index" />
        <div className="flex items-center gap-3">
          <Pill tone="accent">Live</Pill>
          Outlet content with topbar slot push.
        </div>
      </main>
    </TopbarShell>
  ),
};

function ActionsSlotPusher() {
  useTopbarSlot({
    crumb: "Network",
    actions: <Button size="sm">New channel</Button>,
  });
  return null;
}

/**
 * Trailing actions only — route identity remains in the shell Topbar.
 */
export const WithActions: Story = {
  args: {},
  render: () => (
    <TopbarShell>
      <ActionsSlotPusher />
      <main className="px-6 py-5 flex flex-col gap-3 text-[13px] text-muted">
        <PageHead count={5} title="Network" variant="index" />
        Channel list outlet
      </main>
    </TopbarShell>
  ),
};
