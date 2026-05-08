import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { primaryAgentFixture } from "@/systems/agent/mocks";
import { sessionFixtures } from "@/systems/session/mocks";
import type { SessionPayload } from "@/systems/session/types";
import { CenteredSurface } from "@/storybook/story-layout";

import { AgentPageHeader } from "../agent-page-header";

const meta: Meta<typeof AgentPageHeader> = {
  title: "systems/agent/AgentPageHeader",
  component: AgentPageHeader,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" as const },
  },
  args: {
    agent: primaryAgentFixture,
    sessions: sessionFixtures.filter(session => session.agent_name === primaryAgentFixture.name),
    isRefreshing: false,
    isCreatingSession: false,
    newSessionDisabled: false,
    onRefresh: fn(),
    onConfigure: fn(),
    onNewSession: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-(--color-divider) bg-(--color-canvas)">
        {children}
      </div>
    </CenteredSurface>
  );
}

const idleSessions: SessionPayload[] = sessionFixtures
  .filter(session => session.agent_name === primaryAgentFixture.name)
  .map(session => ({ ...session, state: "stopped" as const }));

/**
 * Active agent — at least one session has state === "active", surfaces the ACTIVE chip.
 */
export const Default: Story = {
  render: args => (
    <Frame>
      <AgentPageHeader {...args} />
    </Frame>
  ),
};

/**
 * No live sessions — IDLE chip on the title; the count badge after the name still
 * surfaces the historical session count.
 */
export const Idle: Story = {
  args: { sessions: idleSessions },
  render: args => (
    <Frame>
      <AgentPageHeader {...args} />
    </Frame>
  ),
};

/**
 * Refresh in flight — the Refresh button shows the spinning RefreshCw icon and is disabled.
 */
export const Refreshing: Story = {
  args: { isRefreshing: true },
  render: args => (
    <Frame>
      <AgentPageHeader {...args} />
    </Frame>
  ),
};

/**
 * Workspace not active — `+ New session` is disabled and aria-busy stays false.
 */
export const NoActiveWorkspace: Story = {
  args: { newSessionDisabled: true },
  render: args => (
    <Frame>
      <AgentPageHeader {...args} />
    </Frame>
  ),
};

/**
 * Session creation pending for this agent — `+ New session` reports aria-busy.
 */
export const CreatingSession: Story = {
  args: { isCreatingSession: true },
  render: args => (
    <Frame>
      <AgentPageHeader {...args} />
    </Frame>
  ),
};
