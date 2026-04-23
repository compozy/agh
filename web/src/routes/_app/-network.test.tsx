import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  CreateNetworkChannelResponse,
  NetworkChannel,
  NetworkChannelMessage,
  NetworkChannelsResponse,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkStatus,
} from "@/systems/network";

const routerState = vi.hoisted(() => ({
  searchParams: {} as Record<string, unknown>,
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => Record<string, unknown>)
    | undefined,
}));

const { toast } = vi.hoisted(() => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

let mockActiveWorkspaceId: string | null = "ws_main";
let mockActiveWorkspaceName = "Polybot";

let mockWorkspaceAgents = [
  { name: "polybot-main", prompt: "coordinate", provider: "anthropic" },
  { name: "coder-agent-01", prompt: "code", provider: "openai" },
];

let mockNetworkStatus: NetworkStatus | undefined;
let mockNetworkStatusLoading = false;
let mockNetworkStatusError: Error | null = null;
let mockNetworkChannels: NetworkChannelsResponse | undefined;
let mockNetworkChannelsLoading = false;
let mockNetworkChannelsError: Error | null = null;
let mockChannelDetail: NetworkChannel | undefined;
let mockChannelDetailLoading = false;
let mockChannelDetailError: Error | null = null;
let mockChannelMessages: NetworkChannelMessage[] | undefined;
let mockChannelMessagesLoading = false;
let mockChannelMessagesError: Error | null = null;
let mockNetworkPeers: NetworkPeerSummary[] | undefined;
let mockNetworkPeersLoading = false;
let mockNetworkPeersError: Error | null = null;
let mockPeerDetail: NetworkPeerDetail | undefined;
let mockPeerDetailLoading = false;
let mockPeerDetailError: Error | null = null;
let mockPeerMessages: NetworkChannelMessage[] | undefined;
let mockPeerMessagesLoading = false;
let mockPeerMessagesError: Error | null = null;

const mockCreateNetworkChannelMutateAsync = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let mockCreateNetworkChannelPending = false;
const mockSendNetworkMessageMutateAsync = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let mockSendNetworkMessagePending = false;

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => {
      routerState.validateSearch = opts.validateSearch;
      return {
        component: opts.component,
        useSearch: () =>
          routerState.validateSearch
            ? routerState.validateSearch(routerState.searchParams)
            : routerState.searchParams,
      };
    },
  useNavigate:
    () =>
    async (opts: {
      search?:
        | Record<string, unknown>
        | ((current: Record<string, unknown>) => Record<string, unknown>);
    }) => {
      if (!opts.search) {
        return;
      }

      const current = routerState.validateSearch
        ? routerState.validateSearch(routerState.searchParams)
        : routerState.searchParams;
      routerState.searchParams =
        typeof opts.search === "function" ? opts.search(current) : opts.search;
    },
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
            created_at: "2026-04-13T12:00:00Z",
            id: mockActiveWorkspaceId,
            name: mockActiveWorkspaceName,
            root_dir: "/workspace/polybot",
            updated_at: "2026-04-13T12:00:00Z",
          },
        ]
      : [],
    hasWorkspaces: Boolean(mockActiveWorkspaceId),
    activeWorkspace: mockActiveWorkspaceId
      ? {
          add_dirs: [],
          created_at: "2026-04-13T12:00:00Z",
          id: mockActiveWorkspaceId,
          name: mockActiveWorkspaceName,
          root_dir: "/workspace/polybot",
          updated_at: "2026-04-13T12:00:00Z",
        }
      : undefined,
    activeWorkspaceId: mockActiveWorkspaceId,
    clearActiveWorkspaceSelection: vi.fn(),
    isError: false,
    isLoading: false,
    setActiveWorkspaceId: vi.fn(),
  }),
  useWorkspace: () => ({
    data: mockActiveWorkspaceId
      ? {
          agents: mockWorkspaceAgents,
          sessions: [],
          skills: [],
          workspace: {
            add_dirs: [],
            created_at: "2026-04-13T12:00:00Z",
            id: mockActiveWorkspaceId,
            name: mockActiveWorkspaceName,
            root_dir: "/workspace/polybot",
            updated_at: "2026-04-13T12:00:00Z",
          },
        }
      : undefined,
    error: null,
    isLoading: false,
  }),
}));

