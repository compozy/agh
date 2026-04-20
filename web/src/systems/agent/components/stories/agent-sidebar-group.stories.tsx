import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

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

const mockSessions = (
  <>
    <li>
      <button
        type="button"
        data-testid="story-session-release-notes"
        className="w-full rounded-md px-2 py-1 text-left text-xs text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]"
      >
        Create release notes
      </button>
    </li>
    <li>
      <button
        type="button"
        className="w-full rounded-md px-2 py-1 text-left text-xs text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]"
      >
        Storybook rollout
      </button>
    </li>
    <li>
      <button
        type="button"
        className="w-full rounded-md px-2 py-1 text-left text-xs text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]"
      >
        Review lane
      </button>
    </li>
  </>
);

function Frame({ children }: { children?: React.ReactNode }) {
  return (
    <SidebarSurface>
      <div className="p-3">{children}</div>
    </SidebarSurface>
  );
}

export const ExpandedWithSessions: Story = {
  render: () => (
    <Frame>
      <AgentSidebarGroup
        agent={primaryAgentFixture}
        sessionCount={3}
        onNewSession={() => undefined}
      >
        {mockSessions}
      </AgentSidebarGroup>
    </Frame>
  ),
};

export const Collapsed: Story = {
  render: () => (
    <Frame>
      <AgentSidebarGroup
        agent={primaryAgentFixture}
        sessionCount={3}
        defaultOpen={false}
        onNewSession={() => undefined}
      >
        {mockSessions}
      </AgentSidebarGroup>
    </Frame>
  ),
};

export const NoSessions: Story = {
  render: () => (
    <Frame>
      <AgentSidebarGroup agent={primaryAgentFixture} onNewSession={() => undefined} />
    </Frame>
  ),
};

export const DisabledNewSession: Story = {
  render: () => (
    <Frame>
      <AgentSidebarGroup
        agent={primaryAgentFixture}
        newSessionDisabled
        onNewSession={() => undefined}
      />
    </Frame>
  ),
};

export const ExpandCollapseInteraction: Story = {
  tags: ["play-fn"],
  render: () => (
    <Frame>
      <AgentSidebarGroup
        agent={primaryAgentFixture}
        sessionCount={3}
        defaultOpen={false}
        onNewSession={() => undefined}
      >
        {mockSessions}
      </AgentSidebarGroup>
    </Frame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const trigger = await canvas.findByTestId(
      `agent-sidebar-group-trigger-${primaryAgentFixture.name}`
    );
    await expect(trigger).toHaveAttribute("aria-expanded", "false");
    await userEvent.click(trigger);
    await expect(trigger).toHaveAttribute("aria-expanded", "true");
    await expect(canvas.getByTestId("story-session-release-notes")).toBeVisible();
  },
};

export const NewSessionAction: Story = {
  tags: ["play-fn"],
  render: () => {
    return (
      <Frame>
        <AgentSidebarGroup
          agent={primaryAgentFixture}
          onNewSession={name => {
            (window as Window & { __newSessionPayload?: string }).__newSessionPayload = name;
          }}
        />
      </Frame>
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const action = await canvas.findByTestId(
      `agent-sidebar-group-new-session-${primaryAgentFixture.name}`
    );
    await userEvent.click(action);
    await expect((window as Window & { __newSessionPayload?: string }).__newSessionPayload).toBe(
      primaryAgentFixture.name
    );
  },
};
