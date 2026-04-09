import type { Meta, StoryObj } from "@storybook/react-vite";

import { StatusDot } from "../status-dot";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof StatusDot> = {
  title: "components/design-system/StatusDot",
  component: StatusDot,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "A tiny signal marker for health, warning, utility, and danger cues within dense system rows.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-2xl">
        <Story />
      </StoryFrame>
    ),
  ],
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const statusTones = ["neutral", "amber", "green", "violet", "danger"] as const;

/**
 * Default neutral status marker.
 */
export const Default: Story = {
  args: {
    tone: "neutral",
  },
};

/**
 * Tone palette for common AGH signal states.
 */
export const SignalPalette: Story = {
  args: {},
  render: () => (
    <div className="flex w-full items-center gap-5">
      {statusTones.map(tone => (
        <div className="flex items-center gap-2" key={tone}>
          <StatusDot tone={tone} />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
            {tone}
          </span>
        </div>
      ))}
    </div>
  ),
};
