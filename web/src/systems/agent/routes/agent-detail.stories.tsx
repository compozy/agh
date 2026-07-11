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
          "Agent detail route stories rendered through the real router. Covers the live sessions table, the right-rail MCP panel, the empty/loading branches, and the not-found state.",
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

/**
 * Default agent detail page for the payout-operations agent , sessions table, status pill, stats grid.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters(fraudAgentRoute),
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-page-header-status")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-page-toolbar")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-sessions-table")).resolves.toBeDefined();
  },
};

/**
 * Agent that has no sessions yet , empty state inside the sessions panel + IDLE status pill.
 */
export const NoSessions: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(complianceAgentRoute),
    ...storybookMswParameters({
      session: [aghApiMock.get("/api/sessions", () => HttpResponse.json(sessionPage([])))],
    }),
  },
  render: () => <StorybookWorkspaceSetup />,
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
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-sessions-loading")).resolves.toBeDefined();
  },
};

/**
 * Agent detail loading branch , `/api/agents/:name` is in flight while the shell stays mounted.
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
  render: () => <StorybookWorkspaceSetup />,
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
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-detail-not-found")).resolves.toBeDefined();
  },
};

/**
 * Failed-session branch , at least one session has a populated failure payload, surfacing the FAILED chip.
 */
export const WithFailedSession: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
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
  render: () => <StorybookWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(
      canvas.findByTestId("agent-session-status-sess_fraud_failed")
    ).resolves.toHaveTextContent("FAILED");
  },
};

/**
 * Live agents list returning many agents , confirms the sidebar still resolves the active row when the
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
  render: () => <StorybookWorkspaceSetup />,
};
