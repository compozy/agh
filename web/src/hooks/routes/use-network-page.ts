import {
  startTransition,
  useCallback,
  useDeferredValue,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import {
  buildPeerCapabilityViews,
  createNetworkChannelDraft,
  filterNetworkMessagesByKind,
  formatChannelMemberCount,
  formatNetworkDateTime,
  formatNetworkRelativeTime,
  getNetworkRoomKey,
  getPeerDisplayName,
  getPeerPresenceTone,
  matchesChannelSearch,
  matchesPeerSearch,
  sortAgentsForNetwork,
  sortNetworkChannels,
  sortNetworkPeers,
  toNetworkKindFilter,
  useCreateNetworkChannel,
  useNetworkChannel,
  useNetworkChannelMessages,
  useNetworkChannels,
  useNetworkPeer,
  useNetworkPeerMessages,
  useNetworkPeers,
  useNetworkStatus,
  useSendNetworkMessage,
} from "@/systems/network";
import type {
  NetworkActiveRoom,
  NetworkChannel,
  NetworkChannelSummary,
  NetworkDetailsTab,
  NetworkKindFilter,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkRoomField,
  NetworkRoomKindMetric,
  NetworkRoomListItem,
  NetworkRoomMember,
  NetworkSendRequest,
  NetworkTimelineMessage,
} from "@/systems/network";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";

const STARRED_CHANNELS_STORAGE_KEY = "network:starred-channels";
const ROOM_READ_AT_STORAGE_KEY = "network:room-read-at";
const TIMELINE_LIMIT = 120;

interface NetworkRouteSearch {
  channel?: string;
  details?: "closed";
  kind?: NetworkKindFilter;
  peer?: string;
}

function normalizeSearchValue(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function readStringArray(storageKey: string): string[] {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(storageKey) ?? "[]");
    return Array.isArray(parsed)
      ? parsed.filter(item => typeof item === "string" && item.trim() !== "")
      : [];
  } catch {
    return [];
  }
}

function writeStringArray(storageKey: string, values: string[]) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(values));
  } catch {
    // Persistence is best effort only.
  }
}

function readStringRecord(storageKey: string): Record<string, string> {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(storageKey) ?? "{}");
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return {};
    }

    return Object.entries(parsed).reduce<Record<string, string>>((acc, [key, value]) => {
      if (typeof value === "string") {
        acc[key] = value;
      }
      return acc;
    }, {});
  } catch {
    return {};
  }
}

function writeStringRecord(storageKey: string, values: Record<string, string>) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(values));
  } catch {
    // Persistence is best effort only.
  }
}

