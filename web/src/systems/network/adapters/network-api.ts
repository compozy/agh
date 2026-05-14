import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  CreateNetworkChannelRequest,
  CreateNetworkChannelResponse,
  NetworkChannel,
  NetworkChannelsResponse,
  NetworkConversationMessagesQuery,
  NetworkDirectRoomDetail,
  NetworkDirectRoomMessage,
  NetworkDirectRoomSummary,
  NetworkPeerDetail,
  NetworkPeerSummary,
  NetworkResolveDirectRoomRequest,
  NetworkSendRequest,
  NetworkSendResponse,
  NetworkStatus,
  NetworkThreadDetail,
  NetworkThreadMessage,
  NetworkThreadSummary,
  NetworkWorkDetail,
} from "../types";

export class NetworkApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "NetworkApiError";
  }
}

export async function getNetworkStatus(signal?: AbortSignal): Promise<NetworkStatus> {
  const { data, error, response } = await apiClient.GET("/api/network/status", { signal });

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network status", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch network status").network;
}

export async function listNetworkChannels(
  workspaceId: string,
  signal?: AbortSignal
): Promise<NetworkChannelsResponse> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels",
    {
      params: { path: { workspace_id: workspaceId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network channels", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch network channels");
}

export async function getNetworkChannel(
  workspaceId: string,
  channel: string,
  signal?: AbortSignal
): Promise<NetworkChannel> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}",
    {
      params: { path: { workspace_id: workspaceId, channel } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load channel "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load channel "${channel}"`).channel;
}

export interface NetworkThreadsListQuery {
  after?: string;
  limit?: number;
}

export async function listNetworkThreads(
  workspaceId: string,
  channel: string,
  query: NetworkThreadsListQuery = {},
  signal?: AbortSignal
): Promise<NetworkThreadSummary[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/threads",
    {
      params: {
        path: { workspace_id: workspaceId, channel },
        query: toThreadsListQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load threads for "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load threads for "${channel}"`).threads;
}

export async function getNetworkThread(
  workspaceId: string,
  channel: string,
  threadId: string,
  signal?: AbortSignal
): Promise<NetworkThreadDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/threads/{thread_id}",
    {
      params: { path: { workspace_id: workspaceId, channel, thread_id: threadId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Thread not found: ${threadId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load thread "${threadId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load thread "${threadId}"`).thread;
}

export async function listNetworkThreadMessages(
  workspaceId: string,
  channel: string,
  threadId: string,
  query: NetworkConversationMessagesQuery = {},
  signal?: AbortSignal
): Promise<NetworkThreadMessage[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/threads/{thread_id}/messages",
    {
      params: {
        path: { workspace_id: workspaceId, channel, thread_id: threadId },
        query: toConversationMessageQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Thread not found: ${threadId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load thread messages for "${threadId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load thread messages for "${threadId}"`)
    .messages;
}

export interface NetworkDirectsListQuery {
  after?: string;
  limit?: number;
  peer_id?: string;
}

export async function listNetworkDirectRooms(
  workspaceId: string,
  channel: string,
  query: NetworkDirectsListQuery = {},
  signal?: AbortSignal
): Promise<NetworkDirectRoomSummary[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/directs",
    {
      params: {
        path: { workspace_id: workspaceId, channel },
        query: toDirectsListQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load direct rooms for "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load direct rooms for "${channel}"`)
    .directs;
}

export async function getNetworkDirectRoom(
  workspaceId: string,
  channel: string,
  directId: string,
  signal?: AbortSignal
): Promise<NetworkDirectRoomDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/directs/{direct_id}",
    {
      params: { path: { workspace_id: workspaceId, channel, direct_id: directId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Direct room not found: ${directId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load direct room "${directId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load direct room "${directId}"`).direct;
}

export async function listNetworkDirectRoomMessages(
  workspaceId: string,
  channel: string,
  directId: string,
  query: NetworkConversationMessagesQuery = {},
  signal?: AbortSignal
): Promise<NetworkDirectRoomMessage[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/directs/{direct_id}/messages",
    {
      params: {
        path: { workspace_id: workspaceId, channel, direct_id: directId },
        query: toConversationMessageQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Direct room not found: ${directId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load direct messages for "${directId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load direct messages for "${directId}"`)
    .messages;
}

export async function resolveNetworkDirectRoom(
  workspaceId: string,
  channel: string,
  body: NetworkResolveDirectRoomRequest,
  signal?: AbortSignal
): Promise<NetworkDirectRoomDetail> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/network/channels/{channel}/directs/resolve",
    {
      params: { path: { workspace_id: workspaceId, channel } },
      body,
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to resolve direct room in "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to resolve direct room in "${channel}"`)
    .direct;
}

export async function getNetworkWork(
  workspaceId: string,
  workId: string,
  signal?: AbortSignal
): Promise<NetworkWorkDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/work/{work_id}",
    {
      params: { path: { workspace_id: workspaceId, work_id: workId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Network work not found: ${workId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load network work "${workId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load network work "${workId}"`).work;
}

export async function listNetworkPeers(
  workspaceId: string,
  channel?: string,
  signal?: AbortSignal
): Promise<NetworkPeerSummary[]> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/peers",
    {
      params: {
        path: { workspace_id: workspaceId },
        query: channel ? { channel } : undefined,
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network peers", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch network peers").peers;
}

export async function getNetworkPeer(
  workspaceId: string,
  peerId: string,
  signal?: AbortSignal
): Promise<NetworkPeerDetail> {
  const { data, error, response } = await apiClient.GET(
    "/api/workspaces/{workspace_id}/network/peers/{peer_id}",
    {
      params: { path: { workspace_id: workspaceId, peer_id: peerId } },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Peer not found: ${peerId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load peer "${peerId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load peer "${peerId}"`).peer;
}

export async function createNetworkChannel(
  workspaceId: string,
  body: CreateNetworkChannelRequest,
  signal?: AbortSignal
): Promise<CreateNetworkChannelResponse> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/network/channels",
    {
      params: { path: { workspace_id: workspaceId } },
      body: { ...body, workspace_id: workspaceId },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to create network channel", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to create network channel");
}

export async function sendNetworkMessage(
  workspaceId: string,
  body: NetworkSendRequest,
  signal?: AbortSignal
): Promise<NetworkSendResponse> {
  const { data, error, response } = await apiClient.POST(
    "/api/workspaces/{workspace_id}/network/send",
    {
      params: { path: { workspace_id: workspaceId } },
      body: { ...body, workspace_id: workspaceId },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to send network message", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to send network message");
}

function toThreadsListQuery(query: NetworkThreadsListQuery) {
  const supported: { after?: string; limit?: number } = {};
  if (query.after) {
    supported.after = query.after;
  }
  if (query.limit != null) {
    supported.limit = query.limit;
  }
  return supported;
}

function toDirectsListQuery(query: NetworkDirectsListQuery) {
  const supported: { after?: string; limit?: number; peer_id?: string } = {};
  if (query.after) {
    supported.after = query.after;
  }
  if (query.limit != null) {
    supported.limit = query.limit;
  }
  if (query.peer_id) {
    supported.peer_id = query.peer_id;
  }
  return supported;
}

function toConversationMessageQuery(query: NetworkConversationMessagesQuery) {
  const supported: {
    after?: string;
    before?: string;
    kind?: string;
    limit?: number;
    work_id?: string;
  } = {};
  if (query.after) {
    supported.after = query.after;
  }
  if (query.before) {
    supported.before = query.before;
  }
  if (query.kind) {
    supported.kind = query.kind;
  }
  if (query.work_id) {
    supported.work_id = query.work_id;
  }
  if (query.limit != null) {
    supported.limit = query.limit;
  }
  return supported;
}
