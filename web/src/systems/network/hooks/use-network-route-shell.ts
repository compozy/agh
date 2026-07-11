import { useEffect, useMemo } from "react";
import { useChildMatches, useNavigate, useParams } from "@tanstack/react-router";

import type { ChannelTab } from "../components/shell/channel-tabs-types";
import type { NetworkChannelSummary, NetworkRecentEntry } from "../types";
import { useActiveWorkspace } from "@/systems/workspace";
import { useNetworkPage, type UseNetworkPageResult } from "./use-network-page";

const TAB_ROUTES: ReadonlyArray<{ id: ChannelTab; pathFragment: string }> = [
  { id: "threads", pathFragment: "/threads" },
  { id: "directs", pathFragment: "/directs" },
  { id: "activity", pathFragment: "/activity" },
];

function detectActiveTab(pathname: string): ChannelTab {
  for (const tab of TAB_ROUTES) {
    if (pathname.includes(tab.pathFragment)) {
      return tab.id;
    }
  }
  return "threads";
}

export interface NetworkRouteShellResult {
  page: UseNetworkPageResult;
  activeChannel: NetworkChannelSummary | null;
  activeTab: ChannelTab;
  /** Active thread route id when the URL targets `/threads/$threadId`. */
  activeThreadId: string | null;
  /** Active direct-room route id when the URL targets `/directs/$directId`. */
  activeDirectId: string | null;
  activeWorkspaceId: string | null;
  hasUnread: (channelId: string) => boolean;
}

export function useNetworkRouteShell(): NetworkRouteShellResult {
  const { activeWorkspaceId, selectedWorkspaceId, setActiveWorkspaceId } = useActiveWorkspace();
  const navigate = useNavigate();
  const childMatches = useChildMatches();
  const childParams = useParams({ strict: false }) as {
    workspaceId?: string;
    channel?: string;
    threadId?: string;
    directId?: string;
  };
  const childPathname = childMatches.at(-1)?.pathname ?? "";
  const routeWorkspaceId = childParams.workspaceId ?? activeWorkspaceId;
  const routeWorkspaceAllowed =
    childParams.workspaceId == null ||
    selectedWorkspaceId == null ||
    selectedWorkspaceId === childParams.workspaceId;
  const page = useNetworkPage(routeWorkspaceId, { enabled: routeWorkspaceAllowed });

  useEffect(() => {
    if (!childParams.workspaceId || childParams.workspaceId === activeWorkspaceId) {
      return;
    }
    if (selectedWorkspaceId !== null && selectedWorkspaceId !== childParams.workspaceId) {
      void navigate({ to: "/network" });
      return;
    }
    setActiveWorkspaceId(childParams.workspaceId);
  }, [
    activeWorkspaceId,
    childParams.workspaceId,
    navigate,
    selectedWorkspaceId,
    setActiveWorkspaceId,
  ]);

  useEffect(() => {
    if (childParams.workspaceId != null && childParams.channel != null) {
      return;
    }
    const target = page.firstVisibleChannel?.channel;
    if (!target || !routeWorkspaceId) {
      return;
    }
    void navigate({
      params: { workspaceId: routeWorkspaceId, channel: target },
      to: "/network/$workspaceId/$channel/threads",
    });
  }, [
    routeWorkspaceId,
    childParams.channel,
    childParams.workspaceId,
    navigate,
    page.firstVisibleChannel,
  ]);

  return useMemo(() => {
    const activeChannel =
      childParams.workspaceId == null || childParams.workspaceId === routeWorkspaceId
        ? (page.channels.find(channel => channel.channel === childParams.channel) ?? null)
        : null;
    const activeTab = detectActiveTab(childPathname);
    const activeThreadId = childParams.threadId ?? null;
    const activeDirectId = childParams.directId ?? null;

    return {
      page,
      activeChannel,
      activeTab,
      activeThreadId,
      activeDirectId,
      activeWorkspaceId: routeWorkspaceId,
      // A dot means "known unread" from the bounded embedded recents projection.
      // Absence deliberately does not claim the whole channel catalog is read.
      hasUnread: (channelId: string): boolean =>
        page.recents.some(recent => recent.channel === channelId && recent.hasUnread),
    };
  }, [
    routeWorkspaceId,
    childParams.channel,
    childParams.workspaceId,
    childParams.threadId,
    childParams.directId,
    childPathname,
    page,
  ]);
}

export type { NetworkRecentEntry };
