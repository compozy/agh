import type { Meta, StoryObj } from "@storybook/react-vite";

import { MonoBadge, type MonoBadgeTone } from "../mono-badge";

const meta: Meta<typeof MonoBadge> = {
  title: "ui/MonoBadge",
  component: MonoBadge,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Inline mono pill for identifiers (agent IDs, versions, protocol names) and tinted status badges. 6px radius, JetBrains Mono 11px/500 at 0.06em tracking.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const TONES: MonoBadgeTone[] = [
  "default",
  "neutral",
  "accent",
  "success",
  "warning",
  "danger",
  "info",
];

export const Default: Story = {
  args: {
    children: "agent-42",
  },
};

export const AllTones: Story = {
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {TONES.map(tone => (
        <MonoBadge key={tone} tone={tone}>
          {tone}
        </MonoBadge>
      ))}
    </div>
  ),
};

export const LowercaseIdentifier: Story = {
  args: {
    uppercase: false,
    children: "agh-network/v0",
  },
};

export const BesideLabel: Story = {
  render: () => (
    <div className="flex items-center gap-2 text-[color:var(--color-text-secondary)]">
      <span className="text-sm">Running</span>
      <MonoBadge tone="accent">v0.2.1</MonoBadge>
    </div>
  ),
};
