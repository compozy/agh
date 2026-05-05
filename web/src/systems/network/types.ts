import type { OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type NetworkStatusResponse = OperationResponse<"getNetworkStatus", 200>;
export type NetworkStatus = NetworkStatusResponse["network"];

export type NetworkChannelsResponse = OperationResponse<"listNetworkChannels", 200>;
export type NetworkChannelSummary = NetworkChannelsResponse["channels"][number];

export type NetworkChannelDetailResponse = OperationResponse<"getNetworkChannel", 200>;
export type NetworkChannel = NetworkChannelDetailResponse["channel"];

export type NetworkThreadsResponse = OperationResponse<"listNetworkThreads", 200>;
export type NetworkThreadSummary = NetworkThreadsResponse["threads"][number];
export type NetworkThreadMessagesResponse = OperationResponse<"listNetworkThreadMessages", 200>;
export type NetworkDirectRoomsResponse = OperationResponse<"listNetworkDirectRooms", 200>;
export type NetworkDirectRoomSummary = NetworkDirectRoomsResponse["directs"][number];
export type NetworkDirectRoomMessagesResponse = OperationResponse<
  "listNetworkDirectRoomMessages",
  200
>;

export type NetworkChannelMessagesResponse = NetworkThreadMessagesResponse;
export type NetworkChannelMessage = NetworkChannelMessagesResponse["messages"][number];
export interface NetworkConversationMessagesQuery {
  after?: string | null | undefined;
  before?: string | null | undefined;
  include_presence?: boolean | null;
  kind?: string | null | undefined;
  limit?: number | null | undefined;
  work_id?: string | null | undefined;
}
export type NetworkChannelMessagesQuery = NetworkConversationMessagesQuery;
export type NetworkTimelineMessage = NetworkChannelMessage;

export type NetworkPeersResponse = OperationResponse<"listNetworkPeers", 200>;
export type NetworkPeerSummary = NetworkPeersResponse["peers"][number];

export type NetworkPeerDetailResponse = OperationResponse<"getNetworkPeer", 200>;
export type NetworkPeerDetail = NetworkPeerDetailResponse["peer"];
export type NetworkPeerMessagesResponse = NetworkDirectRoomMessagesResponse;
export type NetworkPeerMessagesQuery = NetworkConversationMessagesQuery;

export type NetworkPeerCard = NetworkPeerSummary["peer_card"];
export type NetworkCapabilityBrief = NetworkPeerCard["capabilities"][number];

export type NetworkCapabilityCatalog = NonNullable<NetworkPeerDetail["capability_catalog"]>;
export type NetworkCapability = NetworkCapabilityCatalog["capabilities"][number];

export interface NetworkPeerCapabilityView {
  id: string;
  summary: string;
  detail: NetworkCapability | null;
}

export type CreateNetworkChannelRequest = OperationRequestBody<"createNetworkChannel">;
export type CreateNetworkChannelResponse = OperationResponse<"createNetworkChannel", 201>;
export type NetworkSendRequest = OperationRequestBody<"sendNetworkMessage">;
export type NetworkSendResponse = OperationResponse<"sendNetworkMessage", 200>;

export interface NetworkCreateChannelDraft {
  channelName: string;
  purpose: string;
  selectedAgentNames: string[];
}

export type NetworkRoomType = "channel" | "peer";
export type NetworkDetailsTab = "about" | "members" | "wire";
export type NetworkSignalTone = "accent" | "success" | "warning" | "danger" | "info" | "neutral";
export type NetworkKindFilter =
  | "all"
  | "say"
  | "direct"
  | "receipt"
  | "capability"
  | "greet"
  | "whois"
  | "trace";

export interface NetworkRoomListItem {
  id: string;
  isStarred: boolean;
  key: string;
  lastActivityAt: string | null;
  meta: string;
  preview: string;
  roomType: NetworkRoomType;
  subtitle: string;
  title: string;
  tone: NetworkSignalTone;
  unreadCount: number;
}

export interface NetworkRoomMember {
  id: string;
  lastSeen: string | null;
  local: boolean;
  sessionId: string | null;
  subtitle: string;
  title: string;
  tone: NetworkSignalTone;
}

export interface NetworkRoomField {
  label: string;
  mono?: boolean;
  tone?: NetworkSignalTone;
  value: string;
}

export interface NetworkRoomKindMetric {
  count: number;
  kind: Exclude<NetworkKindFilter, "all">;
}

export interface NetworkActiveRoom {
  aboutFields: NetworkRoomField[];
  canCompose: boolean;
  canStar: boolean;
  capabilities: NetworkPeerCapabilityView[];
  channel: string;
  composeHint: string | null;
  composePlaceholder: string;
  description: string;
  id: string;
  introBody: string;
  introTitle: string;
  isStarred: boolean;
  key: string;
  kindCounts: NetworkRoomKindMetric[];
  lastActivityAt: string | null;
  lastPresenceAt: string | null;
  memberCount: number;
  members: NetworkRoomMember[];
  messageCount: number;
  messages: NetworkTimelineMessage[];
  presenceCount: number;
  preview: string;
  purpose: string | null;
  roomType: NetworkRoomType;
  subtitle: string;
  title: string;
  wireFields: NetworkRoomField[];
}
