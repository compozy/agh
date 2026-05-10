import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithTopbar as render } from "@/test/render-with-topbar";

import type {
  BridgeDetailResponse,
  BridgeProvider,
  BridgeRoute,
  BridgeSecretBinding,
  BridgesListResponse,
  CreateBridgeResponse,
  TestBridgeDeliveryResponse,
  UpdateBridgeResponse,
} from "@/systems/bridges";

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

let mockBridgesData: BridgesListResponse | undefined;
let mockBridgesLoading = false;
let mockBridgesError: Error | null = null;

let mockProvidersData: BridgeProvider[] | undefined;
let mockProvidersLoading = false;
let mockProvidersError: Error | null = null;

let mockBridgeDetail: BridgeDetailResponse | undefined;
let mockBridgeDetailLoading = false;
let mockBridgeDetailError: Error | null = null;

let mockBridgeRoutes: BridgeRoute[] | undefined;
let mockBridgeRoutesLoading = false;
let mockBridgeRoutesError: Error | null = null;
let mockSecretBindingsData: BridgeSecretBinding[] | undefined;
let mockSecretBindingsLoading = false;
let mockSecretBindingsError: Error | null = null;

const mockCreateBridgeMutateAsync = vi.fn();
const mockUpdateBridgeMutateAsync = vi.fn();
const mockPutBridgeSecretBindingMutateAsync = vi.fn();
const mockDeleteBridgeSecretBindingMutateAsync = vi.fn();
const mockEnableBridgeMutateAsync = vi.fn();
const mockDisableBridgeMutateAsync = vi.fn();
const mockRestartBridgeMutateAsync = vi.fn();
const mockTestDeliveryMutateAsync = vi.fn();
let mockCreateBridgePending = false;
let mockUpdateBridgePending = false;
let mockPutBridgeSecretBindingPending = false;
let mockDeleteBridgeSecretBindingPending = false;
let mockEnableBridgePending = false;
let mockDisableBridgePending = false;
let mockRestartBridgePending = false;
let mockTestDeliveryPending = false;

let mockActiveWorkspaceId: string | null = "ws_test";
let mockActiveWorkspaceName = "test-workspace";

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
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
      data: mockBridgesData,
      error: mockBridgesError,
      isLoading: mockBridgesLoading,
    }),
    useBridgeProviders: () => ({
      data: mockProvidersData,
      error: mockProvidersError,
      isLoading: mockProvidersLoading,
    }),
    useBridge: () => ({
      data: mockBridgeDetail,
      error: mockBridgeDetailError,
      isLoading: mockBridgeDetailLoading,
    }),
    useBridgeRoutes: () => ({
      data: mockBridgeRoutes,
      error: mockBridgeRoutesError,
      isLoading: mockBridgeRoutesLoading,
    }),
    useBridgeSecretBindings: () => ({
      data: mockSecretBindingsData,
      error: mockSecretBindingsError,
      isLoading: mockSecretBindingsLoading,
    }),
    useBridgeHealthStream: vi.fn(),
    useCreateBridge: () => ({
      isPending: mockCreateBridgePending,
      mutateAsync: mockCreateBridgeMutateAsync,
    }),
    useUpdateBridge: () => ({
      isPending: mockUpdateBridgePending,
      mutateAsync: mockUpdateBridgeMutateAsync,
    }),
    usePutBridgeSecretBinding: () => ({
      isPending: mockPutBridgeSecretBindingPending,
      mutateAsync: mockPutBridgeSecretBindingMutateAsync,
    }),
    useDeleteBridgeSecretBinding: () => ({
      isPending: mockDeleteBridgeSecretBindingPending,
      mutateAsync: mockDeleteBridgeSecretBindingMutateAsync,
    }),
    useEnableBridge: () => ({
      isPending: mockEnableBridgePending,
      mutateAsync: mockEnableBridgeMutateAsync,
    }),
    useDisableBridge: () => ({
      isPending: mockDisableBridgePending,
      mutateAsync: mockDisableBridgeMutateAsync,
    }),
    useRestartBridge: () => ({
      isPending: mockRestartBridgePending,
      mutateAsync: mockRestartBridgeMutateAsync,
    }),
    useTestBridgeDelivery: () => ({
      isPending: mockTestDeliveryPending,
      mutateAsync: mockTestDeliveryMutateAsync,
    }),
  };
});

