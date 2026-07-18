import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UseAgentDetailPageResult } from "@/hooks/routes/use-agent-detail-page";
import { primaryAgentFixture } from "@/systems/agent/testing";
import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";

let childMatches: Array<{ routeId: string }> = [];
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
    AgentDetailHeader: ({
      metricsUnavailable,
      activeCount,
    }: {
      metricsUnavailable?: boolean;
      activeCount: number;
    }) => (
      <div
        data-testid="agent-detail-header"
        data-metrics-unavailable={metricsUnavailable ? "true" : "false"}
        data-active-count={activeCount}
      />
    ),
    AgentDiagnosticsBanner: () => <div data-testid="agent-diagnostics-banner" />,
    AgentOverviewTab: ({
      sessions,
      sessionsTotal,
      activeSessionsTotal,
      failedSessionsTotal,
      metricsUnavailable,
      metricsLoading,
      sessionsError,
    }: {
      sessions: SessionPayload[];
      sessionsTotal: number;
      activeSessionsTotal: number;
      failedSessionsTotal: number | null;
      metricsUnavailable: boolean;
      metricsLoading?: boolean;
      sessionsError: boolean;
    }) => (
      <div
        data-testid="agent-overview-tab"
        data-session-ids={sessions.map(session => session.id).join(",")}
        data-total={sessionsTotal}
        data-active={activeSessionsTotal}
        data-failed={failedSessionsTotal ?? "null"}
        data-metrics-unavailable={metricsUnavailable ? "true" : "false"}
        data-metrics-loading={metricsLoading ? "true" : "false"}
        data-sessions-error={sessionsError ? "true" : "false"}
      />
    ),
    useAgentInstructionsTab: () => ({
      promptWordCount: "0 words",
      soulMissing: false,
      heartbeatMissing: false,
      soul: {
        resourceKey: '["ws-test","coder","soul"]',
        payload: undefined,
        isLoading: false,
        isError: false,
        history: undefined,
        onValidate: vi.fn(),
        onSave: vi.fn(),
        onRestore: vi.fn(),
        onRetry: vi.fn(),
      },
      heartbeat: {
        resourceKey: '["ws-test","coder","heartbeat"]',
        payload: undefined,
        isLoading: false,
        isError: false,
        history: undefined,
        onValidate: vi.fn(),
        onSave: vi.fn(),
        onRestore: vi.fn(),
        onRetry: vi.fn(),
        onWake: vi.fn(),
        wakePending: false,
        status: undefined,
      },
    }),
    AgentInstructionsTab: () => <div data-testid="agent-instructions-tab" />,
    AgentConfigurationTab: () => <div data-testid="agent-configuration-tab" />,
    AgentSessionsTab: () => <div data-testid="agent-sessions-tab" />,
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
    failedSessionsTotal: 1,
    runtimeSeconds: 3600,
    metricsUnavailable: false,
    metricsLoading: false,
    metricsError: false,
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

  it("Should replace the detail shell for nested session routes", () => {
    childMatches = [{ routeId: "/_app/agents/$name/sessions/$id" }];

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-detail-outlet")).toBeInTheDocument();
    expect(mockUseAgentDetailPage).not.toHaveBeenCalled();
  });

  it("Should keep the detail shell mounted under the settings overlay outlet", () => {
    childMatches = [{ routeId: "/_app/agents/$name/settings" }];

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-detail-page")).toBeInTheDocument();
    expect(screen.getByTestId("agent-detail-header")).toBeInTheDocument();
    expect(screen.getByTestId("agent-detail-outlet")).toBeInTheDocument();
    expect(mockUseAgentDetailPage).toHaveBeenCalled();
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
    expect(screen.getByTestId("agent-detail-header")).toBeInTheDocument();
    expect(screen.getByTestId("agent-detail-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute(
      "data-session-ids",
      "sess-normal"
    );
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-total", "205");
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-active", "7");
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-failed", "1");
    // Route identity (back-to-Agents breadcrumb) now comes from the match
    // chain's `beforeLoad` crumb, not a topbar slot field; the detail route
    // only publishes trailing actions.
    const slot = mockUseTopbarSlot.mock.calls.at(-1)?.[0];
    expect(slot?.actions).toBeTruthy();
  });

  it("Should render loading and not-found states", () => {
    mockUseAgentDetailPage.mockReturnValue(makePage({ agent: undefined, agentLoading: true }));
    render(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-detail-loading")).toBeInTheDocument();

    const view = render(<AgentDetailRoute />);
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ agent: undefined, agentLoading: false, agentError: new Error("missing") })
    );
    view.rerender(<AgentDetailRoute />);
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ agent: undefined, agentLoading: false, agentError: null })
    );
    view.rerender(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-detail-not-found")).toBeInTheDocument();
  });

  it("Should keep overview mounted when sessions error", () => {
    mockUseAgentDetailPage.mockReturnValue(makePage({ sessions: [], sessionsError: true }));
    render(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-overview-tab")).toBeInTheDocument();
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute("data-sessions-error", "true");
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute(
      "data-metrics-unavailable",
      "false"
    );
    expect(screen.getByTestId("agent-detail-header")).toHaveAttribute(
      "data-metrics-unavailable",
      "false"
    );
    expect(screen.getByTestId("agent-tab-sessions")).toHaveTextContent("205");
  });

  it("Should omit the Sessions tab count when catalog metrics are unavailable", () => {
    mockUseAgentDetailPage.mockReturnValue(
      makePage({
        sessionsTotal: 0,
        activeSessionsTotal: 0,
        metricsUnavailable: true,
        metricsError: true,
        failedSessionsTotal: null,
        runtimeSeconds: null,
      })
    );
    render(<AgentDetailRoute />);
    expect(screen.getByTestId("agent-tab-sessions")).toHaveTextContent("Sessions");
    expect(screen.getByTestId("agent-tab-sessions")).not.toHaveTextContent("0");
    expect(screen.getByTestId("agent-detail-header")).toHaveAttribute(
      "data-metrics-unavailable",
      "true"
    );
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute(
      "data-metrics-unavailable",
      "true"
    );
    expect(screen.getByTestId("agent-overview-tab")).toHaveAttribute(
      "data-sessions-error",
      "false"
    );
  });

  it("Should place diagnostics above the body PageHead and outside tab content", () => {
    mockUseAgentDetailPage.mockReturnValue(
      makePage({
        agent: {
          ...primaryAgentFixture,
          diagnostics: [
            {
              error_kind: "schema",
              message: "Invalid permissions value",
              path: "permissions",
            },
          ],
        },
      })
    );
    render(<AgentDetailRoute />);

    const banner = screen.getByTestId("agent-diagnostics-banner");
    const header = screen.getByTestId("agent-detail-header");
    const tabs = screen.getByTestId("agent-detail-tabs");
    const body = screen.getByTestId("agent-detail-body");

    expect(banner.compareDocumentPosition(header) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
    expect(header.compareDocumentPosition(tabs) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
    expect(body.contains(banner)).toBe(false);
  });
});
