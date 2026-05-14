import { useEffect, useMemo } from "react";
import { useChildMatches, useNavigate, useParams } from "@tanstack/react-router";

import type { ChannelTab } from "../components/shell/channel-tabs-types";
import type { NetworkChannelSummary, NetworkRecentEntry } from "../types";
import { useActiveWorkspace } from "@/systems/workspace";
import { useLastRead } from "./use-last-read";
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
  const page = useNetworkPage();
  const { activeWorkspaceId, setActiveWorkspaceId } = useActiveWorkspace();
  const { lastReadAt } = useLastRead();
  const navigate = useNavigate();
  const childMatches = useChildMatches();
  const childParams = useParams({ strict: false }) as {
    workspaceId?: string;
    channel?: string;
    threadId?: string;
    directId?: string;
  };
  const childPathname = childMatches.at(-1)?.pathname ?? "";

  useEffect(() => {
    if (!childParams.workspaceId || childParams.workspaceId === activeWorkspaceId) {
      return;
    }
    setActiveWorkspaceId(childParams.workspaceId);
  }, [activeWorkspaceId, childParams.workspaceId, setActiveWorkspaceId]);

  useEffect(() => {
    if (childParams.workspaceId != null && childParams.channel != null) {
      return;
    }
    const target = page.firstVisibleChannel?.channel;
    if (!target || !activeWorkspaceId) {
      return;
    }
    void navigate({
      params: { workspaceId: activeWorkspaceId, channel: target },
      to: "/network/$workspaceId/$channel/threads",
    });
  }, [
    activeWorkspaceId,
    childParams.channel,
    childParams.workspaceId,
    navigate,
    page.firstVisibleChannel,
  ]);

  return useMemo(() => {
    const activeChannel =
      childParams.workspaceId === activeWorkspaceId
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
      activeWorkspaceId,
      hasUnread: (channelId: string): boolean => {
        const summary = page.channels.find(channel => channel.channel === channelId);
        if (!summary?.last_activity_at) {
          return false;
        }
        const lastActivityMs = new Date(summary.last_activity_at).getTime();
        if (Number.isNaN(lastActivityMs)) {
          return false;
        }

        let lastReadMs = 0;
        for (const recent of page.recents) {
          if (recent.channel !== channelId) {
            continue;
          }
          const stamp = lastReadAt({
            channel: recent.channel,
            surface: recent.surface,
            containerId: recent.containerId,
          });
          if (!stamp) {
            continue;
          }
          const value = new Date(stamp).getTime();
          if (!Number.isNaN(value) && value > lastReadMs) {
            lastReadMs = value;
          }
        }

        return lastActivityMs > lastReadMs;
      },
    };
  }, [
    activeWorkspaceId,
    childParams.channel,
    childParams.workspaceId,
    childParams.threadId,
    childParams.directId,
    childPathname,
    lastReadAt,
    page,
  ]);
}

export type { NetworkRecentEntry };