function safeDate(value?: string | null): number {
  if (!value) {
    return 0;
  }

  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

function makeUnreadCount(
  roomKey: string,
  lastActivityAt: string | null | undefined,
  readMarkers: Record<string, string>,
  selectedRoomKey: string | null
) {
  if (!lastActivityAt || selectedRoomKey === roomKey) {
    return 0;
  }

  return safeDate(lastActivityAt) > safeDate(readMarkers[roomKey]) ? 1 : 0;
}

function summarizeChannelMeta(channel: NetworkChannelSummary) {
  if (channel.last_activity_at) {
    return formatNetworkRelativeTime(channel.last_activity_at);
  }

  return channel.message_count && channel.message_count > 0 ? "materialized" : "idle";
}

function summarizePeerMeta(peer: NetworkPeerSummary) {
  if (peer.last_seen) {
    return formatNetworkRelativeTime(peer.last_seen);
  }

  return peer.local ? "local" : "offline";
}

function summarizeChannelPreview(channel: NetworkChannelSummary) {
  return (
    channel.last_message_preview?.trim() || channel.purpose?.trim() || "No timeline events yet"
  );
}

function summarizePeerPreview(peer: NetworkPeerSummary) {
  return `#${peer.channel}`;
}

function makeChannelRoomItem(
  channel: NetworkChannelSummary,
  isStarred: boolean,
  readMarkers: Record<string, string>,
  selectedRoomKey: string | null
): NetworkRoomListItem {
  const key = getNetworkRoomKey("channel", channel.channel);

  return {
    id: channel.channel,
    isStarred,
    key,
    lastActivityAt: channel.last_activity_at ?? null,
    meta: summarizeChannelMeta(channel),
    preview: summarizeChannelPreview(channel),
    roomType: "channel",
    subtitle: `${formatChannelMemberCount(channel)} · ${channel.message_count ?? 0} msgs`,
    title: channel.channel,
    tone: channel.message_count && channel.message_count > 0 ? "accent" : "neutral",
    unreadCount: makeUnreadCount(key, channel.last_activity_at, readMarkers, selectedRoomKey),
  };
}

function makePeerRoomItem(
  peer: NetworkPeerSummary,
  readMarkers: Record<string, string>,
  selectedRoomKey: string | null
): NetworkRoomListItem {
  const key = getNetworkRoomKey("peer", peer.peer_id);

  return {
    id: peer.peer_id,
    isStarred: false,
    key,
    lastActivityAt: peer.last_seen ?? null,
    meta: summarizePeerMeta(peer),
    preview: summarizePeerPreview(peer),
    roomType: "peer",
    subtitle: peer.local ? "Local peer" : "Remote peer",
    title: getPeerDisplayName(peer),
    tone: getPeerPresenceTone(peer),
    unreadCount: makeUnreadCount(key, peer.last_seen, readMarkers, selectedRoomKey),
  };
}

function makeMemberFromPeer(peer: NetworkPeerSummary): NetworkRoomMember {
  return {
    id: peer.peer_id,
    lastSeen: peer.last_seen ?? null,
    local: peer.local,
    sessionId: peer.session_id ?? null,
    subtitle: peer.local ? `Local · #${peer.channel}` : `Remote · #${peer.channel}`,
    title: getPeerDisplayName(peer),
    tone: getPeerPresenceTone(peer),
  };
}

function pickChannelSenderSessionId(channel: NetworkChannel | undefined) {
  return channel?.sessions?.[0]?.id ?? null;
}

function pickPeerSenderSessionId(
  peer: NetworkPeerDetail | undefined,
  allPeers: NetworkPeerSummary[]
) {
  if (!peer) {
    return null;
  }

  if (peer.local && peer.session_id) {
    return peer.session_id;
  }

  return (
    allPeers.find(candidate => candidate.channel === peer.channel && candidate.local)?.session_id ??
    null
  );
}

function summarizeKindCounts(messages: NetworkTimelineMessage[]): NetworkRoomKindMetric[] {
  const counts = new Map<NetworkRoomKindMetric["kind"], number>();

  for (const message of messages) {
    const kind = toNetworkKindFilter(message.kind);
    if (!kind) {
      continue;
    }
    counts.set(kind, (counts.get(kind) ?? 0) + 1);
  }

  return [...counts.entries()]
    .map(([kind, count]) => ({ kind, count }))
    .sort((left, right) => right.count - left.count || left.kind.localeCompare(right.kind));
}

function summarizeChannelWireFields(
  room: NetworkChannelSummary,
  detail: NetworkChannel | undefined,
  lastActivityAt: string | null
): NetworkRoomField[] {
  return [
    {
      label: "Workspace",
      mono: true,
      value: detail?.workspace_id ?? room.workspace_id ?? "unassigned",
    },
    {
      label: "Created By",
      mono: true,
      value: detail?.created_by ?? room.created_by ?? "system",
    },
    {
      label: "Sessions",
      mono: true,
      value: String(detail?.session_count ?? room.session_count ?? 0),
    },
    {
      label: "Peers",
      mono: true,
      value: `${detail?.local_peer_count ?? room.local_peer_count ?? 0} local / ${
        detail?.remote_peer_count ?? room.remote_peer_count ?? 0
      } remote`,
    },
    {
      label: "Last Activity",
      value: lastActivityAt ? formatNetworkDateTime(lastActivityAt) : "Unavailable",
    },
  ];
}

function summarizePeerWireFields(peer: NetworkPeerDetail | undefined): NetworkRoomField[] {
  if (!peer) {
    return [];
  }

  return [
    { label: "Sent", mono: true, value: String(peer.metrics.sent ?? 0) },
    { label: "Received", mono: true, value: String(peer.metrics.received ?? 0) },
    { label: "Delivered", mono: true, tone: "success", value: String(peer.metrics.delivered ?? 0) },
    { label: "Rejected", mono: true, tone: "danger", value: String(peer.metrics.rejected ?? 0) },
    { label: "Channel", mono: true, value: peer.channel ?? "unknown" },
  ];
}

function makeChannelActiveRoom({
  detail,
  filteredMessages,
  isStarred,
  rawMessages,
  room,
}: {
  detail: NetworkChannel | undefined;
  filteredMessages: NetworkTimelineMessage[];
  isStarred: boolean;
  rawMessages: NetworkTimelineMessage[];
  room: NetworkChannelSummary;
}): NetworkActiveRoom {
  const lastActivityAt =
    detail?.last_activity_at ??
    room.last_activity_at ??
    rawMessages[rawMessages.length - 1]?.timestamp ??
    null;
  const purpose = detail?.purpose?.trim() || room.purpose?.trim() || null;
  const members = sortNetworkPeers(detail?.peers ?? []).map(makeMemberFromPeer);

  return {
    aboutFields: [
      {
        label: "Purpose",
        value: purpose ?? "No purpose has been recorded for this room yet.",
      },
      {
        label: "Created",
        value: detail?.created_at ? formatNetworkDateTime(detail.created_at) : "Unavailable",
      },
    ],
    canCompose: pickChannelSenderSessionId(detail) !== null,
    canStar: true,
    capabilities: [],
    channel: room.channel,
    composeHint:
      pickChannelSenderSessionId(detail) !== null
        ? "Broadcasts send through the first local session in this channel."
        : "This channel has no local session available for composing yet.",
    composePlaceholder: `Send a broadcast to #${room.channel}`,
    description: purpose ?? `Coordination room for #${room.channel}.`,
    id: room.channel,
    introBody:
      purpose ??
      "Materialize this room with a short operator note so other agents know how to use it.",
    introTitle: `Welcome to #${room.channel}`,
    isStarred,
    key: getNetworkRoomKey("channel", room.channel),
    kindCounts:
      detail?.kind_counts
        ?.map(metric =>
          metric.kind ? { count: metric.count, kind: toNetworkKindFilter(metric.kind) } : null
        )
        .filter(
          (metric): metric is NetworkRoomKindMetric => metric !== null && metric.kind !== null
        )
        .map(metric => ({ kind: metric.kind, count: metric.count })) ??
      summarizeKindCounts(rawMessages),
    lastActivityAt,
    memberCount: detail?.peer_count ?? room.peer_count ?? members.length,
    members,
    messageCount: detail?.message_count ?? room.message_count ?? rawMessages.length,
    messages: filteredMessages,
    preview: summarizeChannelPreview(room),
    purpose,
    roomType: "channel",
    subtitle: purpose ?? `${formatChannelMemberCount(room)} active`,
    title: room.channel,
    wireFields: summarizeChannelWireFields(room, detail, lastActivityAt),
  };
}

function makePeerActiveRoom({
  channelPeers,
  detail,
  filteredMessages,
  rawMessages,
  room,
}: {
  channelPeers: NetworkPeerSummary[];
  detail: NetworkPeerDetail | undefined;
  filteredMessages: NetworkTimelineMessage[];
  rawMessages: NetworkTimelineMessage[];
  room: NetworkPeerSummary;
}): NetworkActiveRoom {
  const lastActivityAt =
    rawMessages[rawMessages.length - 1]?.timestamp ?? detail?.last_seen ?? room.last_seen ?? null;
  const capabilities = buildPeerCapabilityViews(
    detail?.peer_card.capabilities,
    detail?.capability_catalog
  );

  return {
    aboutFields: [
      { label: "Peer ID", mono: true, value: room.peer_id },
      { label: "Channel", mono: true, value: room.channel },
      {
        label: "Last Seen",
        value: detail?.last_seen ? formatNetworkDateTime(detail.last_seen) : "Unavailable",
      },
    ],
    canCompose: pickPeerSenderSessionId(detail, channelPeers) !== null,
    canStar: false,
    capabilities,
    channel: room.channel,
    composeHint:
      pickPeerSenderSessionId(detail, channelPeers) !== null
        ? `Direct messages send through the first local session in #${room.channel}.`
        : "No local peer is available in this channel to address this room yet.",
    composePlaceholder: `Send a direct message to ${getPeerDisplayName(room)}`,
    description: `Directed timeline for ${getPeerDisplayName(room)} on #${room.channel}.`,
    id: room.peer_id,
    introBody: room.local
      ? "This is a local peer lane. Use it for targeted coordination and handoff acknowledgements."
      : "This peer is visible on the network. Direct messages stay scoped to this lane.",
    introTitle: `Direct thread with ${getPeerDisplayName(room)}`,
    isStarred: false,
    key: getNetworkRoomKey("peer", room.peer_id),
    kindCounts: summarizeKindCounts(rawMessages),
    lastActivityAt,
    memberCount: channelPeers.length,
    members: sortNetworkPeers(channelPeers).map(makeMemberFromPeer),
    messageCount: rawMessages.length,
    messages: filteredMessages,
    preview: summarizePeerPreview(room),
    purpose: null,
    roomType: "peer",
    subtitle: room.local ? "Local peer" : "Remote peer",
    title: getPeerDisplayName(room),
    wireFields: summarizePeerWireFields(detail),
  };
}

function validateNetworkSearch(search: Record<string, unknown>): NetworkRouteSearch {
  const kindValue = normalizeSearchValue(search.kind);
  const normalizedKind =
    kindValue === "all" || (kindValue && toNetworkKindFilter(kindValue))
      ? (kindValue as NetworkKindFilter)
      : undefined;

  return {
    channel: normalizeSearchValue(search.channel),
    details: search.details === "closed" ? "closed" : undefined,
    kind: normalizedKind === "all" ? undefined : normalizedKind,
    peer: normalizeSearchValue(search.peer),
  };
}

function useNetworkPage(search: NetworkRouteSearch = {}) {
  const navigate = useNavigate({ from: "/network" });
  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const [sidebarQuery, setSidebarQuery] = useState("");
  const [detailsTab, setDetailsTab] = useState<NetworkDetailsTab>("about");
  const [composeDraft, setComposeDraft] = useState("");
  const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState(createNetworkChannelDraft);
  const [starredChannels, setStarredChannels] = useState(() =>
    readStringArray(STARRED_CHANNELS_STORAGE_KEY)
  );
  const [readMarkers, setReadMarkers] = useState(() => readStringRecord(ROOM_READ_AT_STORAGE_KEY));

  const deferredSidebarQuery = useDeferredValue(sidebarQuery);

  const updateSearch = useCallback(
    (updater: (current: NetworkRouteSearch) => NetworkRouteSearch) => {
      void navigate({
        search: current =>
          updater(validateNetworkSearch((current ?? {}) as Record<string, unknown>)),
        to: "/network",
      });
    },
    [navigate]
  );

  const networkStatusQuery = useNetworkStatus();
  const networkStatus = networkStatusQuery.data;
  const isNetworkEnabled = networkStatus?.enabled === true;
  const isNetworkDisabled = networkStatus?.enabled === false;

  const networkChannelsQuery = useNetworkChannels({ enabled: isNetworkEnabled });
  const networkPeersQuery = useNetworkPeers(undefined, { enabled: isNetworkEnabled });
  const createChannelMutation = useCreateNetworkChannel();
  const sendMessageMutation = useSendNetworkMessage();
  const workspaceDetailQuery = useWorkspace(activeWorkspaceId ?? "", {
    enabled: Boolean(activeWorkspaceId),
  });

  const allChannels = networkChannelsQuery.data?.channels ?? [];
  const allPeers = networkPeersQuery.data ?? [];
  const workspaceAgents = workspaceDetailQuery.data?.agents ?? [];
  const sortedAgents = useMemo(() => sortAgentsForNetwork(workspaceAgents), [workspaceAgents]);

  const filteredChannels = useMemo(
    () =>
      sortNetworkChannels(
        allChannels.filter(channel => matchesChannelSearch(channel, deferredSidebarQuery))
      ),
    [allChannels, deferredSidebarQuery]
  );
  const filteredPeers = useMemo(
    () => sortNetworkPeers(allPeers.filter(peer => matchesPeerSearch(peer, deferredSidebarQuery))),
    [allPeers, deferredSidebarQuery]
  );

  const selectedRoomKey =
    search.peer != null
      ? getNetworkRoomKey("peer", search.peer)
      : search.channel != null
        ? getNetworkRoomKey("channel", search.channel)
        : null;
  const activeKind = search.kind ?? "all";
  const isDetailsOpen = search.details !== "closed";

  const starredChannelRooms = useMemo(
    () =>
      filteredChannels
        .filter(channel => starredChannels.includes(channel.channel))
        .map(channel => makeChannelRoomItem(channel, true, readMarkers, selectedRoomKey)),
    [filteredChannels, readMarkers, selectedRoomKey, starredChannels]
  );
  const channelRooms = useMemo(
    () =>
      filteredChannels
        .filter(channel => !starredChannels.includes(channel.channel))
        .map(channel => makeChannelRoomItem(channel, false, readMarkers, selectedRoomKey)),
    [filteredChannels, readMarkers, selectedRoomKey, starredChannels]
  );
  const directRooms = useMemo(
    () => filteredPeers.map(peer => makePeerRoomItem(peer, readMarkers, selectedRoomKey)),
    [filteredPeers, readMarkers, selectedRoomKey]
  );

  const activeRoomItem = useMemo(() => {
    if (search.peer) {
      return directRooms.find(room => room.id === search.peer) ?? null;
    }
    if (search.channel) {
      return (
        [...starredChannelRooms, ...channelRooms].find(room => room.id === search.channel) ?? null
      );
    }

    return starredChannelRooms[0] ?? channelRooms[0] ?? directRooms[0] ?? null;
  }, [channelRooms, directRooms, search.channel, search.peer, starredChannelRooms]);

  useEffect(() => {
    if (!activeRoomItem) {
      return;
    }

    const hasSelectedTarget =
      (activeRoomItem.roomType === "channel" && search.channel === activeRoomItem.id) ||
      (activeRoomItem.roomType === "peer" && search.peer === activeRoomItem.id);

    if (hasSelectedTarget) {
      return;
    }

    updateSearch(current => ({
      ...current,
      channel: activeRoomItem.roomType === "channel" ? activeRoomItem.id : undefined,
      peer: activeRoomItem.roomType === "peer" ? activeRoomItem.id : undefined,
    }));
  }, [activeRoomItem, search.channel, search.peer, updateSearch]);

  const activeChannelSummary = useMemo(
    () =>
      activeRoomItem?.roomType === "channel"
        ? allChannels.find(channel => channel.channel === activeRoomItem.id)
        : undefined,
    [activeRoomItem, allChannels]
  );
  const activePeerSummary = useMemo(
    () =>
      activeRoomItem?.roomType === "peer"
        ? allPeers.find(peer => peer.peer_id === activeRoomItem.id)
        : undefined,
    [activeRoomItem, allPeers]
  );

  const channelDetailQuery = useNetworkChannel(activeChannelSummary?.channel ?? "", {
    enabled:
      isNetworkEnabled && activeRoomItem?.roomType === "channel" && Boolean(activeChannelSummary),
  });
  const channelMessagesQuery = useNetworkChannelMessages(activeChannelSummary?.channel ?? "", {
    enabled:
      isNetworkEnabled && activeRoomItem?.roomType === "channel" && Boolean(activeChannelSummary),
    query: { limit: TIMELINE_LIMIT },
  });
  const peerDetailQuery = useNetworkPeer(activePeerSummary?.peer_id ?? "", {
    enabled: isNetworkEnabled && activeRoomItem?.roomType === "peer" && Boolean(activePeerSummary),
  });
  const peerMessagesQuery = useNetworkPeerMessages(activePeerSummary?.peer_id ?? "", {
    enabled: isNetworkEnabled && activeRoomItem?.roomType === "peer" && Boolean(activePeerSummary),
    query: { limit: TIMELINE_LIMIT },
  });

  const rawMessages = useMemo(() => {
    if (activeRoomItem?.roomType === "peer") {
      return peerMessagesQuery.data ?? [];
    }

    return channelMessagesQuery.data ?? [];
  }, [activeRoomItem?.roomType, channelMessagesQuery.data, peerMessagesQuery.data]);

  const filteredMessages = useMemo(
    () => filterNetworkMessagesByKind(rawMessages, activeKind),
    [activeKind, rawMessages]
  );

  const activeRoom = useMemo(() => {
    if (activeRoomItem?.roomType === "channel" && activeChannelSummary) {
      return makeChannelActiveRoom({
        detail: channelDetailQuery.data,
        filteredMessages,
        isStarred: starredChannels.includes(activeChannelSummary.channel),
        rawMessages,
        room: activeChannelSummary,
      });
    }

    if (activeRoomItem?.roomType === "peer" && activePeerSummary) {
      return makePeerActiveRoom({
        channelPeers: allPeers.filter(peer => peer.channel === activePeerSummary.channel),
        detail: peerDetailQuery.data,
        filteredMessages,
        rawMessages,
        room: activePeerSummary,
      });
    }

    return null;
  }, [
    activeChannelSummary,
    activePeerSummary,
    activeRoomItem?.roomType,
    allPeers,
    channelDetailQuery.data,
    filteredMessages,
    peerDetailQuery.data,
    rawMessages,
    starredChannels,
  ]);

  useEffect(() => {
    if (!activeRoom?.key || !activeRoom.lastActivityAt) {
      return;
    }

    const roomKey = activeRoom.key;
    const lastActivityAt = activeRoom.lastActivityAt;

    setReadMarkers(current => {
      if (current[roomKey] === lastActivityAt) {
        return current;
      }

      const next = { ...current, [roomKey]: lastActivityAt };
      writeStringRecord(ROOM_READ_AT_STORAGE_KEY, next);
      return next;
    });
  }, [activeRoom?.key, activeRoom?.lastActivityAt]);

  useEffect(() => {
    setComposeDraft("");
    setDetailsTab("about");
  }, [activeRoom?.key]);

  const handleSelectRoom = useCallback(
    (room: NetworkRoomListItem) => {
      updateSearch(current => ({
        ...current,
        channel: room.roomType === "channel" ? room.id : undefined,
        peer: room.roomType === "peer" ? room.id : undefined,
      }));
    },
    [updateSearch]
  );

  const handleToggleStarChannel = useCallback((channel: string) => {
    setStarredChannels(current => {
      const next = current.includes(channel)
        ? current.filter(value => value !== channel)
        : [channel, ...current];
      writeStringArray(STARRED_CHANNELS_STORAGE_KEY, next);
      return next;
    });
  }, []);

  const handleToggleDetails = useCallback(() => {
    updateSearch(current => ({
      ...current,
      details: current.details === "closed" ? undefined : "closed",
    }));
  }, [updateSearch]);

  const handleSetKind = useCallback(
    (kind: NetworkKindFilter) => {
      updateSearch(current => ({
        ...current,
        kind: kind === "all" ? undefined : kind,
      }));
    },
    [updateSearch]
  );

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
    const purpose = createDraft.purpose.trim();
    if (!channelName) {
      toast.error("Provide a channel name before creating the channel.");
      return;
    }
    if (!purpose) {
      toast.error("Provide a room purpose before creating the channel.");
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
        purpose,
        workspace_id: activeWorkspaceId,
      });

      startTransition(() => {
        setSidebarQuery("");
        updateSearch(current => ({
          ...current,
          channel: result.channel.channel,
          kind: undefined,
          peer: undefined,
        }));
      });
      setCreateDialogOpen(false);
      setCreateDraft(createNetworkChannelDraft());
      toast.success(`Created #${result.channel.channel}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create network channel");
    }
  };

  const composeRequest = useMemo<NetworkSendRequest | null>(() => {
    if (!activeRoom) {
      return null;
    }

    if (activeRoom.roomType === "channel") {
      const sessionId = pickChannelSenderSessionId(channelDetailQuery.data);
      if (!sessionId) {
        return null;
      }

      return {
        body: { text: composeDraft.trim() },
        channel: activeRoom.channel,
        kind: "say",
        session_id: sessionId,
      };
    }

    const sessionId = pickPeerSenderSessionId(peerDetailQuery.data, allPeers);
    if (!sessionId) {
      return null;
    }

    return {
      body: { text: composeDraft.trim() },
      channel: activeRoom.channel,
      kind: "direct",
      session_id: sessionId,
      to: activeRoom.id,
    };
  }, [activeRoom, allPeers, channelDetailQuery.data, composeDraft, peerDetailQuery.data]);

  const handleComposeSubmit = async () => {
    const text = composeDraft.trim();
    if (!text) {
      return;
    }
    if (!composeRequest) {
      toast.error("This room has no local session available for sending.");
      return;
    }

    try {
      await sendMessageMutation.mutateAsync(composeRequest);
      setComposeDraft("");
      toast.success(
        activeRoom?.roomType === "channel" ? "Broadcast sent." : "Direct message sent."
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to send network message");
    }
  };

  const roomError =
    activeRoomItem?.roomType === "peer"
      ? (peerDetailQuery.error ?? peerMessagesQuery.error ?? null)
      : (channelDetailQuery.error ?? channelMessagesQuery.error ?? null);
  const isRoomLoading =
    activeRoomItem?.roomType === "peer"
      ? peerDetailQuery.isLoading && !peerDetailQuery.data
      : channelDetailQuery.isLoading && !channelDetailQuery.data;
  const isTimelineLoading =
    activeRoomItem?.roomType === "peer"
      ? peerMessagesQuery.isLoading
      : channelMessagesQuery.isLoading;

  return {
    activeKind,
    activeRoom,
    channelRooms,
    composeDraft,
    createDraft,
    detailsTab,
    directRooms,
    handleComposeSubmit,
    handleCreateChannel,
    handleOpenCreateDialog,
    handleSelectRoom,
    handleSetKind,
    handleToggleDetails,
    handleToggleStarChannel,
    isComposePending: sendMessageMutation.isPending,
    isCreateDialogOpen,
    isCreatePending: createChannelMutation.isPending,
    isDetailsOpen,
    isNetworkDisabled,
    isNetworkEnabled,
    isPageLoading: networkStatusQuery.isLoading && !networkStatus,
    isRoomLoading,
    isTimelineLoading,
    networkStatus,
    pageError: networkStatus ? null : networkStatusQuery.error,
    roomError,
    selectedRoomKey: activeRoomItem?.key ?? null,
    setComposeDraft,
    setCreateDialogOpen,
    setCreateDraft,
    setDetailsTab,
    setSidebarQuery,
    sidebarQuery,
    sortedAgents,
    starredChannelRooms,
    workspaceName: activeWorkspace?.name ?? null,
  };
}

export { useNetworkPage, validateNetworkSearch };
export type { NetworkRouteSearch };
