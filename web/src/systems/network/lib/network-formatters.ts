import type { AgentPayload } from "@/systems/agent";

import type {
  NetworkCapabilityBrief,
  NetworkCapabilityCatalog,
  NetworkChannel,
  NetworkChannelSummary,
  NetworkCreateChannelDraft,
  NetworkKindFilter,
  NetworkPeerCapabilityView,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkSignalTone,
  NetworkTimelineMessage,
  NetworkStatus,
  NetworkRoomType,
} from "../types";

const numberFormatter = new Intl.NumberFormat("en-US");
const timeFormatter = new Intl.DateTimeFormat("en-US", {
  hour: "numeric",
  minute: "2-digit",
});

export const NETWORK_KIND_FILTERS: Exclude<NetworkKindFilter, "all">[] = [
  "say",
  "direct",
  "receipt",
  "capability",
  "greet",
  "whois",
  "trace",
];

const NETWORK_KIND_LABELS: Record<Exclude<NetworkKindFilter, "all">, string> = {
  capability: "recipe",
  direct: "direct",
  greet: "greet",
  receipt: "receipt",
  say: "say",
  trace: "trace",
  whois: "whois",
};

const NETWORK_KIND_TONES: Record<Exclude<NetworkKindFilter, "all">, NetworkSignalTone> = {
  capability: "info",
  direct: "accent",
  greet: "success",
  receipt: "warning",
  say: "neutral",
  trace: "info",
  whois: "warning",
};

interface NetworkMetricCard {
  detail?: string;
  label: string;
  value: string;
}

function parseTimestampOrZero(value?: string | null): number {
  if (!value) {
    return 0;
  }

  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

export function createNetworkChannelDraft(): NetworkCreateChannelDraft {
  return {
    channelName: "",
    purpose: "",
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

export function formatChannelMemberCount(
  channel?: Pick<NetworkChannel, "peer_count" | "session_count"> | null
): string {
  const count = channel?.peer_count ?? channel?.session_count ?? 0;
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

export function getPeerPresenceTone(
  peer: Pick<NetworkPeerSummary, "last_seen" | "local">
): NetworkSignalTone {
  if (peer.local) {
    return "accent";
  }

  if (!peer.last_seen) {
    return "neutral";
  }

  const parsed = new Date(peer.last_seen);
  if (Number.isNaN(parsed.getTime())) {
    return "neutral";
  }

  return Date.now() - parsed.getTime() <= 60_000 ? "success" : "neutral";
}

export function getNetworkStatusTone(status?: string | null): NetworkSignalTone {
  switch (status?.trim()) {
    case "online":
    case "running":
      return "success";
    case "starting":
    case "degraded":
      return "warning";
    case "offline":
    case "stopped":
      return "danger";
    default:
      return "neutral";
  }
}

export function getNetworkMetricCards(
  status: NetworkStatus | undefined,
  channelCount: number
): NetworkMetricCard[] {
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

export function getNetworkRoomKey(roomType: NetworkRoomType, id: string): string {
  return `${roomType}:${id.trim()}`;
}

export function sortNetworkChannels(channels: NetworkChannelSummary[]) {
  return [...channels].sort((left, right) => {
    const leftActivity = parseTimestampOrZero(left.last_activity_at);
    const rightActivity = parseTimestampOrZero(right.last_activity_at);

    if (leftActivity !== rightActivity) {
      return rightActivity - leftActivity;
    }

    if ((left.message_count ?? 0) !== (right.message_count ?? 0)) {
      return (right.message_count ?? 0) - (left.message_count ?? 0);
    }

    return left.channel.localeCompare(right.channel);
  });
}

export function matchesChannelSearch(channel: NetworkChannelSummary, query: string) {
  if (!query) {
    return true;
  }

  const normalized = query.toLowerCase();
  return (
    channel.channel.toLowerCase().includes(normalized) ||
    channel.purpose?.toLowerCase().includes(normalized) === true ||
    channel.last_message_preview?.toLowerCase().includes(normalized) === true
  );
}

export function sortNetworkPeers(peers: NetworkPeerSummary[]) {
  return [...peers].sort((left, right) => {
    if (left.local !== right.local) {
      return left.local ? -1 : 1;
    }

    const leftSeen = parseTimestampOrZero(left.last_seen);
    const rightSeen = parseTimestampOrZero(right.last_seen);

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

export function formatNetworkKindLabel(kind: string): string {
  const normalized = toNetworkKindFilter(kind);
  return normalized ? NETWORK_KIND_LABELS[normalized] : kind;
}

export function getNetworkKindTone(kind: string): NetworkSignalTone {
  const normalized = toNetworkKindFilter(kind);
  return normalized ? NETWORK_KIND_TONES[normalized] : "neutral";
}

export function toNetworkKindFilter(kind: string): Exclude<NetworkKindFilter, "all"> | null {
  if (NETWORK_KIND_FILTERS.includes(kind as Exclude<NetworkKindFilter, "all">)) {
    return kind as Exclude<NetworkKindFilter, "all">;
  }

  return null;
}

export function filterNetworkMessagesByKind(
  messages: NetworkTimelineMessage[],
  kind: NetworkKindFilter
): NetworkTimelineMessage[] {
  if (kind === "all") {
    return messages;
  }

  return messages.filter(message => message.kind === kind);
}

export function getNetworkMessagePrimaryText(message: NetworkTimelineMessage): string {
  const preview = message.preview_text?.trim();
  if (preview) {
    return preview;
  }

  const text = message.text?.trim();
  if (text) {
    return text;
  }

  return `(${formatNetworkKindLabel(message.kind)})`;
}

export function getMessageAuthorInitial(
  message: Pick<NetworkTimelineMessage, "display_name" | "peer_from">
): string {
  const author = (message.display_name ?? message.peer_from).trim();
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

export function buildPeerCapabilityViews(
  brief: readonly NetworkCapabilityBrief[] | undefined,
  catalog: NetworkCapabilityCatalog | null | undefined
): NetworkPeerCapabilityView[] {
  const briefList = brief ?? [];
  const catalogList = catalog?.capabilities ?? [];

  const byId = new Map<string, NetworkPeerCapabilityView>();
  for (const entry of briefList) {
    byId.set(entry.id, { id: entry.id, summary: entry.summary, detail: null });
  }

  for (const detail of catalogList) {
    const existing = byId.get(detail.id);
    if (existing) {
      existing.detail = detail;
      if (!existing.summary) {
        existing.summary = detail.summary;
      }
      continue;
    }

    byId.set(detail.id, { id: detail.id, summary: detail.summary, detail });
  }

  return [...byId.values()].sort((left, right) => left.id.localeCompare(right.id));
}

export function hasCapabilityDetail(view: NetworkPeerCapabilityView): boolean {
  const detail = view.detail;
  if (!detail) {
    return false;
  }

  return Boolean(
    detail.outcome ||
    detail.version ||
    (detail.requirements?.length ?? 0) > 0 ||
    (detail.context_needed?.length ?? 0) > 0 ||
    (detail.artifacts_expected?.length ?? 0) > 0 ||
    (detail.execution_outline?.length ?? 0) > 0 ||
    (detail.constraints?.length ?? 0) > 0 ||
    (detail.examples?.length ?? 0) > 0
  );
}