vi.mock("@/systems/agent", () => ({
  AgentIcon: ({ provider }: { provider: string }) => (
    <span data-testid={`agent-icon-${provider}`} />
  ),
}));

vi.mock("@/systems/network", async () => {
  const actual = await vi.importActual("@/systems/network");

  return {
    ...actual,
    useNetworkStatus: () => ({
      data: mockNetworkStatus,
      error: mockNetworkStatusError,
      isLoading: mockNetworkStatusLoading,
      refetch: vi.fn(),
    }),
    useNetworkChannels: () => ({
      data: mockNetworkChannels,
      error: mockNetworkChannelsError,
      isLoading: mockNetworkChannelsLoading,
      refetch: vi.fn(),
    }),
    useNetworkChannel: () => ({
      data: mockChannelDetail,
      error: mockChannelDetailError,
      isLoading: mockChannelDetailLoading,
      refetch: vi.fn(),
    }),
    useNetworkChannelMessages: () => ({
      data: mockChannelMessages,
      error: mockChannelMessagesError,
      isLoading: mockChannelMessagesLoading,
      refetch: vi.fn(),
    }),
    useNetworkPeers: () => ({
      data: mockNetworkPeers,
      error: mockNetworkPeersError,
      isLoading: mockNetworkPeersLoading,
      refetch: vi.fn(),
    }),
    useNetworkPeer: () => ({
      data: mockPeerDetail,
      error: mockPeerDetailError,
      isLoading: mockPeerDetailLoading,
      refetch: vi.fn(),
    }),
    useNetworkPeerMessages: () => ({
      data: mockPeerMessages,
      error: mockPeerMessagesError,
      isLoading: mockPeerMessagesLoading,
      refetch: vi.fn(),
    }),
    useCreateNetworkChannel: () => ({
      isPending: mockCreateNetworkChannelPending,
      mutateAsync: mockCreateNetworkChannelMutateAsync,
    }),
    useSendNetworkMessage: () => ({
      isPending: mockSendNetworkMessagePending,
      mutateAsync: mockSendNetworkMessageMutateAsync,
    }),
  };
});

import { Route } from "./network";

const NetworkPage = (Route as unknown as { component: () => ReactNode }).component;

function makeChannelSummary(
  overrides: Partial<NetworkChannelsResponse["channels"][number]> = {}
): NetworkChannelsResponse["channels"][number] {
  return {
    channel: "coord.core",
    created_at: "2026-04-13T09:00:00Z",
    created_by: "polybot-main",
    last_activity_at: "2026-04-13T10:45:00Z",
    last_message_preview: "Dispatching task deploy-api-v2.3 to coder-agent-01",
    local_peer_count: 1,
    message_count: 6,
    peer_count: 2,
    purpose: "Coordinate release handoffs and verification.",
    remote_peer_count: 1,
    session_count: 1,
    workspace_id: "ws_main",
    ...overrides,
  };
}

function makePeerSummary(overrides: Partial<NetworkPeerSummary> = {}): NetworkPeerSummary {
  return {
    channel: "coord.core",
    display_name: "Remote Reviewer",
    joined_at: "2026-04-13T09:10:00Z",
    last_seen: "2026-04-13T10:40:00Z",
    local: false,
    peer_card: {
      artifacts_supported: ["capability"],
      capabilities: [{ id: "chat", summary: "Coordinates review handoffs." }],
      display_name: "Remote Reviewer",
      peer_id: "peer_remote",
      profiles_supported: ["default"],
      trust_modes_supported: ["relay"],
    },
    peer_id: "peer_remote",
    session_id: "sess_remote",
    ...overrides,
  };
}

