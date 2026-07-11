import { useMemo } from "react";

import type { NetworkChannelsResponse, NetworkRecentEntry, NetworkSurface } from "../types";
import { useLastRead } from "./use-last-read";

export const RECENTS_LIMIT_FOR_TESTS = 5;
type EmbeddedRecent = NonNullable<NetworkChannelsResponse["recents"]>[number];

function safeTimestamp(value?: string | null): number {
  if (!value) return 0;
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? 0 : parsed;
}

function isNetworkSurface(value: string): value is NetworkSurface {
  return value === "thread" || value === "direct";
}

function participantLabel(recent: EmbeddedRecent): string {
  if (recent.surface === "direct") {
    const peers = [recent.peer_a, recent.peer_b].filter(Boolean);
    return peers.length > 0 ? peers.join(" + ") : "two-party";
  }
  const count = recent.participant_count ?? 0;
  if (count <= 0) return "open";
  return count === 1 ? "1 peer" : `${count} peers`;
}

function preview(recent: EmbeddedRecent): string {
  const text = recent.last_message_preview?.trim();
  if (text) return text;
  if (recent.surface === "thread" && recent.title?.trim()) return recent.title.trim();
  return recent.surface === "thread" ? "New public thread" : "Direct room";
}

export interface UseNetworkRecentsResult {
  recents: NetworkRecentEntry[];
  isLoading: boolean;
}

export function useNetworkRecents(
  embedded: ReadonlyArray<EmbeddedRecent>,
  options: {
    workspaceId?: string | null;
    enabled?: boolean;
    limit?: number;
    isLoading?: boolean;
  } = {}
): UseNetworkRecentsResult {
  const enabled = options.enabled ?? true;
  const limit = options.limit ?? RECENTS_LIMIT_FOR_TESTS;
  const { lastReadAt } = useLastRead({ workspaceId: options.workspaceId });
  const recents = useMemo(() => {
    if (!enabled) return [];
    return embedded
      .filter(recent => isNetworkSurface(recent.surface))
      .map(recent => {
        const surface = recent.surface as NetworkSurface;
        const lastActivityAt = recent.last_activity_at ?? null;
        return {
          surface,
          channel: recent.channel,
          containerId: recent.container_id,
          preview: preview(recent),
          lastActivityAt,
          hasUnread:
            safeTimestamp(lastActivityAt) >
            safeTimestamp(
              lastReadAt({ channel: recent.channel, surface, containerId: recent.container_id })
            ),
          participantLabel: participantLabel(recent),
        } satisfies NetworkRecentEntry;
      })
      .slice(0, limit);
  }, [embedded, enabled, lastReadAt, limit]);
  return { recents, isLoading: enabled && (options.isLoading ?? false) };
}
