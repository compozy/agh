import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Spinner, useTopbarSlot } from "@agh/ui";

import type { TopbarRouteContext } from "@/types/topbar";
import {
  DaemonDown,
  NetworkEmpty,
  ThreadOverlay,
  useNetworkListFilters,
  useNetworkRouteView,
} from "@/systems/network";
import { NetworkListFiltersProvider } from "@/systems/network/contexts/network-list-filters-context";
import { NetworkInspector, NetworkShell } from "@/systems/network/components/shell";
import { preloadNetworkRootRoute } from "./-network-preload";

export const Route = createFileRoute("/_app/network")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Network", to: "/network" } },
  }),
  loader: ({ context }) => preloadNetworkRootRoute(context.queryClient),
  component: NetworkRouteShell,
});

function NetworkRouteShell() {
  const view = useNetworkRouteView();
  const { page, activeChannel, activeTab, activeThreadId, activeDirectId, hasUnread } = view;
  const workspaceId = view.activeWorkspaceId ?? "";
  const {
    inspector,
    members: channelMembers,
    threads: channelThreads,
    directs: channelDirects,
  } = view.inspectorView;
  const showInspectorInRightRail = !view.showOverlayInRightRail && inspector.open;

  useTopbarSlot({
    actions: page.status ? view.networkCreate.action : undefined,
  });

  const activeChannelKey = activeChannel?.channel ?? null;
  const filters = useNetworkListFilters({
    workspaceId,
    channel: activeChannelKey ?? "",
    enabled: Boolean(activeChannelKey),
  });

  if (page.isStatusLoading) {
    return (
      <>
        <div
          aria-label="Loading network workspace"
          className="flex min-h-0 flex-1 items-center justify-center"
          data-testid="network-loading"
          role="status"
        >
          <Spinner aria-hidden="true" className="size-5 text-subtle" />
        </div>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.statusError || !page.status) {
    return (
      <>
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="network-error"
        >
          <DaemonDown />
        </div>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.isNetworkDisabled) {
    return (
      <>
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="network-disabled-state"
        >
          <NetworkEmpty disabledByAdmin onOpenSettings={view.networkCreate.openNetworkSettings} />
        </div>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.channels.length === 0 && !page.isChannelsLoading) {
    return (
      <>
        <NetworkListFiltersProvider value={filters}>
          <NetworkShell
            workspaceId={workspaceId}
            activeChannel={null}
            activeChannelDetail={null}
            activeDirectId={null}
            activeTab="threads"
            directCount={null}
            directs={[]}
            hasUnread={() => false}
            inspectorOpen={false}
            loading={{ channels: false, directs: false, recents: false }}
            isPinned={() => false}
            onInspectorToggle={() => undefined}
            onTogglePinned={page.togglePinned}
            openWorkCount={0}
            pinnedChannels={[]}
            recents={[]}
            rightRailMode="thread"
            rightRailOpen={false}
            selfSessionId={null}
            threadCount={null}
            unpinnedChannels={[]}
          >
            <div
              className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
              data-testid="network-no-channels-state"
            >
              <NetworkEmpty
                className="max-w-xl"
                onOpenSettings={view.networkCreate.openNetworkSettings}
              />
            </div>
          </NetworkShell>
        </NetworkListFiltersProvider>
        {view.networkCreate.dialog}
      </>
    );
  }

  const rightRailContent =
    view.showOverlayInRightRail && activeChannel && activeThreadId ? (
      <ThreadOverlay
        workspaceId={workspaceId}
        channel={activeChannel.channel}
        fullPage={false}
        threadId={activeThreadId}
      />
    ) : showInspectorInRightRail && activeChannel ? (
      <NetworkInspector
        activeTab={inspector.tab}
        channel={activeChannel.channel}
        directs={channelDirects.directs}
        isActivityLoading={channelThreads.isLoading || channelDirects.isLoading}
        isMembersLoading={channelMembers.isLoading}
        isWorkLoading={view.openWork.isLoading}
        members={channelMembers.members}
        onClose={inspector.close}
        onTabChange={inspector.setTab}
        threads={channelThreads.threads}
        workCount={view.openWork.openCount}
        workEntries={view.openWork.entries}
      />
    ) : null;

  const threadCount = activeChannelKey ? filters.threadsQuery.total : null;
  const directCount = activeChannelKey ? filters.directsQuery.total : null;

  return (
    <>
      <NetworkListFiltersProvider value={filters}>
        <NetworkShell
          workspaceId={workspaceId}
          activeChannel={activeChannel}
          activeChannelDetail={null}
          activeDirectId={activeDirectId}
          activeTab={activeTab}
          directCount={directCount}
          directs={view.railView.directs.directs}
          hasUnread={hasUnread}
          inspectorOpen={inspector.open}
          loading={{
            channels: page.isChannelsLoading,
            directs: view.railView.directs.isLoading,
            recents: page.isRecentsLoading,
          }}
          isPinned={page.isPinned}
          onInspectorToggle={inspector.toggle}
          onTogglePinned={page.togglePinned}
          openWorkCount={view.openWork.openCount}
          pinnedChannels={page.pinnedChannels}
          recents={page.recents}
          rightRailContent={rightRailContent}
          rightRailMode={view.showOverlayInRightRail ? "thread" : "inspector"}
          rightRailOpen={view.showOverlayInRightRail || showInspectorInRightRail}
          selfSessionId={view.railView.session.session?.sessionId ?? null}
          threadCount={threadCount}
          unpinnedChannels={page.unpinnedChannels}
        >
          <Outlet />
        </NetworkShell>
      </NetworkListFiltersProvider>
      {view.networkCreate.dialog}
    </>
  );
}
