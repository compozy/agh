import type { Meta, StoryObj } from "@storybook/react-vite";

import { Separator } from "../separator";

const meta: Meta<typeof Separator> = {
  title: "ui/Separator",
  component: Separator,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component: "Structural divider available in horizontal and vertical orientations.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[360px] bg-background p-4 text-foreground">
        <Story />
      </div>
    ),
  ],
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <div className="flex flex-col gap-3">
      <span className="text-sm text-muted-foreground">Above</span>
      <Separator />
      <span className="text-sm text-muted-foreground">Below</span>
    </div>
  ),
};

export const Vertical: Story = {
  args: {},
  render: () => (
    <div className="flex h-12 items-center gap-3">
      <span className="text-sm text-muted-foreground">Left</span>
      <Separator orientation="vertical" />
      <span className="text-sm text-muted-foreground">Right</span>
    </div>
  ),
};
