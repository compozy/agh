import { AlertTriangle, Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Empty } from "@agh/ui";
import {
  type NetworkRouteSearch,
  useNetworkPage,
  validateNetworkSearch,
} from "@/hooks/routes/use-network-page";
import {
  NetworkCreateChannelDialog,
  NetworkWorkspaceShell,
  toggleDraftAgent,
} from "@/systems/network";

export const Route = createFileRoute("/_app/network")({
  validateSearch: validateNetworkSearch,
  component: NetworkPage,
});

function NetworkPage() {
  const page = useNetworkPage(Route.useSearch() as NetworkRouteSearch);

  if (page.isPageLoading) {
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

  if (page.pageError || !page.networkStatus) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="network-error"
      >
        <Empty
          className="max-w-xl"
          description={page.pageError?.message ?? "Failed to load network status"}
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
          description="Enable the embedded network in AGH config to inspect rooms, peers, and multi-kind wire traffic."
          icon={NetworkIcon}
          title="Network disabled"
        />
      </div>
    );
  }

  return (
    <>
      <NetworkWorkspaceShell
        activeKind={page.activeKind}
        activeRoom={page.activeRoom}
        channelRooms={page.channelRooms}
        composeDraft={page.composeDraft}
        detailsTab={page.detailsTab}
        directRooms={page.directRooms}
        isComposePending={page.isComposePending}
        isDetailsOpen={page.isDetailsOpen}
        isRoomLoading={page.isRoomLoading}
        isTimelineLoading={page.isTimelineLoading}
        onComposeDraftChange={page.setComposeDraft}
        onComposeSubmit={page.handleComposeSubmit}
        onOpenCreateDialog={page.handleOpenCreateDialog}
        onSelectDetailsTab={page.setDetailsTab}
        onSelectKind={page.handleSetKind}
        onSelectRoom={page.handleSelectRoom}
        onSidebarQueryChange={page.setSidebarQuery}
        onToggleDetails={page.handleToggleDetails}
        onToggleStarChannel={page.handleToggleStarChannel}
        roomError={page.roomError}
        selectedRoomKey={page.selectedRoomKey}
        sidebarQuery={page.sidebarQuery}
        starredChannelRooms={page.starredChannelRooms}
        status={page.networkStatus}
      />

      <NetworkCreateChannelDialog
        agents={page.sortedAgents}
        canSubmit={
          Boolean(page.networkStatus?.enabled) &&
          page.createDraft.channelName.trim() !== "" &&
          page.createDraft.purpose.trim() !== "" &&
          page.createDraft.selectedAgentNames.length > 0
        }
        draft={page.createDraft}
        isSubmitting={page.isCreatePending}
        onChannelNameChange={channelName =>
          page.setCreateDraft(currentDraft => ({
            ...currentDraft,
            channelName,
          }))
        }
        onOpenChange={page.setCreateDialogOpen}
        onPurposeChange={purpose =>
          page.setCreateDraft(currentDraft => ({
            ...currentDraft,
            purpose,
          }))
        }
        onSubmit={page.handleCreateChannel}
        onToggleAgent={agentName =>
          page.setCreateDraft(currentDraft => toggleDraftAgent(currentDraft, agentName))
        }
        open={page.isCreateDialogOpen}
        workspaceName={page.workspaceName}
      />
    </>
  );
}
