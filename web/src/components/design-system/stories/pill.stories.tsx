import type { Meta, StoryObj } from "@storybook/react-vite";

import { Pill } from "../pill";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof Pill> = {
  title: "components/design-system/Pill",
  component: Pill,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Compact mono pill treatments used for filters, tags, and operating-state labels across AGH surfaces.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-3xl">
        <Story />
      </StoryFrame>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default pill treatment for neutral metadata.
 */
export const Default: Story = {
  args: {
    children: "Filters",
    kind: "filter",
    tone: "neutral",
    emphasis: "muted",
  },
};

/**
 * Essential tone and size combinations used in the first-pass foundation layer.
 */
export const Palette: Story = {
  args: {},
  render: () => (
    <div className="flex w-full flex-wrap gap-2">
      <Pill emphasis="strong" kind="filter" tone="amber">
        Foundations
      </Pill>
      <Pill kind="filter">Panels</Pill>
      <Pill emphasis="strong" kind="tag" tone="green">
        Stable
      </Pill>
      <Pill emphasis="strong" kind="tag" tone="violet">
        Utility
      </Pill>
      <Pill emphasis="strong" kind="state" tone="danger">
        Attention
      </Pill>
    </div>
  ),
};
