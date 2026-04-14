import type { AgentPayload } from "@/systems/agent";

import type {
  NetworkChannel,
  NetworkChannelMessage,
  NetworkChannelSummary,
  NetworkCreateChannelDraft,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkStatus,
} from "../types";

const numberFormatter = new Intl.NumberFormat("en-US");
const timeFormatter = new Intl.DateTimeFormat("en-US", {
  hour: "numeric",
  minute: "2-digit",
});

export function createNetworkChannelDraft(): NetworkCreateChannelDraft {
  return {
    channelName: "",
    selectedAgentNames: [],
  };
}

export function toggleDraftAgent(
  draft: NetworkCreateChannelDraft,
  agentName: string
): NetworkCreateChannelDraft {
  const selectedAgentNames = draft.selectedAgentNames.includes(agentName)
    ? draft.selectedAgentNames.filter(name => name !== agentName)
    : [...draft.selectedAgentNames, agentName];

  return {
    ...draft,
    selectedAgentNames,
  };
}

export function formatNetworkNumber(value?: number | null): string {
  return numberFormatter.format(value ?? 0);
}

export function formatChannelPeerCount(value?: number | null): string {
  const count = value ?? 0;
  return `${formatNetworkNumber(count)} ${count === 1 ? "peer" : "peers"}`;
}

export function formatChannelMemberCount(channel?: Pick<NetworkChannel, "peer_count">): string {
  const count = channel?.peer_count ?? 0;
  return `${formatNetworkNumber(count)} ${count === 1 ? "member" : "members"}`;
}

export function formatNetworkClockTime(value?: string | null): string {
  if (!value) {
    return "Unknown";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "Unknown";
  }

  return timeFormatter.format(parsed);
}

