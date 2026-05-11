import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChevronDown, ListChecksIcon, NetworkIcon } from "lucide-react";
import { useEffect } from "react";

import { Button, Pill } from "@agh/ui";
import { LaneTabs } from "../lane-tabs";
import { SearchInput } from "../search-input";
import { Topbar, TopbarOverflowIcon, TopbarSlotProvider, useTopbarSlot } from "../topbar";

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

/**
 * Auto-resolved count from `useNavCounts()`. The route declares its
 * `navCountKey` and the shell threads the resolved value via `navCount`.
 */
export const AutoResolvedNavCount: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <Topbar
        navCount={42}
        route={{
          title: "Tasks",
          icon: ListChecksIcon,
          navCountKey: "tasks",
        }}
      />
    </TopbarSlotProvider>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "When the slot omits `count` and the route declares `navCountKey`, the shell passes the resolved count through `navCount`.",
      },
    },
  },
};

function DetailModeSetup() {
  useTopbarSlot({
    back: () => undefined,
    meta: (
      <>
        <span className="font-mono text-[10.5px] text-(--faint)">task_01H</span>
        <span className="text-[12px] text-(--muted)">created 2h ago</span>
      </>
    ),
    actions: <Button size="sm">Run</Button>,
    overflow: (
      <Button aria-label="More" size="sm" variant="ghost">
        <TopbarOverflowIcon className="size-3.5" />
      </Button>
    ),
  });
  return null;
}

/**
 * Detail-mode topbar (ADR-005 §5/§8) — back chevron, meta line, overflow menu.
 */
export const DetailMode: Story = {
  args: {},
  render: () => (
    <TopbarSlotProvider>
      <DetailModeSetup />
      <Topbar
        route={{
          title: "Reconcile order ledger",
          icon: ChevronDown,
        }}
      />
    </TopbarSlotProvider>
  ),
};
