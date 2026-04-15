import { Hash, Network as NetworkIcon, Plus, Users } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { MetricStrip, PillButton } from "@/components/design-system";
import { Button } from "@agh/ui";
import {
  NetworkChannelDetailPanel,
  NetworkChannelsListPanel,
  NetworkCreateChannelDialog,
  NetworkEmptyState,
  NetworkPeerDetailPanel,
  NetworkPeersListPanel,
  toggleDraftAgent,
} from "@/systems/network";
import { useNetworkPage } from "@/hooks/routes/use-network-page";
import { WorkspacePageShell } from "@/systems/workspace";

export const Route = createFileRoute("/_app/network")({
  component: NetworkPage,
});

function NetworkPage() {
  const page = useNetworkPage();

  return (
    <>
      <WorkspacePageShell
        title="Network"
        icon={<NetworkIcon className="size-4" />}
        count={page.headerCount}
        controls={
          <div className="flex items-center gap-1.5" data-testid="network-tab-pills">
            <PillButton
              active={page.activeTab === "channels"}
              data-testid="network-tab-channels"
              onClick={() => page.setActiveTab("channels")}
            >
              Channels
            </PillButton>
            <PillButton
              active={page.activeTab === "peers"}
              data-testid="network-tab-peers"
              onClick={() => page.setActiveTab("peers")}
            >
              Peers
            </PillButton>
          </div>
        }
        meta={
          page.activeTab === "channels" && page.isNetworkEnabled ? (
            <Button
              className="border-[color:var(--color-accent)] bg-transparent text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)] hover:text-[color:var(--color-accent)]"
              data-testid="open-network-create-dialog"
              onClick={page.handleOpenCreateDialog}
              size="lg"
              type="button"
              variant="outline"
            >
              <Plus className="size-4" />
              Channel
            </Button>
          ) : null
        }
      >
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="border-b border-[color:var(--color-divider)] p-4">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              {page.pageMetrics.map(metric => (
                <MetricStrip
                  key={metric.label}
                  label={metric.label}
                  value={metric.value}
                  detail={metric.detail}
                />
              ))}
            </div>
          </div>

          {page.isNetworkDisabled ? (
            <NetworkEmptyState
              description="Enable the embedded network in AGH config to inspect channels, peers, and runtime message flow."
              icon={<NetworkIcon className="size-5" />}
              testId="network-disabled-state"
              title="Network disabled"
            />
          ) : page.activeTab === "channels" ? (
            <div className="flex min-h-0 flex-1 overflow-hidden">
              <NetworkChannelsListPanel
                channels={page.visibleChannels}
                errorMessage={page.channelsListError?.message ?? null}
                isLoading={page.isChannelsListLoading}
                onSearchChange={page.setChannelSearchQuery}
                onSelectChannel={page.setSelectedChannel}
                searchQuery={page.channelSearchQuery}
                selectedChannel={page.effectiveSelectedChannel}
              />
              {page.isChannelsListLoading ? (
                <NetworkChannelDetailPanel
                  channel={undefined}
                  error={null}
                  isLoading={true}
                  isMessagesLoading={false}
                  messages={[]}
                />
              ) : page.channelsListError ? (
                <NetworkChannelDetailPanel
                  channel={undefined}
                  error={page.channelsListError}
                  isLoading={false}
                  isMessagesLoading={false}
                  messages={[]}
                />
              ) : page.visibleChannels.length === 0 && page.channelSearchQuery !== "" ? (
                <NetworkEmptyState
                  description="Adjust the current search to inspect another materialized network channel."
                  icon={<Hash className="size-5" />}
                  testId="network-channels-empty-state"
                  title="No channels found"
                />
              ) : page.visibleChannels.length === 0 ? (
                <NetworkEmptyState
                  actionLabel="Create Channel"
                  description="Create your first channel to enable agent-to-agent coordination inside the active workspace."
                  icon={<Hash className="size-5" />}
                  onAction={page.handleOpenCreateDialog}
                  testId="network-channels-empty-state"
                  title="No channels yet"
                />
              ) : (
                <NetworkChannelDetailPanel
                  channel={page.channelDetail.channel}
                  error={page.channelDetail.error}
                  isLoading={page.channelDetail.isLoading}
                  isMessagesLoading={page.channelDetail.isMessagesLoading}
                  messages={page.channelDetail.messages}
                />
              )}
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 overflow-hidden">
              <NetworkPeersListPanel
                errorMessage={page.peersListError?.message ?? null}
                isLoading={page.isPeersListLoading}
                onSearchChange={page.setPeerSearchQuery}
                onSelectPeer={page.setSelectedPeerId}
                peers={page.visiblePeers}
                searchQuery={page.peerSearchQuery}
                selectedPeerId={page.effectiveSelectedPeerId}
              />
              {page.isPeersListLoading ? (
                <NetworkPeerDetailPanel error={null} isLoading={true} peer={undefined} />
              ) : page.peersListError ? (
                <NetworkPeerDetailPanel
                  error={page.peersListError}
                  isLoading={false}
                  peer={undefined}
                />
              ) : page.visiblePeers.length === 0 && page.peerSearchQuery !== "" ? (
                <NetworkEmptyState
                  description="Adjust the current search to inspect another visible network peer."
                  icon={<Users className="size-5" />}
                  testId="network-peers-empty-state"
                  title="No peers found"
                />
              ) : page.visiblePeers.length === 0 ? (
                <NetworkEmptyState
                  description="Peers are discovered automatically when agents join the network. Start a channel session to make local peers visible."
                  icon={<Users className="size-5" />}
                  testId="network-peers-empty-state"
                  title="No peers connected"
                />
              ) : (
                <NetworkPeerDetailPanel
                  error={page.peerDetail.error}
                  isLoading={page.peerDetail.isLoading}
                  peer={page.peerDetail.peer}
                />
              )}
            </div>
          )}
        </div>
      </WorkspacePageShell>

      <NetworkCreateChannelDialog
        agents={page.sortedAgents}
        canSubmit={page.canSubmitCreate}
        draft={page.createDraft}
        isSubmitting={page.isCreatePending}
        onChannelNameChange={channelName =>
          page.setCreateDraft(currentDraft => ({
            ...currentDraft,
            channelName,
          }))
        }
        onOpenChange={page.setCreateDialogOpen}
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
