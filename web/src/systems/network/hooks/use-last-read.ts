import { useCallback, useEffect, useState } from "react";

import type { NetworkSurface } from "../types";

const LAST_READ_STORAGE_KEY = "network:last-read";
const KEY_SEPARATOR = ":";

export interface NetworkLastReadKey {
  channel: string;
  surface: NetworkSurface;
  containerId: string;
}

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
  return [key.channel, key.surface, key.containerId].join(KEY_SEPARATOR);
}

export interface UseLastReadResult {
  lastReadAt(key: NetworkLastReadKey): string | null;
  markRead(key: NetworkLastReadKey, timestamp: string | null | undefined): void;
}

export function useLastRead(): UseLastReadResult {
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
    (key: NetworkLastReadKey) => state[buildLastReadStorageKey(key)] ?? null,
    [state]
  );

  const markRead = useCallback((key: NetworkLastReadKey, timestamp: string | null | undefined) => {
    if (!timestamp) {
      return;
    }
    const storageKey = buildLastReadStorageKey(key);
    setState(current => {
      if (current[storageKey] === timestamp) {
        return current;
      }
      const next = { ...current, [storageKey]: timestamp };
      writeLastReadMap(next);
      return next;
    });
  }, []);

  return { lastReadAt, markRead };
}

export const LAST_READ_STORAGE_KEY_FOR_TESTS = LAST_READ_STORAGE_KEY;