export function formatNetworkDateTime(value?: string | null): string {
  if (!value) {
    return "Unavailable";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "Unavailable";
  }

  return parsed.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatNetworkRelativeTime(value?: string | null): string {
  if (!value) {
    return "Unavailable";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "Unavailable";
  }

  const diffMs = Date.now() - parsed.getTime();
  if (diffMs < 0) {
    return "just now";
  }

  const diffSeconds = Math.floor(diffMs / 1000);
  if (diffSeconds < 5) {
    return "just now";
  }
  if (diffSeconds < 60) {
    return `${diffSeconds}s ago`;
  }

  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d ago`;
}

export function getPeerDisplayName(
  peer: Pick<NetworkPeerSummary, "display_name" | "peer_card" | "peer_id">
): string {
  return peer.display_name ?? peer.peer_card.display_name ?? peer.peer_id;
}

export function getPeerTypeLabel(peer: Pick<NetworkPeerSummary, "local">): string {
  return peer.local ? "LOCAL" : "REMOTE";
}

export function getPeerPresenceTone(peer: Pick<NetworkPeerSummary, "last_seen" | "local">) {
  if (peer.local) {
    return "bg-[color:var(--color-accent)]";
  }

  if (!peer.last_seen) {
    return "bg-[color:var(--color-text-tertiary)]";
  }

  const parsed = new Date(peer.last_seen);
  if (Number.isNaN(parsed.getTime())) {
    return "bg-[color:var(--color-text-tertiary)]";
  }

  return Date.now() - parsed.getTime() <= 60_000
    ? "bg-[color:var(--color-success)]"
    : "bg-[color:var(--color-text-tertiary)]";
}

export function getNetworkMetricCards(
  status: NetworkStatus | undefined,
  channelCount: number
): Array<{ detail?: string; label: string; value: string }> {
  const localPeers = status?.local_peers ?? 0;
  const remotePeers = status?.remote_peers ?? 0;

  return [
    {
      detail: `${formatNetworkNumber(localPeers)} local / ${formatNetworkNumber(remotePeers)} remote`,
      label: "Total Peers",
      value: formatNetworkNumber(localPeers + remotePeers),
    },
    {
      detail:
        status?.channels != null
          ? `${formatNetworkNumber(status.channels)} active in runtime`
          : undefined,
      label: "Total Channels",
      value: formatNetworkNumber(channelCount),
    },
    {
      detail:
        status?.messages_sent != null
          ? `${formatNetworkNumber(status.messages_sent)} sent total`
          : undefined,
      label: "Queued Msgs",
      value: formatNetworkNumber(status?.queued_messages ?? 0),
    },
    {
      detail:
        status?.status && status.status !== "" ? status.status.replaceAll("_", " ") : undefined,
      label: "Workers",
      value: formatNetworkNumber(status?.delivery_workers ?? 0),
    },
  ];
}

export function sortNetworkChannels(channels: NetworkChannelSummary[]) {
  return [...channels].sort((left, right) => {
    const leftMessageTime = left.last_message_at ? new Date(left.last_message_at).getTime() : 0;
    const rightMessageTime = right.last_message_at ? new Date(right.last_message_at).getTime() : 0;

    if (leftMessageTime !== rightMessageTime) {
      return rightMessageTime - leftMessageTime;
    }

    if (left.peer_count !== right.peer_count) {
      return right.peer_count - left.peer_count;
    }

    return left.channel.localeCompare(right.channel);
  });
}

export function matchesChannelSearch(channel: NetworkChannelSummary, query: string) {
  if (!query) {
    return true;
  }

  const normalized = query.toLowerCase();
  return channel.channel.toLowerCase().includes(normalized);
}

export function sortNetworkPeers(peers: NetworkPeerSummary[]) {
  return [...peers].sort((left, right) => {
    if (left.local !== right.local) {
      return left.local ? -1 : 1;
    }

    const leftSeen = left.last_seen ? new Date(left.last_seen).getTime() : 0;
    const rightSeen = right.last_seen ? new Date(right.last_seen).getTime() : 0;

    if (leftSeen !== rightSeen) {
      return rightSeen - leftSeen;
    }

    return getPeerDisplayName(left).localeCompare(getPeerDisplayName(right));
  });
}

export function matchesPeerSearch(peer: NetworkPeerSummary, query: string) {
  if (!query) {
    return true;
  }

  const normalized = query.toLowerCase();
  return (
    getPeerDisplayName(peer).toLowerCase().includes(normalized) ||
    peer.peer_id.toLowerCase().includes(normalized) ||
    peer.channel.toLowerCase().includes(normalized)
  );
}

export function getMessageAuthorInitial(
  message: Pick<NetworkChannelMessage, "display_name" | "peer_id">
): string {
  const author = (message.display_name ?? message.peer_id).trim();
  return author.charAt(0).toUpperCase() || "?";
}

export function getChannelDetailDescription(channel: NetworkChannel): string {
  if ((channel.sessions?.length ?? 0) > 0) {
    const sessionCount = channel.sessions?.length ?? 0;
    return `Materialized by ${formatNetworkNumber(sessionCount)} ${
      sessionCount === 1 ? "agent session" : "agent sessions"
    }.`;
  }

  return "Read-only coordination timeline for peers visible in this channel.";
}

export function getPeerHeartbeatLabel(peer: Pick<NetworkPeerDetail, "last_seen">): string {
  if (!peer.last_seen) {
    return "Last heartbeat unavailable";
  }

  const relative = formatNetworkRelativeTime(peer.last_seen);
  return relative === "Unavailable" ? "Last heartbeat unavailable" : `Last heartbeat: ${relative}`;
}

export function getPeerDeliveredRate(peer: Pick<NetworkPeerDetail, "metrics">): string {
  const delivered = peer.metrics.delivered ?? 0;
  const received = peer.metrics.received ?? 0;

  if (received <= 0) {
    return "No traffic yet";
  }

  return `${Math.round((delivered / received) * 100)}% rate`;
}

export function sortAgentsForNetwork(agents: AgentPayload[]) {
  return [...agents].sort((left, right) => left.name.localeCompare(right.name));
}
