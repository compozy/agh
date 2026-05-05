import type { OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type NetworkSurface = "thread" | "direct";

export type NetworkSignalTone = "accent" | "success" | "warning" | "danger" | "info" | "neutral";

export type NetworkKindFilter =
  | "all"
  | "say"
  | "receipt"
  | "capability"
  | "greet"
  | "whois"
  | "trace";

export type NetworkStatusResponse = OperationResponse<"getNetworkStatus", 200>;
export type NetworkStatus = NetworkStatusResponse["network"];

export type NetworkChannelsResponse = OperationResponse<"listNetworkChannels", 200>;
export type NetworkChannelSummary = NetworkChannelsResponse["channels"][number];

export type NetworkChannelDetailResponse = OperationResponse<"getNetworkChannel", 200>;
export type NetworkChannel = NetworkChannelDetailResponse["channel"];

export type NetworkThreadsResponse = OperationResponse<"listNetworkThreads", 200>;
export type NetworkThreadSummary = NetworkThreadsResponse["threads"][number];

export type NetworkThreadDetailResponse = OperationResponse<"getNetworkThread", 200>;
export type NetworkThreadDetail = NetworkThreadDetailResponse["thread"];

export type NetworkThreadMessagesResponse = OperationResponse<"listNetworkThreadMessages", 200>;
export type NetworkThreadMessage = NetworkThreadMessagesResponse["messages"][number];

export type NetworkDirectRoomsResponse = OperationResponse<"listNetworkDirectRooms", 200>;
export type NetworkDirectRoomSummary = NetworkDirectRoomsResponse["directs"][number];

export type NetworkDirectRoomDetailResponse = OperationResponse<"getNetworkDirectRoom", 200>;
export type NetworkDirectRoomDetail = NetworkDirectRoomDetailResponse["direct"];

export type NetworkDirectRoomMessagesResponse = OperationResponse<
  "listNetworkDirectRoomMessages",
  200
>;
export type NetworkDirectRoomMessage = NetworkDirectRoomMessagesResponse["messages"][number];

export type NetworkResolveDirectRoomRequest = OperationRequestBody<"resolveNetworkDirectRoom">;
export type NetworkResolveDirectRoomResponse = OperationResponse<"resolveNetworkDirectRoom", 200>;

export type NetworkWorkResponse = OperationResponse<"getNetworkWork", 200>;
export type NetworkWorkDetail = NetworkWorkResponse["work"];

export type NetworkConversationMessage = NetworkThreadMessage | NetworkDirectRoomMessage;

export type NetworkPeersResponse = OperationResponse<"listNetworkPeers", 200>;
export type NetworkPeerSummary = NetworkPeersResponse["peers"][number];

export type NetworkPeerDetailResponse = OperationResponse<"getNetworkPeer", 200>;
export type NetworkPeerDetail = NetworkPeerDetailResponse["peer"];

export type NetworkPeerCard = NetworkPeerSummary["peer_card"];
export type NetworkCapabilityBrief = NetworkPeerCard["capabilities"][number];

export type NetworkCapabilityCatalog = NonNullable<NetworkPeerDetail["capability_catalog"]>;
export type NetworkCapability = NetworkCapabilityCatalog["capabilities"][number];

export type CreateNetworkChannelRequest = OperationRequestBody<"createNetworkChannel">;
export type CreateNetworkChannelResponse = OperationResponse<"createNetworkChannel", 201>;
export type NetworkSendRequest = OperationRequestBody<"sendNetworkMessage">;
export type NetworkSendResponse = OperationResponse<"sendNetworkMessage", 200>;

export interface NetworkCreateChannelDraft {
  channelName: string;
  purpose: string;
  selectedAgentNames: string[];
}

export interface NetworkConversationMessagesQuery {
  after?: string | null | undefined;
  before?: string | null | undefined;
  kind?: string | null | undefined;
  limit?: number | null | undefined;
  work_id?: string | null | undefined;
}

export type NetworkRouteSurface = NetworkSurface | "activity";

export interface NetworkRecentEntry {
  surface: NetworkSurface;
  channel: string;
  containerId: string;
  preview: string;
  lastActivityAt: string | null;
  hasUnread: boolean;
  participantLabel: string;
}
