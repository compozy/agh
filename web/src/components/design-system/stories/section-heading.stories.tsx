import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "@agh/ui";

import { SectionHeading } from "../section-heading";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof SectionHeading> = {
  title: "components/design-system/SectionHeading",
  component: SectionHeading,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "A heading cluster with mono eyebrow, display title, supporting text, and optional action for major command-surface sections.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-6xl">
        <Story />
      </StoryFrame>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default heading treatment with an inline action.
 */
export const Default: Story = {
  args: {
    eyebrow: "AGH / foundations",
    title: "Command surfaces that feel authored.",
    description:
      "The heading primitive establishes the typography and metadata rhythm used at the top of major panels and routes.",
    action: <Button variant="default">Launch review</Button>,
  },
};

/**
 * Minimal heading composition without an action block.
 */
export const Minimal: Story = {
  args: {
    eyebrow: "Panel / heading",
    title: "Mono metadata and display hierarchy stay consistent.",
    description: "Use the minimal form when the page does not require a secondary CTA.",
  },
};