function makeChannelMessage(overrides: Partial<NetworkChannelMessage> = {}): NetworkChannelMessage {
  return {
    body: { text: "Dispatching task deploy-api-v2.3 to coder-agent-01" },
    channel: "coord.core",
    direction: "sent",
    display_name: "Polybot Main",
    kind: "say",
    local: true,
    message_id: "msg_1",
    peer_from: "peer_local",
    preview_text: "Dispatching task deploy-api-v2.3 to coder-agent-01",
    session_id: "sess_local",
    text: "Dispatching task deploy-api-v2.3 to coder-agent-01",
    timestamp: "2026-04-13T10:42:00Z",
    ...overrides,
  };
}

function makeChannelDetail(overrides: Partial<NetworkChannel> = {}): NetworkChannel {
  return {
    channel: "coord.core",
    created_at: "2026-04-13T09:00:00Z",
    created_by: "polybot-main",
    kind_counts: [
      { kind: "say", count: 1 },
      { kind: "direct", count: 1 },
    ],
    last_activity_at: "2026-04-13T10:45:00Z",
    last_message_preview: "Dispatching task deploy-api-v2.3 to coder-agent-01",
    local_peer_count: 1,
    message_count: 2,
    peer_count: 2,
    peers: [
      makePeerSummary({
        display_name: "Polybot Main",
        local: true,
        peer_card: {
          artifacts_supported: ["capability"],
          capabilities: [{ id: "chat", summary: "Coordinates planning." }],
          display_name: "Polybot Main",
          peer_id: "peer_local",
          profiles_supported: ["default"],
          trust_modes_supported: ["local-first"],
        },
        peer_id: "peer_local",
        session_id: "sess_local",
      }),
      makePeerSummary(),
    ],
    purpose: "Coordinate release handoffs and verification.",
    remote_peer_count: 1,
    session_count: 1,
    sessions: [
      {
        acp_caps: {
          supports_load_session: true,
          supported_models: ["gpt-5.4"],
          supported_modes: ["chat"],
        },
        agent_name: "polybot-main",
        channel: "coord.core",
        created_at: "2026-04-13T09:00:00Z",
        id: "sess_local",
        name: "Polybot Main",
        state: "active",
        updated_at: "2026-04-13T10:45:00Z",
        workspace_id: "ws_main",
        workspace_path: "/workspace/polybot",
      },
    ],
    workspace_id: "ws_main",
    ...overrides,
  };
}

function makePeerDetail(overrides: Partial<NetworkPeerDetail> = {}): NetworkPeerDetail {
  return {
    channel: "coord.core",
    display_name: "Remote Reviewer",
    joined_at: "2026-04-13T09:10:00Z",
    last_seen: "2026-04-13T10:40:00Z",
    local: false,
    metrics: {
      delivered: 2,
      received: 3,
      rejected: 0,
      sent: 1,
    },
    peer_card: makePeerSummary().peer_card,
    capability_catalog: {
      capabilities: [
        {
          artifacts_expected: ["message.text"],
          context_needed: ["channel-history"],
          digest: "sha256:chat",
          execution_outline: ["Review the request", "Send a direct acknowledgement"],
          id: "chat",
          outcome: "Converge on the next review action.",
          summary: "Coordinates review handoffs.",
          version: "1.0.0",
        },
      ],
    },
    peer_id: "peer_remote",
    session_id: "sess_remote",
    ...overrides,
  };
}

