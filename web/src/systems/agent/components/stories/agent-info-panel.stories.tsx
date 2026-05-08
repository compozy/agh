import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { primaryAgentFixture } from "@/systems/agent/mocks";
import type { AgentPayload } from "@/systems/agent/types";
import { StorySurface } from "@/storybook/story-layout";

import { AgentInfoPanel } from "../agent-info-panel";

const richAgentFixture: AgentPayload = {
  ...primaryAgentFixture,
  mcp_servers: [
    {
      name: "filesystem",
      transport: "stdio",
      command: "npx @modelcontextprotocol/server-filesystem",
    },
    { name: "github", transport: "http", url: "https://mcp.github.com" },
    { name: "memory", transport: "stdio", command: "agh-memory --readonly" },
  ],
};

const meta: Meta<typeof AgentInfoPanel> = {
  title: "systems/agent/AgentInfoPanel",
  component: AgentInfoPanel,
  parameters: {
    layout: "fullscreen",
  },
  args: {
    agent: richAgentFixture,
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

interface FrameProps {
  children: ReactNode;
}

function Frame({ children }: FrameProps) {
  return (
    <StorySurface className="flex">
      <div className="flex flex-1 items-center justify-center text-sm text-(--color-text-secondary)">
        Agent detail content
      </div>
      {children}
    </StorySurface>
  );
}

/**
 * Default — three MCP servers rendered as compact rows with transport chips.
 */
export const Default: Story = {
  args: {},
  render: args => (
    <Frame>
      <AgentInfoPanel {...args} />
    </Frame>
  ),
};

/**
 * Empty state — agent declares no MCP servers.
 */
export const Empty: Story = {
  args: { agent: { ...primaryAgentFixture, mcp_servers: [] } },
  render: args => (
    <Frame>
      <AgentInfoPanel {...args} />
    </Frame>
  ),
};
