import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";

import { networkChannelsOptions } from "../lib/query-options";
import type { NetworkChannelSummary } from "../types";

const PINNED_CHANNELS_STORAGE_KEY = "network:pinned-channels";

function readPinnedChannels(): string[] {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(PINNED_CHANNELS_STORAGE_KEY) ?? "[]");
    return Array.isArray(parsed)
      ? parsed.filter(item => typeof item === "string" && item.trim() !== "")
      : [];
  } catch {
    return [];
  }
}

function writePinnedChannels(values: string[]): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(PINNED_CHANNELS_STORAGE_KEY, JSON.stringify(values));
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
  const query = useQuery(networkChannelsOptions(options?.enabled ?? true));
  const [pinnedIds, setPinnedIds] = useState<string[]>(() => readPinnedChannels());

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }

    function handleStorage(event: StorageEvent) {
      if (event.key === PINNED_CHANNELS_STORAGE_KEY) {
        setPinnedIds(readPinnedChannels());
      }
    }

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  const togglePinned = useCallback((channel: string) => {
    setPinnedIds(current => {
      const next = current.includes(channel)
        ? current.filter(value => value !== channel)
        : [channel, ...current];
      writePinnedChannels(next);
      return next;
    });
  }, []);

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
