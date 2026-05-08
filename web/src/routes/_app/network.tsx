import { type ReactNode } from "react";
import { Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Empty, PageHeader } from "@agh/ui";

import {
  DaemonDown,
  NetworkEmpty,
  ThreadOverlay,
  useNetworkInspectorView,
  useNetworkRailView,
  useNetworkRouteShell,
  useOpenWork,
  useThreadViewMode,
} from "@/systems/network";
import { NetworkInspector, NetworkShell } from "@/systems/network/components/shell";

export const Route = createFileRoute("/_app/network")({
  component: NetworkRouteShell,
});

function NetworkPageShell({ count, children }: { count: number | undefined; children: ReactNode }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="network-page-shell">
      <PageHeader
        count={count}
        icon={() => <NetworkIcon className="size-3.5" data-testid="network-page-icon" />}
        title={<span data-testid="network-page-title">Network</span>}
      />
      <div className="flex min-h-0 flex-1">{children}</div>
    </div>
  );
}

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
  const inspectorView = useNetworkInspectorView({
    channel: channelKey,
    enabled: Boolean(channelKey) && !showOverlayInRightRail,
  });
  const railView = useNetworkRailView({ channel: channelKey });
  const {
    inspector,
    members: channelMembers,
    threads: channelThreads,
    directs: channelDirects,
  } = inspectorView;
  const showInspectorInRightRail = !showOverlayInRightRail && inspector.open;

  if (page.isStatusLoading) {
    return (
      <NetworkPageShell count={undefined}>
        <div
          aria-label="Loading network workspace"
          className="flex min-h-0 flex-1 items-center justify-center"
          data-testid="network-loading"
          role="status"
        >
          <Loader2
            aria-hidden="true"
            className="size-5 animate-spin text-(--color-text-tertiary)"
          />
        </div>
      </NetworkPageShell>
    );
  }

  if (page.statusError || !page.status) {
    return (
      <NetworkPageShell count={undefined}>
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="network-error"
        >
          <DaemonDown />
        </div>
      </NetworkPageShell>
    );
  }

  if (page.isNetworkDisabled) {
    return (
      <NetworkPageShell count={undefined}>
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="network-disabled-state"
        >
          <NetworkEmpty />
        </div>
      </NetworkPageShell>
    );
  }

  if (page.channels.length === 0 && !page.isChannelsLoading) {
    return (
      <NetworkPageShell count={0}>
        <NetworkShell
          activeChannel={null}
          activeChannelDetail={null}
          activeDirectId={null}
          activeTab="threads"
          directCount={null}
          directs={[]}
          hasUnread={() => false}
          inspectorOpen={false}
          isChannelsLoading={false}
          isDirectsLoading={false}
          isPinned={() => false}
          isRecentsLoading={false}
          onInspectorToggle={() => undefined}
          onTogglePinned={page.togglePinned}
          openWorkCount={0}
          pinnedChannels={[]}
          recents={[]}
          rightRailMode="thread"
          rightRailOpen={false}
          selfPeerId={null}
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
      </NetworkPageShell>
    );
  }

  const rightRailContent =
    showOverlayInRightRail && activeChannel && activeThreadId ? (
      <ThreadOverlay channel={activeChannel.channel} fullPage={false} threadId={activeThreadId} />
    ) : showInspectorInRightRail && activeChannel ? (
      <NetworkInspector
        activeTab={inspector.tab}
        channel={activeChannel.channel}
        directs={channelDirects.directs}
        isActivityLoading={channelThreads.isLoading || channelDirects.isLoading}
        isMembersLoading={channelMembers.isLoading}
        isWorkLoading={openWork.isLoading}
        members={channelMembers.members}
        onClose={inspector.close}
        onTabChange={inspector.setTab}
        threads={channelThreads.threads}
        workCount={openWork.openCount}
        workEntries={openWork.entries}
      />
    ) : null;

  return (
    <NetworkPageShell count={page.channels.length}>
      <NetworkShell
        activeChannel={activeChannel}
        activeChannelDetail={null}
        activeDirectId={activeDirectId}
        activeTab={activeTab}
        directCount={null}
        directs={railView.directs.directs}
        hasUnread={hasUnread}
        inspectorOpen={inspector.open}
        isChannelsLoading={page.isChannelsLoading}
        isDirectsLoading={railView.directs.isLoading}
        isPinned={page.isPinned}
        isRecentsLoading={page.isRecentsLoading}
        onInspectorToggle={inspector.toggle}
        onTogglePinned={page.togglePinned}
        openWorkCount={openWork.openCount}
        pinnedChannels={page.pinnedChannels}
        recents={page.recents}
        rightRailContent={rightRailContent}
        rightRailMode={showOverlayInRightRail ? "thread" : "inspector"}
        rightRailOpen={showOverlayInRightRail || showInspectorInRightRail}
        selfPeerId={railView.session.session?.peerId ?? null}
        threadCount={null}
        unpinnedChannels={page.unpinnedChannels}
      >
        <Outlet />
      </NetworkShell>
    </NetworkPageShell>
  );
}