function resetMocks() {
  routerState.searchParams = {};

  mockNetworkStatus = {
    channels: 2,
    delivery_workers: 2,
    enabled: true,
    local_peers: 1,
    messages_sent: 12,
    queued_messages: 0,
    remote_peers: 1,
    status: "running",
  };
  mockNetworkStatusLoading = false;
  mockNetworkStatusError = null;

  mockNetworkChannels = {
    channels: [makeChannelSummary(), makeChannelSummary({ channel: "ops.alerts" })],
  };
  mockNetworkChannelsLoading = false;
  mockNetworkChannelsError = null;

  mockChannelDetail = makeChannelDetail();
  mockChannelDetailLoading = false;
  mockChannelDetailError = null;

  mockChannelMessages = [
    makeChannelMessage(),
    makeChannelMessage({
      body: { text: "Received. I am validating the handoff now." },
      direction: "received",
      display_name: "Remote Reviewer",
      kind: "direct",
      local: false,
      message_id: "msg_2",
      peer_from: "peer_remote",
      peer_to: "peer_local",
      preview_text: "Received. I am validating the handoff now.",
      text: "Received. I am validating the handoff now.",
      timestamp: "2026-04-13T10:43:00Z",
    }),
  ];
  mockChannelMessagesLoading = false;
  mockChannelMessagesError = null;

  mockNetworkPeers = [
    makePeerSummary({
      display_name: "Polybot Main",
      local: true,
      peer_card: {
        artifacts_supported: ["capability"],
        capabilities: [{ id: "chat", summary: "Coordinates planning." }],
        display_name: "Polybot Main",
        peer_id: "peer_local",
        profiles_supported: ["default"],
        trust_modes_supported: ["local-first"],
      },
      peer_id: "peer_local",
      session_id: "sess_local",
    }),
    makePeerSummary(),
  ];
  mockNetworkPeersLoading = false;
  mockNetworkPeersError = null;

  mockPeerDetail = makePeerDetail();
  mockPeerDetailLoading = false;
  mockPeerDetailError = null;

  mockPeerMessages = [
    makeChannelMessage({
      body: { text: "Can you validate the handoff now?" },
      kind: "direct",
      peer_to: "peer_remote",
    }),
    makeChannelMessage({
      body: { text: "Received. I am validating the handoff now." },
      direction: "received",
      display_name: "Remote Reviewer",
      kind: "direct",
      local: false,
      message_id: "msg_dm_2",
      peer_from: "peer_remote",
      peer_to: "peer_local",
      preview_text: "Received. I am validating the handoff now.",
      text: "Received. I am validating the handoff now.",
      timestamp: "2026-04-13T10:43:00Z",
    }),
  ];
  mockPeerMessagesLoading = false;
  mockPeerMessagesError = null;

  mockCreateNetworkChannelPending = false;
  mockCreateNetworkChannelMutateAsync.mockReset();
  mockCreateNetworkChannelMutateAsync.mockResolvedValue({
    channel: makeChannelDetail({ channel: "release.canary" }),
  } satisfies CreateNetworkChannelResponse);
  mockSendNetworkMessagePending = false;
  mockSendNetworkMessageMutateAsync.mockReset();
  mockSendNetworkMessageMutateAsync.mockResolvedValue({ message: { id: "msg_sent" } });

  toast.error.mockReset();
  toast.success.mockReset();
  window.localStorage.clear();
}

