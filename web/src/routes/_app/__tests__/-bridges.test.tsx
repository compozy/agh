import * as React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";

import type { BridgeProvider, BridgesListResponse, CreateBridgeResponse } from "@/systems/bridges";

function render(ui: React.ReactElement) {
  return renderWithTopbar(ui);
}

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

let mockBridgesData: BridgesListResponse | undefined;
let mockBridgesLoading = false;
let mockBridgesError: Error | null = null;
const mockRefetchBridges = vi.fn();
const mockFetchNextBridges = vi.fn();

let mockProvidersData: BridgeProvider[] | undefined;
let mockProvidersLoading = false;
let mockProvidersError: Error | null = null;
const mockRefetchProviders = vi.fn();
const mockRefetchManifest = vi.fn();

const mockCreateBridgeMutateAsync = vi.fn();
let mockCreateBridgePending = false;

let mockActiveWorkspaceId: string | null = "ws_test";
let mockActiveWorkspaceName = "test-workspace";

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
  Outlet: () => <div data-testid="bridges-outlet" />,
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
      data-params={JSON.stringify(params)}
      href={typeof to === "string" ? to : "#"}
      {...(props as Record<string, unknown>)}
    >
      {children as React.ReactNode}
    </a>
  ),
}));

vi.mock("sonner", () => ({
  toast,
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: mockActiveWorkspaceId
      ? [
          {
            add_dirs: [],
            created_at: "2026-04-03T12:00:00Z",
            id: mockActiveWorkspaceId,
            name: mockActiveWorkspaceName,
            root_dir: "/workspace",
            updated_at: "2026-04-03T12:00:00Z",
          },
        ]
      : [],
    hasWorkspaces: Boolean(mockActiveWorkspaceId),
    activeWorkspace: mockActiveWorkspaceId
      ? {
          add_dirs: [],
          created_at: "2026-04-03T12:00:00Z",
          id: mockActiveWorkspaceId,
          name: mockActiveWorkspaceName,
          root_dir: "/workspace",
          updated_at: "2026-04-03T12:00:00Z",
        }
      : undefined,
    activeWorkspaceId: mockActiveWorkspaceId,
    clearActiveWorkspaceSelection: vi.fn(),
    isError: false,
    isLoading: false,
    setActiveWorkspaceId: vi.fn(),
  }),
}));

vi.mock("@/systems/bridges", async () => {
  const actual = await vi.importActual("@/systems/bridges");

  return {
    ...actual,
    useBridges: () => ({
      data: mockBridgesData ? { pageParams: [undefined], pages: [mockBridgesData] } : undefined,
      bridgeHealth: mockBridgesData?.bridge_health ?? {},
      bridges: mockBridgesData?.bridges ?? [],
      error: mockBridgesError,
      facets: mockBridgesData?.facets,
      fetchNextPage: mockFetchNextBridges,
      hasNextPage: mockBridgesData?.page.has_more ?? false,
      isFetchingNextPage: false,
      isLoading: mockBridgesLoading,
      refetch: mockRefetchBridges,
      total: mockBridgesData?.page.total ?? 0,
    }),
    useBridgeProviders: () => ({
      data: mockProvidersData,
      error: mockProvidersError,
      isLoading: mockProvidersLoading,
      refetch: mockRefetchProviders,
    }),
    useBridgeHealthStream: vi.fn(),
    useSlackBridgeManifest: () => ({
      data: undefined,
      error: null,
      isLoading: false,
      refetch: mockRefetchManifest,
    }),
    useCreateBridge: () => ({
      isPending: mockCreateBridgePending,
      mutateAsync: mockCreateBridgeMutateAsync,
    }),
  };
});

import { Route } from "../bridges";

const BridgesPage = routeComponent(Route);

