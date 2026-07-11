import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, HttpResponse, type HttpHandler } from "msw";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { aghApiMock } from "@/storybook/openapi-msw";
import { FIXTURE_AGENT_DEFINITION_DIGEST } from "@/systems/agent/mocks";
import type { AgentPayload } from "@/systems/agent/types";
import type { SessionPayload } from "@/systems/session";
import { createMswFetch } from "@/test/msw-fetch";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeBeforeLoad, routeComponent } from "@/test/route-options";

function render(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
      mutations: { retry: false },
    },
  });
  return renderWithTopbar(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>, {
    title: "Agents",
  });
}

function agent(overrides: Partial<AgentPayload> & Pick<AgentPayload, "name">): AgentPayload {
  return {
    provider: "claude",
    prompt: "test",
    origin: "global",
    definition_digest: FIXTURE_AGENT_DEFINITION_DIGEST,
    ...overrides,
  };
}

function session(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    id: "sess-1",
    agent_name: "coder",
    provider: "claude",
    workspace_id: "ws_test",
    workspace_path: "/workspace",
    state: "stopped",
    badge: "idle",
    attachable: true,
    available_commands: [],
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T01:00:00Z",
    ...overrides,
  };
}

let mockAgents: AgentPayload[] = [];
let agentsDelayMs = 0;
let agentsShouldError = false;
let agentsRequestCount = 0;

let mockSessions: SessionPayload[] = [];
let sessionsDelayMs = 0;
let sessionsShouldError = false;
let sessionsRequestCount = 0;

const mockOpenCreate = vi.fn();
let mockActiveWorkspaceId: string | null = "ws_test";

const handlers: HttpHandler[] = [
  aghApiMock.get("/api/agents", async () => {
    agentsRequestCount += 1;
    if (agentsDelayMs > 0) await delay(agentsDelayMs);
    if (agentsShouldError) {
      return HttpResponse.json({ error: "agents unavailable" }, { status: 500 });
    }
    return HttpResponse.json({ agents: mockAgents });
  }),
  aghApiMock.get("/api/sessions", async () => {
    sessionsRequestCount += 1;
    if (sessionsDelayMs > 0) await delay(sessionsDelayMs);
    if (sessionsShouldError) {
      return HttpResponse.json({ error: "sessions unavailable" }, { status: 500 });
    }
    return HttpResponse.json({
      sessions: mockSessions,
      page: { has_more: false, limit: 50, total: mockSessions.length },
    });
  }),
];

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
  childMatches: [] as unknown[],
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => Record<string, unknown>)
    | undefined,
}));

function getValidatedSearch() {
  return routerState.validateSearch
    ? routerState.validateSearch(routerState.searchParams)
    : routerState.searchParams;
}

