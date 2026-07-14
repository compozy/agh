import * as React from "react";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";

import type {
  BridgeDetailResponse,
  BridgeProvider,
  BridgeResolveTargetResponse,
  BridgeRoute,
  BridgeSecretBinding,
  BridgeTargetsResponse,
  BridgeVerifyResponse,
  BridgeWebhookRegistrationResponse,
  BridgesListResponse,
  SendBridgeTestResponse,
  TestBridgeDeliveryResponse,
  UpdateBridgeResponse,
} from "@/systems/bridges";

function render(ui: React.ReactElement) {
  return renderWithTopbar(ui, { title: "brg_support" });
}

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    info: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  },
}));

let mockBridgesData: BridgesListResponse | undefined;
let mockBridgeDetail: BridgeDetailResponse | undefined;
let mockBridgeDetailLoading = false;
let mockBridgeDetailError: Error | null = null;
let mockBridgeRoutes: BridgeRoute[] | undefined;
let mockBridgeRoutesLoading = false;
let mockBridgeRoutesError: Error | null = null;
let mockBridgeTargets: BridgeTargetsResponse | undefined;
let mockBridgeTargetsLoading = false;
let mockBridgeTargetsError: Error | null = null;
let mockSecretBindingsData: BridgeSecretBinding[] | undefined;
let mockSecretBindingsLoading = false;
let mockSecretBindingsError: Error | null = null;
let mockProvidersData: BridgeProvider[] | undefined;
let mockProvidersError: Error | null = null;
let mockProvidersLoading = false;

const mockUpdateBridgeMutateAsync = vi.fn();
const mockPutBridgeSecretBindingMutateAsync = vi.fn();
const mockDeleteBridgeSecretBindingMutateAsync = vi.fn();
const mockEnableBridgeMutateAsync = vi.fn();
const mockDisableBridgeMutateAsync = vi.fn();
const mockRestartBridgeMutateAsync = vi.fn();
const mockResolveBridgeTargetMutateAsync = vi.fn();
const mockTestDeliveryMutateAsync = vi.fn();
const mockSendTestMutateAsync = vi.fn();
const mockVerifyBridgeMutateAsync = vi.fn();
const mockRegisterWebhookMutateAsync = vi.fn();
let mockUpdateBridgePending = false;
let mockPutBridgeSecretBindingPending = false;
let mockDeleteBridgeSecretBindingPending = false;
let mockEnableBridgePending = false;
let mockDisableBridgePending = false;
let mockRestartBridgePending = false;
let mockResolveBridgeTargetPending = false;
let mockTestDeliveryPending = false;
let mockSendTestPending = false;
let mockVerifyBridgePending = false;
let mockRegisterWebhookPending = false;

const routerState = vi.hoisted(() => ({
  navigateMock: vi.fn(),
  params: { id: "brg_support" },
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
    useParams: () => routerState.params,
  }),
  useNavigate: () => routerState.navigateMock,
}));

