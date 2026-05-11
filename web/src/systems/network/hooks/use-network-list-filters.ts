import { useCallback, useEffect, useMemo, useState } from "react";

import { createFilter, type Filter as ReuiFilter } from "@agh/ui/components/reui/filters";

import { useActiveNetworkSession } from "./use-active-session";
import { useLastRead, type NetworkLastReadKey } from "./use-last-read";
import type { NetworkDirectRoomSummary, NetworkSurface, NetworkThreadSummary } from "../types";

const PINNED_STORAGE_KEY = "network:pinned-items";

export type NetworkListSort = "recent_activity" | "created" | "alphabetical";

export type NetworkFilterKey = "has_work" | "mentions_me" | "pinned" | "unread";

export const NETWORK_FILTER_KEYS = [
  "has_work",
  "mentions_me",
  "pinned",
  "unread",
] as const satisfies ReadonlyArray<NetworkFilterKey>;

export interface NetworkListFilterCounts {
  all: number;
  hasWork: number;
  me: number;
  pinned: number;
  unread: number;
}

export type NetworkChipFilter = ReuiFilter<boolean>;

type PinnedStore = Record<string, true>;

function pinnedKey(channel: string, surface: NetworkSurface, id: string): string {
  return `${channel}:${surface}:${id}`;
}

function readPinned(): PinnedStore {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(PINNED_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    const out: PinnedStore = {};
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (value === true) {
        out[key] = true;
      }
    }
    return out;
  } catch {
    return {};
  }
}

function writePinned(store: PinnedStore): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(PINNED_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // best-effort
  }
}

function isKnownChipKey(field: string): field is NetworkFilterKey {
  return (NETWORK_FILTER_KEYS as ReadonlyArray<string>).includes(field);
}

function chipKeySet(filters: ReadonlyArray<NetworkChipFilter>): Set<NetworkFilterKey> {
  const out = new Set<NetworkFilterKey>();
  for (const filter of filters) {
    if (isKnownChipKey(filter.field)) {
      out.add(filter.field);
    }
  }
  return out;
}

export function createNetworkChipFilter(key: NetworkFilterKey): NetworkChipFilter {
  return createFilter<boolean>(key, "is", [true]);
}

export interface UseNetworkListFiltersArgs {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
}

export interface UseNetworkListFiltersResult {
  filters: NetworkChipFilter[];
  sort: NetworkListSort;
  counts: NetworkListFilterCounts;
  setFilters: (next: NetworkChipFilter[]) => void;
  setSort: (next: NetworkListSort) => void;
  filteredThreads: NetworkThreadSummary[];
  filteredDirects: NetworkDirectRoomSummary[];
  pin: (surface: NetworkSurface, id: string) => void;
  unpin: (surface: NetworkSurface, id: string) => void;
  isPinned: (surface: NetworkSurface, id: string) => boolean;
  markAllRead: () => void;
  isMarkAllReadDisabled: boolean;
}

function compareTimestampDesc(a: string | null | undefined, b: string | null | undefined): number {
  const left = a ? new Date(a).getTime() : 0;
  const right = b ? new Date(b).getTime() : 0;
  return right - left;
}

function compareTimestampAsc(a: string | null | undefined, b: string | null | undefined): number {
  return -compareTimestampDesc(a, b);
}

function applyThreadSort(
  threads: ReadonlyArray<NetworkThreadSummary>,
  sort: NetworkListSort
): NetworkThreadSummary[] {
  const copy = [...threads];
  if (sort === "alphabetical") {
    copy.sort((left, right) => (left.title ?? "").localeCompare(right.title ?? ""));
  } else if (sort === "created") {
    copy.sort((left, right) => compareTimestampAsc(left.opened_at, right.opened_at));
  } else {
    copy.sort((left, right) => compareTimestampDesc(left.last_activity_at, right.last_activity_at));
  }
  return copy;
}

function applyDirectSort(
  directs: ReadonlyArray<NetworkDirectRoomSummary>,
  sort: NetworkListSort
): NetworkDirectRoomSummary[] {
  const copy = [...directs];
  if (sort === "alphabetical") {
    copy.sort((left, right) =>
      `${left.peer_a}↔${left.peer_b}`.localeCompare(`${right.peer_a}↔${right.peer_b}`)
    );
  } else if (sort === "created") {
    copy.sort((left, right) => compareTimestampAsc(left.opened_at, right.opened_at));
  } else {
    copy.sort((left, right) => compareTimestampDesc(left.last_activity_at, right.last_activity_at));
  }
  return copy;
}

/**
 * Per-channel filter / sort / mark-all-read state for the network list views.
 *
 * Filters compose: each `NetworkChipFilter` in `filters` represents an active
 * boolean toggle (`has_work`, `mentions_me`, `pinned`, `unread`). An empty
 * `filters` array means no filter — all rows pass. Filters and sort run
 * client-side over the loaded list; server-side push down is documented as a
 * follow-up TechSpec so the response envelope stays unchanged for now.
 */