vi.mock("@tanstack/react-router", () => ({
  Outlet: () => <div data-testid="agents-outlet" />,
  createFileRoute:
    () =>
    (opts: {
      component: () => React.ReactNode;
      beforeLoad?: () => unknown;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validateSearch = opts.validateSearch;

      return {
        beforeLoad: opts.beforeLoad,
        component: opts.component,
        useSearch: () => {
          const [search, setSearch] = React.useState(getValidatedSearch());

          React.useEffect(() => {
            routerState.searchListeners.add(setSearch);
            return () => {
              routerState.searchListeners.delete(setSearch);
            };
          }, []);

          return search;
        },
      };
    },
  useChildMatches: () => routerState.childMatches,
  useNavigate:
    () =>
    async (options: {
      params?: Record<string, unknown>;
      search?:
        | Record<string, unknown>
        | ((current: Record<string, unknown>) => Record<string, unknown>);
      to: string;
    }) => {
      if (typeof options.search === "function") {
        routerState.searchParams = options.search(getValidatedSearch());
      } else if (options.search) {
        routerState.searchParams = options.search;
      }

      const nextSearch = getValidatedSearch();
      for (const listener of routerState.searchListeners) {
        listener(nextSearch);
      }

      routerState.navigateMock(options);
    },
  Link: ({ to, params, children, ...props }: Record<string, unknown>) => {
    let href = typeof to === "string" ? to : "#";
    if (params && typeof params === "object") {
      for (const [key, value] of Object.entries(params)) {
        href = href.replace(`$${key}`, encodeURIComponent(String(value)));
      }
    }
    return (
      <a
        href={href}
        data-params={JSON.stringify(params)}
        {...(props as Record<string, unknown>)}
        onClick={event => event.preventDefault()}
      >
        {children as React.ReactNode}
      </a>
    );
  },
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: [
      {
        id: "ws_test",
        root_dir: "/workspace",
        add_dirs: [],
        name: "test-workspace",
        created_at: "2026-04-03T12:00:00Z",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
    hasWorkspaces: true,
    activeWorkspace: mockActiveWorkspaceId
      ? {
          id: "ws_test",
          root_dir: "/workspace",
          add_dirs: [],
          name: "test-workspace",
          created_at: "2026-04-03T12:00:00Z",
          updated_at: "2026-04-03T12:00:00Z",
        }
      : undefined,
    activeWorkspaceId: mockActiveWorkspaceId,
    setActiveWorkspaceId: vi.fn(),
    clearActiveWorkspaceSelection: vi.fn(),
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/agent/hooks/use-agent-create-host", () => ({
  useAgentCreateHost: () => ({ openDialog: mockOpenCreate }),
}));

vi.mock("@/systems/agent/components/agent-create-host", () => ({
  AgentCreateHostProvider: ({ children }: { children: React.ReactNode }) => children,
}));

import { Route } from "../agents";

const AgentsPage = routeComponent(Route);

describe("Agents fleet route", () => {
  beforeEach(() => {
    mockAgents = [
      agent({
        name: "release-captain",
        category_path: ["Engineering", "Release"],
        provider: "anthropic",
        model: "claude-sonnet-4-5",
      }),
      agent({
        name: "code-reviewer",
        category_path: ["Engineering"],
        provider: "openai",
      }),
      agent({ name: "triage-bot", provider: "openai", origin: "workspace" }),
    ];
    agentsDelayMs = 0;
    agentsShouldError = false;
    agentsRequestCount = 0;
    mockSessions = [
      session({ id: "a", agent_name: "release-captain", state: "active", badge: "running" }),
      session({ id: "b", agent_name: "release-captain", state: "stopped" }),
      session({ id: "c", agent_name: "code-reviewer", state: "stopped" }),
    ];
    sessionsDelayMs = 0;
    sessionsShouldError = false;
    sessionsRequestCount = 0;
    mockActiveWorkspaceId = "ws_test";
    mockOpenCreate.mockReset();
    routerState.navigateMock.mockReset();
    routerState.searchParams = {};
    routerState.childMatches = [];
    routerState.searchListeners.clear();
    vi.stubGlobal(
      "fetch",
      createMswFetch(() => handlers)
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("Should register the Agents topbar contract", () => {
    expect(routeBeforeLoad(Route)()).toMatchObject({ topbar: { title: "Agents" } });
  });

  it("Should ask for a workspace before querying the fleet", () => {
    mockActiveWorkspaceId = null;
    render(<AgentsPage />);
    expect(screen.getByTestId("agents-no-workspace")).toHaveTextContent("No workspace selected");
    expect(agentsRequestCount).toBe(0);
    expect(sessionsRequestCount).toBe(0);
  });

  it("Should render skeleton then loaded rows and navigate on row click", async () => {
    const user = userEvent.setup();
    agentsDelayMs = 30;
    render(<AgentsPage />);
    expect(screen.getByTestId("agent-fleet-loading")).toBeInTheDocument();
    expect(screen.getByTestId("agent-fleet-toolbar")).toBeInTheDocument();

    const list = await screen.findByTestId("agent-fleet-list");
    const row = within(list).getByTestId("agent-fleet-row-code-reviewer");
    const link = within(row).getByTestId("agent-fleet-row-link-code-reviewer");
    expect(link).toHaveAttribute("aria-label", "code-reviewer, Idle, 0 of 1 sessions active");
    expect(within(row).getAllByRole("link")).toHaveLength(1);
    expect(within(row).queryAllByRole("button")).toHaveLength(0);
    expect(link).toContainElement(screen.getByTestId("agent-fleet-sessions-code-reviewer"));
    await user.click(screen.getByTestId("agent-fleet-status-code-reviewer"));
    expect(link).toHaveAttribute("href", "/agents/code-reviewer");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ name: "code-reviewer" }));
  });

  it("Should AND-compose search and filters through URL state", async () => {
    const user = userEvent.setup();
    render(<AgentsPage />);
    await screen.findByTestId("agent-fleet-list");
    expect(screen.getByTestId("topbar-count")).toHaveTextContent("3");

    await user.type(screen.getByTestId("agent-fleet-search"), "release");
    await waitFor(() => {
      expect(routerState.navigateMock).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(getValidatedSearch()).toMatchObject({ q: "release" });
    });

    routerState.searchParams = { q: "release", category: "Engineering / Release", status: "idle" };
    act(() => {
      for (const listener of routerState.searchListeners) {
        listener(getValidatedSearch());
      }
    });

    await waitFor(() => {
      expect(screen.queryByTestId("agent-fleet-row-release-captain")).not.toBeInTheDocument();
      expect(screen.getByTestId("agent-fleet-filtered-empty")).toBeInTheDocument();
    });
    expect(screen.getByTestId("topbar-count")).toHaveTextContent("3");
  });

  it("Should keep rows and omit status when sessions fail, including with a status URL filter", async () => {
    sessionsShouldError = true;
    routerState.searchParams = { status: "active" };
    render(<AgentsPage />);

    await screen.findByTestId("agent-fleet-list");
    expect(screen.getByTestId("agent-fleet-sessions-notice")).toHaveTextContent(
      "Session status unavailable"
    );
    expect(screen.getByTestId("agent-fleet-sessions-release-captain")).toHaveTextContent("—");
    expect(screen.queryByTestId("agent-fleet-status-release-captain")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-fleet-row-link-release-captain")).toHaveAttribute(
      "aria-label",
      "release-captain, session status unavailable"
    );
    expect(screen.getAllByTestId(/agent-fleet-row-link-/)).toHaveLength(3);
    expect(sessionsRequestCount).toBe(1);
  });

  it("Should keep skeleton geometry until session-derived signals load", async () => {
    sessionsDelayMs = 30;
    render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-fleet-status-code-reviewer")).not.toBeInTheDocument();
    expect(await screen.findByTestId("agent-fleet-list")).toBeInTheDocument();
  });

  it("Should distinguish first-run empty from filtered empty", async () => {
    const user = userEvent.setup();
    mockAgents = [];
    const { unmount } = render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-empty")).toBeInTheDocument();
    expect(screen.getByText("No agents yet")).toBeInTheDocument();
    expect(screen.queryByTestId("topbar-count")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agents-topbar-create")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("agent-fleet-empty-create"));
    expect(mockOpenCreate).toHaveBeenCalledOnce();
    unmount();

    mockAgents = [agent({ name: "coder" })];
    mockOpenCreate.mockReset();
    routerState.searchParams = { q: "missing" };
    render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-filtered-empty")).toBeInTheDocument();
    expect(screen.getByText("No agents match")).toBeInTheDocument();
    await user.click(screen.getByTestId("agent-fleet-clear-filters"));
    expect(routerState.navigateMock).toHaveBeenCalled();
  });

  it("Should name the agents failure and retry", async () => {
    const user = userEvent.setup();
    mockAgents = [];
    agentsShouldError = true;
    render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-error")).toBeInTheDocument();
    expect(screen.getByText("Couldn't load agents")).toBeInTheDocument();
    expect(screen.getByText("agents unavailable")).toBeInTheDocument();
    agentsShouldError = false;
    mockAgents = [agent({ name: "recovered" })];
    await user.click(screen.getByTestId("agent-fleet-error-retry"));
    expect(await screen.findByTestId("agent-fleet-list")).toBeInTheDocument();
    expect(agentsRequestCount).toBe(2);
  });

  it("Should show Invalid diagnostics while keeping the row navigable", async () => {
    mockAgents = [
      agent({
        name: "broken",
        diagnostics: [{ error_kind: "parse", message: "bad", path: "AGENT.md" }],
      }),
    ];
    mockSessions = [];
    render(<AgentsPage />);
    await screen.findByTestId("agent-fleet-list");
    expect(screen.getByTestId("agent-fleet-invalid-broken")).toHaveTextContent("Invalid");
    expect(screen.getByTestId("agent-fleet-row-link-broken")).toHaveAttribute(
      "href",
      "/agents/broken"
    );
  });

  it("Should open create from the topbar CTA", async () => {
    const user = userEvent.setup();
    render(<AgentsPage />);
    await screen.findByTestId("agent-fleet-list");
    const actions = screen.getByTestId("agents-topbar-actions");
    await user.click(within(actions).getByTestId("agents-topbar-create"));
    expect(mockOpenCreate).toHaveBeenCalledOnce();
  });

  it("Should outlet child routes without rendering the list", () => {
    routerState.childMatches = [{ id: "/_app/agents/$name" }];
    render(<AgentsPage />);
    expect(screen.getByTestId("agents-outlet")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-fleet-page")).not.toBeInTheDocument();
  });

  it("Should focus search when / is pressed outside editable targets", async () => {
    const user = userEvent.setup();
    render(<AgentsPage />);
    const search = await screen.findByTestId("agent-fleet-search");
    expect(search).not.toHaveFocus();
    await user.keyboard("/");
    expect(search).toHaveFocus();
  });

  it("Should restore the search draft from URL navigation and cancel a stale debounce", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { q: "triage" };
    render(<AgentsPage />);
    const search = await screen.findByTestId("agent-fleet-search");
    expect(search).toHaveValue("triage");

    await user.clear(search);
    await user.type(search, "stale");
    routerState.searchParams = { q: "release" };
    act(() => {
      for (const listener of routerState.searchListeners) {
        listener(getValidatedSearch());
      }
    });

    await waitFor(() => expect(search).toHaveValue("release"));
    await new Promise(resolve => setTimeout(resolve, 250));
    expect(getValidatedSearch()).toEqual({ q: "release", category: undefined, status: undefined });
  });
});
