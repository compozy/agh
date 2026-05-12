import type { Meta, StoryObj } from "@storybook/react-vite";
import { ActivityIcon, GaugeIcon, ServerIcon } from "lucide-react";

import { KpiCard } from "../kpi-card";

const meta: Meta<typeof KpiCard> = {
  title: "components/custom/KpiCard",
  component: KpiCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Single dashboard KPI tile per metric. Inter UC eyebrow label + 28 px display value at the warm `--fg-strong` ink. Flat on `--canvas-soft`, no border — depth comes from the warm-surface ramp.",
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
      <KpiCard icon={ActivityIcon} label="Active sessions" value="14" detail="2 since last hour" />
      <KpiCard
        icon={ServerIcon}
        label="Daemon uptime"
        value="3d 04h"
        detail="last restart 2026-05-07"
      />
      <KpiCard icon={GaugeIcon} label="Queue depth" value="0" detail="all workers idle" />
    </>
  ),
};

/**
 * Single tile with trailing slot (count chip, delta marker).
 */
export const WithDetail: Story = {
  args: {},
  render: () => <KpiCard label="Models configured" value="3" detail="Anthropic, OpenAI, Local" />,
};
