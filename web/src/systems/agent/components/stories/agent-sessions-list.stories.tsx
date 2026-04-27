import type { ReactNode } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { sessionFixtures } from "@/systems/session/mocks";
import type { SessionPayload } from "@/systems/session/types";
import { PanelSurface } from "@/storybook/story-layout";

import { AgentSessionsList } from "../agent-sessions-list";

const codexSessions: SessionPayload[] = sessionFixtures.filter(
  session => session.agent_name === "codex-agent"
);

const fallbackCodexSession: SessionPayload = {
  id: "sess-storybook-base",
  name: "Storybook rollout",
  agent_name: "codex-agent",
  provider: "codex",
  workspace_id: "ws_storybook",
  workspace_path: "/workspaces/agh2",
  state: "active",
  created_at: "2026-04-17T16:00:00Z",
  updated_at: "2026-04-17T18:10:00Z",
};

const failureBaseSession = codexSessions[0] ?? fallbackCodexSession;

const codexSessionsWithFailure: SessionPayload[] = [
  ...codexSessions,
  {
    ...failureBaseSession,
    id: "sess-storybook-failed",
    name: "Failed verification",
    state: "stopped",
    stop_reason: "agent_crashed",
    failure: { kind: "agent_crashed", summary: "agent terminated unexpectedly" },
    activity: {
      elapsed_seconds: 142,
      idle_seconds: 0,
      iteration_current: 4,
      iteration_max: 6,
      last_activity_at: "2026-04-17T18:42:00Z",
      last_activity_kind: "tool",
      last_progress_at: "2026-04-17T18:42:00Z",
    },
    updated_at: "2026-04-17T18:42:00Z",
  },
];

const meta: Meta<typeof AgentSessionsList> = {
  title: "systems/agent/AgentSessionsList",
  component: AgentSessionsList,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" as const },
  },
  args: {
    agentName: "codex-agent",
    sessions: codexSessions,
    isLoading: false,
    isError: false,
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

interface FrameProps {
  children: ReactNode;
}

function Frame({ children }: FrameProps) {
  return (
    <PanelSurface>
      <div className="flex w-full flex-col gap-4 px-6 py-5">{children}</div>
    </PanelSurface>
  );
}

/**
 * Default — table of sessions sorted by last activity with status chips and metadata.
 */
export const Default: Story = {
  args: {},
  render: args => (
    <Frame>
      <AgentSessionsList {...args} />
    </Frame>
  ),
};

/**
 * One session has a failure payload — surfaces the FAILED chip via the danger tone.
 */
export const WithFailure: Story = {
  args: { sessions: codexSessionsWithFailure },
  render: args => (
    <Frame>
      <AgentSessionsList {...args} />
    </Frame>
  ),
};

/**
 * Empty state — no sessions for the agent yet.
 */
export const Empty: Story = {
  args: { sessions: [] },
  render: args => (
    <Frame>
      <AgentSessionsList {...args} />
    </Frame>
  ),
};

/**
 * Loading skeleton while sessions are being fetched.
 */
export const Loading: Story = {
  args: { isLoading: true, sessions: [] },
  render: args => (
    <Frame>
      <AgentSessionsList {...args} />
    </Frame>
  ),
};

/**
 * Error fallback when the sessions query rejects.
 */
export const Error: Story = {
  args: { isError: true, sessions: [] },
  render: args => (
    <Frame>
      <AgentSessionsList {...args} />
    </Frame>
  ),
};
