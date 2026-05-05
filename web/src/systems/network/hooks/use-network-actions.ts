import { useCallback, useMemo } from "react";
import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "@agh/ui";

import {
  createNetworkChannel,
  NetworkApiError,
  resolveNetworkDirectRoom,
  sendNetworkMessage,
} from "@/systems/network/adapters/network-api";
import { networkKeys } from "@/systems/network/lib/query-keys";
import { sessionKeys } from "@/systems/session";
import type {
  CreateNetworkChannelRequest,
  NetworkConversationMessage,
  NetworkResolveDirectRoomRequest,
  NetworkResolveDirectRoomResponse,
  NetworkSendRequest,
  NetworkSendResponse,
  NetworkSurface,
} from "@/systems/network/types";

const THREAD_COLLISION_TOAST = "Couldn't open this thread. Try again.";

export interface OptimisticMessageMeta {
  /** `"pending"` while in flight, `"failed"` after the server rejects the send. */
  optimistic: "pending" | "failed";
}

export type OptimisticConversationMessage = NetworkConversationMessage & OptimisticMessageMeta;

function isOptimisticMessage(
  message: NetworkConversationMessage
): message is OptimisticConversationMessage {
  return (message as Partial<OptimisticConversationMessage>).optimistic != null;
}

function generateClientMessageId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  // Defensive fallback for environments without WebCrypto (still UUID-shaped).
  const random = () => Math.floor(Math.random() * 0xffffffff).toString(16);
  return `${random()}${random()}-${random()}-${random()}-${random()}-${random()}${random()}${random()}`;
}

function buildSendRequest(
  input: SendNetworkMessageInput,
  clientMessageId: string
): NetworkSendRequest {
  const base: NetworkSendRequest = {
    body: { text: input.text },
    channel: input.channel,
    id: clientMessageId,
    kind: "say",
    session_id: input.sessionId,
    surface: input.surface,
  };
  if (input.surface === "thread" && input.threadId) {
    return { ...base, thread_id: input.threadId };
  }
  if (input.surface === "direct" && input.directId) {
    return { ...base, direct_id: input.directId };
  }
  return base;
}

function buildOptimisticMessage(
  input: SendNetworkMessageInput,
  clientMessageId: string,
  timestamp: string
): OptimisticConversationMessage {
  const base: OptimisticConversationMessage = {
    body: { text: input.text },
    channel: input.channel,
    direction: "sent",
    display_name: input.displayName,
    kind: "say",
    local: true,
    message_id: clientMessageId,
    optimistic: "pending",
    peer_from: input.peerFrom,
    preview_text: input.text,
    session_id: input.sessionId,
    surface: input.surface,
    text: input.text,
    timestamp,
  };
  if (input.surface === "thread") {
    base.thread_id = input.threadId;
  }
  if (input.surface === "direct") {
    base.direct_id = input.directId;
    if (input.peerTo) {
      base.peer_to = input.peerTo;
    }
  }
  return base;
}

function activeContainerKey(input: SendNetworkMessageInput) {
  if (input.surface === "thread") {
    return networkKeys.threadMessages(input.channel, input.threadId);
  }
  return networkKeys.directMessages(input.channel, input.directId);
}

function applyOptimistic(
  queryClient: QueryClient,
  input: SendNetworkMessageInput,
  optimistic: OptimisticConversationMessage
) {
  queryClient.setQueriesData<NetworkConversationMessage[] | undefined>(
    { queryKey: activeContainerKey(input) },
    previous => {
      if (!previous) {
        return previous;
      }
      return [...previous, optimistic];
    }
  );
}

function replaceOptimisticOnSuccess(
  queryClient: QueryClient,
  input: SendNetworkMessageInput,
  clientMessageId: string,
  timestamp: string
) {
  queryClient.setQueriesData<NetworkConversationMessage[] | undefined>(
    { queryKey: activeContainerKey(input) },
    previous => {
      if (!previous) {
        return previous;
      }
      return previous.map(message => {
        if (message.message_id !== clientMessageId) {
          return message;
        }
        const canonical: NetworkConversationMessage = {
          ...message,
          timestamp,
        };
        if (isOptimisticMessage(canonical)) {
          delete (canonical as Partial<OptimisticConversationMessage>).optimistic;
        }
        return canonical;
      });
    }
  );
}

function markOptimisticFailed(
  queryClient: QueryClient,
  input: SendNetworkMessageInput,
  clientMessageId: string
) {
  queryClient.setQueriesData<NetworkConversationMessage[] | undefined>(
    { queryKey: activeContainerKey(input) },
    previous => {
      if (!previous) {
        return previous;
      }
      return previous.map(message => {
        if (message.message_id !== clientMessageId) {
          return message;
        }
        return {
          ...message,
          optimistic: "failed",
        } as OptimisticConversationMessage;
      });
    }
  );
}

