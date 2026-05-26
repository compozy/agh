import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UseAgentDetailPageResult } from "@/hooks/routes/use-agent-detail-page";
import { primaryAgentFixture } from "@/systems/agent/testing";
import type { SessionPayload } from "@/systems/session";
import { primarySessionFixture } from "@/systems/session/testing";

let childMatches: Array<{ id: string }> = [];
const mockUseAgentDetailPage = vi.fn();
const mockUseTopbarSlot = vi.hoisted(() => vi.fn());

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

vi.mock("@agh/ui", async importOriginal => {
  const actual = await importOriginal<typeof import("@agh/ui")>();
  return {
    ...actual,
    useTopbarSlot: mockUseTopbarSlot,
  };
});

vi.mock("@/systems/agent", () => ({
  AgentInfoInspector: () => <aside data-testid="agent-info-inspector" />,
  AgentPageActions: () => <div data-testid="agent-page-actions" />,
  AgentPageStatusPill: ({ sessions }: { sessions: SessionPayload[] }) => (
    <span data-testid="agent-page-status-pill">{sessions.length}</span>
  ),
  AgentSessionsList: ({
    sessions,
    isLoading,
    isError,
    emptyTitle,
    emptyDescription,
  }: {
    sessions: SessionPayload[];
    isLoading: boolean;
    isError: boolean;
    emptyTitle?: ReactNode;
    emptyDescription?: ReactNode;
  }) => (
    <div
      data-testid="agent-sessions-list"
      data-loading={isLoading}
      data-error={isError}
      data-session-ids={sessions.map(session => session.id).join(",")}
    >
      {sessions.map(session => (
        <span key={session.id}>{session.id}</span>
      ))}
      {emptyTitle ? <span data-testid="agent-sessions-empty-title">{emptyTitle}</span> : null}
      {emptyDescription ? (
        <span data-testid="agent-sessions-empty-description">{emptyDescription}</span>
      ) : null}
    </div>
  ),
  AgentStatsGrid: ({ sessions }: { sessions: SessionPayload[] }) => (
    <div
      data-testid="agent-stats-grid"
      data-session-ids={sessions.map(session => session.id).join(",")}
    />
  ),
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
    mockUseTopbarSlot.mockReset();
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

  it("omits memory extraction sessions from agent metrics and list", () => {
    const normalSession = {
      ...primarySessionFixture,
      id: "sess-normal",
      type: "user",
      state: "active",
    } satisfies SessionPayload;
    const legacyDreamSession = {
      ...primarySessionFixture,
      id: "sess-dream",
      name: "Memory extractor",
      type: "dream",
      state: "active",
    } satisfies SessionPayload;
    const spawnedMemorySession = {
      ...primarySessionFixture,
      id: "sess-memory",
      name: "Memory extractor",
      type: "spawned",
      state: "stopped",
      lineage: {
        parent_session_id: "sess-normal",
        root_session_id: "sess-normal",
        spawn_depth: 1,
        spawn_role: "memory-extractor",
        ttl_expires_at: "2026-04-17T20:00:00Z",
        auto_stop_on_parent: true,
        spawn_budget: {
          max_children: 4,
          max_depth: 1,
          ttl_seconds: 7200,
        },
        permission_policy: {
          tools: [],
          skills: [],
          mcp_servers: [],
          workspace_paths: [],
          network_channels: [],
          sandbox_profiles: [],
        },
      },
    } satisfies SessionPayload;
    mockUseAgentDetailPage.mockReturnValue(
      makePage({ sessions: [legacyDreamSession, spawnedMemorySession, normalSession] })
    );

    render(<AgentDetailRoute />);

    const topbarSlot = mockUseTopbarSlot.mock.calls.at(-1)?.[0];
    expect(topbarSlot).toMatchObject({ count: 1 });
    expect(topbarSlot?.tabs.props.sessions.map((session: SessionPayload) => session.id)).toEqual([
      "sess-normal",
    ]);
    expect(screen.getByTestId("agent-stats-grid")).toHaveAttribute(
      "data-session-ids",
      "sess-normal"
    );
    expect(screen.getByTestId("agent-sessions-list")).toHaveAttribute(
      "data-session-ids",
      "sess-normal"
    );
    expect(screen.queryByTestId("agent-session-view-toggle")).not.toBeInTheDocument();
  });
});
