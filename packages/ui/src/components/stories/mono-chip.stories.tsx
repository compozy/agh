import type { Meta, StoryObj } from "@storybook/react-vite";

import { MonoChip } from "@agh/ui";

const meta: Meta<typeof MonoChip> = {
  title: "ui/MonoChip",
  component: MonoChip,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Neutral inline chip — mirrors `.mono-chip` (default tone) in `docs/design/web-inspiration/styles/app.css`. Use for capability descriptors and tag rows. For tinted semantic variants use `MonoBadge`.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: { children: "code" },
};

export const Row: Story = {
  render: () => (
    <div className="flex flex-wrap items-center gap-1">
      {["code", "shell", "file.read", "file.write", "plan.delegate"].map(label => (
        <MonoChip key={label}>{label}</MonoChip>
      ))}
    </div>
  ),
};
