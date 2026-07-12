import * as React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { loopRunFixtures } from "@/systems/loops/mocks/fixtures";
import type { LoopRunsFilter } from "@/systems/loops/types";

const routerState = vi.hoisted(() => ({
  listeners: new Set<(search: Record<string, unknown>) => void>(),
  search: {} as Record<string, unknown>,
  validate: undefined as ((search: Record<string, unknown>) => Record<string, unknown>) | undefined,
}));

const useLoopRunsMock = vi.hoisted(() => vi.fn());

function validatedSearch(): Record<string, unknown> {
  return routerState.validate ? routerState.validate(routerState.search) : routerState.search;
}

vi.mock("@tanstack/react-router", () => ({
  Outlet: () => <div data-testid="loop-runs-outlet" />,
  Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
    <a
      href={typeof to === "string" ? to : "#"}
      data-params={JSON.stringify(params)}
      {...(props as Record<string, unknown>)}
    >
      {children as React.ReactNode}
    </a>
  ),
  createFileRoute:
    () =>
    (options: {
      component: () => React.ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validate = options.validateSearch;
      return {
        component: options.component,
        useSearch: () => {
          const [search, setSearch] = React.useState(validatedSearch());
          React.useEffect(() => {
            routerState.listeners.add(setSearch);
            return () => {
              routerState.listeners.delete(setSearch);
            };
          }, []);
          return search;
        },
      };
    },
  useChildMatches: () => [],
  useNavigate:
    () =>
    async (options: {
      search?:
        | Record<string, unknown>
        | ((current: Record<string, unknown>) => Record<string, unknown>);
    }) => {
      routerState.search =
        typeof options.search === "function"
          ? options.search(validatedSearch())
          : (options.search ?? {});
      const next = validatedSearch();
      for (const listener of routerState.listeners) listener(next);
    },
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspace: { id: "ws_1", name: "Workspace one" },
    activeWorkspaceId: "ws_1",
  }),
}));

vi.mock("@/systems/loops", async () => {
  const actual = await vi.importActual("@/systems/loops");
  return {
    ...actual,
    useLoopRuns: useLoopRunsMock,
  };
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../loop-runs";

const LoopRunsPage = routeComponent(Route);

describe("LoopRunsPage origin filters", () => {
  beforeEach(() => {
    routerState.listeners.clear();
    routerState.search = {};
    useLoopRunsMock.mockReset();
    useLoopRunsMock.mockImplementation((_workspaceId: string, _filters: LoopRunsFilter) => ({
      data: {
        runs: loopRunFixtures,
        aggregates: { total: 0, live: 0, terminal: 0, succeeded: 0, failed: 0 },
      },
      isLoading: false,
      error: null,
    }));
  });

  it("Should persist session origin filters in URL search and the runs query", async () => {
    const user = userEvent.setup();
    render(<LoopRunsPage />);

    await user.selectOptions(screen.getByLabelText("Run origin"), "session");
    expect(validatedSearch()).toEqual({ origin: "session", origin_session: undefined });
    expect(screen.getByLabelText("Origin session id")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Origin session id"), "session_42");
    expect(validatedSearch()).toEqual({ origin: "session", origin_session: "session_42" });
    expect(useLoopRunsMock).toHaveBeenLastCalledWith(
      "ws_1",
      { origin: "session", origin_session: "session_42" },
      true
    );
  });

  it("Should clear the origin session when switching back to catalog", async () => {
    routerState.search = { origin: "session", origin_session: "session_42" };
    const user = userEvent.setup();
    render(<LoopRunsPage />);

    await user.selectOptions(screen.getByLabelText("Run origin"), "catalog");
    expect(validatedSearch()).toEqual({ origin: "catalog", origin_session: undefined });
    expect(screen.queryByLabelText("Origin session id")).not.toBeInTheDocument();
  });
});
