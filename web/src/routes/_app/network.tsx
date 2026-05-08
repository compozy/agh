import { type ReactNode } from "react";
import { Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Empty, PageHeader } from "@agh/ui";

import { DaemonDown, NetworkEmpty, ThreadOverlay, useNetworkRouteView } from "@/systems/network";
import { NetworkInspector, NetworkShell } from "@/systems/network/components/shell";

export const Route = createFileRoute("/_app/network")({
  component: NetworkRouteShell,
});

function NetworkPageShell({
  count,
  meta,
  children,
}: {
  count: number | undefined;
  meta?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="network-page-shell">
      <PageHeader
        count={count}
        icon={() => <NetworkIcon className="size-3.5" data-testid="network-page-icon" />}
        meta={meta}
        title={<span data-testid="network-page-title">Network</span>}
      />
      <div className="flex min-h-0 flex-1">{children}</div>
    </div>
  );
}

function NetworkRouteShell() {
  const view = useNetworkRouteView();
  const { page, activeChannel, activeTab, activeThreadId, activeDirectId, hasUnread } = view;
  const {
    inspector,
    members: channelMembers,
    threads: channelThreads,
    directs: channelDirects,
  } = view.inspectorView;
  const showInspectorInRightRail = !view.showOverlayInRightRail && inspector.open;

  if (page.isStatusLoading) {
    return (
      <>
        <NetworkPageShell count={undefined}>
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
        </NetworkPageShell>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.statusError || !page.status) {
    return (
      <>
        <NetworkPageShell count={undefined}>
          <div
            className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
            data-testid="network-error"
          >
            <DaemonDown />
          </div>
        </NetworkPageShell>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.isNetworkDisabled) {
    return (
      <>
        <NetworkPageShell count={undefined}>
          <div
            className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
            data-testid="network-disabled-state"
          >
            <NetworkEmpty onOpenSettings={view.networkCreate.openNetworkSettings} />
          </div>
        </NetworkPageShell>
        {view.networkCreate.dialog}
      </>
    );
  }

  if (page.channels.length === 0 && !page.isChannelsLoading) {
    return (
      <>
        <NetworkPageShell count={0} meta={view.networkCreate.action}>
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
        {view.networkCreate.dialog}
      </>
    );
  }

  const rightRailContent =
    view.showOverlayInRightRail && activeChannel && activeThreadId ? (
      <ThreadOverlay channel={activeChannel.channel} fullPage={false} threadId={activeThreadId} />
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

  return (
    <>
      <NetworkPageShell count={page.channels.length} meta={view.networkCreate.action}>
        <NetworkShell
          activeChannel={activeChannel}
          activeChannelDetail={null}
          activeDirectId={activeDirectId}
          activeTab={activeTab}
          directCount={null}
          directs={view.railView.directs.directs}
          hasUnread={hasUnread}
          inspectorOpen={inspector.open}
          isChannelsLoading={page.isChannelsLoading}
          isDirectsLoading={view.railView.directs.isLoading}
          isPinned={page.isPinned}
          isRecentsLoading={page.isRecentsLoading}
          onInspectorToggle={inspector.toggle}
          onTogglePinned={page.togglePinned}
          openWorkCount={view.openWork.openCount}
          pinnedChannels={page.pinnedChannels}
          recents={page.recents}
          rightRailContent={rightRailContent}
          rightRailMode={view.showOverlayInRightRail ? "thread" : "inspector"}
          rightRailOpen={view.showOverlayInRightRail || showInspectorInRightRail}
          selfPeerId={view.railView.session.session?.peerId ?? null}
          threadCount={null}
          unpinnedChannels={page.unpinnedChannels}
        >
          <Outlet />
        </NetworkShell>
      </NetworkPageShell>
      {view.networkCreate.dialog}
    </>
  );
}
