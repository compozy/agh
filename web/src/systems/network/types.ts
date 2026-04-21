import type { OperationQuery, OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type NetworkStatusResponse = OperationResponse<"getNetworkStatus", 200>;
export type NetworkStatus = NetworkStatusResponse["network"];

export type NetworkChannelsResponse = OperationResponse<"listNetworkChannels", 200>;
export type NetworkChannelSummary = NetworkChannelsResponse["channels"][number];

export type NetworkChannelDetailResponse = OperationResponse<"getNetworkChannel", 200>;
export type NetworkChannel = NetworkChannelDetailResponse["channel"];

export type NetworkChannelMessagesResponse = OperationResponse<"listNetworkChannelMessages", 200>;
export type NetworkChannelMessage = NetworkChannelMessagesResponse["messages"][number];
export type NetworkChannelMessagesQuery = OperationQuery<"listNetworkChannelMessages">;

export type NetworkPeersResponse = OperationResponse<"listNetworkPeers", 200>;
export type NetworkPeerSummary = NetworkPeersResponse["peers"][number];

export type NetworkPeerDetailResponse = OperationResponse<"getNetworkPeer", 200>;
export type NetworkPeerDetail = NetworkPeerDetailResponse["peer"];

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

export interface NetworkCreateChannelDraft {
  channelName: string;
  selectedAgentNames: string[];
}

export type NetworkTab = "channels" | "peers";
