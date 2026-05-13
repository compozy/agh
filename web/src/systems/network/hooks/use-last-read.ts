import { useCallback, useEffect, useState } from "react";

import { useActiveWorkspace } from "@/systems/workspace";

import type { NetworkSurface } from "../types";

const LAST_READ_STORAGE_KEY = "network:last-read";
const KEY_SEPARATOR = ":";

export interface NetworkLastReadKey {
  workspaceId: string;
  channel: string;
  surface: NetworkSurface;
  containerId: string;
}

export type NetworkLastReadLookupKey = Omit<NetworkLastReadKey, "workspaceId">;

type LastReadState = Record<string, string>;

function readLastReadMap(): LastReadState {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(LAST_READ_STORAGE_KEY) ?? "{}");
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return {};
    }

    const map: LastReadState = {};
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === "string") {
        map[key] = value;
      }
    }
    return map;
  } catch {
    return {};
  }
}

function writeLastReadMap(state: LastReadState) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(LAST_READ_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // best-effort persistence
  }
}

export function buildLastReadStorageKey(key: NetworkLastReadKey): string {
  return [key.workspaceId, key.channel, key.surface, key.containerId].join(KEY_SEPARATOR);
}

export interface UseLastReadResult {
  lastReadAt(key: NetworkLastReadLookupKey): string | null;
  markRead(key: NetworkLastReadLookupKey, timestamp: string | null | undefined): void;
}

export function useLastRead(): UseLastReadResult {
  const { activeWorkspaceId } = useActiveWorkspace();
  const [state, setState] = useState<LastReadState>(() => readLastReadMap());

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }

    function handleStorage(event: StorageEvent) {
      if (event.key === LAST_READ_STORAGE_KEY) {
        setState(readLastReadMap());
      }
    }

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  const lastReadAt = useCallback(
    (key: NetworkLastReadLookupKey) => {
      if (!activeWorkspaceId) {
        return null;
      }
      return state[buildLastReadStorageKey({ ...key, workspaceId: activeWorkspaceId })] ?? null;
    },
    [activeWorkspaceId, state]
  );

  const markRead = useCallback(
    (key: NetworkLastReadLookupKey, timestamp: string | null | undefined) => {
      if (!timestamp || !activeWorkspaceId) {
        return;
      }
      const storageKey = buildLastReadStorageKey({ ...key, workspaceId: activeWorkspaceId });
      setState(current => {
        if (current[storageKey] === timestamp) {
          return current;
        }
        const next = { ...current, [storageKey]: timestamp };
        writeLastReadMap(next);
        return next;
      });
    },
    [activeWorkspaceId]
  );

  return { lastReadAt, markRead };
}

export const LAST_READ_STORAGE_KEY_FOR_TESTS = LAST_READ_STORAGE_KEY;
