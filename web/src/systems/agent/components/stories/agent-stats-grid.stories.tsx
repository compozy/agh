import type { Meta, StoryObj } from "@storybook/react-vite";

import { sessionFixtures } from "@/systems/session/mocks";
import type { SessionPayload } from "@/systems/session/types";
import { CenteredSurface } from "@/storybook/story-layout";

import { AgentStatsGrid } from "../agent-stats-grid";

const meta: Meta<typeof AgentStatsGrid> = {
  title: "systems/agent/AgentStatsGrid",
  component: AgentStatsGrid,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-4xl">{children}</div>
    </CenteredSurface>
  );
}

const codexSessions: SessionPayload[] = sessionFixtures.filter(
  session => session.agent_name === "codex-agent"
);

const richSessions: SessionPayload[] = codexSessions.map(session => ({
  ...session,
  activity: {
    elapsed_seconds: 1820,
    idle_seconds: 30,
    iteration_current: 12,
    iteration_max: 30,
    last_activity_at: "2026-04-17T18:10:00Z",
    last_activity_kind: "tool",
    last_progress_at: "2026-04-17T18:10:00Z",
  },
}));

const failedSessions: SessionPayload[] = [
  ...richSessions,
  {
    ...richSessions[0],
    id: "sess-failure",
    state: "stopped",
    stop_reason: "agent_crashed",
    failure: { kind: "agent_crashed", summary: "broker disconnect" },
    activity: {
      elapsed_seconds: 412,
      idle_seconds: 0,
      iteration_current: 2,
      iteration_max: 6,
      last_activity_at: "2026-04-17T18:55:00Z",
      last_activity_kind: "tool",
      last_progress_at: "2026-04-17T18:55:00Z",
    },
  },
];

/**
 * Default — agent has one active session; runtime accumulates from `activity.elapsed_seconds`.
 */
export const Default: Story = {
  args: { sessions: richSessions },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};

/**
 * Failure tone surfaces when at least one session has a populated failure payload.
 */
export const WithFailure: Story = {
  args: { sessions: failedSessions },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};

/**
 * Empty state — no sessions yet, every metric falls back to the em-dash placeholder.
 */
export const Empty: Story = {
  args: { sessions: [] },
  render: args => (
    <Frame>
      <AgentStatsGrid {...args} />
    </Frame>
  ),
};
