// Suite: cross-workspace session Return route integration
// Invariant: a Return history intent selects only its exact session owner during the navigation that carries it.
// Boundary IN: TanStack Router, Link/history state, route beforeLoad/loader, Query cache, and active-workspace store.
// Boundary OUT: HTTP adapters and rendered session transcript, owned by their system suites.

import { QueryClient } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  Link,
  Outlet,
  RouterProvider,
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  type HistoryState,
} from "@tanstack/react-router";
import { beforeEach, describe, expect, it } from "vitest";

import {
  createSessionReturnHistoryState,
  sessionKeys,
  type SessionPayload,
} from "@/systems/session";
import { useActiveWorkspaceStore, workspaceKeys, type WorkspacePayload } from "@/systems/workspace";
import { routeBeforeLoad, routeLoader } from "@/test/route-options";
import type { AgentSessionRouteLoaderData } from "../-agent-session-route-loader";
import { Route as ProductionSessionRoute } from "../agents.$name.sessions.$id";

const BENCH_SESSION_ID = "sess-40e90687024bfb24";
const BENCH_WORKSPACE_ID = "ws_74a58ac2bf973937";
const PRIMARY_SESSION_ID = "sess-5ec18f5f2a13fe16";
const PRIMARY_WORKSPACE_ID = "ws_06366aad69887872";

interface TestRouterContext {
  queryClient: QueryClient;
}

interface SessionRouteContextExtension {
  sessionReturnWorkspaceId?: string;
}

function makeWorkspace(id: string, name: string): WorkspacePayload {
  return {
    id,
    name,
    root_dir: `/workspace/${name}`,
    add_dirs: [],
    created_at: "2026-07-13T00:00:00Z",
    updated_at: "2026-07-13T00:00:00Z",
  };
}

type OwnedSession = SessionPayload & { workspace_id: string };

function makeSession(id: string, workspaceId: string, name: string): OwnedSession {
  return {
    id,
    name,
    agent_name: "general",
    provider: "codex",
    workspace_id: workspaceId,
    workspace_path: `/workspace/${name}`,
    state: "active",
    badge: "running",
    attachable: true,
    available_commands: [],
    created_at: "2026-07-13T00:00:00Z",
    updated_at: "2026-07-13T00:00:00Z",
  };
}

function seedSessionRouteQueries(queryClient: QueryClient): void {
  const sessions = [
    makeSession(BENCH_SESSION_ID, BENCH_WORKSPACE_ID, "bench-ops"),
    makeSession(PRIMARY_SESSION_ID, PRIMARY_WORKSPACE_ID, "primary"),
  ];
  queryClient.setQueryData(workspaceKeys.list(), [
    makeWorkspace(BENCH_WORKSPACE_ID, "bench-ops"),
    makeWorkspace(PRIMARY_WORKSPACE_ID, "primary"),
  ]);
  for (const session of sessions) {
    queryClient.setQueryData(sessionKeys.byId(session.id), session);
    queryClient.setQueryData(sessionKeys.detail(session.workspace_id, session.id), session);
    queryClient.setQueryData(sessionKeys.transcript(session.workspace_id, session.id), {
      pages: [],
      pageParams: [],
    });
  }
}

function buildSessionReturnRouter() {
  const causes: Array<"enter" | "preload" | "stay"> = [];
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  seedSessionRouteQueries(queryClient);

  const rootRoute = createRootRouteWithContext<TestRouterContext>()({
    component: () => <Outlet />,
  });
  const beforeLoad = routeBeforeLoad<{
    params: { id: string; name: string };
    location: { state: HistoryState };
  }>(ProductionSessionRoute);
  const loadSessionRoute = routeLoader<{
    context: TestRouterContext & SessionRouteContextExtension;
    params: { id: string };
    preload: boolean;
  }>(ProductionSessionRoute);
  const agentRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "agents/$name",
    component: () => <Outlet />,
  });
  const sessionRoute = createRoute({
    getParentRoute: () => agentRoute,
    path: "sessions/$id",
    beforeLoad: args => {
      causes.push(args.cause);
      return beforeLoad(args) as SessionRouteContextExtension;
    },
    loader: args => loadSessionRoute(args) as Promise<AgentSessionRouteLoaderData>,
    component: SessionRouteHarness,
  });
  const routeTree = rootRoute.addChildren([agentRoute.addChildren([sessionRoute])]);
  const router = createRouter({
    routeTree,
    context: { queryClient },
    history: createMemoryHistory({
      initialEntries: [`/agents/general/sessions/${BENCH_SESSION_ID}`],
    }),
    defaultPreloadStaleTime: 0,
  });

  return { causes, router };

  function SessionRouteHarness() {
    const { id } = sessionRoute.useParams();
    return (
      <main>
        <p>Loaded session: {id}</p>
        {id === BENCH_SESSION_ID ? (
          <Link
            to="/agents/$name/sessions/$id"
            params={{ name: "general", id: PRIMARY_SESSION_ID }}
            state={createSessionReturnHistoryState(PRIMARY_SESSION_ID, PRIMARY_WORKSPACE_ID)}
          >
            Return to primary
          </Link>
        ) : (
          <Link to="/agents/$name/sessions/$id" params={{ name: "general", id: BENCH_SESSION_ID }}>
            Open bench permalink
          </Link>
        )}
      </main>
    );
  }
}

describe("cross-workspace session Return router integration", () => {
  beforeEach(() => {
    localStorage.clear();
    useActiveWorkspaceStore.setState({ selectedWorkspaceId: BENCH_WORKSPACE_ID });
  });

  it("selects the destination workspace when Return changes params within the same route", async () => {
    const { causes, router } = buildSessionReturnRouter();
    render(<RouterProvider router={router} />);

    await screen.findByText(`Loaded session: ${BENCH_SESSION_ID}`);
    fireEvent.click(screen.getByRole("link", { name: "Return to primary" }));

    await screen.findByText(`Loaded session: ${PRIMARY_SESSION_ID}`);
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(`/agents/general/sessions/${PRIMARY_SESSION_ID}`);
    });
    expect(router.state.location.state).toMatchObject({
      sessionReturn: {
        sessionId: PRIMARY_SESSION_ID,
        workspaceId: PRIMARY_WORKSPACE_ID,
      },
    });
    expect(causes.at(-1)).toBe("stay");
    expect(useActiveWorkspaceStore.getState().selectedWorkspaceId).toBe(PRIMARY_WORKSPACE_ID);
    expect(localStorage.getItem("agh:active-workspace")).toContain(PRIMARY_WORKSPACE_ID);
  });

  it("does not carry Return selection authority into the next navigation without intent", async () => {
    const { router } = buildSessionReturnRouter();
    render(<RouterProvider router={router} />);

    await screen.findByText(`Loaded session: ${BENCH_SESSION_ID}`);
    fireEvent.click(screen.getByRole("link", { name: "Return to primary" }));
    await screen.findByText(`Loaded session: ${PRIMARY_SESSION_ID}`);
    fireEvent.click(screen.getByRole("link", { name: "Open bench permalink" }));

    await screen.findByText(`Loaded session: ${BENCH_SESSION_ID}`);
    expect(router.state.location.state).not.toHaveProperty("sessionReturn");
    expect(useActiveWorkspaceStore.getState().selectedWorkspaceId).toBe(PRIMARY_WORKSPACE_ID);
  });
});
