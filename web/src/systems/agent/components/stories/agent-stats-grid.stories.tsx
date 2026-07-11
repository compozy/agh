import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import { AgentStatsGrid } from "../agent-stats-grid";

const meta: Meta<typeof AgentStatsGrid> = {
  title: "systems/agent/components/AgentStatsGrid",
  component: AgentStatsGrid,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-4xl">{children}</div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  args: {
    total: 205,
    active: 7,
    resumable: 13,
    lastActivityAt: new Date().toISOString(),
  },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};

export const Idle: Story = {
  args: {
    total: 42,
    active: 0,
    resumable: 4,
    lastActivityAt: "2026-04-17T18:10:00Z",
  },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};

export const Empty: Story = {
  args: { total: 0, active: 0, resumable: 0, lastActivityAt: null },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};
