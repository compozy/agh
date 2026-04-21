import type {
  CreateNetworkChannelResponse,
  NetworkCapabilityCatalog,
  NetworkChannel,
  NetworkChannelMessage,
  NetworkChannelsResponse,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkStatus,
} from "../types";

const primaryPeerCard = {
  peer_id: "peer_storybook_local",
  display_name: "Storyboard",
  artifacts_supported: ["capability"],
  capabilities: [
    { id: "chat", summary: "Coordinates chat-first collaboration." },
    { id: "tools", summary: "Runs tool-driven implementation steps." },
  ],
  profiles_supported: ["default"],
  trust_modes_supported: ["local-first"],
};

const remotePeerCard = {
  peer_id: "peer_storybook_remote",
  display_name: "Remote Reviewer",
  artifacts_supported: ["capability"],
  capabilities: [{ id: "chat", summary: "Participates in channel discussion." }],
  profiles_supported: ["default"],
  trust_modes_supported: ["relay"],
};

const primaryCapabilityCatalog: NetworkCapabilityCatalog = {
  capabilities: [
    {
      id: "chat",
      summary: "Coordinates chat-first collaboration.",
      outcome: "Peers converge on a shared plan inside a channel.",
      version: "1.0.0",
      digest: "sha256:chat",
      context_needed: ["channel-history", "active-workspace"],
      artifacts_expected: ["message.text"],
      execution_outline: [
        "Acknowledge the incoming request.",
        "Summarize the conversation so far.",
        "Propose the next coordinated step.",
      ],
      constraints: ["Latency-sensitive"],
      examples: ["Standup sync", "Rollout review"],
      requirements: [],
    },
    {
      id: "tools",
      summary: "Runs tool-driven implementation steps.",
      outcome: "Delegated implementation step runs with reproducible output.",
      version: "0.2.0",
      digest: "sha256:tools",
      context_needed: ["workspace-root"],
      artifacts_expected: ["tool-call.summary"],
      execution_outline: ["Pick the next step.", "Run the tool.", "Report the result."],
      constraints: [],
      examples: ["Run `make verify`"],
      requirements: ["chat"],
    },
  ],
};

export const networkStatusFixture: NetworkStatus = {
  enabled: true,
  status: "online",
  channels: 2,
  local_peers: 1,
  remote_peers: 2,
  delivery_workers: 2,
  queued_messages: 1,
  messages_sent: 24,
  configured_default_channel: "storybook",
  effective_default_channel: "storybook",
  effective_default_source: "workspace",
};

export const networkChannelsFixture: NetworkChannelsResponse = {
  channels: [
    {
      channel: "storybook",
      last_message_at: "2026-04-17T18:10:00Z",
      message_count: 8,
      peer_count: 2,
      local_peer_count: 1,
      remote_peer_count: 1,
      session_count: 2,
    },
    {
      channel: "release",
      last_message_at: "2026-04-17T17:45:00Z",
      message_count: 3,
      peer_count: 1,
      local_peer_count: 0,
      remote_peer_count: 1,
      session_count: 1,
    },
  ],
};

export const networkChannelFixture: NetworkChannel = {
  channel: "storybook",
  last_message_at: "2026-04-17T18:10:00Z",
  message_count: 8,
  peer_count: 2,
  local_peer_count: 1,
  remote_peer_count: 1,
  session_count: 2,
  peers: [
    {
      channel: "storybook",
      display_name: "Storyboard",
      joined_at: "2026-04-17T17:40:00Z",
      last_seen: "2026-04-17T18:10:00Z",
      local: true,
      peer_card: primaryPeerCard,
      peer_id: primaryPeerCard.peer_id,
      session_id: "sess-storybook",
    },
    {
      channel: "storybook",
      display_name: "Remote Reviewer",
      joined_at: "2026-04-17T17:42:00Z",
      last_seen: "2026-04-17T18:09:00Z",
      local: false,
      peer_card: remotePeerCard,
      peer_id: remotePeerCard.peer_id,
      session_id: "sess-reviewer",
    },
  ],
  sessions: [
    {
      id: "sess-storybook",
      name: "Storybook rollout",
      agent_name: "codex-agent",
      channel: "storybook",
      state: "active",
      created_at: "2026-04-17T17:40:00Z",
      updated_at: "2026-04-17T18:10:00Z",
      workspace_id: "ws_storybook",
      workspace_path: "/Users/pedro/Dev/compozy/agh2",
      acp_caps: {
        supports_load_session: true,
        supported_models: ["gpt-5.4"],
        supported_modes: ["chat"],
      },
    },
    {
      id: "sess-reviewer",
      name: "Review lane",
      agent_name: "claude-agent",
      channel: "storybook",
      state: "active",
      created_at: "2026-04-17T17:42:00Z",
      updated_at: "2026-04-17T18:09:00Z",
      workspace_id: "ws_storybook",
      workspace_path: "/Users/pedro/Dev/compozy/agh2",
      acp_caps: {
        supports_load_session: true,
        supported_models: ["claude-opus"],
        supported_modes: ["chat"],
      },
    },
  ],
};

export const networkChannelMessagesFixture: NetworkChannelMessage[] = [
  {
    message_id: "msg_storybook_1",
    channel: "storybook",
    peer_id: primaryPeerCard.peer_id,
    display_name: "Storyboard",
    local: true,
    session_id: "sess-storybook",
    text: "Preview build is green. Starting the system stories batch now.",
    timestamp: "2026-04-17T18:00:00Z",
  },
  {
    message_id: "msg_storybook_2",
    channel: "storybook",
    peer_id: remotePeerCard.peer_id,
    display_name: "Remote Reviewer",
    local: false,
    session_id: "sess-reviewer",
    text: "Received. I will cover bridges and automation edge cases.",
    timestamp: "2026-04-17T18:08:00Z",
  },
];

export const networkPeersFixture: NetworkPeerSummary[] = [
  {
    channel: "storybook",
    display_name: "Storyboard",
    joined_at: "2026-04-17T17:40:00Z",
    last_seen: "2026-04-17T18:10:00Z",
    local: true,
    peer_card: primaryPeerCard,
    peer_id: primaryPeerCard.peer_id,
    session_id: "sess-storybook",
  },
  {
    channel: "storybook",
    display_name: "Remote Reviewer",
    joined_at: "2026-04-17T17:42:00Z",
    last_seen: "2026-04-17T18:09:00Z",
    local: false,
    peer_card: remotePeerCard,
    peer_id: remotePeerCard.peer_id,
    session_id: "sess-reviewer",
  },
];

export const networkPeerFixture: NetworkPeerDetail = {
  channel: "storybook",
  display_name: "Storyboard",
  joined_at: "2026-04-17T17:40:00Z",
  last_seen: "2026-04-17T18:10:00Z",
  local: true,
  metrics: {
    sent: 12,
    received: 9,
    delivered: 10,
    rejected: 1,
  },
  peer_card: primaryPeerCard,
  capability_catalog: primaryCapabilityCatalog,
  peer_id: primaryPeerCard.peer_id,
  session_id: "sess-storybook",
};

export const createNetworkChannelFixture: CreateNetworkChannelResponse = {
  channel: networkChannelFixture,
};
