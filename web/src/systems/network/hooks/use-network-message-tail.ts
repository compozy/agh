import { useQuery, useQueryClient, type InfiniteData, type QueryKey } from "@tanstack/react-query";

import { listNetworkDirectRoomMessages, listNetworkThreadMessages } from "../adapters/network-api";
import { networkKeys } from "../lib/query-keys";
import {
  NETWORK_DEFAULT_TIMELINE_LIMIT,
  networkDirectMessagesOptions,
  networkThreadMessagesOptions,
} from "../lib/query-options";
import type {
  NetworkConversationMessage,
  NetworkConversationMessageFilters,
  NetworkConversationMessagesResponse,
  NetworkMessagePageParam,
  NetworkSurface,
} from "../types";

const TAIL_REFRESH_INTERVAL = 5_000;
type MessageData = InfiniteData<NetworkConversationMessagesResponse, NetworkMessagePageParam>;

function mergeMessages(
  current: ReadonlyArray<NetworkConversationMessage>,
  incoming: ReadonlyArray<NetworkConversationMessage>
): NetworkConversationMessage[] {
  const byId = new Map(current.map(message => [message.message_id, message]));
  for (const message of incoming) byId.set(message.message_id, message);
  return [...byId.values()];
}

export function rollNetworkMessageTail(
  previous: MessageData,
  incoming: NetworkConversationMessage[]
): MessageData {
  if (incoming.length === 0 || !previous.pages[0]) return previous;
  const pages = previous.pages.map(page => ({ ...page, messages: [...page.messages] }));
  pages[0]!.messages = mergeMessages(pages[0]!.messages, incoming);
  for (let index = 0; index < pages.length; index += 1) {
    const page = pages[index]!;
    const limit = Math.max(1, page.page.limit || NETWORK_DEFAULT_TIMELINE_LIMIT);
    if (page.messages.length <= limit) continue;
    const overflow = page.messages.slice(0, page.messages.length - limit);
    page.messages = page.messages.slice(-limit);
    const next = pages[index + 1];
    if (next) {
      next.messages = mergeMessages(next.messages, overflow);
    } else {
      pages.push({ messages: overflow, page: { ...page.page } });
    }
  }
  const pageParams = [...previous.pageParams];
  while (pageParams.length < pages.length) {
    const newerPage = pages[pageParams.length - 1];
    pageParams.push(newerPage?.messages[0] ? { before: newerPage.messages[0].message_id } : null);
  }
  return { pages, pageParams };
}

function mergeTailResponse(
  previous: MessageData,
  response: NetworkConversationMessagesResponse,
  initialProbe: boolean
): MessageData {
  const next = rollNetworkMessageTail(previous, response.messages);
  if (!initialProbe || previous.pages.length !== 1 || next.pages.length === 0) return next;
  const lastIndex = next.pages.length - 1;
  const lastPage = next.pages[lastIndex];
  if (lastPage) next.pages[lastIndex] = { ...lastPage, page: response.page };
  return next;
}

function latestTailId(data: MessageData | undefined): string | undefined {
  const messages = data?.pages[0]?.messages ?? [];
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message && !("optimistic" in message)) return message.message_id;
  }
  return undefined;
}

interface UseNetworkMessageTailArgs {
  workspaceId: string;
  channel: string;
  surface: NetworkSurface | null | undefined;
  containerId: string;
  filters: NetworkConversationMessageFilters;
  enabled: boolean;
}

export function useNetworkMessageTail({
  workspaceId,
  channel,
  surface,
  containerId,
  filters,
  enabled,
}: UseNetworkMessageTailArgs): { isFetching: boolean; error: Error | null } {
  const queryClient = useQueryClient();
  const normalizedFilters = {
    ...filters,
    limit: filters.limit ?? NETWORK_DEFAULT_TIMELINE_LIMIT,
  };
  const canonicalKey: QueryKey =
    surface === "thread"
      ? networkThreadMessagesOptions(workspaceId, channel, containerId, normalizedFilters).queryKey
      : networkDirectMessagesOptions(workspaceId, channel, containerId, normalizedFilters).queryKey;
  const tailKey =
    surface === "thread"
      ? networkKeys.threadMessageTail(workspaceId, channel, containerId, normalizedFilters)
      : networkKeys.directMessageTail(workspaceId, channel, containerId, normalizedFilters);

  const query = useQuery({
    queryKey: tailKey,
    queryFn: async ({ signal }) => {
      const current = queryClient.getQueryData<MessageData>(canonicalKey);
      const after = latestTailId(current);
      const query = { ...normalizedFilters, ...(after ? { after } : {}) };
      const response =
        surface === "thread"
          ? await listNetworkThreadMessages(workspaceId, channel, containerId, query, signal)
          : await listNetworkDirectRoomMessages(workspaceId, channel, containerId, query, signal);
      queryClient.setQueryData<MessageData>(canonicalKey, previous =>
        previous ? mergeTailResponse(previous, response, !after) : previous
      );
      return null;
    },
    enabled: enabled && Boolean(workspaceId) && Boolean(channel) && Boolean(containerId),
    refetchInterval: TAIL_REFRESH_INTERVAL,
    refetchOnWindowFocus: true,
    staleTime: 0,
  });
  return { isFetching: query.isFetching, error: query.error ?? null };
}
