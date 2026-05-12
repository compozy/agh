import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { CenteredSurface } from "@/storybook/story-layout";
import { primaryAgentFixture } from "@/systems/agent/mocks";
import type { SessionPayload } from "@/systems/session/types";

import { AgentPageActions, AgentPageStatusPill } from "../agent-page-header";

const idleSessions: SessionPayload[] = [];

const activeSessions: SessionPayload[] = [
  {
    id: "sess-active-1",
    name: "Launch hold triage",
    agent_name: primaryAgentFixture.name,
    provider: primaryAgentFixture.provider,
    workspace_id: "ws-launch",
    workspace_path: "/repos/launch",
    state: "active",
    created_at: "2026-04-17T16:00:00Z",
    updated_at: "2026-04-17T18:10:00Z",
  },
];

const meta: Meta<typeof AgentPageActions> = {
  title: "systems/agent/AgentPageHeader",
  component: AgentPageActions,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Topbar-slot building blocks for the agent detail route. `AgentPageStatusPill` flips between IDLE and ACTIVE based on the active session count; `AgentPageActions` clusters refresh, configure, and primary new-session buttons.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Actions — the right-side toolbar cluster routes push into the topbar
 * `actions` slot. Refresh + configure use `outline` icon-buttons; new-session
 * is the default action button.
 */
export const Actions: Story = {
  args: {
    agent: primaryAgentFixture,
    isRefreshing: false,
    onRefresh: fn(),
    onConfigure: fn(),
    onNewSession: fn(),
    isCreatingSession: false,
    newSessionDisabled: false,
  },
  render: args => (
    <CenteredSurface>
      <div className="flex w-full max-w-3xl items-center justify-end">
        <AgentPageActions {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * RefreshingAndCreating — both async actions are in their busy state:
 * refresh button shows the spinning icon, new-session is `aria-busy`.
 */
export const RefreshingAndCreating: Story = {
  args: {
    agent: primaryAgentFixture,
    isRefreshing: true,
    onRefresh: fn(),
    onConfigure: fn(),
    onNewSession: fn(),
    isCreatingSession: true,
    newSessionDisabled: true,
  },
  render: args => (
    <CenteredSurface>
      <div className="flex w-full max-w-3xl items-center justify-end">
        <AgentPageActions {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * StatusIdle — `AgentPageStatusPill` rendered without any active sessions:
 * neutral mono pill reading `IDLE`.
 */
export const StatusIdle: StoryObj<typeof AgentPageStatusPill> = {
  args: {},
  render: () => (
    <CenteredSurface>
      <AgentPageStatusPill sessions={idleSessions} />
    </CenteredSurface>
  ),
};

/**
 * StatusActive — at least one session is active; the pill flips to
 * the success tone and reads `ACTIVE`.
 */
export const StatusActive: StoryObj<typeof AgentPageStatusPill> = {
  args: {},
  render: () => (
    <CenteredSurface>
      <AgentPageStatusPill sessions={activeSessions} />
    </CenteredSurface>
  ),
};
