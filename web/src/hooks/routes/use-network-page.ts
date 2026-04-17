import { startTransition, useDeferredValue, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

import {
  createNetworkChannelDraft,
  getNetworkMetricCards,
  matchesChannelSearch,
  matchesPeerSearch,
  sortAgentsForNetwork,
  sortNetworkChannels,
  sortNetworkPeers,
  useCreateNetworkChannel,
  useNetworkChannel,
  useNetworkChannelMessages,
  useNetworkChannels,
  useNetworkPeer,
  useNetworkPeers,
  useNetworkStatus,
} from "@/systems/network";
import type { NetworkTab } from "@/systems/network";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";

function useNetworkPage() {
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
    (isNetworkStatusLoading || (networkChannelsQuery.isLoading && !networkChannelsQuery.data));
  const channelsListError = !isNetworkDisabled
    ? (networkStatusError ?? (!networkChannelsQuery.data ? networkChannelsQuery.error : null))
    : null;
  const isPeersListLoading =
    !isNetworkDisabled &&
    (isNetworkStatusLoading || (networkPeersQuery.isLoading && !networkPeersQuery.data));
  const peersListError = !isNetworkDisabled
    ? (networkStatusError ?? (!networkPeersQuery.data ? networkPeersQuery.error : null))
    : null;
  const channelDetail = {
    channel: channelDetailQuery.data,
    error: channelDetailQuery.error ?? channelMessagesQuery.error ?? null,
    isLoading: channelDetailQuery.isLoading && !channelDetailQuery.data,
    isMessagesLoading: channelMessagesQuery.isLoading,
    messages: channelMessagesQuery.data ?? [],
  };
  const peerDetail = {
    error: peerDetailQuery.error ?? null,
    isLoading: peerDetailQuery.isLoading && !peerDetailQuery.data,
    peer: peerDetailQuery.data,
  };
  const refetchNetworkStatus = networkStatusQuery.refetch;
  const refetchNetworkChannels = networkChannelsQuery.refetch;
  const refetchChannelDetail = channelDetailQuery.refetch;
  const refetchChannelMessages = channelMessagesQuery.refetch;

  useEffect(() => {
    if (!isNetworkEnabled || activeTab !== "channels" || !effectiveSelectedChannel) {
      return;
    }

    void refetchNetworkStatus();
    void refetchNetworkChannels();
    void refetchChannelDetail();
    void refetchChannelMessages();
  }, [
    activeTab,
    effectiveSelectedChannel,
    isNetworkEnabled,
    refetchChannelDetail,
    refetchChannelMessages,
    refetchNetworkChannels,
    refetchNetworkStatus,
  ]);

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

  return {
    activeTab,
    canSubmitCreate,
    channelsListError,
    channelDetail,
    channelSearchQuery,
    createDraft,
    effectiveSelectedChannel,
    effectiveSelectedPeerId,
    handleCreateChannel,
    handleOpenCreateDialog,
    headerCount,
    isChannelsListLoading,
    isCreateDialogOpen,
    isCreatePending: createChannelMutation.isPending,
    isNetworkDisabled,
    isNetworkEnabled,
    isPeersListLoading,
    pageMetrics,
    peerDetail,
    peerSearchQuery,
    peersListError,
    setActiveTab,
    setChannelSearchQuery,
    setCreateDialogOpen,
    setCreateDraft,
    setPeerSearchQuery,
    setSelectedChannel,
    setSelectedPeerId,
    sortedAgents,
    visibleChannels,
    visiblePeers,
    workspaceName: activeWorkspace?.name ?? null,
  };
}

export { useNetworkPage };
