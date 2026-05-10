import type { Meta, StoryObj } from "@storybook/react-vite";

import { PriorityBars } from "../priority-bars";

const meta: Meta<typeof PriorityBars> = {
  title: "components/custom/PriorityBars",
  component: PriorityBars,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Four-bar priority indicator (low / medium / high / urgent). Filled bars adopt the requested signal tone; unfilled bars use `--line` so the rest position stays quiet on the warm canvas.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="flex items-end gap-6 bg-background p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * All four levels rendered side by side using the default accent tone.
 */
export const Levels: Story = {
  args: {},
  render: () => (
    <>
      <PriorityBars level="low" />
      <PriorityBars level="medium" />
      <PriorityBars level="high" />
      <PriorityBars level="urgent" />
    </>
  ),
};

/**
 * Tone variants — desaturated signal palette only, never raw saturated colors.
 */
export const Tones: Story = {
  args: {},
  render: () => (
    <>
      <PriorityBars level="urgent" tone="accent" />
      <PriorityBars level="urgent" tone="success" />
      <PriorityBars level="urgent" tone="warning" />
      <PriorityBars level="urgent" tone="danger" />
      <PriorityBars level="urgent" tone="info" />
      <PriorityBars level="urgent" tone="neutral" />
    </>
  ),
};
