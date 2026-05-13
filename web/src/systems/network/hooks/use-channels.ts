import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useActiveWorkspace } from "@/systems/workspace";

import { networkChannelsOptions } from "../lib/query-options";
import type { NetworkChannelSummary } from "../types";

const PINNED_CHANNELS_STORAGE_KEY = "network:pinned-channels";
type PinnedChannelsState = Record<string, string[]>;

function readPinnedChannelsState(): PinnedChannelsState {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(PINNED_CHANNELS_STORAGE_KEY) ?? "{}");
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return {};
    }
    const state: PinnedChannelsState = {};
    for (const [workspaceId, channels] of Object.entries(parsed)) {
      if (typeof workspaceId !== "string" || !Array.isArray(channels)) {
        continue;
      }
      const clean = channels.filter(item => typeof item === "string" && item.trim() !== "");
      if (clean.length > 0) {
        state[workspaceId] = clean;
      }
    }
    return state;
  } catch {
    return {};
  }
}

function readPinnedChannels(workspaceId: string | null | undefined): string[] {
  if (!workspaceId) {
    return [];
  }
  return readPinnedChannelsState()[workspaceId] ?? [];
}

function writePinnedChannels(workspaceId: string, values: string[]): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    const state = readPinnedChannelsState();
    const next = { ...state };
    if (values.length === 0) {
      delete next[workspaceId];
    } else {
      next[workspaceId] = values;
    }
    window.localStorage.setItem(PINNED_CHANNELS_STORAGE_KEY, JSON.stringify(next));
  } catch {
    // localStorage is best-effort; stay quiet on quota or privacy mode failures.
  }
}

function compareChannelsAlphabetical(left: NetworkChannelSummary, right: NetworkChannelSummary) {
  return left.channel.localeCompare(right.channel);
}

export interface UseNetworkChannelsResult {
  channels: NetworkChannelSummary[];
  pinned: NetworkChannelSummary[];
  unpinned: NetworkChannelSummary[];
  pinnedIds: ReadonlyArray<string>;
  isPinned: (channel: string) => boolean;
  togglePinned: (channel: string) => void;
  isLoading: boolean;
  isError: boolean;
  error: Error | null;
}

export function useNetworkChannels(options?: { enabled?: boolean }): UseNetworkChannelsResult {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled = (options?.enabled ?? true) && activeWorkspaceId != null;
  const query = useQuery(networkChannelsOptions(workspaceId, enabled));
  const [pinnedIds, setPinnedIds] = useState<string[]>(() => readPinnedChannels(activeWorkspaceId));

  useEffect(() => {
    setPinnedIds(readPinnedChannels(activeWorkspaceId));
  }, [activeWorkspaceId]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }

    function handleStorage(event: StorageEvent) {
      if (event.key === PINNED_CHANNELS_STORAGE_KEY) {
        setPinnedIds(readPinnedChannels(activeWorkspaceId));
      }
    }

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [activeWorkspaceId]);

  const togglePinned = useCallback(
    (channel: string) => {
      if (!activeWorkspaceId) {
        return;
      }
      setPinnedIds(current => {
        const next = current.includes(channel)
          ? current.filter(value => value !== channel)
          : [channel, ...current];
        writePinnedChannels(activeWorkspaceId, next);
        return next;
      });
    },
    [activeWorkspaceId]
  );

  const isPinned = useCallback((channel: string) => pinnedIds.includes(channel), [pinnedIds]);

  const channels = useMemo(() => {
    const list = query.data?.channels ?? [];
    return [...list].sort(compareChannelsAlphabetical);
  }, [query.data]);

  const pinned = useMemo(
    () => channels.filter(channel => pinnedIds.includes(channel.channel)),
    [channels, pinnedIds]
  );
  const unpinned = useMemo(
    () => channels.filter(channel => !pinnedIds.includes(channel.channel)),
    [channels, pinnedIds]
  );

  return {
    channels,
    pinned,
    unpinned,
    pinnedIds,
    isPinned,
    togglePinned,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error ?? null,
  };
}

export const PINNED_CHANNELS_STORAGE_KEY_FOR_TESTS = PINNED_CHANNELS_STORAGE_KEY;
