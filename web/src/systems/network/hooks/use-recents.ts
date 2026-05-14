import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";

import { useActiveWorkspace } from "@/systems/workspace";

import { networkDirectsOptions, networkThreadsOptions } from "../lib/query-options";
import type {
  NetworkChannelSummary,
  NetworkDirectRoomSummary,
  NetworkRecentEntry,
  NetworkSurface,
  NetworkThreadSummary,
} from "../types";
import { useLastRead } from "./use-last-read";

const RECENTS_LIMIT = 5;

function safeTimestamp(value?: string | null): number {
  if (!value) {
    return 0;
  }
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

function pickPreview(value: string | undefined | null, fallback: string): string {
  const trimmed = value?.trim() ?? "";
  return trimmed === "" ? fallback : trimmed;
}

function describeThreadParticipants(thread: NetworkThreadSummary): string {
  const count = thread.participant_count ?? 0;
  if (count <= 0) {
    return "open";
  }
  return count === 1 ? "1 peer" : `${count} peers`;
}

function describeDirectParticipants(direct: NetworkDirectRoomSummary): string {
  const peers = [direct.peer_a, direct.peer_b].filter(Boolean);
  if (peers.length === 0) {
    return "two-party";
  }
  return peers.join(" + ");
}

function toThreadEntry(
  channel: string,
  thread: NetworkThreadSummary,
  hasUnread: boolean
): NetworkRecentEntry {
  return {
    surface: "thread" satisfies NetworkSurface,
    channel,
    containerId: thread.thread_id,
    preview: pickPreview(
      thread.last_message_preview,
      thread.title ? thread.title : "New public thread"
    ),
    lastActivityAt: thread.last_activity_at ?? thread.opened_at ?? null,
    hasUnread,
    participantLabel: describeThreadParticipants(thread),
  };
}

function toDirectEntry(
  channel: string,
  direct: NetworkDirectRoomSummary,
  hasUnread: boolean
): NetworkRecentEntry {
  return {
    surface: "direct" satisfies NetworkSurface,
    channel,
    containerId: direct.direct_id,
    preview: pickPreview(direct.last_message_preview, "Direct room"),
    lastActivityAt: direct.last_activity_at ?? direct.opened_at ?? null,
    hasUnread,
    participantLabel: describeDirectParticipants(direct),
  };
}

export interface UseNetworkRecentsResult {
  recents: NetworkRecentEntry[];
  isLoading: boolean;
}

export function useNetworkRecents(
  channels: ReadonlyArray<NetworkChannelSummary>,
  options?: { enabled?: boolean; limit?: number }
): UseNetworkRecentsResult {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled = (options?.enabled ?? true) && activeWorkspaceId != null;
  const limit = options?.limit ?? RECENTS_LIMIT;
  const { lastReadAt } = useLastRead();

  const threadQueries = useQueries({
    queries: channels.map(channel => ({
      ...networkThreadsOptions(workspaceId, channel.channel, { limit }, enabled),
    })),
  });
  const directQueries = useQueries({
    queries: channels.map(channel => ({
      ...networkDirectsOptions(workspaceId, channel.channel, { limit }, enabled),
    })),
  });

  const recents = useMemo(() => {
    if (!enabled) {
      return [];
    }

    const merged: NetworkRecentEntry[] = [];

    channels.forEach((channel, index) => {
      const threadResult = threadQueries[index];
      const directResult = directQueries[index];

      const threads = threadResult?.data ?? [];
      for (const thread of threads) {
        const hasUnread =
          safeTimestamp(thread.last_activity_at) >
          safeTimestamp(
            lastReadAt({
              channel: channel.channel,
              surface: "thread",
              containerId: thread.thread_id,
            })
          );
        merged.push(toThreadEntry(channel.channel, thread, hasUnread));
      }

      const directs = directResult?.data ?? [];
      for (const direct of directs) {
        const hasUnread =
          safeTimestamp(direct.last_activity_at) >
          safeTimestamp(
            lastReadAt({
              channel: channel.channel,
              surface: "direct",
              containerId: direct.direct_id,
            })
          );
        merged.push(toDirectEntry(channel.channel, direct, hasUnread));
      }
    });

    merged.sort(
      (left, right) => safeTimestamp(right.lastActivityAt) - safeTimestamp(left.lastActivityAt)
    );
    return merged.slice(0, limit);
  }, [channels, directQueries, enabled, lastReadAt, limit, threadQueries]);

  const isLoading = useMemo(() => {
    if (!enabled || channels.length === 0) {
      return false;
    }
    return [...threadQueries, ...directQueries].some(query => query.isLoading);
  }, [channels.length, directQueries, enabled, threadQueries]);

  return { recents, isLoading };
}

export const RECENTS_LIMIT_FOR_TESTS = RECENTS_LIMIT;
