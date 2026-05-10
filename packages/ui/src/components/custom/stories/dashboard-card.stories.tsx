import type { Meta, StoryObj } from "@storybook/react-vite";
import { ActivityIcon, GaugeIcon, ServerIcon } from "lucide-react";

import { DashboardCard } from "../dashboard-card";

const meta: Meta<typeof DashboardCard> = {
  title: "components/custom/DashboardCard",
  component: DashboardCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Single dashboard tile per metric. Mono UPPERCASE eyebrow label by default; large display value uses `--text-display-2xl`-equivalent at the warm `--fg-strong` ink. Stays flat on `--canvas-soft` with a 1px `--line` ring; never reach for the SaaS hero-metric template.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="grid w-[760px] grid-cols-3 gap-3 bg-background p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Three real operator metrics laid out in a grid — one accent / status icon per tile, no background colour drift.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <>
      <DashboardCard
        icon={ActivityIcon}
        label="Active sessions"
        value="14"
        detail="2 since last hour"
      />
      <DashboardCard
        icon={ServerIcon}
        label="Daemon uptime"
        value="3d 04h"
        detail="last restart 2026-05-07"
      />
      <DashboardCard icon={GaugeIcon} label="Queue depth" value="0" detail="all workers idle" />
    </>
  ),
};

/**
 * Sentence-case label for places where the dashboard rhythm is broken (a settings card row).
 */
export const SentenceLabel: Story = {
  args: {},
  render: () => (
    <DashboardCard
      labelCase="sentence"
      label="Models configured"
      value="3"
      detail="Anthropic, OpenAI, Local"
    />
  ),
};
