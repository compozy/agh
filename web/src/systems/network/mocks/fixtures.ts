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

const remoteCapabilityCatalog: NetworkCapabilityCatalog = {
  capabilities: [
    {
      id: "chat",
      summary: "Participates in channel discussion.",
      outcome: "Remote peers can acknowledge and continue directed coordination.",
      version: "1.0.0",
      digest: "sha256:remote-chat",
      context_needed: ["channel-history"],
      artifacts_expected: ["message.text"],
      execution_outline: [
        "Read the direct request.",
        "Acknowledge the lane ownership.",
        "Respond with the next coordination step.",
      ],
      constraints: ["Network-visible only"],
      examples: ["Review handoff", "Escalation follow-up"],
      requirements: [],
    },
  ],
};

const primaryCapabilityCatalog: NetworkCapabilityCatalog = {
  capabilities: [
    {
      id: "chat",
      summary: "Coordinates chat-first collaboration.",
      outcome: "Peers converge on a shared plan inside a room.",
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
  status: "running",
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
      created_at: "2026-04-17T17:40:00Z",
      created_by: "codex-agent",
      last_activity_at: "2026-04-17T18:10:00Z",
      last_message_preview: "Preview build is green. Starting the system stories batch now.",
      local_peer_count: 1,
      message_count: 8,
      peer_count: 2,
      purpose: "Coordinate the AGH runtime storybook refresh.",
      remote_peer_count: 1,
      session_count: 2,
      workspace_id: "ws_storybook",
    },
    {
      channel: "release",
      created_at: "2026-04-17T16:55:00Z",
      created_by: "claude-agent",
      last_activity_at: "2026-04-17T17:45:00Z",
      last_message_preview: "Waiting for the canary hash before promoting.",
      local_peer_count: 0,
      message_count: 3,
      peer_count: 1,
      purpose: "Coordinate release handoffs and canary verification.",
      remote_peer_count: 1,
      session_count: 1,
      workspace_id: "ws_storybook",
    },
  ],
};

export const networkChannelFixture: NetworkChannel = {
  channel: "storybook",
  created_at: "2026-04-17T17:40:00Z",
  created_by: "codex-agent",
  kind_counts: [
    { kind: "say", count: 2 },
    { kind: "direct", count: 1 },
    { kind: "capability", count: 1 },
  ],
  last_activity_at: "2026-04-17T18:10:00Z",
  last_message_preview: "Preview build is green. Starting the system stories batch now.",
  local_peer_count: 1,
  message_count: 8,
  peer_count: 2,
  peers: [
    {
      channel: "storybook",
      display_name: "Storyboard",
      joined_at: "2026-04-17T17:40:00Z",
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
  purpose: "Coordinate the AGH runtime storybook refresh.",
  remote_peer_count: 1,
  session_count: 2,
  sessions: [
    {
      id: "sess-storybook",
      name: "Storybook rollout",
      agent_name: "codex-agent",
      provider: "codex",
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
      provider: "claude",
      channel: "storybook",
      state: "active",
      created_at: "2026-04-17T17:42:00Z",
      updated_at: "2026-04-17T18:09:00Z",
      workspace_id: "ws_storybook",
      workspace_path: "/Users/pedro/Dev/compozy/agh2",
      acp_caps: {
        supports_load_session: true,
        supported_models: ["claude-sonnet-4-6"],
        supported_modes: ["chat"],
      },
    },
  ],
  workspace_id: "ws_storybook",
};

export const networkChannelMessagesFixture: NetworkChannelMessage[] = [
  {
    body: { text: "Preview build is green. Starting the system stories batch now." },
    channel: "storybook",
    direction: "sent",
    display_name: "Storybook rollout",
    kind: "say",
    local: true,
    message_id: "msg_storybook_1",
    peer_from: primaryPeerCard.peer_id,
    preview_text: "Preview build is green. Starting the system stories batch now.",
    session_id: "sess-storybook",
    text: "Preview build is green. Starting the system stories batch now.",
    timestamp: "2026-04-17T18:00:00Z",
  },
  {
    body: { text: "Received. I will cover bridges and automation edge cases." },
    channel: "storybook",
    direction: "received",
    display_name: "Remote Reviewer",
    kind: "direct",
    local: false,
    message_id: "msg_storybook_2",
    peer_from: remotePeerCard.peer_id,
    peer_to: primaryPeerCard.peer_id,
    preview_text: "Received. I will cover bridges and automation edge cases.",
    text: "Received. I will cover bridges and automation edge cases.",
    timestamp: "2026-04-17T18:08:00Z",
  },
  {
    body: {
      capability: {
        id: "tools",
        summary: "Run tool-driven verification passes.",
        outcome: "Verification output is reported back to the room.",
        version: "0.2.0",
        digest: "sha256:tools",
        execution_outline: ["Pick the next check", "Run the command", "Report the result"],
      },
    },
    channel: "storybook",
    direction: "received",
    display_name: "Remote Reviewer",
    kind: "capability",
    local: false,
    message_id: "msg_storybook_3",
    peer_from: remotePeerCard.peer_id,
    preview_text: "Run tool-driven verification passes.",
    timestamp: "2026-04-17T18:09:00Z",
  },
];

export const networkPeerMessagesFixture: NetworkChannelMessage[] = [
  {
    body: { text: "Can you validate the bridge edge cases next?" },
    channel: "storybook",
    direction: "sent",
    display_name: "Storybook rollout",
    kind: "direct",
    local: true,
    message_id: "msg_dm_1",
    peer_from: primaryPeerCard.peer_id,
    peer_to: remotePeerCard.peer_id,
    preview_text: "Can you validate the bridge edge cases next?",
    session_id: "sess-storybook",
    text: "Can you validate the bridge edge cases next?",
    timestamp: "2026-04-17T18:03:00Z",
  },
  {
    body: { text: "Received. I will cover bridges and automation edge cases." },
    channel: "storybook",
    direction: "received",
    display_name: "Remote Reviewer",
    kind: "direct",
    local: false,
    message_id: "msg_dm_2",
    peer_from: remotePeerCard.peer_id,
    peer_to: primaryPeerCard.peer_id,
    preview_text: "Received. I will cover bridges and automation edge cases.",
    text: "Received. I will cover bridges and automation edge cases.",
    timestamp: "2026-04-17T18:08:00Z",
  },
];

export const networkPeersFixture: NetworkPeerSummary[] = [
  {
    channel: "storybook",
    display_name: "Storyboard",
    joined_at: "2026-04-17T17:40:00Z",
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

export const networkRemotePeerFixture: NetworkPeerDetail = {
  channel: "storybook",
  display_name: "Remote Reviewer",
  joined_at: "2026-04-17T17:42:00Z",
  last_seen: "2026-04-17T18:09:00Z",
  local: false,
  metrics: {
    sent: 4,
    received: 6,
    delivered: 5,
    rejected: 0,
  },
  peer_card: remotePeerCard,
  capability_catalog: remoteCapabilityCatalog,
  peer_id: remotePeerCard.peer_id,
  session_id: "sess-reviewer",
};

export const createNetworkChannelFixture: CreateNetworkChannelResponse = {
  channel: networkChannelFixture,
};
