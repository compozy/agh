import type {
  BridgeDeliveryDefaults,
  BridgeProvider,
  BridgeRoute,
  BridgeScope,
  BridgeStatus,
} from "@/systems/bridges/types";

type BridgeTone = "amber" | "danger" | "green" | "neutral" | "violet";

function normalizeText(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

export function buildBridgeProviderKey(
  provider: Pick<BridgeProvider, "extension_name" | "platform">
) {
  return `${provider.extension_name}::${provider.platform}`;
}

export function findBridgeProviderByKey(providers: BridgeProvider[], providerKey: string) {
  return providers.find(provider => buildBridgeProviderKey(provider) === providerKey);
}

export function isBridgeProviderSelectable(provider: BridgeProvider): boolean {
  return provider.enabled && provider.health !== "unhealthy";
}

export function bridgeStatusTone(status: BridgeStatus): BridgeTone {
  switch (status) {
    case "ready":
      return "green";
    case "auth_required":
      return "violet";
    case "error":
      return "danger";
    case "starting":
    case "degraded":
      return "amber";
    case "disabled":
    default:
      return "neutral";
  }
}

export function bridgeStatusLabel(status: BridgeStatus): string {
  return status.replaceAll("_", " ");
}

export function bridgeScopeTone(scope: BridgeScope): BridgeTone {
  return scope === "workspace" ? "violet" : "neutral";
}

export function bridgeProviderHealthTone(health?: string): BridgeTone {
  switch (health) {
    case "healthy":
      return "green";
    case "unhealthy":
      return "danger";
    default:
      return "neutral";
  }
}

export function bridgeProviderStateTone(state?: string): BridgeTone {
  switch (state) {
    case "active":
      return "green";
    case "error":
      return "danger";
    case "registered":
      return "violet";
    case "enabled":
      return "amber";
    default:
      return "neutral";
  }
}

export function normalizeBridgeDeliveryDefaults(value: unknown): BridgeDeliveryDefaults {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }

  const candidate = value as Record<string, unknown>;
  const mode = candidate.mode;

  return {
    group_id: normalizeText(candidate.group_id),
    mode: mode === "direct-send" || mode === "reply" ? mode : undefined,
    peer_id: normalizeText(candidate.peer_id),
    thread_id: normalizeText(candidate.thread_id),
  };
}

export function compactBridgeDeliveryDefaults(
  value: BridgeDeliveryDefaults
): BridgeDeliveryDefaults | undefined {
  const normalized = {
    group_id: normalizeText(value.group_id),
    mode: value.mode,
    peer_id: normalizeText(value.peer_id),
    thread_id: normalizeText(value.thread_id),
  };

  if (!normalized.group_id && !normalized.mode && !normalized.peer_id && !normalized.thread_id) {
    return undefined;
  }

  return normalized;
}

export interface BridgeRoutingPolicy {
  include_group: boolean;
  include_peer: boolean;
  include_thread: boolean;
}

export function describeBridgeRoutingPolicy(policy: BridgeRoutingPolicy): string {
  const parts: string[] = [];

  if (policy.include_peer) {
    parts.push("peer");
  }
  if (policy.include_group) {
    parts.push("group");
  }
  if (policy.include_thread) {
    parts.push("thread");
  }

  return parts.length > 0 ? parts.join(" + ") : "No routing dimensions enabled";
}

export function describeBridgeDeliveryDefaults(value: unknown): string {
  const defaults = normalizeBridgeDeliveryDefaults(value);
  const parts: string[] = [];

  if (defaults.mode) {
    parts.push(defaults.mode);
  }
  if (defaults.peer_id) {
    parts.push(`peer:${defaults.peer_id}`);
  }
  if (defaults.group_id) {
    parts.push(`group:${defaults.group_id}`);
  }
  if (defaults.thread_id) {
    parts.push(`thread:${defaults.thread_id}`);
  }

  return parts.length > 0 ? parts.join(" · ") : "No delivery defaults configured";
}

export function formatBridgeDateTime(value?: string | null): string {
  if (!value) {
    return "Unavailable";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("en-US", {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
    year: "numeric",
  });
}

export function formatBridgeRelativeTime(value?: string | null): string {
  if (!value) {
    return "Never";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const diffMs = date.getTime() - Date.now();
  if (Math.abs(diffMs) < 1000 * 60) {
    return "Just now";
  }

  const diffMinutes = Math.floor(Math.abs(diffMs) / (1000 * 60));
  if (diffMinutes < 60) {
    return diffMs >= 0 ? `In ${diffMinutes}m` : `${diffMinutes}m ago`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return diffMs >= 0 ? `In ${diffHours}h` : `${diffHours}h ago`;
  }

  const diffDays = Math.floor(diffHours / 24);
  return diffMs >= 0 ? `In ${diffDays}d` : `${diffDays}d ago`;
}

export function describeBridgeRouteTarget(
  route: Pick<BridgeRoute, "group_id" | "peer_id" | "thread_id">
) {
  const parts: string[] = [];

  if (route.peer_id) {
    parts.push(`peer:${route.peer_id}`);
  }
  if (route.group_id) {
    parts.push(`group:${route.group_id}`);
  }
  if (route.thread_id) {
    parts.push(`thread:${route.thread_id}`);
  }

  return parts.length > 0 ? parts.join(" · ") : "default target";
}

export function describeBridgeTestTarget(
  target: Pick<BridgeDeliveryDefaults, "group_id" | "mode" | "peer_id" | "thread_id">
): string {
  const parts: string[] = [];

  if (target.mode) {
    parts.push(target.mode);
  }
  if (target.peer_id) {
    parts.push(`peer:${target.peer_id}`);
  }
  if (target.group_id) {
    parts.push(`group:${target.group_id}`);
  }
  if (target.thread_id) {
    parts.push(`thread:${target.thread_id}`);
  }

  return parts.length > 0 ? parts.join(" · ") : "Bridge defaults";
}
