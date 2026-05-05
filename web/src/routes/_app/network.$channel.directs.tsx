import { useState } from "react";
import { Plus } from "lucide-react";
import { createFileRoute, Outlet, useParams } from "@tanstack/react-router";

import { Button } from "@agh/ui";

import {
  DirectsEmpty,
  DirectsList,
  NewDirectDialog,
  useActiveNetworkSession,
  useNetworkDirects,
} from "@/systems/network";

interface DirectDetailParams {
  directId?: string;
}

export const Route = createFileRoute("/_app/network/$channel/directs")({
  component: NetworkChannelDirectsRoute,
});

function NetworkChannelDirectsRoute() {
  const { channel } = Route.useParams();
  const detailParams = useParams({ strict: false }) as DirectDetailParams;
  const directsQuery = useNetworkDirects(channel);
  const activeSession = useActiveNetworkSession(channel);
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

  const showEmpty = !directsQuery.isLoading && directsQuery.directs.length === 0;
  const sessionId = activeSession.session?.sessionId ?? "";

  return (
    <section
      aria-label={`Direct rooms in #${channel}`}
      className="flex min-h-0 flex-1 flex-col"
      data-testid="network-directs-tab"
    >
      <header className="flex items-center justify-end border-b border-[color:var(--color-divider)] px-5 py-2">
        <Button
          aria-label="Open new direct room"
          data-testid="network-directs-new-direct"
          disabled={!sessionId}
          onClick={() => setNewDirectOpen(true)}
          size="sm"
          type="button"
          variant="outline"
        >
          <Plus aria-hidden="true" className="size-3.5" />
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
          activeDirectId={null}
          channel={channel}
          directs={directsQuery.directs}
          isLoading={directsQuery.isLoading}
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