import { Route } from "../bridges";

function makeBridge(overrides: Partial<BridgesListResponse["bridges"][number]> = {}) {
  return {
    created_at: "2026-04-13T12:00:00Z",
    dm_policy: "open" as const,
    display_name: "Support",
    enabled: true,
    extension_name: "ext-telegram",
    id: "brg_support",
    platform: "telegram",
    provider_config: {
      mode: "bot",
      webhook_url: "https://example.test/webhook",
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
      {
        description: "Optional webhook secret",
        name: "webhook_secret",
        required: false,
      },
    ],
    state: "active",
    ...overrides,
  };
}

function makeRoute(overrides: Partial<BridgeRoute> = {}): BridgeRoute {
  return {
    agent_name: "support-agent",
    bridge_instance_id: "brg_support",
    created_at: "2026-04-13T12:00:00Z",
    last_activity_at: "2026-04-13T12:15:00Z",
    peer_id: "peer_123",
    routing_key_hash: "abc123",
    scope: "workspace",
    session_id: "sess_123",
    updated_at: "2026-04-13T12:15:00Z",
    workspace_id: "ws_test",
    ...overrides,
  };
}

function makeSecretBinding(overrides: Partial<BridgeSecretBinding> = {}): BridgeSecretBinding {
  return {
    binding_name: "bot_token",
    bridge_instance_id: "brg_support",
    created_at: "2026-04-13T12:00:00Z",
    kind: "bot_token",
    updated_at: "2026-04-13T12:10:00Z",
    secret_ref: "vault:bridges/brg_support/bot_token",
    ...overrides,
  };
}

const BridgesPage = (Route as unknown as { component: () => React.ReactNode }).component;