vi.mock("sonner", () => ({
  toast,
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: [
      {
        add_dirs: [],
        created_at: "2026-04-03T12:00:00Z",
        id: "ws_test",
        name: "test-workspace",
        root_dir: "/workspace",
        updated_at: "2026-04-03T12:00:00Z",
      },
    ],
    hasWorkspaces: true,
    activeWorkspace: {
      add_dirs: [],
      created_at: "2026-04-03T12:00:00Z",
      id: "ws_test",
      name: "test-workspace",
      root_dir: "/workspace",
      updated_at: "2026-04-03T12:00:00Z",
    },
    activeWorkspaceId: "ws_test",
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
      error: null,
      isLoading: false,
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
    useBridgeTargets: () => ({
      data: mockBridgeTargets,
      error: mockBridgeTargetsError,
      isLoading: mockBridgeTargetsLoading,
    }),
    useBridgeSecretBindings: () => ({
      data: mockSecretBindingsData,
      error: mockSecretBindingsError,
      isLoading: mockSecretBindingsLoading,
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
    useResolveBridgeTarget: () => ({
      isPending: mockResolveBridgeTargetPending,
      mutateAsync: mockResolveBridgeTargetMutateAsync,
    }),
    useTestBridgeDelivery: () => ({
      isPending: mockTestDeliveryPending,
      mutateAsync: mockTestDeliveryMutateAsync,
    }),
    useSendBridgeTest: () => ({
      isPending: mockSendTestPending,
      mutateAsync: mockSendTestMutateAsync,
    }),
    useVerifyBridge: () => ({
      isPending: mockVerifyBridgePending,
      mutateAsync: mockVerifyBridgeMutateAsync,
    }),
    useRegisterBridgeWebhook: () => ({
      isPending: mockRegisterWebhookPending,
      mutateAsync: mockRegisterWebhookMutateAsync,
    }),
  };
});

import { Route } from "../bridges.$id";

const BridgeDetailPage = routeComponent(Route);

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
    webhook_public_url: "https://example.test/webhook",
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

function makeTargets(
  overrides: Partial<BridgeTargetsResponse["targets"][number]> = {}
): BridgeTargetsResponse {
  const target = {
    bridge_id: "brg_support",
    canonical_route: "telegram:channel:support",
    capabilities: ["direct-send", "reply"],
    display_name: "Support room",
    last_seen_at: "2026-04-13T12:16:00Z",
    normalized: "support room",
    qualifier: "telegram",
    target_type: "channel",
    updated_at: "2026-04-13T12:16:00Z",
    ...overrides,
  };

  return {
    bridge_id: "brg_support",
    cache_stale: false,
    generated_at: "2026-04-13T12:16:00Z",
    last_successful_refresh_at: "2026-04-13T12:16:00Z",
    targets: [target],
    total: 1,
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

describe("BridgeDetailRoute", () => {
  beforeEach(() => {
    routerState.params = { id: "brg_support" };
    routerState.navigateMock.mockReset();
    mockBridgesData = {
      bridge_health: {
        brg_support: makeHealth(),
      },
      bridges: [makeBridge()],
      facets: {
        platforms: { telegram: 1 },
        statuses: {
          auth_required: 0,
          degraded: 0,
          disabled: 0,
          error: 0,
          ready: 1,
          starting: 0,
        },
      },
      page: { has_more: false, limit: 50, total: 1 },
    };
    mockBridgeDetail = {
      bridge: makeBridge(),
      health: makeHealth(),
    };
    mockBridgeDetailLoading = false;
    mockBridgeDetailError = null;
    mockBridgeRoutes = [makeRoute()];
    mockBridgeRoutesLoading = false;
    mockBridgeRoutesError = null;
    mockBridgeTargets = makeTargets();
    mockBridgeTargetsLoading = false;
    mockBridgeTargetsError = null;
    mockSecretBindingsData = [makeSecretBinding()];
    mockSecretBindingsLoading = false;
    mockSecretBindingsError = null;
    mockProvidersData = [makeProvider()];
    mockProvidersError = null;
    mockProvidersLoading = false;
    mockUpdateBridgePending = false;
    mockPutBridgeSecretBindingPending = false;
    mockDeleteBridgeSecretBindingPending = false;
    mockEnableBridgePending = false;
    mockDisableBridgePending = false;
    mockRestartBridgePending = false;
    mockResolveBridgeTargetPending = false;
    mockTestDeliveryPending = false;
    mockSendTestPending = false;
    mockVerifyBridgePending = false;
    mockRegisterWebhookPending = false;

    mockUpdateBridgeMutateAsync.mockReset();
    mockPutBridgeSecretBindingMutateAsync.mockReset();
    mockDeleteBridgeSecretBindingMutateAsync.mockReset();
    mockEnableBridgeMutateAsync.mockReset();
    mockDisableBridgeMutateAsync.mockReset();
    mockRestartBridgeMutateAsync.mockReset();
    mockResolveBridgeTargetMutateAsync.mockReset();
    mockTestDeliveryMutateAsync.mockReset();
    mockSendTestMutateAsync.mockReset();
    mockVerifyBridgeMutateAsync.mockReset();
    mockRegisterWebhookMutateAsync.mockReset();
    toast.success.mockReset();
    toast.error.mockReset();
    toast.info.mockReset();
    toast.warning.mockReset();
    routerState.params.id = "brg_support";

    mockTestDeliveryMutateAsync.mockResolvedValue({
      delivery_target: {
        bridge_instance_id: "brg_support",
        mode: "reply",
        peer_id: "peer_123",
      },
      message: "Ping",
      status: "resolved",
    } satisfies TestBridgeDeliveryResponse);
    mockSendTestMutateAsync.mockResolvedValue({
      bridge_instance_id: "brg_support",
      delivery_id: "delivery_test_support",
      delivery_target: {
        bridge_instance_id: "brg_support",
        mode: "reply",
        peer_id: "peer_123",
      },
      remote_message_id: "remote_test_support",
      status: "delivered",
    } satisfies SendBridgeTestResponse);
    mockVerifyBridgeMutateAsync.mockResolvedValue({
      bridge_instance_id: "brg_support",
      checks: [
        { check: "provider.identity", remediation: "", status: "pass" },
        {
          check: "webhook.secret",
          remediation: "Bind webhook_secret, then verify again.",
          status: "fail",
        },
      ],
    } satisfies BridgeVerifyResponse);
    mockRegisterWebhookMutateAsync.mockResolvedValue({
      bridge_instance_id: "brg_support",
      remediation: "",
      status: "pass",
    } satisfies BridgeWebhookRegistrationResponse);
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
    mockResolveBridgeTargetMutateAsync.mockResolvedValue({
      result: {
        ambiguous: false,
        match: makeTargets().targets[0],
        step: 2,
      },
    } satisfies BridgeResolveTargetResponse);
  });

  it("renders bridge detail with back navigation", () => {
    render(<BridgeDetailPage />);

    const detailPanel = screen.getByTestId("bridge-detail-panel");
    expect(within(detailPanel).getByText("Support")).toBeInTheDocument();
    expect(within(detailPanel).getByText("support-agent")).toBeInTheDocument();
    expect(screen.getByTestId("bridge-route-sess_123")).toBeInTheDocument();
    expect(screen.getByLabelText("Back to bridges")).toBeInTheDocument();
  });

  it("navigates back to the bridges list", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.click(screen.getByLabelText("Back to bridges"));
    expect(routerState.navigateMock).toHaveBeenCalledWith({ to: "/bridges" });
  });

  it("renders the no routes detail variant when the bridge has no routes", () => {
    mockBridgeRoutes = [];

    render(<BridgeDetailPage />);

    expect(screen.getByTestId("bridge-routes-empty")).toHaveTextContent("No routes");
  });

  it("keeps setup facts unknown while provider or binding reads are unavailable", () => {
    mockProvidersData = undefined;
    mockProvidersError = new Error("Provider catalog unavailable");
    const view = render(<BridgeDetailPage />);

    expect(screen.getByTestId("bridge-provider-runtime-unavailable")).toBeInTheDocument();
    expect(screen.getByText("Setup status is not available yet.")).toBeInTheDocument();
    expect(screen.queryByText("UNBOUND")).not.toBeInTheDocument();

    mockProvidersData = [makeProvider()];
    mockProvidersError = null;
    mockSecretBindingsData = undefined;
    mockSecretBindingsError = new Error("Secret bindings unavailable");
    view.rerender(<BridgeDetailPage />);

    expect(screen.getByTestId("bridge-secret-bindings-unavailable")).toBeInTheDocument();
    expect(screen.getByText("Setup status is not available yet.")).toBeInTheDocument();
    expect(screen.queryByText("UNBOUND")).not.toBeInTheDocument();
  });

  it("opens test delivery and shows the resolved target result", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

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

  it("sends a real test message separately from the dry-run resolver", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("open-send-test-btn"));
    expect(screen.getByTestId("bridge-send-test-dialog")).toBeInTheDocument();
    await user.type(screen.getByTestId("test-delivery-message"), "Provider ping");
    await user.click(screen.getByTestId("submit-send-test"));

    await waitFor(() => {
      expect(mockSendTestMutateAsync).toHaveBeenCalledWith({
        data: {
          message: "Provider ping",
          target: {
            bridge_instance_id: "brg_support",
          },
        },
        id: "brg_support",
      });
    });
    expect(mockTestDeliveryMutateAsync).not.toHaveBeenCalled();
    expect(screen.getByTestId("bridge-send-test-result")).toHaveTextContent(
      "delivery_test_support"
    );
    expect(screen.getByTestId("bridge-send-test-result")).toHaveTextContent("remote_test_support");
    expect(toast.success).toHaveBeenCalledWith("Sent a test message through Support.");
  });

  it("runs verification and renders each check on its mapped secret card", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("verify-bridge-btn"));
    await waitFor(() => {
      expect(mockVerifyBridgeMutateAsync).toHaveBeenCalledWith({ id: "brg_support" });
    });

    expect(screen.getByTestId("bridge-secret-check-bot_token-provider.identity")).toHaveTextContent(
      "PASS"
    );
    expect(
      screen.getByTestId("bridge-secret-check-webhook_secret-webhook.secret")
    ).toHaveTextContent("Bind webhook_secret, then verify again.");
    expect(screen.getByTestId("bridge-setup-item-verified")).toHaveTextContent("FAILED");
    expect(toast.error).toHaveBeenCalledWith("Verification found setup issues for Support.");

    await user.type(
      screen.getByTestId("bridge-secret-env-input-webhook_secret"),
      "telegram-webhook-secret"
    );
    await user.click(screen.getByTestId("save-bridge-secret-webhook_secret"));
    await waitFor(() => {
      expect(mockPutBridgeSecretBindingMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({ bindingName: "webhook_secret", id: "brg_support" })
      );
    });
    expect(
      screen.queryByTestId("bridge-secret-check-bot_token-provider.identity")
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("bridge-setup-item-verified")).toHaveTextContent("ACTION NEEDED");
  });

  it.each([
    ["warn", "warning", "Verification completed with warnings for Support."],
    ["skipped", "info", "Verification checks were skipped for Support."],
  ] as const)(
    "reports a %s verification without a false success",
    async (status, tone, message) => {
      mockVerifyBridgeMutateAsync.mockResolvedValueOnce({
        bridge_instance_id: "brg_support",
        checks: [{ check: "provider.identity", remediation: "Check provider state.", status }],
      } satisfies BridgeVerifyResponse);
      const user = userEvent.setup();
      render(<BridgeDetailPage />);

      await user.click(screen.getByTestId("verify-bridge-btn"));

      await waitFor(() => expect(toast[tone]).toHaveBeenCalledWith(message));
      expect(toast.success).not.toHaveBeenCalled();
    }
  );

  it("invalidates verification evidence after an external provider config change", async () => {
    mockVerifyBridgeMutateAsync.mockResolvedValueOnce({
      bridge_instance_id: "brg_support",
      checks: [{ check: "provider.identity", remediation: "", status: "pass" }],
    } satisfies BridgeVerifyResponse);
    const user = userEvent.setup();
    const view = render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("verify-bridge-btn"));
    expect(
      await screen.findByTestId("bridge-secret-check-bot_token-provider.identity")
    ).toBeInTheDocument();

    mockBridgeDetail = {
      bridge: makeBridge({
        provider_config: {
          mode: "bot",
          webhook: { public_url: "https://example.test/rotated-webhook" },
        },
        updated_at: "2026-04-13T13:00:00Z",
      }),
      health: makeHealth(),
    };
    view.rerender(<BridgeDetailPage />);

    expect(
      screen.queryByTestId("bridge-secret-check-bot_token-provider.identity")
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("bridge-setup-item-verified")).toHaveTextContent("ACTION NEEDED");
  });

  it("discards a late verification response after a secret mutation", async () => {
    let resolveVerification!: (result: BridgeVerifyResponse) => void;
    mockVerifyBridgeMutateAsync.mockReturnValueOnce(
      new Promise<BridgeVerifyResponse>(resolve => {
        resolveVerification = resolve;
      })
    );
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("verify-bridge-btn"));
    await user.type(
      screen.getByTestId("bridge-secret-env-input-webhook_secret"),
      "rotated-webhook-secret"
    );
    await user.click(screen.getByTestId("save-bridge-secret-webhook_secret"));
    await waitFor(() => expect(mockPutBridgeSecretBindingMutateAsync).toHaveBeenCalledOnce());

    await act(async () => {
      resolveVerification({
        bridge_instance_id: "brg_support",
        checks: [{ check: "provider.identity", remediation: "", status: "pass" }],
      });
    });

    expect(
      screen.queryByTestId("bridge-secret-check-bot_token-provider.identity")
    ).not.toBeInTheDocument();
    expect(toast.success).not.toHaveBeenCalledWith("Verified bridge Support.");
  });

  it("discards a late verification response after navigating to another bridge", async () => {
    let resolveVerification!: (result: BridgeVerifyResponse) => void;
    mockVerifyBridgeMutateAsync.mockReturnValueOnce(
      new Promise<BridgeVerifyResponse>(resolve => {
        resolveVerification = resolve;
      })
    );
    const user = userEvent.setup();
    const view = render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("verify-bridge-btn"));
    routerState.params.id = "brg_other";
    mockBridgeDetail = {
      bridge: makeBridge({ display_name: "Other", id: "brg_other" }),
      health: makeHealth({ bridge_instance_id: "brg_other" }),
    };
    view.rerender(<BridgeDetailPage />);

    await act(async () => {
      resolveVerification({
        bridge_instance_id: "brg_support",
        checks: [{ check: "provider.identity", remediation: "", status: "pass" }],
      });
    });

    expect(toast.success).not.toHaveBeenCalledWith("Verified bridge Support.");
    expect(screen.getByTestId("bridge-detail-panel")).toHaveTextContent("Other");
  });

  it("preserves Telegram registration across lifecycle and non-provider edits", async () => {
    mockBridgeDetail = {
      bridge: makeBridge({ enabled: false, status: "disabled" }),
      health: makeHealth({ status: "disabled" }),
    };
    const user = userEvent.setup();
    const view = render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("register-bridge-webhook-btn"));
    await waitFor(() => {
      expect(mockRegisterWebhookMutateAsync).toHaveBeenCalledWith({ id: "brg_support" });
    });

    expect(screen.getByTestId("bridge-setup-item-webhook")).toHaveTextContent("COMPLETE");
    expect(toast.success).toHaveBeenCalledWith("Registered the webhook for Support.");

    await user.click(screen.getByTestId("setup-enable-bridge-btn"));
    await waitFor(() => {
      expect(mockEnableBridgeMutateAsync).toHaveBeenCalledWith({ id: "brg_support" });
    });
    mockBridgeDetail = {
      bridge: makeBridge({ enabled: true, status: "ready", updated_at: "2026-04-13T13:00:00Z" }),
      health: makeHealth({ status: "ready" }),
    };
    mockProvidersData = [
      makeProvider({ health: "degraded", health_message: "Provider health is refreshing." }),
    ];
    view.rerender(<BridgeDetailPage />);

    expect(screen.getByTestId("bridge-setup-item-webhook")).toHaveTextContent("COMPLETE");
    expect(screen.getByTestId("register-bridge-webhook-btn")).toHaveTextContent("Disable first");

    await user.click(screen.getByTestId("verify-bridge-btn"));
    await waitFor(() => expect(mockVerifyBridgeMutateAsync).toHaveBeenCalledOnce());
    expect(screen.getByTestId("bridge-setup-item-webhook")).toHaveTextContent("COMPLETE");

    const updatedBridge = makeBridge({
      delivery_defaults: {
        progress: {
          grouping: "accumulate",
          tool_progress: "all",
        },
      },
      display_name: "Support Ops",
      provider_config: {
        webhook: { public_url: "https://example.test/webhook" },
        mode: "bot",
      },
      updated_at: "2026-04-13T13:15:00Z",
    });
    mockUpdateBridgeMutateAsync.mockResolvedValueOnce({
      bridge: updatedBridge,
      health: makeHealth({ status: "ready" }),
    } satisfies UpdateBridgeResponse);

    await user.click(screen.getByTestId("edit-bridge-btn"));
    await user.clear(screen.getByTestId("bridge-edit-display-name-input"));
    await user.type(screen.getByTestId("bridge-edit-display-name-input"), "Support Ops");
    await user.selectOptions(
      screen.getByTestId("bridge-edit-delivery-progress-mode-select"),
      "all"
    );
    await user.click(screen.getByTestId("submit-bridge-edit"));
    await waitFor(() => expect(mockUpdateBridgeMutateAsync).toHaveBeenCalledOnce());

    mockBridgeDetail = {
      bridge: updatedBridge,
      health: makeHealth({ status: "ready" }),
    };
    view.rerender(<BridgeDetailPage />);

    expect(screen.getByTestId("bridge-setup-item-webhook")).toHaveTextContent("COMPLETE");
    expect(screen.getByTestId("register-bridge-webhook-btn")).toHaveTextContent("Disable first");
  });

  it("reports uncertain Telegram registration without claiming completion", async () => {
    mockBridgeDetail = {
      bridge: makeBridge({ enabled: false, status: "disabled" }),
      health: makeHealth({ status: "disabled" }),
    };
    mockRegisterWebhookMutateAsync.mockResolvedValueOnce({
      bridge_instance_id: "brg_support",
      remediation: "Provider acceptance timed out; verify before enabling.",
      status: "warn",
    } satisfies BridgeWebhookRegistrationResponse);
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("register-bridge-webhook-btn"));

    await waitFor(() =>
      expect(toast.warning).toHaveBeenCalledWith(
        "Provider acceptance timed out; verify before enabling."
      )
    );
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("resets route-local drafts and dialogs when the bridge id changes", async () => {
    const user = userEvent.setup();
    const view = render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("open-send-test-btn"));
    await user.type(screen.getByTestId("test-delivery-message"), "Bridge A draft");
    expect(screen.getByTestId("bridge-send-test-dialog")).toBeInTheDocument();

    routerState.params.id = "brg_other";
    mockBridgeDetail = {
      bridge: makeBridge({ display_name: "Other", id: "brg_other" }),
      health: makeHealth({ bridge_instance_id: "brg_other" }),
    };
    view.rerender(<BridgeDetailPage />);

    expect(screen.queryByTestId("bridge-send-test-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("bridge-detail-panel")).toHaveTextContent("Other");
  });

  it("resolves bridge target names from the target directory section", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

    await user.type(screen.getByTestId("bridge-target-resolve-input"), "Support room");
    await user.click(screen.getByTestId("bridge-target-resolve-submit"));

    await waitFor(() => {
      expect(mockResolveBridgeTargetMutateAsync).toHaveBeenCalledWith({
        data: { name: "Support room" },
        id: "brg_support",
      });
    });

    expect(screen.getByTestId("bridge-target-resolve-result")).toHaveTextContent(
      "telegram:channel:support"
    );
    expect(toast.success).toHaveBeenCalledWith("Resolved target Support room.");
  });

  it("edits mutable bridge fields and prompts for restart", async () => {
    const user = userEvent.setup();
    render(<BridgeDetailPage />);

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
            webhook: {
              public_url: "https://example.test/webhook",
            },
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
    render(<BridgeDetailPage />);

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
    render(<BridgeDetailPage />);

    await user.click(screen.getByTestId("disable-bridge-btn"));
    await waitFor(() => {
      expect(mockDisableBridgeMutateAsync).toHaveBeenCalledWith({
        id: "brg_support",
      });
    });
  });
});
