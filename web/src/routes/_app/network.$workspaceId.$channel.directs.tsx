import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";
import { Network as NetworkIcon, Plus } from "lucide-react";
import { useState } from "react";

import { Button, Eyebrow } from "@agh/ui";

import {
  DirectsEmpty,
  DirectsList,
  NewDirectDialog,
  useNetworkChannelDirectsRoute,
  useNetworkListFiltersContext,
} from "@/systems/network";
import type { TopbarRouteContext } from "@/types/topbar";

interface DirectDetailParams {
  directId?: string;
}

export const Route = createFileRoute("/_app/network/$workspaceId/$channel/directs")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `#${params.channel} · Directs`, icon: NetworkIcon },
  }),
  component: NetworkChannelDirectsRoute,
});

function NetworkChannelDirectsRoute() {
  const { workspaceId, channel } = Route.useParams();
  const detailParams = useParams({ strict: false }) as DirectDetailParams;
  const route = useNetworkChannelDirectsRoute(channel);
  const { filteredDirects } = useNetworkListFiltersContext();
  const [newDirectOpen, setNewDirectOpen] = useState(false);

  if (detailParams.directId) {
    return (
      <section
        aria-label={`Direct room ${detailParams.directId} in #${channel}`}
        className="flex min-h-0 flex-1 flex-col"
        data-testid="network-direct-detail-slot"
      >
        <Outlet />
      </section>
    );
  }

  const directsQuery = route.directs;
  const activeSession = route.session;
  const channelMembers = route.members;
  const visibleDirects = filteredDirects;
  const showEmpty = !directsQuery.isLoading && visibleDirects.length === 0;
  const sessionId = activeSession.session?.sessionId ?? "";
  const totalDirects = directsQuery.directs.length;
  const subheaderLabel =
    totalDirects === 1
      ? "1 DIRECT ROOM IN THIS CHANNEL"
      : `${totalDirects} DIRECT ROOMS IN THIS CHANNEL`;

  return (
    <section
      aria-label={`Direct rooms in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-directs-tab"
    >
      <header
        className="flex items-center justify-between gap-3 border-b border-line px-5 py-2"
        data-testid="network-directs-subheader"
      >
        <Eyebrow className="text-subtle">{subheaderLabel}</Eyebrow>
        <Button
          aria-label="Open new direct room"
          data-testid="network-directs-new-direct"
          disabled={!sessionId}
          onClick={() => setNewDirectOpen(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Plus aria-hidden="true" className="size-3" />
          New direct
        </Button>
      </header>

      {showEmpty ? (
        <div className="flex flex-1 items-center justify-center px-6 py-10">
          <DirectsEmpty
            className="max-w-md"
            onNewDirect={sessionId ? () => setNewDirectOpen(true) : undefined}
          />
        </div>
      ) : (
        <DirectsList
          workspaceId={workspaceId}
          activeDirectId={null}
          channel={channel}
          directs={visibleDirects}
          isLoading={directsQuery.isLoading}
          members={channelMembers.members}
          selfPeerId={activeSession.session?.peerId}
        />
      )}

      <NewDirectDialog
        channel={channel}
        onOpenChange={setNewDirectOpen}
        open={newDirectOpen && Boolean(sessionId)}
        selfPeerId={activeSession.session?.peerId}
        sessionId={sessionId}
      />
    </section>
  );
}
