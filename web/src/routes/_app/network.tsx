import { Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Empty } from "@agh/ui";

import {
  DaemonDown,
  NetworkEmpty,
  ThreadOverlay,
  useNetworkRouteShell,
  useOpenWork,
  useThreadViewMode,
  WorkInspector,
} from "@/systems/network";
import { NetworkShell } from "@/systems/network/components/shell";

export const Route = createFileRoute("/_app/network")({
  component: NetworkRouteShell,
});

function NetworkRouteShell() {
  const { page, activeChannel, activeTab, activeThreadId, activeDirectId, hasUnread } =
    useNetworkRouteShell();
  const viewMode = useThreadViewMode();
  const showOverlayInRightRail = activeThreadId != null && viewMode === "overlay";
  const containerSurface = activeThreadId
    ? ("thread" as const)
    : activeDirectId
      ? ("direct" as const)
      : null;
  const containerId = activeThreadId ?? activeDirectId ?? null;
  const channelKey = activeChannel?.channel ?? null;
  const openWork = useOpenWork({
    channel: channelKey,
    surface: containerSurface,
    containerId,
    enabled: Boolean(channelKey) && containerSurface != null,
  });
  const showInspectorInRightRail = !showOverlayInRightRail && openWork.openCount > 0;

  if (page.isStatusLoading) {
    return (
      <div
        aria-label="Loading network workspace"
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="network-loading"
        role="status"
      >
        <Loader2
          aria-hidden="true"
          className="size-5 animate-spin text-[color:var(--color-text-tertiary)]"
        />
      </div>
    );
  }

  if (page.statusError || !page.status) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="network-error"
      >
        <DaemonDown />
      </div>
    );
  }

  if (page.isNetworkDisabled) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="network-disabled-state"
      >
        <NetworkEmpty />
      </div>
    );
  }

  if (page.channels.length === 0 && !page.isChannelsLoading) {
    return (
      <NetworkShell
        activeChannel={null}
        activeChannelDetail={null}
        activeTab="threads"
        directCount={null}
        hasUnread={() => false}
        isChannelsLoading={false}
        isPinned={() => false}
        isRecentsLoading={false}
        onTogglePinned={page.togglePinned}
        openWorkCount={0}
        pinnedChannels={[]}
        recents={[]}
        rightRailMode="thread"
        rightRailOpen={false}
        threadCount={null}
        unpinnedChannels={[]}
      >
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="network-no-channels-state"
        >
          <Empty
            className="max-w-xl"
            description="Create one or accept an invite."
            icon={NetworkIcon}
            title="No channels yet."
          />
        </div>
      </NetworkShell>
    );
  }

  const rightRailContent =
    showOverlayInRightRail && activeChannel && activeThreadId ? (
      <ThreadOverlay channel={activeChannel.channel} fullPage={false} threadId={activeThreadId} />
    ) : showInspectorInRightRail ? (
      <WorkInspector entries={openWork.entries} isLoading={openWork.isLoading} />
    ) : null;

  return (
    <NetworkShell
      activeChannel={activeChannel}
      activeChannelDetail={null}
      activeTab={activeTab}
      directCount={null}
      hasUnread={hasUnread}
      isChannelsLoading={page.isChannelsLoading}
      isPinned={page.isPinned}
      isRecentsLoading={page.isRecentsLoading}
      onTogglePinned={page.togglePinned}
      openWorkCount={openWork.openCount}
      pinnedChannels={page.pinnedChannels}
      recents={page.recents}
      rightRailContent={rightRailContent}
      rightRailMode={showOverlayInRightRail ? "thread" : "work"}
      rightRailOpen={showOverlayInRightRail || showInspectorInRightRail}
      threadCount={null}
      unpinnedChannels={page.unpinnedChannels}
    >
      <Outlet />
    </NetworkShell>
  );
}
