import type { ChannelTab } from "../components/shell/channel-tabs-types";

export interface NetworkWindowLocation {
  pathname: string;
  search: Record<string, unknown>;
}

export interface ParsedNetworkWindowLocation {
  workspaceId: string | null;
  channel: string | null;
  activeTab: ChannelTab;
  activeThreadId: string | null;
  activeDirectId: string | null;
  view: "full" | undefined;
}

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function parseNetworkWindowLocation(
  location: NetworkWindowLocation
): ParsedNetworkWindowLocation {
  const match = /^\/network\/([^/]+)\/([^/]+)\/(threads|directs|activity)(?:\/([^/]+))?$/.exec(
    location.pathname
  );
  if (!match) {
    return {
      workspaceId: null,
      channel: null,
      activeTab: "threads",
      activeThreadId: null,
      activeDirectId: null,
      view: undefined,
    };
  }

  const workspaceId = decodePathSegment(match[1]);
  const channel = decodePathSegment(match[2]);
  const activeTab = match[3] as ChannelTab;
  const detailId = match[4] ? decodePathSegment(match[4]) : null;
  return {
    workspaceId,
    channel,
    activeTab,
    activeThreadId: activeTab === "threads" ? detailId : null,
    activeDirectId: activeTab === "directs" ? detailId : null,
    view: activeTab === "threads" && location.search.view === "full" ? "full" : undefined,
  };
}

export function networkThreadsLocation(
  workspaceId: string,
  channel: string,
  threadId?: string,
  view?: "full"
): NetworkWindowLocation {
  const base = `/network/${encodeURIComponent(workspaceId)}/${encodeURIComponent(channel)}/threads`;
  return {
    pathname: threadId ? `${base}/${encodeURIComponent(threadId)}` : base,
    search: view ? { view } : {},
  };
}

export interface NetworkWindowTrail {
  /** Parent crumbs for the drill-in head (empty at the root). */
  parents: ReadonlyArray<{ id: "root" | "channel"; label: string }>;
  /** Leaf title (window H1). */
  leaf: string;
  /** Whether the head shows the back affordance (any drilled-in level). */
  drilledIn: boolean;
}

/**
 * Drill-in trail for the unified 44px head (os/pagehead contract §03).
 * Selecting a channel is rail navigation, not a drill — the head stays at the
 * root level (`Network` + channel count). The trail only appears once a
 * conversation (thread or direct) is open: Network / #channel / <leaf>.
 */
export function networkWindowTrail(
  location: ParsedNetworkWindowLocation,
  conversationLabel?: string | null
): NetworkWindowTrail {
  const inConversation = Boolean(location.activeThreadId || location.activeDirectId);
  if (!location.channel || !inConversation) {
    return { parents: [], leaf: "Network", drilledIn: false };
  }
  return {
    parents: [
      { id: "root", label: "Network" },
      { id: "channel", label: `#${location.channel}` },
    ],
    leaf: conversationLabel ?? (location.activeThreadId ? "Thread" : "Direct"),
    drilledIn: true,
  };
}