function discardOptimistic(
  queryClient: QueryClient,
  input: SendNetworkMessageInput,
  clientMessageId: string
) {
  queryClient.setQueriesData<NetworkConversationMessage[] | undefined>(
    { queryKey: activeContainerKey(input) },
    previous => {
      if (!previous) {
        return previous;
      }
      return previous.filter(message => message.message_id !== clientMessageId);
    }
  );
}

export interface SendNetworkMessageThreadInput {
  surface: "thread";
  channel: string;
  threadId: string;
  sessionId: string;
  text: string;
  peerFrom: string;
  displayName?: string;
  /** When provided, replaces an existing optimistic message id (used by retry). */
  clientMessageId?: string;
}

export interface SendNetworkMessageDirectInput {
  surface: "direct";
  channel: string;
  directId: string;
  sessionId: string;
  text: string;
  peerFrom: string;
  peerTo?: string;
  displayName?: string;
  clientMessageId?: string;
}

export type SendNetworkMessageInput = SendNetworkMessageThreadInput | SendNetworkMessageDirectInput;

export interface SendNetworkMessageResult {
  clientMessageId: string;
  response: NetworkSendResponse;
}

export interface UseSendNetworkMessageResult {
  send: (input: SendNetworkMessageInput) => Promise<SendNetworkMessageResult>;
  retry: (
    input: SendNetworkMessageInput,
    clientMessageId: string
  ) => Promise<SendNetworkMessageResult>;
  discard: (input: SendNetworkMessageInput, clientMessageId: string) => void;
  isSending: boolean;
}

function invalidateContainerQueries(queryClient: QueryClient, input: SendNetworkMessageInput) {
  if (input.surface === "thread") {
    return Promise.all([
      queryClient.invalidateQueries({
        queryKey: networkKeys.threadMessages(input.channel, input.threadId),
      }),
      queryClient.invalidateQueries({
        queryKey: networkKeys.threadDetail(input.channel, input.threadId),
      }),
      queryClient.invalidateQueries({
        queryKey: networkKeys.threadsList(input.channel),
      }),
    ]);
  }
  return Promise.all([
    queryClient.invalidateQueries({
      queryKey: networkKeys.directMessages(input.channel, input.directId),
    }),
    queryClient.invalidateQueries({
      queryKey: networkKeys.directDetail(input.channel, input.directId),
    }),
    queryClient.invalidateQueries({
      queryKey: networkKeys.directsList(input.channel),
    }),
  ]);
}

export function useSendNetworkMessage(): UseSendNetworkMessageResult {
  const queryClient = useQueryClient();

  const mutation = useMutation<SendNetworkMessageResult, Error, SendNetworkMessageInput>({
    mutationFn: async (input: SendNetworkMessageInput) => {
      const clientMessageId = input.clientMessageId ?? generateClientMessageId();
      const isRetry = input.clientMessageId != null;
      const timestamp = new Date().toISOString();
      const optimistic = buildOptimisticMessage(input, clientMessageId, timestamp);

      if (isRetry) {
        // The retry path already has the optimistic placeholder in cache; flip
        // it back to "pending" so the danger-tint and inline retry/discard
        // disappear while the second attempt is in flight.
        queryClient.setQueriesData<NetworkConversationMessage[] | undefined>(
          { queryKey: activeContainerKey(input) },
          previous => {
            if (!previous) {
              return previous;
            }
            return previous.map(message => {
              if (message.message_id !== clientMessageId) {
                return message;
              }
              return { ...optimistic, message_id: clientMessageId };
            });
          }
        );
      } else {
        applyOptimistic(queryClient, input, optimistic);
      }

      try {
        const request = buildSendRequest(input, clientMessageId);
        const response = await sendNetworkMessage(request);
        replaceOptimisticOnSuccess(queryClient, input, clientMessageId, timestamp);
        return { clientMessageId, response };
      } catch (error) {
        markOptimisticFailed(queryClient, input, clientMessageId);
        throw error;
      }
    },
    onSettled: (_data, _error, variables) => {
      void invalidateContainerQueries(queryClient, variables);
      void queryClient.invalidateQueries({ queryKey: sessionKeys.lists() });
    },
  });

  const send = useCallback(
    (input: SendNetworkMessageInput) => mutation.mutateAsync(input),
    [mutation]
  );
  const retry = useCallback(
    (input: SendNetworkMessageInput, clientMessageId: string) =>
      mutation.mutateAsync({ ...input, clientMessageId }),
    [mutation]
  );
  const discard = useCallback(
    (input: SendNetworkMessageInput, clientMessageId: string) => {
      discardOptimistic(queryClient, input, clientMessageId);
    },
    [queryClient]
  );

  return useMemo(
    () => ({
      send,
      retry,
      discard,
      isSending: mutation.isPending,
    }),
    [send, retry, discard, mutation.isPending]
  );
}

