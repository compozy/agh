import type { Meta, StoryObj } from "@storybook/react-vite";

import { SidebarSurface } from "@/storybook/story-layout";
import { primaryAgentFixture } from "@/systems/agent/mocks";

import { AgentSidebarGroup } from "../agent-sidebar-group";

const meta: Meta<typeof AgentSidebarGroup> = {
  title: "systems/agent/AgentSidebarGroup",
  component: AgentSidebarGroup,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AgentSidebarGroupFrame({ children }: { children?: React.ReactNode }) {
  return (
    <SidebarSurface>
      <div className="p-3">
        <AgentSidebarGroup agent={primaryAgentFixture} onNewSession={() => undefined}>
          {children}
        </AgentSidebarGroup>
      </div>
    </SidebarSurface>
  );
}

export const Default: Story = {
  render: () => (
    <AgentSidebarGroupFrame>
      <li>
        <button
          type="button"
          className="w-full rounded-md px-2 py-1 text-left text-xs text-muted-foreground hover:bg-[color:var(--color-hover)] hover:text-foreground"
        >
          Create release notes
        </button>
      </li>
      <li>
        <button
          type="button"
          className="w-full rounded-md px-2 py-1 text-left text-xs text-muted-foreground hover:bg-[color:var(--color-hover)] hover:text-foreground"
        >
          Storybook rollout
        </button>
      </li>
      <li>
        <button
          type="button"
          className="w-full rounded-md px-2 py-1 text-left text-xs text-muted-foreground hover:bg-[color:var(--color-hover)] hover:text-foreground"
        >
          Review lane
        </button>
      </li>
    </AgentSidebarGroupFrame>
  ),
};

export const EmptyGroup: Story = {
  render: () => <AgentSidebarGroupFrame />,
};
