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
  NetworkChannelMessage,
  NetworkChannelMessagesQuery,
  NetworkChannelsResponse,
  NetworkDirectRoomsResponse,
  NetworkPeerDetail,
  NetworkPeerMessagesQuery,
  NetworkPeerSummary,
  NetworkSendRequest,
  NetworkSendResponse,
  NetworkStatus,
  NetworkThreadsResponse,
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

export async function listNetworkChannels(signal?: AbortSignal): Promise<NetworkChannelsResponse> {
  const { data, error, response } = await apiClient.GET("/api/network/channels", { signal });

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network channels", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch network channels");
}

export async function getNetworkChannel(
  channel: string,
  signal?: AbortSignal
): Promise<NetworkChannel> {
  const { data, error, response } = await apiClient.GET("/api/network/channels/{channel}", {
    params: { path: { channel } },
    signal,
  });

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

export async function listNetworkChannelMessages(
  channel: string,
  query: NetworkChannelMessagesQuery = {},
  signal?: AbortSignal
): Promise<NetworkChannelMessage[]> {
  const threads = await listNetworkThreads(channel, { limit: 1 }, signal);
  const thread = threads.threads.at(0);
  if (!thread) {
    return [];
  }

  const { data, error, response } = await apiClient.GET(
    "/api/network/channels/{channel}/threads/{thread_id}/messages",
    {
      params: {
        path: { channel, thread_id: thread.thread_id },
        query: toConversationMessageQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load messages for "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load messages for "${channel}"`).messages;
}

async function listNetworkThreads(
  channel: string,
  query: { after?: string; limit?: number } = {},
  signal?: AbortSignal
): Promise<NetworkThreadsResponse> {
  const { data, error, response } = await apiClient.GET("/api/network/channels/{channel}/threads", {
    params: { path: { channel }, query },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load threads for "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load threads for "${channel}"`);
}

export async function listNetworkPeers(
  channel?: string,
  signal?: AbortSignal
): Promise<NetworkPeerSummary[]> {
  const { data, error, response } = await apiClient.GET("/api/network/peers", {
    params: {
      query: channel ? { channel } : undefined,
    },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to fetch network peers", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to fetch network peers").peers;
}

export async function getNetworkPeer(
  peerId: string,
  signal?: AbortSignal
): Promise<NetworkPeerDetail> {
  const { data, error, response } = await apiClient.GET("/api/network/peers/{peer_id}", {
    params: { path: { peer_id: peerId } },
    signal,
  });

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

export async function listNetworkPeerMessages(
  peerId: string,
  query: NetworkPeerMessagesQuery = {},
  signal?: AbortSignal
): Promise<NetworkChannelMessage[]> {
  const peer = await getNetworkPeer(peerId, signal);
  const channel = peer.channel;
  if (!channel) {
    return [];
  }
  const directs = await listNetworkDirectRooms(channel, { peer_id: peerId, limit: 1 }, signal);
  const direct = directs.directs.at(0);
  if (!direct) {
    return [];
  }

  const { data, error, response } = await apiClient.GET(
    "/api/network/channels/{channel}/directs/{direct_id}/messages",
    {
      params: {
        path: { channel, direct_id: direct.direct_id },
        query: toConversationMessageQuery(query),
      },
      signal,
    }
  );

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Peer not found: ${peerId}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load direct history for "${peerId}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load direct history for "${peerId}"`)
    .messages;
}

async function listNetworkDirectRooms(
  channel: string,
  query: { after?: string; limit?: number; peer_id?: string } = {},
  signal?: AbortSignal
): Promise<NetworkDirectRoomsResponse> {
  const { data, error, response } = await apiClient.GET("/api/network/channels/{channel}/directs", {
    params: { path: { channel }, query },
    signal,
  });

  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new NetworkApiError(`Channel not found: ${channel}`, 404);
    }

    throw new NetworkApiError(
      defaultApiErrorMessage(`Failed to load direct rooms for "${channel}"`, response, error),
      response.status
    );
  }

  return requireResponseData(data, response, `Failed to load direct rooms for "${channel}"`);
}

function toConversationMessageQuery(query: NetworkChannelMessagesQuery) {
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

export async function createNetworkChannel(
  body: CreateNetworkChannelRequest,
  signal?: AbortSignal
): Promise<CreateNetworkChannelResponse> {
  const { data, error, response } = await apiClient.POST("/api/network/channels", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to create network channel", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to create network channel");
}

export async function sendNetworkMessage(
  body: NetworkSendRequest,
  signal?: AbortSignal
): Promise<NetworkSendResponse> {
  const { data, error, response } = await apiClient.POST("/api/network/send", {
    body,
    signal,
  });

  if (apiRequestFailed(response, error)) {
    throw new NetworkApiError(
      defaultApiErrorMessage("Failed to send network message", response, error),
      response.status
    );
  }

  return requireResponseData(data, response, "Failed to send network message");
}
