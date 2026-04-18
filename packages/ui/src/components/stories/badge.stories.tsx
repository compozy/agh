import type { Meta, StoryObj } from "@storybook/react-vite";

import { Badge } from "../badge";

const meta: Meta<typeof Badge> = {
  title: "ui/Badge",
  component: Badge,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component: "Inline status pill for labels, counts, and quick state markers.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: { children: "Beta", variant: "default" },
};

export const Variants: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2 bg-background p-4 text-foreground">
      <Badge variant="default">Default</Badge>
      <Badge variant="secondary">Secondary</Badge>
      <Badge variant="destructive">Destructive</Badge>
      <Badge variant="outline">Outline</Badge>
      <Badge variant="ghost">Ghost</Badge>
      <Badge variant="link">Link</Badge>
    </div>
  ),
};
