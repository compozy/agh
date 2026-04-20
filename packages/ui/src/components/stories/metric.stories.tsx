import type { Meta, StoryObj } from "@storybook/react-vite";

import { Metric, type MetricTone } from "../metric";

const meta: Meta<typeof Metric> = {
  title: "ui/Metric",
  component: Metric,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Metric card — mono eyebrow + Inter 24px/700 value + optional inline detail or subtext. Per DESIGN.md §4 Metric Cards.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const TONES: MetricTone[] = ["default", "accent", "success", "warning", "danger"];

export const Default: Story = {
  args: {
    label: "Sessions",
    value: "12",
  },
  render: args => (
    <div className="w-[220px]">
      <Metric {...args} />
    </div>
  ),
};

export const WithDetail: Story = {
  args: {
    label: "Credits",
    value: "56.4%",
    detail: "+4.2%",
    tone: "success",
  },
  render: args => (
    <div className="w-[240px]">
      <Metric {...args} />
    </div>
  ),
};

export const WithSubtext: Story = {
  args: {
    label: "Queue depth",
    value: "08",
    subtext: "3 in progress · 5 pending review",
  },
  render: args => (
    <div className="w-[320px]">
      <Metric {...args} />
    </div>
  ),
};

export const Tones: Story = {
  render: () => (
    <div className="grid gap-3 md:grid-cols-3">
      {TONES.map(tone => (
        <Metric key={tone} label={tone} value="08" tone={tone} subtext={`tone=${tone}`} />
      ))}
    </div>
  ),
};

export const DashboardRow: Story = {
  render: () => (
    <div className="grid gap-3 md:grid-cols-4">
      <Metric label="Channels" value="24" />
      <Metric label="Peers online" value="18" tone="success" detail="+3" />
      <Metric label="Queued runs" value="05" tone="warning" />
      <Metric label="Failures / 1h" value="00" tone="danger" subtext="Last failure: 2h ago" />
    </div>
  ),
};
