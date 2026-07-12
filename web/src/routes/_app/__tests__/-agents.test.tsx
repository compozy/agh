import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, HttpResponse, type HttpHandler } from "msw";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { aghApiMock } from "@/storybook/openapi-msw";
import { FIXTURE_AGENT_DEFINITION_DIGEST } from "@/systems/agent/mocks";
import type { AgentPayload } from "@/systems/agent/types";
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

interface CatalogSession {
  agent_name: string;
  state: "active" | "stopped";
}

function session(overrides: Partial<CatalogSession> = {}): CatalogSession {
  return {
    agent_name: "coder",
    state: "stopped",
    ...overrides,
  };
}

let mockAgents: AgentPayload[] = [];
let agentsDelayMs = 0;
let agentsShouldError = false;
let agentsRequestCount = 0;

let mockSessions: CatalogSession[] = [];
let sessionsAvailable = true;

const mockOpenCreate = vi.fn();
let mockActiveWorkspaceId: string | null = "ws_test";

const handlers: HttpHandler[] = [
  aghApiMock.get("/api/agents/catalog", async ({ request }) => {
    agentsRequestCount += 1;
    if (agentsDelayMs > 0) await delay(agentsDelayMs);
    if (agentsShouldError) {
      return HttpResponse.json({ error: "agents unavailable" }, { status: 500 });
    }
    const url = new URL(request.url);
    const query = url.searchParams.get("q")?.trim().toLowerCase() ?? "";
    const category = url.searchParams.get("category")?.trim() ?? "";
    const status = url.searchParams.get("status")?.trim() ?? "";
    const requestedLimit = Number(url.searchParams.get("limit") ?? "50");
    const start = Number(url.searchParams.get("cursor") ?? "0");
    const counts = new Map<string, { active: number; total: number }>();
    for (const item of mockSessions) {
      const current = counts.get(item.agent_name) ?? { active: 0, total: 0 };
      current.total += 1;
      if (item.state === "active") current.active += 1;
      counts.set(item.agent_name, current);
    }
    const filtered = mockAgents.filter(item => {
      const itemCategory = item.category_path?.join(" / ") ?? "";
      const itemCounts = counts.get(item.name) ?? { active: 0, total: 0 };
      if (
        query &&
        !item.name.toLowerCase().includes(query) &&
        !itemCategory.toLowerCase().includes(query)
      ) {
        return false;
      }
      if (category && itemCategory !== category) return false;
      if (sessionsAvailable && status === "active" && itemCounts.active === 0) return false;
      if (sessionsAvailable && status === "idle" && itemCounts.active > 0) return false;
      return true;
    });
    const pageAgents = filtered.slice(start, start + requestedLimit);
    const next = start + pageAgents.length;
    return HttpResponse.json({
      agents: pageAgents.map(item => ({
        agent: item,
        ...(sessionsAvailable
          ? { sessions: counts.get(item.name) ?? { active: 0, total: 0 } }
          : {}),
      })),
      facets: {
        categories: [...new Set(mockAgents.flatMap(item => item.category_path?.join(" / ") ?? []))],
        total: mockAgents.length,
        active: sessionsAvailable
          ? mockAgents.filter(item => (counts.get(item.name)?.active ?? 0) > 0).length
          : 0,
        idle: sessionsAvailable
          ? mockAgents.filter(item => (counts.get(item.name)?.active ?? 0) === 0).length
          : 0,
      },
      page: {
        has_more: next < filtered.length,
        limit: requestedLimit,
        total: filtered.length,
        ...(next < filtered.length ? { next_cursor: String(next) } : {}),
      },
      sessions_available: sessionsAvailable,
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
  useAgentCreateHost: () => ({ openDialog: mockOpenCreate, openForDuplicate: vi.fn() }),
}));

vi.mock("@/systems/agent/components/agent-create-host", () => ({
  AgentCreateHostProvider: ({ children }: { children: React.ReactNode }) => children,
}));

const mockOpenNewSession = vi.fn();

vi.mock("@/systems/session", async importOriginal => {
  const actual = await importOriginal<typeof import("@/systems/session")>();
  return {
    ...actual,
    useSessionCreate: () => ({
      openForAgent: mockOpenNewSession,
      isCreating: false,
      pendingAgentName: null,
      hasActiveWorkspace: true,
    }),
  };
});

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
      session({ agent_name: "release-captain", state: "active" }),
      session({ agent_name: "release-captain", state: "stopped" }),
      session({ agent_name: "code-reviewer", state: "stopped" }),
    ];
    sessionsAvailable = true;
    mockActiveWorkspaceId = "ws_test";
    mockOpenCreate.mockReset();
    mockOpenNewSession.mockReset();
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

  it("Should ask for a workspace before querying the fleet", async () => {
    mockActiveWorkspaceId = null;
    render(<AgentsPage />);
    expect(await screen.findByTestId("agents-no-workspace")).toHaveTextContent(
      "No workspace selected"
    );
    await waitFor(() => {
      expect(agentsRequestCount).toBe(0);
    });
  });

  it("Should render skeleton then loaded rows with sibling new-session action", async () => {
    const user = userEvent.setup();
    agentsDelayMs = 30;
    render(<AgentsPage />);
    expect(screen.getByTestId("agent-fleet-loading")).toBeInTheDocument();
    expect(screen.getByTestId("agent-fleet-toolbar")).toBeInTheDocument();
    expect(screen.queryByTestId("listing-view-toggle")).not.toBeInTheDocument();

    const list = await screen.findByTestId("agent-fleet-list");
    expect(screen.getByTestId("agents-page-head")).toHaveTextContent("Operate");
    expect(screen.getByTestId("listing-view-toggle")).toBeInTheDocument();
    const row = within(list).getByTestId("agent-fleet-row-code-reviewer");
    const link = within(row).getByTestId("agent-fleet-row-link-code-reviewer");
    expect(link).toHaveAttribute("aria-label", "code-reviewer, Idle, 0 of 1 sessions active");
    expect(within(row).getAllByRole("link")).toHaveLength(1);
    const newSession = within(row).getByTestId("agent-fleet-new-session-code-reviewer");
    expect(newSession).toBeInTheDocument();
    expect(link).not.toContainElement(newSession);
    expect(link).not.toContainElement(screen.getByTestId("agent-fleet-sessions-code-reviewer"));
    await user.click(newSession);
    expect(mockOpenNewSession).toHaveBeenCalledWith("code-reviewer");
    expect(link).toHaveAttribute("href", "/agents/code-reviewer");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ name: "code-reviewer" }));
  });

  it("Should persist view=cards and keep origin plus full aria on cards", async () => {
    const user = userEvent.setup();
    render(<AgentsPage />);
    await screen.findByTestId("agent-fleet-list");

    await user.click(screen.getByTestId("listing-view-cards"));
    await waitFor(() => {
      expect(getValidatedSearch()).toMatchObject({ view: "cards" });
    });

    expect(await screen.findByTestId("agent-fleet-card-grid")).toBeInTheDocument();
    const card = screen.getByTestId("agent-fleet-card-triage-bot");
    expect(within(card).getByTestId("agent-fleet-card-meta-triage-bot")).toHaveTextContent(
      "Workspace"
    );
    const cardLink = within(card).getByTestId("agent-fleet-card-link-triage-bot");
    expect(cardLink).toHaveAttribute("aria-label", "triage-bot, Idle, 0 of 0 sessions active");
    const cardNewSession = within(card).getByTestId("agent-fleet-new-session-triage-bot");
    expect(cardLink).not.toContainElement(cardNewSession);

    routerState.searchParams = { view: "cards" };
    act(() => {
      for (const listener of routerState.searchListeners) {
        listener(getValidatedSearch());
      }
    });
    expect(screen.getByTestId("agent-fleet-card-grid")).toBeInTheDocument();
  });

  it("Should preserve card geometry while a restored cards view is loading", async () => {
    agentsDelayMs = 30;
    routerState.searchParams = { view: "cards" };

    render(<AgentsPage />);

    const loading = screen.getByTestId("agent-fleet-loading");
    expect(loading).toHaveAttribute("aria-label", "Loading agents");
    expect(loading.children).toHaveLength(6);
    expect(await screen.findByTestId("agent-fleet-card-grid")).toBeInTheDocument();
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
    await waitFor(() => expect(screen.getByTestId("topbar-count")).toHaveTextContent("1"));

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
    expect(screen.getByTestId("topbar-count")).toHaveTextContent("0");
  });

  it("Should keep rows and omit status when sessions fail, including with a status URL filter", async () => {
    sessionsAvailable = false;
    routerState.searchParams = { status: "active" };
    render(<AgentsPage />);

    await screen.findByTestId("agent-fleet-list");
    expect(screen.getByTestId("agent-fleet-sessions-notice")).toHaveTextContent(
      "Session status unavailable"
    );
    expect(screen.getByTestId("agent-fleet-sessions-release-captain")).toHaveTextContent("--");
    expect(screen.queryByTestId("agent-fleet-status-release-captain")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-fleet-row-link-release-captain")).toHaveAttribute(
      "aria-label",
      "release-captain, session status unavailable"
    );
    expect(screen.getAllByTestId(/agent-fleet-row-link-/)).toHaveLength(3);
    expect(agentsRequestCount).toBe(1);
  });

  it("Should keep skeleton geometry until catalog counts load", async () => {
    agentsDelayMs = 30;
    render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-fleet-status-code-reviewer")).not.toBeInTheDocument();
    expect(await screen.findByTestId("agent-fleet-list")).toBeInTheDocument();
  });

  it("Should load the next backend page without recomputing the catalog in the client", async () => {
    const user = userEvent.setup();
    mockAgents = Array.from({ length: 51 }, (_, index) =>
      agent({ name: `agent-${String(index + 1).padStart(2, "0")}` })
    );
    mockSessions = [];

    render(<AgentsPage />);
    await screen.findByTestId("agent-fleet-row-agent-50");
    expect(screen.queryByTestId("agent-fleet-row-agent-51")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("agent-fleet-load-more"));
    expect(await screen.findByTestId("agent-fleet-row-agent-51")).toBeInTheDocument();
    expect(agentsRequestCount).toBe(2);
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
    expect(
      screen.getByText("Try a different search or clear the active filters.")
    ).toBeInTheDocument();
    await user.click(screen.getByTestId("agent-fleet-clear-filters"));
    expect(routerState.navigateMock).toHaveBeenCalled();
  });

  it("Should keep page head and search when agents fail, without view toggle", async () => {
    const user = userEvent.setup();
    mockAgents = [];
    agentsShouldError = true;
    render(<AgentsPage />);
    expect(await screen.findByTestId("agent-fleet-error")).toBeInTheDocument();
    expect(screen.getByTestId("agents-page-head")).toBeInTheDocument();
    expect(screen.getByTestId("agent-fleet-search")).toBeInTheDocument();
    expect(screen.queryByTestId("listing-view-toggle")).not.toBeInTheDocument();
    expect(screen.getByText("Couldn't load agents")).toBeInTheDocument();
    expect(
      screen.getByText("The agents request failed. Check the daemon connection and try again.")
    ).toBeInTheDocument();
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
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 250));
    });
    expect(getValidatedSearch()).toEqual({
      q: "release",
      category: undefined,
      status: undefined,
      view: undefined,
    });
  });
});