describe("BridgesPage", () => {
  beforeEach(() => {
    vi.useRealTimers();
    mockBridgesData = {
      bridge_health: {
        brg_support: makeHealth(),
      },
      bridges: [makeBridge()],
    };
    mockBridgesLoading = false;
    mockBridgesError = null;
    mockProvidersData = [makeProvider()];
    mockProvidersLoading = false;
    mockProvidersError = null;
    mockBridgeDetail = {
      bridge: makeBridge(),
      health: makeHealth(),
    };
    mockBridgeDetailLoading = false;
    mockBridgeDetailError = null;
    mockBridgeRoutes = [makeRoute()];
    mockBridgeRoutesLoading = false;
    mockBridgeRoutesError = null;
    mockSecretBindingsData = [makeSecretBinding()];
    mockSecretBindingsLoading = false;
    mockSecretBindingsError = null;
    mockCreateBridgePending = false;
    mockUpdateBridgePending = false;
    mockPutBridgeSecretBindingPending = false;
    mockDeleteBridgeSecretBindingPending = false;
    mockEnableBridgePending = false;
    mockDisableBridgePending = false;
    mockRestartBridgePending = false;
    mockTestDeliveryPending = false;
    mockActiveWorkspaceId = "ws_test";
    mockActiveWorkspaceName = "test-workspace";

    mockCreateBridgeMutateAsync.mockReset();
    mockUpdateBridgeMutateAsync.mockReset();
    mockPutBridgeSecretBindingMutateAsync.mockReset();
    mockDeleteBridgeSecretBindingMutateAsync.mockReset();
    mockEnableBridgeMutateAsync.mockReset();
    mockDisableBridgeMutateAsync.mockReset();
    mockRestartBridgeMutateAsync.mockReset();
    mockTestDeliveryMutateAsync.mockReset();
    toast.success.mockReset();
    toast.error.mockReset();

    mockCreateBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ id: "brg_created", status: "starting" }),
      health: makeHealth({ bridge_instance_id: "brg_created", status: "starting" }),
    } satisfies CreateBridgeResponse);
    mockTestDeliveryMutateAsync.mockResolvedValue({
      delivery_target: {
        bridge_instance_id: "brg_support",
        mode: "reply",
        peer_id: "peer_123",
      },
      message: "Ping",
      status: "resolved",
    } satisfies TestBridgeDeliveryResponse);
    mockUpdateBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ display_name: "Support Ops" }),
      health: makeHealth(),
    } satisfies UpdateBridgeResponse);
    mockPutBridgeSecretBindingMutateAsync.mockResolvedValue(makeSecretBinding());
    mockDeleteBridgeSecretBindingMutateAsync.mockResolvedValue(undefined);
    mockEnableBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ enabled: true, status: "starting" }),
      health: makeHealth({ status: "starting" }),
    } satisfies BridgeDetailResponse);
    mockDisableBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ enabled: false, status: "disabled" }),
      health: makeHealth({ status: "disabled" }),
    } satisfies BridgeDetailResponse);
    mockRestartBridgeMutateAsync.mockResolvedValue({
      bridge: makeBridge({ status: "starting" }),
      health: makeHealth({ status: "starting" }),
    } satisfies BridgeDetailResponse);
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
    mockBridgesData = {
      bridge_health: {},
      bridges: [],
    };

    render(<BridgesPage />);

    expect(screen.getByTestId("bridges-empty-state")).toBeInTheDocument();
    expect(screen.getByText("No bridges configured")).toBeInTheDocument();
    expect(screen.getByText("Telegram")).toBeInTheDocument();
  });

  it("renders the selected bridge detail and route list", () => {
    render(<BridgesPage />, { title: "Bridges" });

    const detailPanel = screen.getByTestId("bridge-detail-panel");

    expect(screen.getByText("Bridges")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-list-panel")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-item-brg_support")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Support")).toBeInTheDocument();
    expect(within(detailPanel).getByText("support-agent")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Open direct messages")).toBeInTheDocument();
    expect(within(detailPanel).getByTestId("bridge-detail-provider-config")).toHaveTextContent(
      '"mode": "bot"'
    );
    expect(within(detailPanel).getByTestId("bridge-detail-secret-slots")).toHaveTextContent(
      "bot_token"
    );
    expect(screen.getByTestId("bridge-route-sess_123")).toBeInTheDocument();
  });

  it("renders the no routes detail variant when the selected bridge has no routes", () => {
    mockBridgeRoutes = [];

    render(<BridgesPage />);

    expect(screen.getByTestId("bridge-routes-empty")).toHaveTextContent("No routes");
  });

  it("creates a bridge with provider config and shows the persisted values in the UI", async () => {
    const user = userEvent.setup();
    mockBridgesData = {
      bridge_health: {},
      bridges: [],
    };

    render(<BridgesPage />);

    await user.click(screen.getByTestId("bridge-empty-create-btn"));

    expect(screen.getByTestId("bridge-create-dialog")).toBeInTheDocument();

    await user.selectOptions(screen.getByTestId("bridge-dm-policy-select"), "allowlist");
    fireEvent.change(screen.getByTestId("bridge-provider-config-input"), {
      target: {
        value: '{"mode":"bot","webhook_url":"https://example.test/webhook"}',
      },
    });

    mockCreateBridgeMutateAsync.mockImplementationOnce(async payload => {
      const createdBridge = makeBridge({
        dm_policy: payload.dm_policy,
        id: "brg_created",
        provider_config: payload.provider_config,
        status: "starting",
      });

      mockBridgesData = {
        bridge_health: {
          brg_created: makeHealth({
            bridge_instance_id: "brg_created",
            status: "starting",
          }),
        },
        bridges: [createdBridge],
      };
      mockBridgeDetail = {
        bridge: createdBridge,
        health: makeHealth({
          bridge_instance_id: "brg_created",
          status: "starting",
        }),
      };
      mockBridgeRoutes = [];

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
      expect(mockCreateBridgeMutateAsync).toHaveBeenCalledWith({
        delivery_defaults: undefined,
        dm_policy: "allowlist",
        display_name: "Telegram",
        enabled: true,
        extension_name: "ext-telegram",
        platform: "telegram",
        provider_config: {
          mode: "bot",
          webhook_url: "https://example.test/webhook",
        },
        routing_policy: { include_group: true, include_peer: true, include_thread: true },
        scope: "workspace",
        workspace_id: "ws_test",
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId("bridge-detail-panel")).toHaveTextContent(
        "Allowlisted direct messages only"
      );
    });

    expect(screen.getByTestId("bridge-detail-provider-config")).toHaveTextContent(
      '"webhook_url": "https://example.test/webhook"'
    );
    expect(toast.success).toHaveBeenCalledWith("Created bridge Support.");
  });

  it("blocks workspace-scoped bridge creation when the active workspace disappears", async () => {
    const user = userEvent.setup();
    mockBridgesData = {
      bridge_health: {},
      bridges: [],
    };

    const { rerender } = render(<BridgesPage />);

    await user.click(screen.getByTestId("bridge-empty-create-btn"));

    mockActiveWorkspaceId = null;
    mockActiveWorkspaceName = "";
    rerender(<BridgesPage />);

    await user.click(screen.getByTestId("submit-bridge-create"));

    expect(mockCreateBridgeMutateAsync).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith(
      "Select an active workspace before creating a workspace-scoped bridge."
    );
  });

  it("opens test delivery and shows the resolved target result", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.click(screen.getByTestId("open-test-delivery-btn"));

    expect(screen.getByTestId("bridge-test-delivery-dialog")).toBeInTheDocument();

    await user.clear(screen.getByTestId("test-delivery-message"));
    await user.type(screen.getByTestId("test-delivery-message"), "Ping");
    await user.click(screen.getByTestId("submit-test-delivery"));

    await waitFor(() => {
      expect(mockTestDeliveryMutateAsync).toHaveBeenCalledWith({
        data: {
          message: "Ping",
          target: {
            bridge_instance_id: "brg_support",
          },
        },
        id: "brg_support",
      });
    });

    expect(screen.getByTestId("bridge-test-delivery-result")).toHaveTextContent("peer:peer_123");
    expect(toast.success).toHaveBeenCalledWith("Resolved delivery target for Support.");
  });

  it("edits mutable bridge fields and prompts for restart", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.click(screen.getByTestId("edit-bridge-btn"));

    expect(screen.getByTestId("bridge-edit-dialog")).toBeInTheDocument();

    await user.clear(screen.getByTestId("bridge-edit-display-name-input"));
    await user.type(screen.getByTestId("bridge-edit-display-name-input"), "Support Ops");
    await user.click(screen.getByTestId("submit-bridge-edit"));

    await waitFor(() => {
      expect(mockUpdateBridgeMutateAsync).toHaveBeenCalledWith({
        data: {
          delivery_defaults: null,
          display_name: "Support Ops",
          dm_policy: "open",
          provider_config: {
            mode: "bot",
            webhook_url: "https://example.test/webhook",
          },
          routing_policy: { include_group: true, include_peer: true, include_thread: true },
        },
        id: "brg_support",
      });
    });

    expect(toast.success).toHaveBeenCalledWith(
      "Updated bridge Support Ops. Restart to apply changes."
    );
    expect(screen.getByTestId("bridge-restart-required")).toBeInTheDocument();
  });

  it("writes secret bindings and clears the restart hint after restart", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.clear(screen.getByTestId("bridge-secret-env-input-bot_token"));
    await user.type(screen.getByTestId("bridge-secret-env-input-bot_token"), "telegram-token");
    await user.click(screen.getByTestId("save-bridge-secret-bot_token"));

    await waitFor(() => {
      expect(mockPutBridgeSecretBindingMutateAsync).toHaveBeenCalledWith({
        bindingName: "bot_token",
        data: {
          kind: "bot_token",
          secret_ref: "vault:bridges/brg_support/bot_token",
          secret_value: "telegram-token",
        },
        id: "brg_support",
      });
    });

    expect(screen.getByTestId("bridge-restart-required")).toBeInTheDocument();

    await user.click(screen.getByTestId("restart-bridge-btn"));

    await waitFor(() => {
      expect(mockRestartBridgeMutateAsync).toHaveBeenCalledWith({
        id: "brg_support",
      });
    });

    expect(toast.success).toHaveBeenCalledWith("Restarted bridge Support.");
    expect(screen.queryByTestId("bridge-restart-required")).not.toBeInTheDocument();
  });

  it("disables the selected bridge", async () => {
    const user = userEvent.setup();
    render(<BridgesPage />);

    await user.click(screen.getByTestId("disable-bridge-btn"));
    await waitFor(() => {
      expect(mockDisableBridgeMutateAsync).toHaveBeenCalledWith({
        id: "brg_support",
      });
    });
    expect(toast.success).toHaveBeenCalledWith("Disabled bridge Support.");
  });
});