describe("network route", () => {
  beforeEach(() => {
    resetMocks();
  });

  it("renders the unified room workspace around the default channel", () => {
    render(<NetworkPage />);

    expect(screen.getByTestId("network-workspace")).toBeInTheDocument();
    expect(screen.getByTestId("network-room-channel-coord.core")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("network-room-header")).getByText("#coord.core")
    ).toBeInTheDocument();
    expect(screen.getByTestId("network-room-intro")).toHaveTextContent(
      "Coordinate release handoffs and verification."
    );
    expect(screen.getByTestId("network-message-msg_1")).toHaveTextContent(
      "Dispatching task deploy-api-v2.3 to coder-agent-01"
    );
    expect(screen.getByTestId("network-details-panel")).toBeInTheDocument();
  });

  it("exposes workspace navigation, filters, tabs, and composer controls accessibly", () => {
    render(<NetworkPage />);

    const channelRow = screen.getByTestId("network-room-channel-coord.core");
    expect(channelRow).not.toHaveAttribute("role", "button");
    expect(within(channelRow).getByRole("button", { name: /coord\.core/i })).toHaveAttribute(
      "aria-current",
      "page"
    );
    expect(within(channelRow).getByRole("button", { name: "Star channel" })).toBeInTheDocument();

    expect(screen.getByLabelText("Close room details")).toBeInTheDocument();
    expect(screen.getByLabelText("Network message composer")).toBe(
      screen.getByTestId("network-composer-input")
    );

    const kindFilters = screen.getByRole("group", { name: "Timeline kind filters" });
    expect(within(kindFilters).getByRole("button", { name: "All" })).toHaveAttribute(
      "aria-pressed",
      "true"
    );

    const detailsTabs = screen.getByRole("tablist", { name: "Room detail tabs" });
    expect(within(detailsTabs).getByRole("tab", { name: "about" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  it("renders the disabled state when the network is turned off", () => {
    mockNetworkStatus = { enabled: false, status: "offline" };

    render(<NetworkPage />);

    expect(screen.getByTestId("network-disabled-state")).toHaveTextContent("Network disabled");
    expect(screen.queryByTestId("network-workspace")).not.toBeInTheDocument();
  });

  it("creates a channel with purpose from the redesigned dialog", async () => {
    const user = userEvent.setup();

    render(<NetworkPage />);

    await user.click(screen.getByTestId("network-open-create-dialog"));
    fireEvent.change(screen.getByTestId("network-channel-name-input"), {
      target: { value: "release.canary" },
    });
    fireEvent.change(screen.getByTestId("network-channel-purpose-input"), {
      target: { value: "Coordinate release handoff" },
    });
    await user.click(screen.getByTestId("network-agent-option-polybot-main"));
    await user.click(screen.getByTestId("network-create-channel-submit"));

    await waitFor(() => {
      expect(mockCreateNetworkChannelMutateAsync).toHaveBeenCalledWith({
        agent_names: ["polybot-main"],
        channel: "release.canary",
        purpose: "Coordinate release handoff",
        workspace_id: "ws_main",
      });
    });
  });

  it("sends a broadcast from the active channel composer", async () => {
    const user = userEvent.setup();

    render(<NetworkPage />);

    fireEvent.change(screen.getByTestId("network-composer-input"), {
      target: { value: "Please verify the rollout plan." },
    });
    await user.click(screen.getByTestId("network-composer-submit"));

    await waitFor(() => {
      expect(mockSendNetworkMessageMutateAsync).toHaveBeenCalledWith({
        body: { text: "Please verify the rollout plan." },
        channel: "coord.core",
        kind: "say",
        session_id: "sess_local",
      });
    });
  });

  it("renders a peer room and sends directed messages there", async () => {
    const user = userEvent.setup();
    routerState.searchParams = { peer: "peer_remote" };

    const view = render(<NetworkPage />);

    expect(
      within(screen.getByTestId("network-room-header")).getByText("Remote Reviewer")
    ).toBeInTheDocument();
    expect(screen.getByTestId("network-room-intro")).toHaveTextContent(
      "Direct thread with Remote Reviewer"
    );

    fireEvent.change(screen.getByTestId("network-composer-input"), {
      target: { value: "Can you own the reviewer pass?" },
    });
    await user.click(screen.getByTestId("network-composer-submit"));

    await waitFor(() => {
      expect(mockSendNetworkMessageMutateAsync).toHaveBeenCalledWith({
        body: { text: "Can you own the reviewer pass?" },
        channel: "coord.core",
        kind: "direct",
        session_id: "sess_local",
        to: "peer_remote",
      });
    });

    routerState.searchParams = { channel: "coord.core" };
    view.rerender(<NetworkPage />);
    expect(
      within(screen.getByTestId("network-room-header")).getByText("#coord.core")
    ).toBeInTheDocument();
  });
});
