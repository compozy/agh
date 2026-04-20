import { Hash, Network as NetworkIcon, Plus, Users } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { Button, Empty, Metric, PageHeader, Pills, SplitPane } from "@agh/ui";
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

export const Route = createFileRoute("/_app/network")({
  component: NetworkPage,
});

function NetworkPage() {
  const page = useNetworkPage();

  const primaryAction =
    page.activeTab === "channels" && page.isNetworkEnabled ? (
      <Button
        data-testid="open-network-create-dialog"
        onClick={page.handleOpenCreateDialog}
        size="sm"
        type="button"
        variant="outline"
      >
        <Plus className="size-3.5" />
        Channel
      </Button>
    ) : null;

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="network-shell">
        <PageHeader
          title={<span data-testid="network-shell-title">Network</span>}
          icon={() => <NetworkIcon className="size-3.5" data-testid="network-shell-icon" />}
          count={page.headerCount}
          controls={
            <Pills
              aria-label="Network tab"
              data-testid="network-tab-pills"
              value={page.activeTab}
              onChange={page.setActiveTab}
              items={[
                { value: "channels", label: "Channels", testId: "network-tab-channels" },
                { value: "peers", label: "Peers", testId: "network-tab-peers" },
              ]}
            />
          }
          meta={primaryAction}
        />

        <div
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
          data-testid="network-shell-body"
        >
          <div className="border-b border-[color:var(--color-divider)] p-4">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              {page.pageMetrics.map(metric => (
                <Metric
                  key={metric.label}
                  data-testid={`network-metric-${metric.label.toLowerCase().replaceAll(" ", "-")}`}
                  label={metric.label}
                  value={metric.value}
                  subtext={metric.detail}
                />
              ))}
            </div>
          </div>

          {page.isNetworkDisabled ? (
            <div
              className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
              data-testid="network-disabled-state"
            >
              <Empty
                className="max-w-xl"
                icon={NetworkIcon}
                title="Network disabled"
                description="Enable the embedded network in AGH config to inspect channels, peers, and runtime message flow."
              />
            </div>
          ) : page.activeTab === "channels" ? (
            <SplitPane
              data-testid="network-split-pane"
              list={
                <NetworkChannelsListPanel
                  channels={page.visibleChannels}
                  errorMessage={page.channelsListError?.message ?? null}
                  isLoading={page.isChannelsListLoading}
                  onSearchChange={page.setChannelSearchQuery}
                  onSelectChannel={page.setSelectedChannel}
                  searchQuery={page.channelSearchQuery}
                  selectedChannel={page.effectiveSelectedChannel}
                />
              }
              detail={
                page.isChannelsListLoading ? (
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
                    icon={Hash}
                    testId="network-channels-empty-state"
                    title="No channels found"
                  />
                ) : page.visibleChannels.length === 0 ? (
                  <NetworkEmptyState
                    actionLabel="Create Channel"
                    description="Create your first channel to enable agent-to-agent coordination inside the active workspace."
                    icon={Hash}
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
                )
              }
            />
          ) : (
            <SplitPane
              data-testid="network-split-pane"
              list={
                <NetworkPeersListPanel
                  errorMessage={page.peersListError?.message ?? null}
                  isLoading={page.isPeersListLoading}
                  onSearchChange={page.setPeerSearchQuery}
                  onSelectPeer={page.setSelectedPeerId}
                  peers={page.visiblePeers}
                  searchQuery={page.peerSearchQuery}
                  selectedPeerId={page.effectiveSelectedPeerId}
                />
              }
              detail={
                page.isPeersListLoading ? (
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
                    icon={Users}
                    testId="network-peers-empty-state"
                    title="No peers found"
                  />
                ) : page.visiblePeers.length === 0 ? (
                  <NetworkEmptyState
                    description="Peers are discovered automatically when agents join the network. Start a channel session to make local peers visible."
                    icon={Users}
                    testId="network-peers-empty-state"
                    title="No peers connected"
                  />
                ) : (
                  <NetworkPeerDetailPanel
                    error={page.peerDetail.error}
                    isLoading={page.peerDetail.isLoading}
                    peer={page.peerDetail.peer}
                  />
                )
              }
            />
          )}
        </div>
      </div>

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
