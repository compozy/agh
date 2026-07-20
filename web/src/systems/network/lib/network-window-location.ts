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

export function networkWindowCrumb(location: ParsedNetworkWindowLocation): string {
  if (!location.channel) return "Network";
  if (location.activeThreadId) return `Network / #${location.channel} / Thread`;
  if (location.activeDirectId) return `Network / #${location.channel} / Direct`;
  const label =
    location.activeTab === "activity"
      ? "Activity"
      : location.activeTab === "directs"
        ? "Directs"
        : "Threads";
  return `Network / #${location.channel} / ${label}`;
}