export interface CreateNetworkThreadInput {
  channel: string;
  sessionId: string;
  text: string;
  peerFrom: string;
  displayName?: string;
}

export interface CreateNetworkThreadResult {
  threadId: string;
  rootMessageId: string;
}

export interface UseCreateNetworkThreadResult {
  createThread: (input: CreateNetworkThreadInput) => Promise<CreateNetworkThreadResult>;
  isCreating: boolean;
}

interface CreateThreadAttemptArgs extends CreateNetworkThreadInput {
  threadId: string;
}

async function attemptCreateThread(
  queryClient: QueryClient,
  args: CreateThreadAttemptArgs
): Promise<CreateNetworkThreadResult> {
  const clientMessageId = generateClientMessageId();
  const timestamp = new Date().toISOString();
  const input: SendNetworkMessageThreadInput = {
    surface: "thread",
    channel: args.channel,
    threadId: args.threadId,
    sessionId: args.sessionId,
    text: args.text,
    peerFrom: args.peerFrom,
    displayName: args.displayName,
  };

  // Seed the per-thread message cache so when we navigate to the new
  // route the optimistic root message is already visible.
  applyOptimistic(queryClient, input, buildOptimisticMessage(input, clientMessageId, timestamp));

  try {
    const request = buildSendRequest(input, clientMessageId);
    await sendNetworkMessage(request);
    replaceOptimisticOnSuccess(queryClient, input, clientMessageId, timestamp);
    return { threadId: args.threadId, rootMessageId: clientMessageId };
  } catch (error) {
    discardOptimistic(queryClient, input, clientMessageId);
    throw error;
  }
}

export function useCreateNetworkThread(): UseCreateNetworkThreadResult {
  const queryClient = useQueryClient();

  const mutation = useMutation<CreateNetworkThreadResult, Error, CreateNetworkThreadInput>({
    mutationFn: async input => {
      const firstAttemptThreadId = `thread_${generateClientMessageId().replace(/-/g, "")}`;
      try {
        return await attemptCreateThread(queryClient, { ...input, threadId: firstAttemptThreadId });
      } catch (firstError) {
        // _techspec.md:1127 — exactly one silent retry, then surface the toast.
        const secondAttemptThreadId = `thread_${generateClientMessageId().replace(/-/g, "")}`;
        try {
          return await attemptCreateThread(queryClient, {
            ...input,
            threadId: secondAttemptThreadId,
          });
        } catch (secondError) {
          toast.error(THREAD_COLLISION_TOAST);
          throw secondError instanceof Error ? secondError : firstError;
        }
      }
    },
    onSettled: (_data, _error, variables) => {
      void queryClient.invalidateQueries({ queryKey: networkKeys.threadsList(variables.channel) });
      void queryClient.invalidateQueries({
        queryKey: networkKeys.channelDetail(variables.channel),
      });
    },
  });

  return useMemo(
    () => ({
      createThread: (input: CreateNetworkThreadInput) => mutation.mutateAsync(input),
      isCreating: mutation.isPending,
    }),
    [mutation]
  );
}

export interface ResolveNetworkDirectRoomInput {
  channel: string;
  body: NetworkResolveDirectRoomRequest;
}

export interface UseResolveNetworkDirectRoomResult {
  resolveRoom: (
    input: ResolveNetworkDirectRoomInput
  ) => Promise<NetworkResolveDirectRoomResponse["direct"]>;
  isResolving: boolean;
  error: Error | null;
}

export function useResolveNetworkDirectRoom(): UseResolveNetworkDirectRoomResult {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (input: ResolveNetworkDirectRoomInput) => {
      return resolveNetworkDirectRoom(input.channel, input.body);
    },
    onSettled: (_data, _error, variables) => {
      void queryClient.invalidateQueries({ queryKey: networkKeys.directsList(variables.channel) });
    },
  });

  const resolveRoom = useCallback(
    (input: ResolveNetworkDirectRoomInput) => mutation.mutateAsync(input),
    [mutation]
  );

  return useMemo(
    () => ({
      resolveRoom,
      isResolving: mutation.isPending,
      error: mutation.error ?? null,
    }),
    [resolveRoom, mutation.isPending, mutation.error]
  );
}

export function useCreateNetworkChannel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateNetworkChannelRequest) => createNetworkChannel(data),
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: networkKeys.all }),
        queryClient.invalidateQueries({ queryKey: sessionKeys.lists() }),
      ]),
  });
}

export { isOptimisticMessage, NetworkApiError, THREAD_COLLISION_TOAST };
export type { NetworkSurface };
