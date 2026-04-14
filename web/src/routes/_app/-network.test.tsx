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
let mockAgents = [
  ...mockWorkspaceAgents,
  { name: "researcher-01", prompt: "research", provider: "openai" },
];

let mockNetworkStatus: NetworkStatus | undefined;
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

const mockCreateNetworkChannelMutateAsync = vi.fn();
let mockCreateNetworkChannelPending = false;

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: ReactNode;
    params?: { id?: string };
    to: string;
    [key: string]: unknown;
  }) => {
    const href = params?.id ? to.replace("$id", params.id) : to;
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("sonner", () => ({
  toast,
}));

vi.mock("@/systems/workspace", () => ({
  WorkspacePageShell: ({
    children,
    controls,
    meta,
    title,
  }: {
    children: ReactNode;
    controls?: ReactNode;
    meta?: ReactNode;
    title: string;
  }) => (
    <div data-testid="workspace-page-shell">
      <div data-testid="workspace-page-shell-header">
        <h1>{title}</h1>
        {controls ? <div data-testid="workspace-page-shell-controls">{controls}</div> : null}
        {meta ? <div data-testid="workspace-page-shell-meta">{meta}</div> : null}
      </div>
      {children}
    </div>
  ),
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
  useAgents: () => ({
    data: mockAgents,
    error: null,
    isLoading: false,
  }),
}));

vi.mock("@/systems/network", async () => {
  const actual = await vi.importActual("@/systems/network");

  return {
    ...actual,
    useNetworkStatus: () => ({
      data: mockNetworkStatus,
      error: null,
      isLoading: false,
    }),
    useNetworkChannels: () => ({
      data: mockNetworkChannels,
      error: mockNetworkChannelsError,
      isLoading: mockNetworkChannelsLoading,
    }),
    useNetworkChannel: () => ({
      data: mockChannelDetail,
      error: mockChannelDetailError,
      isLoading: mockChannelDetailLoading,
    }),
    useNetworkChannelMessages: () => ({
      data: mockChannelMessages,
      error: mockChannelMessagesError,
      isLoading: mockChannelMessagesLoading,
    }),
    useNetworkPeers: () => ({
      data: mockNetworkPeers,
      error: mockNetworkPeersError,
      isLoading: mockNetworkPeersLoading,
    }),
    useNetworkPeer: () => ({
      data: mockPeerDetail,
      error: mockPeerDetailError,
      isLoading: mockPeerDetailLoading,
    }),
    useCreateNetworkChannel: () => ({
      isPending: mockCreateNetworkChannelPending,
      mutateAsync: mockCreateNetworkChannelMutateAsync,
    }),
  };
});

import { Route } from "./network";

function makeChannelSummary(
  overrides: Partial<NetworkChannelsResponse["channels"][number]> = {}
): NetworkChannelsResponse["channels"][number] {
  return {
    channel: "general",
    last_message_at: "2026-04-13T10:45:00Z",
    local_peer_count: 1,
    message_count: 6,
    peer_count: 3,
    remote_peer_count: 2,
    session_count: 1,
    ...overrides,
  };
}

function makeChannelMessage(overrides: Partial<NetworkChannelMessage> = {}): NetworkChannelMessage {
  return {
    channel: "general",
    display_name: "polybot-main",
    intent: "announce",
    local: true,
    message_id: "msg_1",
    peer_id: "peer_local",
    session_id: "sess_local",
    text: "Dispatching task deploy-api-v2.3 to coder-agent-01",
    timestamp: "2026-04-13T10:42:00Z",
    ...overrides,
  };
}

function makeChannel(overrides: Partial<NetworkChannel> = {}): NetworkChannel {
  return {
    channel: "general",
    last_message_at: "2026-04-13T10:45:00Z",
    local_peer_count: 1,
    message_count: 6,
    peer_count: 3,
    peers: [
      {
        channel: "general",
        display_name: "polybot-main",
        joined_at: "2026-04-13T10:00:00Z",
        last_seen: "2026-04-13T10:45:00Z",
        local: true,
        peer_card: {
          artifacts_supported: [],
          capabilities: [],
          peer_id: "peer_local",
          profiles_supported: [],
          trust_modes_supported: [],
        },
        peer_id: "peer_local",
        session_id: "sess_local",
      },
    ],
    remote_peer_count: 2,
    session_count: 1,
    sessions: [
      {
        agent_name: "polybot-main",
        created_at: "2026-04-13T10:00:00Z",
        id: "sess_local",
        state: "active",
        updated_at: "2026-04-13T10:45:00Z",
        workspace_id: "ws_main",
      },
    ],
    ...overrides,
  };
}

function makePeerSummary(overrides: Partial<NetworkPeerSummary> = {}): NetworkPeerSummary {
  return {
    channel: "general",
    display_name: "polybot-main",
    joined_at: "2026-04-13T10:00:00Z",
    last_seen: "2026-04-13T10:45:00Z",
    local: true,
    peer_card: {
      artifacts_supported: [],
      capabilities: [],
      peer_id: "peer_local",
      profiles_supported: [],
      trust_modes_supported: [],
    },
    peer_id: "peer_local",
    session_id: "sess_local",
    ...overrides,
  };
}

function makePeerDetail(overrides: Partial<NetworkPeerDetail> = {}): NetworkPeerDetail {
  return {
    channel: "general",
    display_name: "polybot-main",
    joined_at: "2026-04-13T10:00:00Z",
    last_seen: "2026-04-13T10:47:00Z",
    local: true,
    metrics: {
      delivered: 12,
      received: 12,
      rejected: 0,
      sent: 14,
    },
    peer_card: {
      artifacts_supported: [],
      capabilities: [],
      peer_id: "peer_local",
      profiles_supported: [],
      trust_modes_supported: [],
    },
    peer_id: "peer_local",
    session_id: "sess_local",
    ...overrides,
  };
}

const NetworkPage = (Route as unknown as { component: () => ReactNode }).component;

describe("NetworkPage", () => {
  beforeEach(() => {
    vi.useRealTimers();
    mockActiveWorkspaceId = "ws_main";
    mockActiveWorkspaceName = "Polybot";
    mockWorkspaceAgents = [
      { name: "polybot-main", prompt: "coordinate", provider: "anthropic" },
      { name: "coder-agent-01", prompt: "code", provider: "openai" },
    ];
    mockAgents = [
      ...mockWorkspaceAgents,
      { name: "researcher-01", prompt: "research", provider: "openai" },
    ];

    mockNetworkStatus = {
      channels: 2,
      delivery_workers: 4,
      enabled: true,
      local_peers: 1,
      messages_sent: 42,
      queued_messages: 3,
      remote_peers: 2,
      status: "active",
    };
    mockNetworkChannels = {
      channels: [
        makeChannelSummary(),
        makeChannelSummary({
          channel: "deployments",
          last_message_at: "2026-04-13T10:40:00Z",
          peer_count: 2,
        }),
      ],
    };
    mockNetworkChannelsLoading = false;
    mockNetworkChannelsError = null;
    mockChannelDetail = makeChannel();
    mockChannelDetailLoading = false;
    mockChannelDetailError = null;
    mockChannelMessages = [
      makeChannelMessage(),
      makeChannelMessage({
        display_name: "coder-agent-01",
        local: false,
        message_id: "msg_2",
        peer_id: "peer_coder",
        session_id: undefined,
        text: "Acknowledged. Starting deployment pipeline...",
        timestamp: "2026-04-13T10:43:00Z",
      }),
    ];
    mockChannelMessagesLoading = false;
    mockChannelMessagesError = null;
    mockNetworkPeers = [
      makePeerSummary(),
      makePeerSummary({
        display_name: "coder-agent-01",
        local: false,
        peer_id: "peer_coder",
        session_id: undefined,
      }),
    ];
    mockNetworkPeersLoading = false;
    mockNetworkPeersError = null;
    mockPeerDetail = makePeerDetail();
    mockPeerDetailLoading = false;
    mockPeerDetailError = null;

    mockCreateNetworkChannelPending = false;
    mockCreateNetworkChannelMutateAsync.mockReset();
    mockCreateNetworkChannelMutateAsync.mockResolvedValue({
      channel: makeChannel({
        channel: "deployments",
        last_message_at: null,
        message_count: 0,
      }),
    } satisfies CreateNetworkChannelResponse);
    toast.error.mockReset();
    toast.success.mockReset();
  });

  it("renders loading and error states from the active list query", () => {
    mockNetworkChannelsLoading = true;
    mockNetworkChannels = undefined;
    const { rerender } = render(<NetworkPage />);

    expect(screen.getByTestId("workspace-page-shell")).toBeInTheDocument();
    expect(screen.getByTestId("network-channels-list-loading")).toBeInTheDocument();
    expect(screen.getByTestId("network-channel-loading")).toBeInTheDocument();

    mockNetworkChannelsLoading = false;
    mockNetworkChannels = undefined;
    mockNetworkChannelsError = new Error("network down");
    rerender(<NetworkPage />);

    expect(screen.getByTestId("network-channels-list-error")).toHaveTextContent("network down");
    expect(screen.getByTestId("network-channel-error")).toHaveTextContent("network down");
  });

  it("renders peer loading and error states inside the panel instead of replacing the page", async () => {
    const user = userEvent.setup();
    mockNetworkPeersLoading = true;
    mockNetworkPeers = undefined;
    const { rerender } = render(<NetworkPage />);

    await user.click(screen.getByTestId("network-tab-peers"));

    expect(screen.getByTestId("workspace-page-shell")).toBeInTheDocument();
    expect(screen.getByTestId("network-peers-list-loading")).toBeInTheDocument();
    expect(screen.getByTestId("network-peer-loading")).toBeInTheDocument();

    mockNetworkPeersLoading = false;
    mockNetworkPeers = undefined;
    mockNetworkPeersError = new Error("peer discovery failed");
    rerender(<NetworkPage />);

    expect(screen.getByTestId("network-peers-list-error")).toHaveTextContent(
      "peer discovery failed"
    );
    expect(screen.getByTestId("network-peer-error")).toHaveTextContent("peer discovery failed");
  });

  it("renders the channels view with metrics and the read-only timeline", () => {
    render(<NetworkPage />);

    expect(screen.getByText("Network")).toBeInTheDocument();
    expect(screen.getByTestId("network-tab-channels")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Total Peers")).toBeInTheDocument();
    expect(screen.getByTestId("network-channel-item-general")).toBeInTheDocument();
    expect(screen.getByTestId("network-channel-detail-panel")).toBeInTheDocument();
    expect(
      screen.getByText("This channel is read-only. Use the CLI to send messages.")
    ).toBeInTheDocument();
    expect(screen.getByTestId("network-channel-message-msg_1")).toHaveTextContent(
      "Dispatching task deploy-api-v2.3 to coder-agent-01"
    );
    expect(screen.getByText("View Session")).toHaveAttribute("href", "/session/sess_local");
  });

  it("switches to peers and renders identity and metrics for the selected peer", async () => {
    const user = userEvent.setup();
    render(<NetworkPage />);

    await user.click(screen.getByTestId("network-tab-peers"));

    expect(screen.getByTestId("network-tab-peers")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("network-peers-list-panel")).toBeInTheDocument();
    const detailPanel = screen.getByTestId("network-peer-detail-panel");
    expect(detailPanel).toBeInTheDocument();
    expect(within(detailPanel).getByText("Peer Identity")).toBeInTheDocument();
    expect(within(detailPanel).getByText("Message Statistics")).toBeInTheDocument();
    expect(within(detailPanel).getAllByText("polybot-main").length).toBeGreaterThan(0);
    expect(within(detailPanel).getByText("general")).toBeInTheDocument();
  });

  it("opens the create dialog and submits the selected agents", async () => {
    render(<NetworkPage />);

    fireEvent.click(screen.getByTestId("open-network-create-dialog"));
    expect(screen.getByTestId("network-create-channel-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("network-agent-option-researcher-01")).not.toBeInTheDocument();

    const channelNameInput = screen.getByTestId("network-channel-name-input");
    const firstAgent = screen.getByTestId("network-agent-option-polybot-main");
    const secondAgent = screen.getByTestId("network-agent-option-coder-agent-01");

    fireEvent.change(channelNameInput, { target: { value: "deployments" } });
    fireEvent.click(firstAgent);
    fireEvent.click(secondAgent);

    expect(firstAgent).toHaveAttribute("aria-pressed", "true");
    expect(secondAgent).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByTestId("network-create-channel-submit"));

    await waitFor(() =>
      expect(mockCreateNetworkChannelMutateAsync).toHaveBeenCalledWith({
        agent_names: ["polybot-main", "coder-agent-01"],
        channel: "deployments",
        workspace_id: "ws_main",
      })
    );
    expect(mockCreateNetworkChannelMutateAsync).toHaveBeenCalledOnce();
    expect(toast.success).toHaveBeenCalledWith("Created channel deployments.");
  });

  it("renders truthful empty states for channels and peers", async () => {
    const user = userEvent.setup();
    mockNetworkChannels = { channels: [] };
    mockNetworkPeers = [];

    render(<NetworkPage />);

    expect(screen.getByTestId("network-channels-list-empty")).toBeInTheDocument();
    expect(screen.getByTestId("network-channels-empty-state")).toHaveTextContent("No channels yet");

    await user.click(screen.getByTestId("network-tab-peers"));

    expect(screen.getByTestId("network-peers-list-empty")).toBeInTheDocument();
    expect(screen.getByTestId("network-peers-empty-state")).toHaveTextContent("No peers connected");
  });
});
