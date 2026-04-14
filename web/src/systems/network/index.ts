// Types
export type {
  CreateNetworkChannelRequest,
  CreateNetworkChannelResponse,
  NetworkChannel,
  NetworkChannelDetailResponse,
  NetworkChannelMessage,
  NetworkChannelMessagesQuery,
  NetworkChannelMessagesResponse,
  NetworkChannelsResponse,
  NetworkChannelSummary,
  NetworkCreateChannelDraft,
  NetworkPeerDetail,
  NetworkPeerDetailResponse,
  NetworkPeersResponse,
  NetworkPeerSummary,
  NetworkStatus,
  NetworkStatusResponse,
  NetworkTab,
} from "./types";

// Adapters
export {
  createNetworkChannel,
  getNetworkChannel,
  getNetworkPeer,
  getNetworkStatus,
  listNetworkChannelMessages,
  listNetworkChannels,
  listNetworkPeers,
  NetworkApiError,
} from "./adapters/network-api";

// Query infrastructure
export { networkKeys } from "./lib/query-keys";
export {
  networkChannelDetailOptions,
  networkChannelMessagesOptions,
  networkChannelsOptions,
  networkPeerDetailOptions,
  networkPeersOptions,
  networkStatusOptions,
} from "./lib/query-options";

// Lib
export {
  createNetworkChannelDraft,
  formatChannelMemberCount,
  formatChannelPeerCount,
  formatNetworkClockTime,
  formatNetworkDateTime,
  formatNetworkNumber,
  formatNetworkRelativeTime,
  getChannelDetailDescription,
  getMessageAuthorInitial,
  getNetworkMetricCards,
  getPeerDeliveredRate,
  getPeerDisplayName,
  getPeerHeartbeatLabel,
  getPeerPresenceTone,
  getPeerTypeLabel,
  matchesChannelSearch,
  matchesPeerSearch,
  sortAgentsForNetwork,
  sortNetworkChannels,
  sortNetworkPeers,
  toggleDraftAgent,
} from "./lib/network-formatters";

// Hooks
export {
  useNetworkChannel,
  useNetworkChannelMessages,
  useNetworkChannels,
  useNetworkPeer,
  useNetworkPeers,
  useNetworkStatus,
} from "./hooks/use-network";
export { useCreateNetworkChannel } from "./hooks/use-network-actions";

// Components
export { NetworkChannelDetailPanel } from "./components/network-channel-detail-panel";
export { NetworkChannelsListPanel } from "./components/network-channels-list-panel";
export { NetworkCreateChannelDialog } from "./components/network-create-channel-dialog";
export { NetworkEmptyState } from "./components/network-empty-state";
export { NetworkPeerDetailPanel } from "./components/network-peer-detail-panel";
export { NetworkPeersListPanel } from "./components/network-peers-list-panel";
