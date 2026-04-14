import { Hash, Network as NetworkIcon, Plus, Users } from "lucide-react";
import { startTransition, useDeferredValue, useMemo, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { toast } from "sonner";

import { MetricStrip, PillButton } from "@/components/design-system";
import { Button } from "@/components/ui/button";
import {
  createNetworkChannelDraft,
  getNetworkMetricCards,
  matchesChannelSearch,
  matchesPeerSearch,
  NetworkChannelDetailPanel,
  NetworkChannelsListPanel,
  NetworkCreateChannelDialog,
  NetworkEmptyState,
  NetworkPeerDetailPanel,
  NetworkPeersListPanel,
  sortAgentsForNetwork,
  sortNetworkChannels,
  sortNetworkPeers,
  toggleDraftAgent,
  useCreateNetworkChannel,
  useNetworkChannel,
  useNetworkChannelMessages,
  useNetworkChannels,
  useNetworkPeer,
  useNetworkPeers,
  useNetworkStatus,
} from "@/systems/network";
import type { NetworkTab } from "@/systems/network";
import { useActiveWorkspace, useWorkspace, WorkspacePageShell } from "@/systems/workspace";

export const Route = createFileRoute("/_app/network")({
  component: NetworkPage,
});

function NetworkPage() {
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [activeTab, setActiveTab] = useState<NetworkTab>("channels");
  const [channelSearchQuery, setChannelSearchQuery] = useState("");
  const [peerSearchQuery, setPeerSearchQuery] = useState("");
  const [selectedChannel, setSelectedChannel] = useState<string | null>(null);
  const [selectedPeerId, setSelectedPeerId] = useState<string | null>(null);
  const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState(createNetworkChannelDraft);

  const deferredChannelSearch = useDeferredValue(channelSearchQuery);
  const deferredPeerSearch = useDeferredValue(peerSearchQuery);

  const networkStatusQuery = useNetworkStatus();
  const isNetworkEnabled = networkStatusQuery.data?.enabled === true;
  const isNetworkDisabled = networkStatusQuery.data?.enabled === false;
  const isNetworkStatusLoading = networkStatusQuery.isLoading && !networkStatusQuery.data;
  const networkStatusError = !networkStatusQuery.data ? networkStatusQuery.error : null;

  const networkChannelsQuery = useNetworkChannels({ enabled: isNetworkEnabled });
  const networkPeersQuery = useNetworkPeers(undefined, { enabled: isNetworkEnabled });
  const createChannelMutation = useCreateNetworkChannel();
  const workspaceDetailQuery = useWorkspace(activeWorkspaceId ?? "", {
    enabled: Boolean(activeWorkspaceId),
  });

  const allChannels = networkChannelsQuery.data?.channels ?? [];
  const allPeers = networkPeersQuery.data ?? [];
  const workspaceAgents = workspaceDetailQuery.data?.agents ?? [];
  const sortedAgents = useMemo(() => sortAgentsForNetwork(workspaceAgents), [workspaceAgents]);

  const visibleChannels = useMemo(
    () =>
      sortNetworkChannels(
        allChannels.filter(channel => matchesChannelSearch(channel, deferredChannelSearch))
      ),
    [allChannels, deferredChannelSearch]
  );
  const visiblePeers = useMemo(
    () => sortNetworkPeers(allPeers.filter(peer => matchesPeerSearch(peer, deferredPeerSearch))),
    [allPeers, deferredPeerSearch]
  );

  const effectiveSelectedChannel = useMemo(() => {
    if (selectedChannel && visibleChannels.some(channel => channel.channel === selectedChannel)) {
      return selectedChannel;
    }

    return visibleChannels[0]?.channel ?? null;
  }, [selectedChannel, visibleChannels]);

  const effectiveSelectedPeerId = useMemo(() => {
    if (selectedPeerId && visiblePeers.some(peer => peer.peer_id === selectedPeerId)) {
      return selectedPeerId;
    }

    return visiblePeers[0]?.peer_id ?? null;
  }, [selectedPeerId, visiblePeers]);

  const channelDetailQuery = useNetworkChannel(effectiveSelectedChannel ?? "", {
    enabled: isNetworkEnabled && activeTab === "channels" && Boolean(effectiveSelectedChannel),
  });
  const channelMessagesQuery = useNetworkChannelMessages(effectiveSelectedChannel ?? "", {
    enabled: isNetworkEnabled && activeTab === "channels" && Boolean(effectiveSelectedChannel),
  });
  const peerDetailQuery = useNetworkPeer(effectiveSelectedPeerId ?? "", {
    enabled: isNetworkEnabled && activeTab === "peers" && Boolean(effectiveSelectedPeerId),
  });

  const pageMetrics = getNetworkMetricCards(networkStatusQuery.data, allChannels.length);
  const headerCount = activeTab === "channels" ? allChannels.length : allPeers.length;
  const isChannelsListLoading =
    !isNetworkDisabled &&
    ((isNetworkStatusLoading && !networkStatusQuery.data) ||
      (networkChannelsQuery.isLoading && !networkChannelsQuery.data));
  const channelsListError = !isNetworkDisabled
    ? (networkStatusError ?? (!networkChannelsQuery.data ? networkChannelsQuery.error : null))
    : null;
  const isPeersListLoading =
    !isNetworkDisabled &&
    ((isNetworkStatusLoading && !networkStatusQuery.data) ||
      (networkPeersQuery.isLoading && !networkPeersQuery.data));
  const peersListError = !isNetworkDisabled
    ? (networkStatusError ?? (!networkPeersQuery.data ? networkPeersQuery.error : null))
    : null;

  const handleOpenCreateDialog = () => {
    setCreateDraft(createNetworkChannelDraft());
    setCreateDialogOpen(true);
  };

  const handleCreateChannel = async () => {
    if (!activeWorkspaceId) {
      toast.error("Select an active workspace before creating a channel.");
      return;
    }

    const channelName = createDraft.channelName.trim();
    if (!channelName) {
      toast.error("Provide a channel name before creating the channel.");
      return;
    }

    if (createDraft.selectedAgentNames.length === 0) {
      toast.error("Select at least one local agent before creating the channel.");
      return;
    }

    try {
      const result = await createChannelMutation.mutateAsync({
        agent_names: createDraft.selectedAgentNames,
        channel: channelName,
        workspace_id: activeWorkspaceId,
      });

      startTransition(() => {
        setActiveTab("channels");
        setChannelSearchQuery("");
        setSelectedChannel(result.channel.channel);
      });
      setCreateDialogOpen(false);
      setCreateDraft(createNetworkChannelDraft());
      toast.success(`Created channel ${result.channel.channel}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create network channel");
    }
  };

  const canSubmitCreate =
    isNetworkEnabled &&
    Boolean(activeWorkspaceId) &&
    createDraft.channelName.trim() !== "" &&
    createDraft.selectedAgentNames.length > 0;

  return (
    <>
      <WorkspacePageShell
        title="Network"
        icon={<NetworkIcon className="size-4" />}
        count={headerCount}
        controls={
          <div className="flex items-center gap-1.5" data-testid="network-tab-pills">
            <PillButton
              active={activeTab === "channels"}
              data-testid="network-tab-channels"
              onClick={() => setActiveTab("channels")}
            >
              Channels
            </PillButton>
            <PillButton
              active={activeTab === "peers"}
              data-testid="network-tab-peers"
              onClick={() => setActiveTab("peers")}
            >
              Peers
            </PillButton>
          </div>
        }
        meta={
          activeTab === "channels" && isNetworkEnabled ? (
            <Button
              className="border-[color:var(--color-accent)] bg-transparent text-[color:var(--color-accent)] hover:bg-[color:var(--color-accent-tint)] hover:text-[color:var(--color-accent)]"
              data-testid="open-network-create-dialog"
              onClick={handleOpenCreateDialog}
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
              {pageMetrics.map(metric => (
                <MetricStrip
                  detail={metric.detail}
                  key={metric.label}
                  label={metric.label}
                  value={metric.value}
                />
              ))}
            </div>
          </div>

          {isNetworkDisabled ? (
            <NetworkEmptyState
              description="Enable the embedded network in AGH config to inspect channels, peers, and runtime message flow."
              icon={<NetworkIcon className="size-5" />}
              testId="network-disabled-state"
              title="Network disabled"
            />
          ) : activeTab === "channels" ? (
            <div className="flex min-h-0 flex-1 overflow-hidden">
              <NetworkChannelsListPanel
                channels={visibleChannels}
                errorMessage={channelsListError?.message ?? null}
                isLoading={isChannelsListLoading}
                onSearchChange={setChannelSearchQuery}
                onSelectChannel={setSelectedChannel}
                searchQuery={channelSearchQuery}
                selectedChannel={effectiveSelectedChannel}
              />
              {isChannelsListLoading ? (
                <NetworkChannelDetailPanel
                  channel={undefined}
                  error={null}
                  isLoading={true}
                  isMessagesLoading={false}
                  messages={[]}
                />
              ) : channelsListError ? (
                <NetworkChannelDetailPanel
                  channel={undefined}
                  error={channelsListError}
                  isLoading={false}
                  isMessagesLoading={false}
                  messages={[]}
                />
              ) : allChannels.length === 0 ? (
                <NetworkEmptyState
                  actionLabel="Create Channel"
                  description="Create your first channel to enable agent-to-agent coordination inside the active workspace."
                  icon={<Hash className="size-5" />}
                  onAction={handleOpenCreateDialog}
                  testId="network-channels-empty-state"
                  title="No channels yet"
                />
              ) : visibleChannels.length === 0 ? (
                <NetworkEmptyState
                  description="Adjust the current search to inspect another materialized network channel."
                  icon={<Hash className="size-5" />}
                  testId="network-channels-empty-state"
                  title="No channels found"
                />
              ) : (
                <NetworkChannelDetailPanel
                  channel={channelDetailQuery.data}
                  error={channelDetailQuery.error ?? channelMessagesQuery.error ?? null}
                  isLoading={channelDetailQuery.isLoading && !channelDetailQuery.data}
                  isMessagesLoading={channelMessagesQuery.isLoading}
                  messages={channelMessagesQuery.data ?? []}
                />
              )}
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 overflow-hidden">
              <NetworkPeersListPanel
                errorMessage={peersListError?.message ?? null}
                isLoading={isPeersListLoading}
                onSearchChange={setPeerSearchQuery}
                onSelectPeer={setSelectedPeerId}
                peers={visiblePeers}
                searchQuery={peerSearchQuery}
                selectedPeerId={effectiveSelectedPeerId}
              />
              {isPeersListLoading ? (
                <NetworkPeerDetailPanel error={null} isLoading={true} peer={undefined} />
              ) : peersListError ? (
                <NetworkPeerDetailPanel error={peersListError} isLoading={false} peer={undefined} />
              ) : allPeers.length === 0 ? (
                <NetworkEmptyState
                  description="Peers are discovered automatically when agents join the network. Start a channel session to make local peers visible."
                  icon={<Users className="size-5" />}
                  testId="network-peers-empty-state"
                  title="No peers connected"
                />
              ) : visiblePeers.length === 0 ? (
                <NetworkEmptyState
                  description="Adjust the current search to inspect another visible network peer."
                  icon={<Users className="size-5" />}
                  testId="network-peers-empty-state"
                  title="No peers found"
                />
              ) : (
                <NetworkPeerDetailPanel
                  error={peerDetailQuery.error ?? null}
                  isLoading={peerDetailQuery.isLoading && !peerDetailQuery.data}
                  peer={peerDetailQuery.data}
                />
              )}
            </div>
          )}
        </div>
      </WorkspacePageShell>

      <NetworkCreateChannelDialog
        agents={sortedAgents}
        canSubmit={canSubmitCreate}
        draft={createDraft}
        isSubmitting={createChannelMutation.isPending}
        onChannelNameChange={channelName =>
          setCreateDraft(currentDraft => ({
            ...currentDraft,
            channelName,
          }))
        }
        onOpenChange={setCreateDialogOpen}
        onSubmit={handleCreateChannel}
        onToggleAgent={agentName =>
          setCreateDraft(currentDraft => toggleDraftAgent(currentDraft, agentName))
        }
        open={isCreateDialogOpen}
        workspaceName={activeWorkspace?.name ?? null}
      />
    </>
  );
}
