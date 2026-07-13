import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { expect, userEvent, waitFor, within } from "storybook/test";

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
    const banner = await canvas.findByTestId("agent-diagnostics-banner");
    const header = await canvas.findByTestId("agent-detail-header");
    expect(banner.compareDocumentPosition(header) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
    expect(canvas.queryByTestId("agent-detail-body")?.contains(banner)).toBe(false);
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
 * Overview with exact catalog metrics at zero (not derived from the sessions page).
 */
export const OverviewZeroSessions: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(complianceAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.get("/api/agents/catalog", ({ request }) => {
          const url = new URL(request.url);
          const name = url.searchParams.get("name")?.trim() ?? "";
          const agent = agentFixtures.find(entry => entry.name === storyAgentNames.compliance)!;
          if (name && name !== agent.name) {
            return HttpResponse.json({
              agents: [],
              facets: { active: 0, categories: [], idle: 0, total: 0 },
              page: { has_more: false, limit: 1, total: 0 },
              sessions_available: true,
            });
          }
          return HttpResponse.json({
            agents: [
              {
                agent,
                sessions: {
                  active: 0,
                  failed: 0,
                  runtime_seconds: 0,
                  total: 0,
                },
              },
            ],
            facets: { active: 0, categories: [], idle: 1, total: 1 },
            page: { has_more: false, limit: 1, total: 1 },
            sessions_available: true,
          });
        }),
      ],
      session: [aghApiMock.get("/api/sessions", () => HttpResponse.json(sessionPage([])))],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.findByTestId("agent-overview-tab")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-stats-grid")).resolves.toBeDefined();
  },
};

async function openDetailRuntimeSelector(canvasElement: HTMLElement) {
  const body = within(canvasElement.ownerDocument.body);
  const trigger = await body.findByTestId("agent-detail-runtime-select");
  const openButton =
    within(trigger).queryByRole("button", { name: /^Open runtime selector$/i }) ??
    within(trigger).queryByRole("button", { name: /^Model:/i }) ??
    within(trigger).getByRole("button", { name: /^Provider:/i });
  await waitFor(() => expect(openButton).toBeEnabled(), { timeout: 15_000 });
  await userEvent.click(openButton);
  await expect(await body.findByTestId("runtime-selector-popup")).toBeInTheDocument();
  return body;
}

async function chooseAlternateRuntimeOption(popupRoot: HTMLElement) {
  const popup = within(popupRoot);
  const option = popup
    .getAllByRole("option")
    .find(
      entry =>
        !entry.hasAttribute("aria-selected") || entry.getAttribute("aria-selected") !== "true"
    );
  if (!option) throw new Error("expected an alternate runtime option");
  await userEvent.click(option);
}

/** Runtime selector open on the detail header. */
export const RuntimeSelectorOpen: Story = {
  args: {},
  parameters: appRouteParameters(fraudAgentRoute),
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    await openDetailRuntimeSelector(canvasElement);
  },
};

/** Runtime mutation stays pending while the header keeps the closed trigger. */
export const RuntimeMutationPending: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.put("/api/agents/{name}", async () => {
          await delay(120_000);
          return HttpResponse.json({ error: "unreachable" }, { status: 500 });
        }),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = await openDetailRuntimeSelector(canvasElement);
    await chooseAlternateRuntimeOption(await body.findByTestId("runtime-selector-popup"));
    await expect(await canvas.findByTestId("agent-detail-runtime-pending")).toBeInTheDocument();
  },
};

/** Runtime CAS conflict surfaces server-refresh guidance without poisoning cache. */
export const RuntimeMutationConflict: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.put("/api/agents/{name}", () =>
          HttpResponse.json({ error: "definition digest conflict" }, { status: 409 })
        ),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const body = await openDetailRuntimeSelector(canvasElement);
    await chooseAlternateRuntimeOption(await body.findByTestId("runtime-selector-popup"));
    await expect(await canvas.findByTestId("agent-detail-runtime-conflict")).toBeInTheDocument();
  },
};

/**
 * Catalog metrics loading while session rows remain independent.
 */
export const MetricsLoading: Story = {
  args: {},
  parameters: {
    ...appRouteParameters(fraudAgentRoute),
    ...storybookMswParameters({
      agent: [
        aghApiMock.get("/api/agents/catalog", async () => {
          await delay("infinite");
          return HttpResponse.json({
            agents: [],
            facets: { active: 0, categories: [], idle: 0, total: 0 },
            page: { has_more: false, limit: 1, total: 0 },
            sessions_available: true,
          });
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
 * Sessions list loading branch while catalog metrics stay independent.
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
    await expect(canvas.findByTestId("agent-stats-grid")).resolves.toBeDefined();
    await expect(canvas.findByTestId("agent-overview-live-sessions")).resolves.toBeDefined();
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
          await delay(120_000);
          return HttpResponse.json({ agent: agentFixtures[0]! });
        }),
      ],
    }),
  },
  render: () => <AgentWorkspaceSetup />,
  tags: ["play-fn"],
  play: async ({ canvasElement }) => {
    const body = within(canvasElement.ownerDocument.body);
    await expect(await body.findByTestId("agent-detail-loading")).toBeInTheDocument();
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
