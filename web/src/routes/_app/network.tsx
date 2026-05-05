import { AlertTriangle, Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Outlet } from "@tanstack/react-router";

import { Empty } from "@agh/ui";

import { NetworkShell } from "@/systems/network/components/shell";
import { useNetworkRouteShell } from "@/systems/network";

export const Route = createFileRoute("/_app/network")({
  component: NetworkRouteShell,
});

function NetworkRouteShell() {
  const { page, activeChannel, activeTab, hasUnread } = useNetworkRouteShell();

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
        <Empty
          className="max-w-xl"
          description={page.statusError?.message ?? "Failed to load network status"}
          icon={AlertTriangle}
          title="Unable to load the network workspace"
        />
      </div>
    );
  }

  if (page.isNetworkDisabled) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="network-disabled-state"
      >
        <Empty
          className="max-w-xl"
          description="Enable the embedded network in AGH config to inspect channels, threads, and direct rooms."
          icon={NetworkIcon}
          title="Network disabled"
        />
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
            description="Create a channel from the CLI or extension SDK to start coordinating threads and direct rooms."
            icon={NetworkIcon}
            title="No channels yet."
          />
        </div>
      </NetworkShell>
    );
  }

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
      openWorkCount={0}
      pinnedChannels={page.pinnedChannels}
      recents={page.recents}
      rightRailMode="thread"
      rightRailOpen={false}
      threadCount={null}
      unpinnedChannels={page.unpinnedChannels}
    >
      <Outlet />
    </NetworkShell>
  );
}
