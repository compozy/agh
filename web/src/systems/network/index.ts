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
export { useNetworkDirects, useNetworkDirectDetail } from "./hooks/use-directs";
export type {
  UseNetworkDirectsOptions,
  UseNetworkDirectsResult,
  UseNetworkDirectDetailResult,
} from "./hooks/use-directs";
export { useNetworkMessages } from "./hooks/use-messages";
export type { UseNetworkMessagesArgs, UseNetworkMessagesResult } from "./hooks/use-messages";
export { useNetworkPage } from "./hooks/use-network-page";
export type { UseNetworkPageResult } from "./hooks/use-network-page";
export { useNetworkPresence } from "./hooks/use-network-presence";
export type {
  NetworkPresence,
  NetworkPresenceArgs,
  NetworkPresenceState,
} from "./hooks/use-network-presence";
export { useDirectRoom } from "./hooks/use-direct-room";
export type { UseDirectRoomArgs, UseDirectRoomResult } from "./hooks/use-direct-room";
export { useThreadOverlay } from "./hooks/use-thread-overlay";
export type { UseThreadOverlayArgs, UseThreadOverlayResult } from "./hooks/use-thread-overlay";
export { useNetworkRecents } from "./hooks/use-recents";
export type { UseNetworkRecentsResult } from "./hooks/use-recents";
export { useNetworkRouteShell } from "./hooks/use-network-route-shell";
export type { NetworkRouteShellResult } from "./hooks/use-network-route-shell";
export { useNetworkThreads, useNetworkThreadDetail } from "./hooks/use-threads";
export type {
  UseNetworkThreadsOptions,
  UseNetworkThreadsResult,
  UseNetworkThreadDetailResult,
} from "./hooks/use-threads";
export { THREAD_OVERLAY_BREAKPOINT_PX, useThreadViewMode } from "./hooks/use-thread-view-mode";
export type { ThreadViewMode } from "./hooks/use-thread-view-mode";
export { useCreateNetworkChannel, useSendNetworkMessage } from "./hooks/use-network-actions";

// Lib (timeline composition + formatters)
export {
  buildTimelineEntries,
  isSameDayMessage,
  isSystemKind,
  SYSTEM_KINDS,
} from "./lib/group-messages";
export type {
  TimelineDatePillEntry,
  TimelineEntry,
  TimelineMessageEntry,
  TimelineNewDividerEntry,
  TimelineRowVariant,
} from "./lib/group-messages";
export {
  TIMELINE_GROUP_WINDOW_SECONDS,
  formatDatePill,
  formatTimelineClock,
  formatTimelineClockWithSeconds,
  formatTimelineIso,
  isSameCalendarDay,
  isWithinSeconds,
} from "./lib/format-timestamp";

// Components
export { KindChip } from "./components/kind-chip";
export type { KindChipProps } from "./components/kind-chip";
export { NetworkCreateChannelDialog } from "./components/network-create-channel-dialog";

// Components — timeline subtree
export {
  DatePill,
  HoverToolbar,
  MessageAvatar,
  MessageBodyText,
  MessageRow,
  MessageRowCollapsed,
  MessageRowSystem,
  NewDivider,
  Timeline,
  readMessageBody,
} from "./components/timeline";
export type {
  DatePillProps,
  HoverToolbarHandlers,
  HoverToolbarProps,
  MessageAvatarProps,
  MessageRowCollapsedProps,
  MessageRowDensity,
  MessageRowProps,
  MessageRowSystemProps,
  NewDividerProps,
  TimelineProps,
} from "./components/timeline";

// Components — thread overlay subtree
export {
  ThreadOverlay,
  ThreadOverlayHeader,
  ThreadOverlayReplies,
  ThreadOverlayRoot,
} from "./components/thread-overlay";
export type {
  ThreadOverlayHeaderProps,
  ThreadOverlayProps,
  ThreadOverlayRepliesProps,
  ThreadOverlayRootProps,
} from "./components/thread-overlay";

// Components — list views
export { ThreadsList } from "./components/threads";
export type { ThreadsListProps } from "./components/threads";
export { DirectRoom, DirectsList } from "./components/directs";
export type { DirectRoomProps, DirectsListProps } from "./components/directs";
export { ActivityFeed } from "./components/activity";
export type { ActivityFeedProps } from "./components/activity";
