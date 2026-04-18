import type { Meta, StoryObj } from "@storybook/react-vite";

import { MetricStrip } from "../metric-strip";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof MetricStrip> = {
  title: "components/design-system/MetricStrip",
  component: MetricStrip,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "A compact metric block for dense system summaries, pairing a mono label with a large value and optional supporting detail.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-5xl">
        <Story />
      </StoryFrame>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default metric strip for a primary system signal.
 */
export const Default: Story = {
  args: {
    label: "Credits used",
    value: "56.4%",
    detail: "Tokenized summary blocks for dashboards, trays, and shell overviews.",
    tone: "amber",
  },
};

/**
 * Color-coded signal states for different operating conditions.
 */
export const SignalStates: Story = {
  args: {},
  render: () => (
    <div className="grid w-full gap-3 md:grid-cols-3">
      <MetricStrip detail="Action-oriented highlight" label="Warm signal" tone="amber" value="08" />
      <MetricStrip detail="Healthy steady-state" label="Stable sessions" tone="green" value="24" />
      <MetricStrip detail="Utility emphasis" label="Queued reviews" tone="violet" value="05" />
    </div>
  ),
};
