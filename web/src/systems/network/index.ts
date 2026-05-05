// Types
export type {
  CreateNetworkChannelRequest,
  CreateNetworkChannelResponse,
  NetworkCapability,
  NetworkCapabilityBrief,
  NetworkCapabilityCatalog,
  NetworkChannel,
  NetworkChannelDetailResponse,
  NetworkChannelsResponse,
  NetworkChannelSummary,
  NetworkConversationMessage,
  NetworkConversationMessagesQuery,
  NetworkCreateChannelDraft,
  NetworkDirectRoomDetail,
  NetworkDirectRoomDetailResponse,
  NetworkDirectRoomMessage,
  NetworkDirectRoomMessagesResponse,
  NetworkDirectRoomSummary,
  NetworkDirectRoomsResponse,
  NetworkKindFilter,
  NetworkPeerCard,
  NetworkPeerDetail,
  NetworkPeerDetailResponse,
  NetworkPeerSummary,
  NetworkPeersResponse,
  NetworkRecentEntry,
  NetworkResolveDirectRoomRequest,
  NetworkResolveDirectRoomResponse,
  NetworkRouteSurface,
  NetworkSendRequest,
  NetworkSendResponse,
  NetworkSignalTone,
  NetworkStatus,
  NetworkStatusResponse,
  NetworkSurface,
  NetworkThreadDetail,
  NetworkThreadDetailResponse,
  NetworkThreadMessage,
  NetworkThreadMessagesResponse,
  NetworkThreadsResponse,
  NetworkThreadSummary,
  NetworkWorkDetail,
  NetworkWorkResponse,
} from "./types";

// Adapters
export {
  createNetworkChannel,
  getNetworkChannel,
  getNetworkDirectRoom,
  getNetworkPeer,
  getNetworkStatus,
  getNetworkThread,
  getNetworkWork,
  listNetworkChannels,
  listNetworkDirectRoomMessages,
  listNetworkDirectRooms,
  listNetworkPeers,
  listNetworkThreadMessages,
  listNetworkThreads,
  NetworkApiError,
  resolveNetworkDirectRoom,
  sendNetworkMessage,
} from "./adapters/network-api";
export type { NetworkDirectsListQuery, NetworkThreadsListQuery } from "./adapters/network-api";

// Query infrastructure
export { networkKeys } from "./lib/query-keys";
export {
  networkChannelDetailOptions,
  networkChannelsOptions,
  networkDirectDetailOptions,
  networkDirectMessagesOptions,
  networkDirectsOptions,
  networkPeerDetailOptions,
  networkPeersOptions,
  networkStatusOptions,
  networkThreadDetailOptions,
  networkThreadMessagesOptions,
  networkThreadsOptions,
  networkWorkOptions,
} from "./lib/query-options";

// Lib
export {
  NETWORK_KIND_FILTERS,
  createNetworkChannelDraft,
  formatNetworkClockTime,
  formatNetworkDateTime,
  formatNetworkKindLabel,
  formatNetworkNumber,
  formatNetworkRelativeTime,
  getMessageAuthorInitial,
  getMostRecentTimestamp,
  getNetworkKindTone,
  getNetworkStatusTone,
  getPeerDisplayName,
  getPeerRecencyAt,
  isNetworkRunning,
  sortAgentsForNetwork,
  toNetworkKindFilter,
  toggleDraftAgent,
} from "./lib/network-formatters";
export {
  NETWORK_IDENTITY_PALETTE,
  getIdentityInitial,
  pickIdentityPaletteColors,
  pickIdentityPaletteIndex,
} from "./lib/palette";

// Hooks
export { useNetworkChannels } from "./hooks/use-channels";
export { useLastRead, buildLastReadStorageKey } from "./hooks/use-last-read";
export type { NetworkLastReadKey, UseLastReadResult } from "./hooks/use-last-read";
export { useNetworkPage } from "./hooks/use-network-page";
export type { UseNetworkPageResult } from "./hooks/use-network-page";
export { useNetworkRecents } from "./hooks/use-recents";
export type { UseNetworkRecentsResult } from "./hooks/use-recents";
export { useNetworkRouteShell } from "./hooks/use-network-route-shell";
export type { NetworkRouteShellResult } from "./hooks/use-network-route-shell";
export { useCreateNetworkChannel, useSendNetworkMessage } from "./hooks/use-network-actions";

// Components
export { KindChip } from "./components/kind-chip";
export type { KindChipProps } from "./components/kind-chip";
export { NetworkCreateChannelDialog } from "./components/network-create-channel-dialog";
