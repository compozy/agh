import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UseAgentDetailPageResult } from "@/hooks/routes/use-agent-detail-page";
import { primaryAgentFixture } from "@/systems/agent/testing";
import { primarySessionFixture } from "@/systems/session/testing";

let childMatches: Array<{ id: string }> = [];
const mockUseAgentDetailPage = vi.fn();

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
    useParams: () => ({ name: "codex-agent" }),
  }),
  Outlet: () => <div data-testid="agent-detail-outlet" />,
  useChildMatches: () => childMatches,
}));

vi.mock("@/hooks/routes/use-agent-detail-page", () => ({
  useAgentDetailPage: (name: string) => mockUseAgentDetailPage(name),
}));

vi.mock("@/systems/agent", () => ({
  AgentInfoPanel: () => <aside data-testid="agent-info-panel" />,
  AgentPageHeader: ({ sessions }: { sessions: unknown[] }) => (
    <header data-testid="agent-page-header">{sessions.length}</header>
  ),
  AgentSessionsList: ({ isLoading, isError }: { isLoading: boolean; isError: boolean }) => (
    <div data-testid="agent-sessions-list" data-loading={isLoading} data-error={isError} />
  ),
  AgentStatsGrid: () => <div data-testid="agent-stats-grid" />,
}));

import { Route } from "../agents.$name";

const AgentDetailRoute = (Route as unknown as { component: () => ReactNode }).component;

function makePage(overrides: Partial<UseAgentDetailPageResult> = {}): UseAgentDetailPageResult {
  return {
    agent: primaryAgentFixture,
    agentLoading: false,
    agentError: null,
    sessions: [primarySessionFixture],
    sessionsLoading: false,
    sessionsError: false,
    isRefreshing: false,
    isCreatingForAgent: false,
    newSessionDisabled: false,
    sessionCreate: {
      openForAgent: vi.fn(),
      isCreating: false,
      pendingAgentName: null,
      hasActiveWorkspace: true,
    },
    onRefresh: vi.fn(),
    onConfigure: vi.fn(),
    onNewSession: vi.fn(),
    onGoHome: vi.fn(),
    ...overrides,
  };
}

describe("Agent detail route", () => {
  beforeEach(() => {
    childMatches = [];
    mockUseAgentDetailPage.mockReset();
    mockUseAgentDetailPage.mockReturnValue(makePage());
  });

  it("renders nested child routes without starting the detail page queries", () => {
    childMatches = [{ id: "/_app/agents/$name/sessions/$id" }];

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-detail-outlet")).toBeInTheDocument();
    expect(mockUseAgentDetailPage).not.toHaveBeenCalled();
  });

  it("does not render authoritative stats while sessions are loading", () => {
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ sessions: [], sessionsLoading: true, sessionsError: false })
    );

    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-sessions-list")).toHaveAttribute("data-loading", "true");
    expect(screen.queryByTestId("agent-stats-grid")).not.toBeInTheDocument();
  });

  it("renders stats after session data resolves", () => {
    render(<AgentDetailRoute />);

    expect(screen.getByTestId("agent-stats-grid")).toBeInTheDocument();
  });
});
