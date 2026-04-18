import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";
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
      <SidebarMenuItem>
        <SidebarMenuButton size="sm">Create release notes</SidebarMenuButton>
      </SidebarMenuItem>
      <SidebarMenuSub>
        <SidebarMenuSubItem>
          <SidebarMenuSubButton size="sm">Storybook rollout</SidebarMenuSubButton>
        </SidebarMenuSubItem>
        <SidebarMenuSubItem>
          <SidebarMenuSubButton size="sm">Review lane</SidebarMenuSubButton>
        </SidebarMenuSubItem>
      </SidebarMenuSub>
    </AgentSidebarGroupFrame>
  ),
};

export const EmptyGroup: Story = {
  render: () => <AgentSidebarGroupFrame />,
};
