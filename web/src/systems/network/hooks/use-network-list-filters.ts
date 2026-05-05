import { useCallback, useEffect, useMemo, useState } from "react";

import { useActiveNetworkSession } from "./use-active-session";
import { useLastRead, type NetworkLastReadKey } from "./use-last-read";
import type { NetworkDirectRoomSummary, NetworkSurface, NetworkThreadSummary } from "../types";
import type {
  NetworkListFilter,
  NetworkListFilterCounts,
  NetworkListSort,
} from "../components/shell/list-filter-bar";

const PINNED_STORAGE_KEY = "network:pinned-items";

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

export interface UseNetworkListFiltersArgs {
  channel: string;
  threads: ReadonlyArray<NetworkThreadSummary>;
  directs: ReadonlyArray<NetworkDirectRoomSummary>;
}

export interface UseNetworkListFiltersResult {
  filter: NetworkListFilter;
  sort: NetworkListSort;
  counts: NetworkListFilterCounts;
  setFilter: (next: NetworkListFilter) => void;
  setSort: (next: NetworkListSort) => void;
  filteredThreads: NetworkThreadSummary[];
  filteredDirects: NetworkDirectRoomSummary[];
  pin: (surface: NetworkSurface, id: string) => void;
  unpin: (surface: NetworkSurface, id: string) => void;
  isPinned: (surface: NetworkSurface, id: string) => boolean;
  markAllRead: () => void;
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
 * Filters and sort run client-side over the loaded list — server-side push
 * down is documented as a follow-up TechSpec (see plan §C.1) so the response
 * envelope stays unchanged for now. `Pinned` and `Unread` are derived from
 * the existing client-only stores (`network:pinned-items`, `network:last-read`).
 */
export function useNetworkListFilters({
  channel,
  threads,
  directs,
}: UseNetworkListFiltersArgs): UseNetworkListFiltersResult {
  const [filter, setFilter] = useState<NetworkListFilter>("all");
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
    let scope: NetworkThreadSummary[] = threads.filter(thread => {
      switch (filter) {
        case "all":
          return true;
        case "has_work":
          return thread.open_work_count > 0;
        case "me":
          return isThreadMine(thread);
        case "pinned":
          return isPinned("thread", thread.thread_id);
        case "unread":
          return isThreadUnread(thread);
        default:
          return true;
      }
    });
    scope = applyThreadSort(scope, sort);
    return scope;
  }, [threads, filter, sort, isThreadMine, isPinned, isThreadUnread]);

  const filteredDirects = useMemo(() => {
    let scope: NetworkDirectRoomSummary[] = directs.filter(direct => {
      switch (filter) {
        case "all":
          return true;
        case "has_work":
          return direct.open_work_count > 0;
        case "me":
          return isDirectMine(direct);
        case "pinned":
          return isPinned("direct", direct.direct_id);
        case "unread":
          return isDirectUnread(direct);
        default:
          return true;
      }
    });
    scope = applyDirectSort(scope, sort);
    return scope;
  }, [directs, filter, sort, isDirectMine, isPinned, isDirectUnread]);

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

  return {
    filter,
    sort,
    counts,
    setFilter,
    setSort,
    filteredThreads,
    filteredDirects,
    pin,
    unpin,
    isPinned,
    markAllRead,
  };
}
