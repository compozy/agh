import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UseAgentDetailPageResult } from "@/hooks/routes/use-agent-detail-page";
import { primaryAgentFixture } from "@/systems/agent/testing";
import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";

let childMatches: Array<{ id: string }> = [];
let routeSearch: { tab: string; file: string; filter: string } = {
  tab: "overview",
  file: "agent",
  filter: "all",
};
const mockUseAgentDetailPage = vi.fn();
const mockUseTopbarSlot = vi.hoisted(() => vi.fn());
const mockUseActiveWorkspace = vi.hoisted(() => vi.fn(() => ({ activeWorkspaceId: "ws_test" })));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => unknown;
    }) => ({
      component: opts.component,
      validateSearch: opts.validateSearch,
      useParams: () => ({ name: "codex-agent" }),
      useSearch: () => routeSearch,
    }),
  Outlet: () => <div data-testid="agent-detail-outlet" />,
  useChildMatches: () => childMatches,
}));

vi.mock("@/hooks/routes/use-agent-detail-page", () => ({
  useAgentDetailPage: (name: string, search: unknown) => mockUseAgentDetailPage(name, search),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => mockUseActiveWorkspace(),
}));

vi.mock("@agh/ui", async importOriginal => {
  const actual = await importOriginal<typeof import("@agh/ui")>();
  return {
    ...actual,
    useTopbarSlot: mockUseTopbarSlot,
  };
});

vi.mock("@/systems/agent", async importOriginal => {
  const actual = await importOriginal<typeof import("@/systems/agent")>();
  return {
    ...actual,
    AgentPageActions: () => <div data-testid="agent-page-actions" />,
    AgentPageStatusPill: ({ activeCount }: { activeCount: number }) => (
      <span data-testid="agent-page-status-pill">{activeCount}</span>
    ),
    AgentPageMeta: () => <span data-testid="agent-page-meta" />,
    AgentDiagnosticsBanner: () => <div data-testid="agent-diagnostics-banner" />,
    AgentOverviewTab: ({
      sessions,
      sessionsTotal,
      activeSessionsTotal,
      resumableSessionsTotal,
    }: {
      sessions: SessionPayload[];
      sessionsTotal: number;
      activeSessionsTotal: number;
      resumableSessionsTotal: number;
    }) => (
      <div
        data-testid="agent-overview-tab"
        data-session-ids={sessions.map(session => session.id).join(",")}
        data-total={sessionsTotal}
        data-active={activeSessionsTotal}
        data-resumable={resumableSessionsTotal}
      />
    ),
    AgentInstructionsTab: ({ file }: { file: string }) => (
      <div data-testid="agent-instructions-tab" data-file={file} />
    ),
    AgentConfigurationTab: () => <div data-testid="agent-configuration-tab" />,
    AgentSessionsTab: ({
      sessions,
      filter,
      total,
      active,
      resumable,
      hasMore,
    }: {
      sessions: SessionPayload[];
      filter: string;
      total: number;
      active: number;
      resumable: number;
      hasMore: boolean;
    }) => (
      <div
        data-testid="agent-sessions-tab"
        data-session-ids={sessions.map(session => session.id).join(",")}
        data-filter={filter}
        data-total={total}
        data-active={active}
        data-resumable={resumable}
        data-has-more={hasMore}
      />
    ),
    validateAgentDetailSearch: actual.validateAgentDetailSearch,
  };
});

import { validateAgentDetailSearch } from "@/systems/agent";
import { Route } from "../agents.$name";

const AgentDetailRoute = (Route as unknown as { component: () => ReactNode }).component;

function makePage(overrides: Partial<UseAgentDetailPageResult> = {}): UseAgentDetailPageResult {
  return {
    agent: primaryAgentFixture,
    agentLoading: false,
    agentError: null,
    sessions: [primarySessionFixture],
    sessionsTotal: 205,
    activeSessionsTotal: 7,
    resumableSessionsTotal: 13,
    lastSessionActivityAt: "2026-07-11T12:00:00Z",
    hasMoreSessions: true,
    isLoadingMoreSessions: false,
    onLoadMoreSessions: vi.fn(),
    sessionsLoading: false,
    sessionsError: false,
    search: { tab: "overview", file: "agent", filter: "all" },
    setTab: vi.fn(),
    setFile: vi.fn(),
    setFilter: vi.fn(),
    isCreatingForAgent: false,
    newSessionDisabled: false,
    onNewSession: vi.fn(),
    onEditSettings: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    onBackToAgents: vi.fn(),
    deleteDialog: null,
    ...overrides,
  };
}