export function useNetworkListFilters({
  channel,
  threads,
  directs,
}: UseNetworkListFiltersArgs): UseNetworkListFiltersResult {
  const [filters, setFilters] = useState<NetworkChipFilter[]>([]);
  const [sort, setSort] = useState<NetworkListSort>("recent_activity");
  const [pinnedStore, setPinnedStore] = useState<PinnedStore>(() => readPinned());
  const lastRead = useLastRead();
  const session = useActiveNetworkSession(channel);
  const selfPeerId = session.session?.peerId ?? null;

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }
    function handleStorage(event: StorageEvent) {
      if (event.key === PINNED_STORAGE_KEY) {
        setPinnedStore(readPinned());
      }
    }
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  const isPinned = useCallback(
    (surface: NetworkSurface, id: string) => pinnedStore[pinnedKey(channel, surface, id)] === true,
    [channel, pinnedStore]
  );

  const togglePinned = useCallback(
    (surface: NetworkSurface, id: string, on: boolean) => {
      setPinnedStore(current => {
        const key = pinnedKey(channel, surface, id);
        const isCurrentlyOn = current[key] === true;
        if (isCurrentlyOn === on) {
          return current;
        }
        const next = { ...current };
        if (on) {
          next[key] = true;
        } else {
          delete next[key];
        }
        writePinned(next);
        return next;
      });
    },
    [channel]
  );

  const pin = useCallback(
    (surface: NetworkSurface, id: string) => togglePinned(surface, id, true),
    [togglePinned]
  );
  const unpin = useCallback(
    (surface: NetworkSurface, id: string) => togglePinned(surface, id, false),
    [togglePinned]
  );

  const isThreadUnread = useCallback(
    (thread: NetworkThreadSummary): boolean => {
      const last = lastRead.lastReadAt({
        channel,
        surface: "thread",
        containerId: thread.thread_id,
      });
      if (!thread.last_activity_at) {
        return false;
      }
      if (!last) {
        return true;
      }
      return new Date(thread.last_activity_at).getTime() > new Date(last).getTime();
    },
    [channel, lastRead]
  );

  const isDirectUnread = useCallback(
    (direct: NetworkDirectRoomSummary): boolean => {
      const last = lastRead.lastReadAt({
        channel,
        surface: "direct",
        containerId: direct.direct_id,
      });
      if (!direct.last_activity_at) {
        return false;
      }
      if (!last) {
        return true;
      }
      return new Date(direct.last_activity_at).getTime() > new Date(last).getTime();
    },
    [channel, lastRead]
  );

  const isThreadMine = useCallback(
    (thread: NetworkThreadSummary): boolean => {
      if (!selfPeerId) {
        return false;
      }
      return thread.opened_by_peer_id === selfPeerId;
    },
    [selfPeerId]
  );

  const isDirectMine = useCallback(
    (direct: NetworkDirectRoomSummary): boolean => {
      if (!selfPeerId) {
        return false;
      }
      return direct.peer_a === selfPeerId || direct.peer_b === selfPeerId;
    },
    [selfPeerId]
  );

  const counts = useMemo<NetworkListFilterCounts>(() => {
    const all = threads.length + directs.length;
    let hasWork = 0;
    let me = 0;
    let pinned = 0;
    let unread = 0;
    for (const thread of threads) {
      if (thread.open_work_count > 0) {
        hasWork += 1;
      }
      if (isThreadMine(thread)) {
        me += 1;
      }
      if (isPinned("thread", thread.thread_id)) {
        pinned += 1;
      }
      if (isThreadUnread(thread)) {
        unread += 1;
      }
    }
    for (const direct of directs) {
      if (direct.open_work_count > 0) {
        hasWork += 1;
      }
      if (isDirectMine(direct)) {
        me += 1;
      }
      if (isPinned("direct", direct.direct_id)) {
        pinned += 1;
      }
      if (isDirectUnread(direct)) {
        unread += 1;
      }
    }
    return { all, hasWork, me, pinned, unread };
  }, [threads, directs, isThreadMine, isDirectMine, isPinned, isThreadUnread, isDirectUnread]);

  const filteredThreads = useMemo(() => {
    const active = chipKeySet(filters);
    let scope: NetworkThreadSummary[] = threads.filter(thread => {
      if (active.size === 0) return true;
      if (active.has("has_work") && thread.open_work_count <= 0) return false;
      if (active.has("mentions_me") && !isThreadMine(thread)) return false;
      if (active.has("pinned") && !isPinned("thread", thread.thread_id)) return false;
      if (active.has("unread") && !isThreadUnread(thread)) return false;
      return true;
    });
    scope = applyThreadSort(scope, sort);
    return scope;
  }, [threads, filters, sort, isThreadMine, isPinned, isThreadUnread]);

  const filteredDirects = useMemo(() => {
    const active = chipKeySet(filters);
    let scope: NetworkDirectRoomSummary[] = directs.filter(direct => {
      if (active.size === 0) return true;
      if (active.has("has_work") && direct.open_work_count <= 0) return false;
      if (active.has("mentions_me") && !isDirectMine(direct)) return false;
      if (active.has("pinned") && !isPinned("direct", direct.direct_id)) return false;
      if (active.has("unread") && !isDirectUnread(direct)) return false;
      return true;
    });
    scope = applyDirectSort(scope, sort);
    return scope;
  }, [directs, filters, sort, isDirectMine, isPinned, isDirectUnread]);

  const markAllRead = useCallback(() => {
    for (const thread of filteredThreads) {
      if (!thread.last_activity_at) {
        continue;
      }
      const key: NetworkLastReadKey = {
        channel,
        surface: "thread",
        containerId: thread.thread_id,
      };
      lastRead.markRead(key, thread.last_activity_at);
    }
    for (const direct of filteredDirects) {
      if (!direct.last_activity_at) {
        continue;
      }
      const key: NetworkLastReadKey = {
        channel,
        surface: "direct",
        containerId: direct.direct_id,
      };
      lastRead.markRead(key, direct.last_activity_at);
    }
  }, [channel, filteredThreads, filteredDirects, lastRead]);

  const isMarkAllReadDisabled = counts.unread === 0;

  return {
    filters,
    sort,
    counts,
    setFilters,
    setSort,
    filteredThreads,
    filteredDirects,
    pin,
    unpin,
    isPinned,
    markAllRead,
    isMarkAllReadDisabled,
  };
}
