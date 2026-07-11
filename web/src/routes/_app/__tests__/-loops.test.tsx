import * as React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { loopCatalogFixtures } from "@/systems/loops/mocks/fixtures";
import type { LoopCatalogEntry } from "@/systems/loops/types";
import { renderWithTopbar } from "@/test/render-with-topbar";

function render(ui: React.ReactElement) {
  return renderWithTopbar(ui, { title: "Loops" });
}

let mockLoops: LoopCatalogEntry[] = [];
let mockLoopsTotal = 0;
let mockLoopsLoading = false;
let mockLoopsError: Error | null = null;
const mockRefetchLoops = vi.fn();
const mockFetchNextLoops = vi.fn();
const mockUseLoops = vi.fn();

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  searchListeners: new Set<(search: Record<string, unknown>) => void>(),
  searchParams: {} as Record<string, unknown>,
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
  Outlet: () => <div data-testid="loops-outlet" />,
  createFileRoute:
    () =>
    (opts: {
      component: () => React.ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validateSearch = opts.validateSearch;

      return {
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
  useChildMatches: () => [],
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
  Link: ({ to, params, children, ...props }: Record<string, unknown>) => (
    <a
      href={typeof to === "string" ? to : "#"}
      data-params={JSON.stringify(params)}
      {...(props as Record<string, unknown>)}
    >
      {children as React.ReactNode}
    </a>
  ),
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
    activeWorkspace: {
      id: "ws_test",
      root_dir: "/workspace",
      add_dirs: [],
      name: "test-workspace",
      created_at: "2026-04-03T12:00:00Z",
      updated_at: "2026-04-03T12:00:00Z",
    },
    activeWorkspaceId: "ws_test",
    setActiveWorkspaceId: vi.fn(),
    clearActiveWorkspaceSelection: vi.fn(),
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/loops", async () => {
  const actual = await vi.importActual("@/systems/loops");
  return {
    ...actual,
    useLoops: (...args: unknown[]) => {
      mockUseLoops(...args);
      return {
        data: mockLoops.length > 0 ? { pageParams: [undefined], pages: [] } : undefined,
        facets: {
          categories: { delivery: 1, watch: 1 },
          kinds: { read_only: 1, workspace: 1 },
          statuses: { running: 1, watching: 1 },
        },
        fetchNextPage: mockFetchNextLoops,
        hasNextPage: false,
        isFetchingNextPage: false,
        isLoading: mockLoopsLoading,
        error: mockLoopsError,
        loops: mockLoops,
        refetch: mockRefetchLoops,
        total: mockLoopsTotal,
      };
    },
  };
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../loops";

const LoopsPage = routeComponent(Route);

describe("LoopsPage", () => {
  beforeEach(() => {
    mockLoops = [...loopCatalogFixtures];
    mockLoopsTotal = mockLoops.length;
    mockLoopsLoading = false;
    mockLoopsError = null;
    mockRefetchLoops.mockReset();
    mockFetchNextLoops.mockReset();
    mockUseLoops.mockReset();
    routerState.searchListeners.clear();
    routerState.searchParams = {};
    routerState.navigateMock.mockReset();
  });

  it("Should render listing shell with toolbar and grouped rows", () => {
    render(<LoopsPage />);
    expect(screen.getByTestId("loops-page-head")).toBeInTheDocument();
    expect(screen.getByTestId("listing-toolbar")).toBeInTheDocument();
    expect(screen.getByTestId("loop-catalog")).toBeInTheDocument();
    expect(screen.getByTestId("loop-group-read-only")).toBeInTheDocument();
    expect(screen.getByTestId("loop-group-workspace")).toBeInTheDocument();
  });

  it("Should show the exact server total in page head and topbar Runs link", () => {
    mockLoopsTotal = 12;
    render(<LoopsPage />);
    expect(screen.getByTestId("loops-page-count")).toHaveTextContent("12");
    expect(screen.getByTestId("loops-runs-link")).toBeInTheDocument();
    expect(screen.getByTestId("loops-refresh")).toBeInTheDocument();
  });

  it("Should persist view=cards in URL search", async () => {
    const user = userEvent.setup();
    render(<LoopsPage />);

    await user.click(screen.getByTestId("listing-view-cards"));

    expect(getValidatedSearch()).toMatchObject({ view: "cards" });
    expect(screen.getByTestId("loop-catalog-card-grid")).toBeInTheDocument();
  });

  it("Should persist search query in URL and send it to the server catalog", async () => {
    const user = userEvent.setup();
    render(<LoopsPage />);

    await user.type(screen.getByTestId("loop-search-input"), "software");

    expect(getValidatedSearch()).toMatchObject({ q: "software" });
    expect(mockUseLoops).toHaveBeenLastCalledWith(
      "ws_test",
      expect.objectContaining({ limit: 50, q: "software", sort: "name" }),
      true
    );
  });

  it("Should clear filters from the empty state", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { q: "zzz-no-match" };
    mockLoops = [];
    mockLoopsTotal = 0;
    render(<LoopsPage />);

    expect(screen.getByTestId("loop-catalog-empty")).toBeInTheDocument();
    await user.click(screen.getByTestId("loop-catalog-clear-filters"));
    expect(getValidatedSearch().q).toBeUndefined();
  });

  it("Should restore cards view from URL search", () => {
    routerState.searchParams = { view: "cards" };
    render(<LoopsPage />);
    expect(screen.getByTestId("loop-catalog-card-grid")).toBeInTheDocument();
  });

  it("Should refetch on Refresh", async () => {
    const user = userEvent.setup();
    render(<LoopsPage />);
    await user.click(screen.getByTestId("loops-refresh"));
    expect(mockRefetchLoops).toHaveBeenCalledTimes(1);
  });

  it("Should show empty inventory when the workspace has no loops", () => {
    mockLoops = [];
    mockLoopsTotal = 0;
    render(<LoopsPage />);
    expect(screen.getByTestId("loops-empty")).toBeInTheDocument();
    expect(screen.queryByTestId("listing-toolbar")).not.toBeInTheDocument();
  });

  it("Should show loading and error states", () => {
    mockLoopsLoading = true;
    mockLoops = [];
    mockLoopsTotal = 0;
    const { rerender } = render(<LoopsPage />);
    expect(screen.getByTestId("loops-loading")).toBeInTheDocument();

    mockLoopsLoading = false;
    mockLoopsError = new Error("boom");
    rerender(<LoopsPage />);
    expect(screen.getByTestId("loops-error")).toBeInTheDocument();
  });
});
