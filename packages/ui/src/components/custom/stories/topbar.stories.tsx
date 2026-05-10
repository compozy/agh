import type { Meta, StoryObj } from "@storybook/react-vite";
import { ListChecksIcon, NetworkIcon } from "lucide-react";
import { useEffect } from "react";

import { Button, Pill } from "@agh/ui";
import { LaneTabs } from "../lane-tabs";
import { SearchInput } from "../search-input";
import { Topbar, TopbarSlotProvider, useTopbarSlot } from "../topbar";

const meta: Meta<typeof Topbar> = {
  title: "components/custom/Topbar",
  component: Topbar,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shell-level topbar (ADR-002). Route context (icon, title, count) comes from TanStack Router; live tabs/search/actions are pushed into a single dynamic slot via `useTopbarSlot`. Title is focusable so the shell can move focus on route resolve.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-full bg-background border border-(--line)">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Static route only — icon, title, count.
 */
export const RouteOnly: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <Topbar
        route={{
          title: "Tasks",
          icon: ListChecksIcon,
          getCount: () => 12,
        }}
      />
    </TopbarSlotProvider>
  ),
};

function TabsAndSearchSetup() {
  useTopbarSlot({
    tabs: (
      <LaneTabs
        ariaLabel="Tasks lanes"
        items={[
          { value: "all", label: "All", count: 124 },
          { value: "active", label: "Active", count: 8 },
        ]}
        value="active"
        onChange={() => undefined}
      />
    ),
    search: <SearchInput placeholder="Search tasks..." />,
    actions: <Button size="sm">New task</Button>,
  });
  return null;
}

/**
 * Full slot composition — tabs in the middle, search + actions on the trailing edge.
 */
export const WithSlot: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <TabsAndSearchSetup />
      <Topbar
        route={{
          title: "Tasks",
          icon: ListChecksIcon,
          getCount: () => 124,
        }}
      />
    </TopbarSlotProvider>
  ),
};

function LiveTitleSetup({ count }: { count: number }) {
  useEffect(() => {
    /* re-render trigger only — slot is recomputed each render, the component's bail-out absorbs no-op pushes */
  }, [count]);
  useTopbarSlot({
    title: "Live route title",
    count,
    actions: <Pill tone="accent">Live</Pill>,
  });
  return null;
}

/**
 * Slot overrides title and count for routes that resolve from loader data.
 */
export const LiveTitle: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <LiveTitleSetup count={42} />
      <Topbar
        route={{
          title: "Static fallback",
          icon: NetworkIcon,
          getCount: () => 0,
        }}
      />
    </TopbarSlotProvider>
  ),
};
