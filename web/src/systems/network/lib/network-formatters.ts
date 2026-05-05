import type { AgentPayload } from "@/systems/agent";

import type {
  NetworkConversationMessage,
  NetworkCreateChannelDraft,
  NetworkKindFilter,
  NetworkPeerSummary,
  NetworkSignalTone,
  NetworkStatus,
} from "../types";

const numberFormatter = new Intl.NumberFormat("en-US");
const timeFormatter = new Intl.DateTimeFormat("en-US", {
  hour: "numeric",
  minute: "2-digit",
});

const NETWORK_SUPPORTED_KINDS: ReadonlyArray<Exclude<NetworkKindFilter, "all">> = [
  "say",
  "receipt",
  "capability",
  "greet",
  "whois",
  "trace",
];

export const NETWORK_KIND_FILTERS: ReadonlyArray<Exclude<NetworkKindFilter, "all">> = [
  "say",
  "receipt",
  "capability",
  "whois",
  "trace",
];

const NETWORK_KIND_LABELS: Record<Exclude<NetworkKindFilter, "all">, string> = {
  capability: "capability",
  greet: "greet",
  receipt: "receipt",
  say: "say",
  trace: "trace",
  whois: "whois",
};

const NETWORK_KIND_TONES: Record<Exclude<NetworkKindFilter, "all">, NetworkSignalTone> = {
  capability: "info",
  greet: "success",
  receipt: "warning",
  say: "neutral",
  trace: "info",
  whois: "warning",
};

function parseTimestampOrZero(value?: string | null): number {
  if (!value) {
    return 0;
  }
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

export function getMostRecentTimestamp(
  primaryValue?: string | null,
  secondaryValue?: string | null
): string | null {
  if (!primaryValue) {
    return secondaryValue ?? null;
  }
  if (!secondaryValue) {
    return primaryValue;
  }

  return parseTimestampOrZero(secondaryValue) > parseTimestampOrZero(primaryValue)
    ? secondaryValue
    : primaryValue;
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

export function getPeerRecencyAt(
  peer: Pick<NetworkPeerSummary, "joined_at" | "last_seen"> | null | undefined
): string | null {
  return getMostRecentTimestamp(peer?.last_seen, peer?.joined_at);
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

export function isNetworkRunning(status?: NetworkStatus | null): boolean {
  if (!status) {
    return false;
  }
  return status.enabled === true && (status.status === "running" || status.status === "online");
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
  if ((NETWORK_SUPPORTED_KINDS as ReadonlyArray<string>).includes(kind)) {
    return kind as Exclude<NetworkKindFilter, "all">;
  }

  return null;
}

export function getMessageAuthorInitial(
  message: Pick<NetworkConversationMessage, "display_name" | "peer_from">
): string {
  const author = (message.display_name ?? message.peer_from ?? "").trim();
  return author.charAt(0).toUpperCase() || "?";
}

export function sortAgentsForNetwork(agents: AgentPayload[]) {
  return [...agents].sort((left, right) => left.name.localeCompare(right.name));
}