function makeBridge(overrides: Partial<BridgesListResponse["bridges"][number]> = {}) {
  return {
    created_at: "2026-04-13T12:00:00Z",
    dm_policy: "open" as const,
    display_name: "Support",
    enabled: true,
    extension_name: "ext-telegram",
    id: "brg_support",
    notification_suppress: false,
    platform: "telegram",
    provider_config: {
      mode: "bot",
      webhook: {
        public_url: "https://example.test/webhook",
      },
    },
    routing_policy: { include_group: true, include_peer: true, include_thread: true },
    scope: "workspace" as const,
    status: "ready" as const,
    updated_at: "2026-04-13T12:30:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeHealth(
  overrides: Partial<NonNullable<BridgesListResponse["bridge_health"]>[string]> = {}
) {
  return {
    auth_failures_total: 0,
    bridge_instance_id: "brg_support",
    delivery_backlog: 1,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    last_success_at: "2026-04-13T12:20:00Z",
    route_count: 1,
    status: "ready" as const,
    ...overrides,
  };
}

function makeProvider(overrides: Partial<BridgeProvider> = {}): BridgeProvider {
  return {
    config_schema: {
      schema: "provider-config",
      version: "2026-04-15",
    },
    description: "Provider-specific runtime settings",
    display_name: "Telegram",
    enabled: true,
    extension_name: "ext-telegram",
    health: "healthy",
    health_message: "Webhook and token requirements are healthy.",
    platform: "telegram",
    secret_slots: [
      {
        description: "Bot API token",
        name: "bot_token",
        required: true,
      },
    ],
    state: "active",
    ...overrides,
  };
}

function makeBridgesResponse(
  bridges: BridgesListResponse["bridges"],
  bridgeHealth: BridgesListResponse["bridge_health"],
  total = bridges.length
): BridgesListResponse {
  return {
    bridge_health: bridgeHealth,
    bridges,
    facets: {
      platforms: Object.fromEntries(bridges.map(bridge => [bridge.platform, 1])),
      statuses: {
        auth_required: 0,
        degraded: 0,
        disabled: 0,
        error: 0,
        ready: bridges.length,
        starting: 0,
      },
    },
    page: { has_more: false, limit: 50, total },
  };
}

describe("BridgesPage", () => {
  beforeEach(() => {
    mockBridgesData = makeBridgesResponse([makeBridge()], {
      brg_support: makeHealth(),
    });
    mockBridgesLoading = false;
    mockBridgesError = null;
    mockProvidersData = [makeProvider()];
    mockProvidersLoading = false;
    mockProvidersError = null;
    mockCreateBridgePending = false;
    mockActiveWorkspaceId = "ws_test";
    mockActiveWorkspaceName = "test-workspace";
    mockCreateBridgeMutateAsync.mockReset();
    mockRefetchBridges.mockReset();
    mockFetchNextBridges.mockReset();
    mockRefetchProviders.mockReset();
    mockRefetchManifest.mockReset();
    toast.success.mockReset();
    toast.error.mockReset();
    routerState.searchListeners.clear();
    routerState.searchParams = {};
    routerState.navigateMock.mockReset();

    mockCreateBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ id: "brg_created", status: "starting" }),
      health: makeHealth({ bridge_instance_id: "brg_created", status: "starting" }),
    } satisfies CreateBridgeResponse);
  });

  it("renders listing shell without split pane", () => {
    render(<BridgesPage />);

    expect(screen.getByTestId("bridges-page-head")).toBeInTheDocument();
    expect(screen.getByTestId("listing-toolbar")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-list-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("bridges-split-pane")).not.toBeInTheDocument();
    expect(screen.queryByTestId("bridge-scope-pills")).not.toBeInTheDocument();
  });

  it("renders loading and error states from the list queries", () => {
    mockBridgesLoading = true;
    mockProvidersLoading = true;
    mockBridgesData = undefined;
    mockProvidersData = undefined;
    const { rerender } = render(<BridgesPage />);

    expect(screen.getByTestId("bridges-loading")).toBeInTheDocument();

    mockBridgesLoading = false;
    mockProvidersLoading = false;
    mockBridgesData = undefined;
    mockBridgesError = new Error("boom");
    rerender(<BridgesPage />);

    expect(screen.getByTestId("bridges-error")).toHaveTextContent("boom");
  });

  it("renders the empty state with provider cards when no bridge exists yet", () => {
    mockBridgesData = makeBridgesResponse([], {});

    render(<BridgesPage />);

    expect(screen.getByTestId("bridges-empty-state")).toBeInTheDocument();
    expect(screen.getByText("No bridges configured")).toBeInTheDocument();
    expect(screen.getByText("Telegram")).toBeInTheDocument();
  });

  it("links bridge rows to /bridges/$id", () => {
    render(<BridgesPage />);

    const link = screen.getByRole("link", { name: "Open Support" });
    expect(link).toHaveAttribute("href", "/bridges/$id");
    expect(link).toHaveAttribute("data-params", JSON.stringify({ id: "brg_support" }));
  });

  it("renders the server page unchanged and uses its exact total", () => {
    mockBridgesData = makeBridgesResponse(
      [
        makeBridge({
          display_name: "Global Telegram",
          id: "brg_global",
          scope: "global",
          workspace_id: undefined,
        }),
        makeBridge(),
      ],
      {
        brg_global: makeHealth({ bridge_instance_id: "brg_global" }),
        brg_support: makeHealth(),
      },
      12
    );

    render(<BridgesPage />);

    expect(screen.getByTestId("bridge-item-brg_global")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-item-brg_support")).toBeInTheDocument();
    expect(screen.getByTestId("bridges-page-count")).toHaveTextContent("12");
  });

  it("persists view=cards in URL search", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.click(screen.getByTestId("listing-view-cards"));
    expect(getValidatedSearch()).toMatchObject({ view: "cards" });
    expect(screen.getByTestId("bridge-list-card-grid")).toBeInTheDocument();
  });

  it("restores search and view from URL", () => {
    routerState.searchParams = { q: "Support", view: "cards" };

    render(<BridgesPage />);

    expect(screen.getByTestId("bridge-search-input")).toHaveValue("Support");
    expect(screen.getByTestId("bridge-list-card-grid")).toBeInTheDocument();
  });

  it("creates a bridge and navigates to the detail route", async () => {
    const user = userEvent.setup();
    mockBridgesData = makeBridgesResponse([], {});

    render(<BridgesPage />);

    await user.click(screen.getByTestId("bridge-empty-create-btn"));
    expect(screen.getByTestId("bridge-create-dialog")).toBeInTheDocument();

    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.selectOptions(screen.getByTestId("bridge-dm-policy-select"), "allowlist");
    fireEvent.change(screen.getByTestId("bridge-provider-config-input"), {
      target: {
        value: '{"mode":"bot","webhook":{"public_url":"https://example.test/webhook"}}',
      },
    });
    await user.click(screen.getByTestId("bridge-wizard-next"));

    mockCreateBridgeMutateAsync.mockImplementationOnce(async payload => {
      const createdBridge = makeBridge({
        display_name: payload.display_name,
        dm_policy: payload.dm_policy,
        id: "brg_created",
        provider_config: payload.provider_config,
        status: "starting",
      });

      return {
        bridge: createdBridge,
        health: makeHealth({
          bridge_instance_id: "brg_created",
          status: "starting",
        }),
      } satisfies CreateBridgeResponse;
    });

    await user.click(screen.getByTestId("submit-bridge-create"));

    await waitFor(() => {
      expect(mockCreateBridgeMutateAsync).toHaveBeenCalled();
    });

    expect(toast.success).toHaveBeenCalledWith("Created bridge Telegram.");
    expect(routerState.navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        params: { id: "brg_created" },
        to: "/bridges/$id",
      })
    );
  });

  it("blocks workspace-scoped bridge creation when the active workspace disappears", async () => {
    const user = userEvent.setup();
    mockBridgesData = makeBridgesResponse([], {});

    const { rerender } = render(<BridgesPage />);

    await user.click(screen.getByTestId("bridge-empty-create-btn"));
    await user.click(screen.getByTestId("bridge-wizard-next"));
    await user.click(screen.getByTestId("bridge-wizard-next"));

    mockActiveWorkspaceId = null;
    mockActiveWorkspaceName = "";
    rerender(<BridgesPage />);

    await user.click(screen.getByTestId("submit-bridge-create"));

    expect(mockCreateBridgeMutateAsync).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith(
      "Select an active workspace before creating a workspace-scoped bridge."
    );
  });

  it("refreshes bridges from the topbar action", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.click(screen.getByTestId("bridges-refresh"));
    expect(mockRefetchBridges).toHaveBeenCalled();
    expect(mockRefetchProviders).toHaveBeenCalled();
  });
});
