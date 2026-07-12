import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { expect, within } from "storybook/test";

import {
  storyAgentNames,
  storySessionIds,
  storyWorkspaceIds,
  storyWorkspacePaths,
} from "@/storybook/fintech-scenario";
import { agentFixtures } from "@/systems/agent/mocks";
import { sessionFixtures } from "@/systems/session/mocks";
import type { SessionPayload } from "@/systems/session/types";
import { storybookMswParameters } from "@/storybook/msw";
import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/agent/routes/AgentDetail",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Agent detail route stories rendered through the real router. Covers the four-tab cockpit, Overview/Sessions states, and loading/not-found branches.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const fraudSessions: SessionPayload[] = sessionFixtures.filter(
  session => session.agent_name === storyAgentNames.fraud
);

const fallbackFraudSession: SessionPayload = {
  id: storySessionIds.fraud,
  name: "Payout hold triage",
  agent_name: storyAgentNames.fraud,
  provider: "claude",
  workspace_id: storyWorkspaceIds.risk,
  workspace_path: storyWorkspacePaths.risk,
  state: "active",
  badge: "idle",
  attachable: true,
  available_commands: [],
  created_at: "2026-04-17T16:00:00Z",
  updated_at: "2026-04-17T18:10:00Z",
};

const failureBaseSession = fraudSessions[0] ?? fallbackFraudSession;
function sessionPage(sessions: SessionPayload[]) {
  return {
    sessions,
    page: { has_more: false, limit: 50, total: sessions.length },
  };
}

const fraudAgentRoute = `/agents/${storyAgentNames.fraud}`;
const complianceAgentRoute = `/agents/${storyAgentNames.compliance}`;
const missingAgentRoute = "/agents/ghost-risk-agent";

function AgentWorkspaceSetup() {
  return <StorybookWorkspaceSetup workspaceId={storyWorkspaceIds.risk} />;
}

/**
 * Default agent detail page — Overview cockpit with status pill and toolbar.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters(fraudAgentRoute),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-detail-page")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-overview-tab")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-page-header-status")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-page-toolbar")).resolves.toBeDefined();
    expect(canvas.queryByTestId("agent-info-inspector")).toBeNull();
  },
};

export const InstructionsAgent: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=instructions&file=agent`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-file-agent")).resolves.toBeDefined();
  },
};

export const SoulMissing: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=instructions&file=soul`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-soul-missing")).resolves.toBeDefined();
  },
};

export const SoulEditor: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=instructions&file=soul`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await (await canvas.findByTestId("agent-soul-create")).click();
    await expect(canvas.findByTestId("agent-soul-editor")).resolves.toBeDefined();
  },
};

export const HeartbeatMissing: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=instructions&file=heartbeat`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-heartbeat-missing")).resolves.toBeDefined();
  },
};

export const HeartbeatEditor: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=instructions&file=heartbeat`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await (await canvas.findByTestId("agent-heartbeat-create")).click();
    await expect(canvas.findByTestId("agent-heartbeat-editor")).resolves.toBeDefined();
  },
};

export const Configuration: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=configuration`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-configuration-tab")).resolves.toBeDefined();
  },
};

export const Sessions: Story = {
  args: {},
  parameters: appRouteParameters(`${fraudAgentRoute}?tab=sessions`),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-sessions-tab")).resolves.toBeDefined();
  },
};

export const Diagnostics: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.get("/api/agents/{name}", () =>
          HttpResponse.json({
            agent: {
              ...agentFixtures.find(agent => agent.name === storyAgentNames.fraud)!,
              diagnostics: [
                {
                  error_kind: "frontmatter.invalid",
                  message: "Unknown field in agent definition",
                  path: "AGENT.md:3",
                },
              ],
            },
          })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-diagnostics-banner")).resolves.toBeDefined();
  },
};

export const SessionsError: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/sessions", () =>
          HttpResponse.json({ error: "sessions unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-overview-sessions-notice")).resolves.toBeDefined();
  },
};

/**
 * Agent that has no sessions yet, with an empty state inside the Sessions tab.
 */
export const NoSessions: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(`${complianceAgentRoute}?tab=sessions`),
    ...storybookMswParameters({
      session: [aghApiMock.get("/api/sessions", () => HttpResponse.json(sessionPage([])))],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-sessions-empty")).resolves.toBeDefined();
  },
};

/**
 * Sessions list loading branch while `/api/sessions` is still pending.
 */
export const SessionsLoading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/sessions", async () => {
          await delay("infinite");
          return HttpResponse.json(sessionPage([]));
        }),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-overview-metrics-skeleton")).resolves.toBeDefined();
  },
};

/**
 * Agent detail loading branch: `/api/agents/:name` is in flight while the shell stays mounted.
 */
export const AgentLoading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.get("/api/agents/{name}", async () => {
          await delay("infinite");
          return HttpResponse.json({ agent: agentFixtures[0]! });
        }),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-detail-loading")).resolves.toBeDefined();
  },
};

/**
 * Not-found branch: the agent name does not match anything in the workspace.
 */
export const NotFound: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(missingAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.get("/api/agents/{name}", ({ params }) =>
          HttpResponse.json({ error: `Agent not found: ${String(params.name)}` }, { status: 404 })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-detail-not-found")).resolves.toBeDefined();
  },
};

/**
 * Failed-session branch: at least one session has a populated failure payload, surfacing the FAILED chip.
 */
export const WithFailedSession: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(`${fraudAgentRoute}?tab=sessions`),
    ...storybookMswParameters({
      session: [
        aghApiMock.get("/api/sessions", () =>
          HttpResponse.json(
            sessionPage([
              ...fraudSessions,
              {
                ...failureBaseSession,
                id: "sess_fraud_failed",
                name: "Settlement export retry",
                state: "stopped" as const,
                stop_reason: "agent_crashed" as const,
                failure: {
                  kind: "agent_crashed",
                  summary: "partner settlement export terminated unexpectedly",
                },
                updated_at: "2026-04-17T18:42:00Z",
              },
            ])
          )
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(
      canvas.findByTestId("agent-session-status-sess_fraud_failed")
    ).resolves.toHaveTextContent("FAILED");
  },
};

/**
 * Live agents list returning many agents confirms the sidebar still resolves the active row when the
 * detail route's agent is deeper in the list.
 */
export const ManyAgents: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [aghApiMock.get("/api/agents", () => HttpResponse.json({ agents: agentFixtures }))],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
};