describe("Agent detail route", () => {
  beforeEach(() => {
    childMatches = [];
    routeSearch = { tab: "overview", file: "agent", filter: "all" };
    mockUseAgentDetailPage.mockReset();
    mockUseTopbarSlot.mockReset();
    mockUseAgentDetailPage.mockReturnValue(makePage());
  });

  it("Should validate search defaults and tab/file/filter values", () => {
    expect(validateAgentDetailSearch({})).toEqual({
      tab: "overview",
      file: "agent",
      filter: "all",
    });
    expect(
      validateAgentDetailSearch({ tab: "instructions", file: "soul", filter: "active" })
    ).toEqual({
      tab: "instructions",
      file: "soul",
      filter: "active",
    });
    expect(validateAgentDetailSearch({ tab: "nope", file: "x", filter: "y" })).toEqual({
      tab: "overview",
      file: "agent",
      filter: "all",
    });
  });

  it("Should render nested child routes without starting the detail page queries", () => {
    childMatches = [{ id: "/_app/agents/$name/sessions/$id" }];

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-detail-outlet")).toBeInTheDocument();
    expect(mockUseAgentDetailPage).not.toHaveBeenCalled();
  });

  it("Should render the tabbed detail surface with exact server-owned session totals", () => {
    const normalSession = {
      ...primarySessionFixture,
      id: "sess-normal",
      type: "user",
      state: "active",
    } satisfies SessionPayload;
    mockUseAgentDetailPage.mockReturnValue(makePage({ sessions: [normalSession] }));

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-detail-page")).toBeInTheDocument();
    expect(screen.getByTestId("agent-detail-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute(
      "data-session-ids",
      "sess-normal"
    );
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-total", "205");
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-active", "7");
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-resumable", "13");
    const slot = mockUseTopbarSlot.mock.calls.at(-1)?.[0];
    render(slot.tabs);
    expect(screen.getByTestId("agent-page-status-pill")).toHaveTextContent("7");
    expect(slot?.backLabel).toBe("Agents");
    expect(slot?.actions).toBeTruthy();
  });

  it("Should associate the active detail tab with its tabpanel", () => {
    render(<AgentDetailRoute />);

    const tab = screen.getByRole("tab", { name: "Overview" });
    const panel = screen.getByRole("tabpanel");

    expect(tab).toHaveAttribute("aria-controls", panel.id);
    expect(panel).toHaveAttribute("aria-labelledby", tab.id);
    expect(panel).toContainElement(screen.getByTestId("agent-overview-tab"));
  });

  it("Should render geometry skeleton while the agent is loading", () => {
    mockUseAgentDetailPage.mockReturnValue(makePage({ agent: undefined, agentLoading: true }));
    render(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-detail-loading")).toBeInTheDocument();
  });

  it("Should restore instructions file and sessions filter deep links exactly", () => {
    routeSearch = { tab: "instructions", file: "soul", filter: "all" };
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ search: { tab: "instructions", file: "soul", filter: "all" } })
    );
    const view = render(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-instructions-tab")).toHaveAttribute("data-file", "soul");

    routeSearch = { tab: "sessions", file: "agent", filter: "failed" };
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ search: { tab: "sessions", file: "agent", filter: "failed" } })
    );
    view.rerender(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-sessions-tab")).toHaveAttribute("data-filter", "failed");
    expect(screen.getByTestId("agent-sessions-tab")).toHaveAttribute("data-total", "205");
    expect(screen.getByTestId("agent-sessions-tab")).toHaveAttribute("data-has-more", "true");
  });

  it("Should omit the status pill instead of claiming Idle when sessions are unavailable", () => {
    mockUseAgentDetailPage.mockReturnValue(makePage({ sessions: [], sessionsError: true }));
    render(<AgentDetailRoute />);
    const slot = mockUseTopbarSlot.mock.calls.at(-1)?.[0];
    render(slot.tabs);
    expect(screen.queryByTestId("agent-page-status-pill")).toBeNull();
    expect(screen.getByTestId("agent-page-meta")).toBeVisible();
  });
});
