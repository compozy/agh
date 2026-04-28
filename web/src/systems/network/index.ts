// Types
export type {
  NetworkActiveRoom,
  CreateNetworkChannelRequest,
  CreateNetworkChannelResponse,
  NetworkCapability,
  NetworkCapabilityBrief,
  NetworkCapabilityCatalog,
  NetworkChannel,
  NetworkChannelDetailResponse,
  NetworkChannelMessage,
  NetworkChannelMessagesQuery,
  NetworkChannelMessagesResponse,
  NetworkChannelsResponse,
  NetworkChannelSummary,
  NetworkCreateChannelDraft,
  NetworkDetailsTab,
  NetworkKindFilter,
  NetworkPeerCapabilityView,
  NetworkPeerCard,
  NetworkPeerDetail,
  NetworkPeerDetailResponse,
  NetworkPeerMessagesQuery,
  NetworkPeerMessagesResponse,
  NetworkPeersResponse,
  NetworkPeerSummary,
  NetworkRoomField,
  NetworkRoomKindMetric,
  NetworkRoomListItem,
  NetworkRoomMember,
  NetworkRoomType,
  NetworkSendRequest,
  NetworkSendResponse,
  NetworkSignalTone,
  NetworkStatus,
  NetworkStatusResponse,
  NetworkTimelineMessage,
} from "./types";

// Adapters
export {
  createNetworkChannel,
  getNetworkChannel,
  getNetworkPeer,
  getNetworkStatus,
  listNetworkChannelMessages,
  listNetworkChannels,
  listNetworkPeerMessages,
  listNetworkPeers,
  NetworkApiError,
  sendNetworkMessage,
} from "./adapters/network-api";

// Query infrastructure
export { networkKeys } from "./lib/query-keys";
export {
  networkChannelDetailOptions,
  networkChannelMessagesOptions,
  networkChannelsOptions,
  networkPeerDetailOptions,
  networkPeerMessagesOptions,
  networkPeersOptions,
  networkStatusOptions,
} from "./lib/query-options";

// Lib
export {
  NETWORK_KIND_FILTERS,
  buildPeerCapabilityViews,
  createNetworkChannelDraft,
  filterNetworkMessagesByKind,
  formatChannelMemberCount,
  formatChannelPeerCount,
  formatHistoricalParticipantCount,
  formatNetworkClockTime,
  formatNetworkDateTime,
  formatNetworkKindLabel,
  formatNetworkNumber,
  formatNetworkRelativeTime,
  getChannelRecencyAt,
  getChannelDetailDescription,
  getMostRecentTimestamp,
  getNetworkKindTone,
  getMessageAuthorInitial,
  getNetworkMessagePrimaryText,
  getNetworkMetricCards,
  getNetworkRoomKey,
  getNetworkStatusTone,
  getPeerDeliveredRate,
  getPeerDisplayName,
  getPeerHeartbeatLabel,
  getPeerPresenceTone,
  getPeerRecencyAt,
  getPeerTypeLabel,
  hasCapabilityDetail,
  isHistoricalChannel,
  isPresenceOnlyChannel,
  matchesChannelSearch,
  matchesPeerSearch,
  sortAgentsForNetwork,
  sortNetworkChannels,
  sortNetworkPeers,
  summarizeChannelMeta,
  summarizeChannelPreview,
  summarizeChannelSubtitle,
  toNetworkKindFilter,
  toggleDraftAgent,
} from "./lib/network-formatters";

// Hooks
export {
  useNetworkChannel,
  useNetworkChannelMessages,
  useNetworkChannels,
  useNetworkPeer,
  useNetworkPeerMessages,
  useNetworkPeers,
  useNetworkStatus,
} from "./hooks/use-network";
export { useCreateNetworkChannel, useSendNetworkMessage } from "./hooks/use-network-actions";

// Components
export { NetworkCreateChannelDialog } from "./components/network-create-channel-dialog";
export { NetworkWorkspaceShell } from "./components/network-workspace-shell";
