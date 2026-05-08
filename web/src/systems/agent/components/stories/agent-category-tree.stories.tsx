import type { Meta, StoryObj } from "@storybook/react-vite";

import { SidebarSurface } from "@/storybook/story-layout";
import { agentFixtures } from "@/systems/agent/mocks";
import { sessionFixtures } from "@/systems/session/mocks";

import { withStoryAgentCategories } from "./agent-command-select.stories";
import { AgentCategoryTree } from "../agent-category-tree";

const categorizedAgents = withStoryAgentCategories(agentFixtures);

const meta: Meta<typeof AgentCategoryTree> = {
  title: "systems/agent/AgentCategoryTree",
  component: AgentCategoryTree,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Sidebar tree for grouped agent navigation, including loading and empty states.",
      },
    },
  },
  decorators: [
    Story => (
      <SidebarSurface>
        <Story />
      </SidebarSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Grouped agents show nested categories and active-session dots.
 */
export const Default: Story = {
  args: {
    agents: categorizedAgents,
    agentsLoading: false,
    agentsError: false,
    sessions: sessionFixtures,
  },
};

/**
 * Loading keeps the sidebar height stable while the daemon responds.
 */
export const Loading: Story = {
  args: {
    agents: undefined,
    agentsLoading: true,
    agentsError: false,
    sessions: undefined,
  },
};

/**
 * Error explains that daemon reachability is the blocker.
 */
export const ErrorState: Story = {
  args: {
    agents: undefined,
    agentsLoading: false,
    agentsError: true,
    sessions: undefined,
  },
};

/**
 * Empty state points operators at the bootstrap command.
 */
export const Empty: Story = {
  args: {
    agents: [],
    agentsLoading: false,
    agentsError: false,
    sessions: [],
  },
};
