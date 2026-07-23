import { useEffect, useState } from "react";

const STORAGE_KEY = "session:inspector-open";
const DEFAULT_OPEN = false;

type Store = Record<string, boolean>;

function readStore(): Store {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (parsed === null || typeof parsed !== "object") {
      return {};
    }
    const out: Store = {};
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof value === "boolean") {
        out[key] = value;
      }
    }
    return out;
  } catch {
    return {};
  }
}

function writeStore(store: Store): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(store));
  } catch {
    // localStorage best-effort; ignore quota / privacy mode failures.
  }
}

export interface UseSessionInspectorStateResult {
  open: boolean;
  toggle: () => void;
  close: () => void;
  setOpen: (open: boolean) => void;
}

/**
 * Per-session inspector open/closed preference — default closed, remembered in
 * localStorage so reopening a session restores the last rail visibility.
 */
export function useSessionInspectorState(
  sessionId: string | null | undefined
): UseSessionInspectorStateResult {
  const key = sessionId ?? "";
  const [storedState, setStoredState] = useState<{ key: string; open: boolean }>(() => ({
    key,
    open: key ? (readStore()[key] ?? DEFAULT_OPEN) : DEFAULT_OPEN,
  }));
  const open =
    storedState.key === key
      ? storedState.open
      : key
        ? (readStore()[key] ?? DEFAULT_OPEN)
        : DEFAULT_OPEN;

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }
    function handleStorage(event: StorageEvent) {
      if (event.key !== STORAGE_KEY) {
        return;
      }
      if (!key) {
        setStoredState({ key, open: DEFAULT_OPEN });
        return;
      }
      setStoredState({ key, open: readStore()[key] ?? DEFAULT_OPEN });
    }
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [key]);

  const persist = (next: boolean) => {
    setStoredState({ key, open: next });
    if (!key) return;
    const store = readStore();
    store[key] = next;
    writeStore(store);
  };

  return {
    open,
    toggle: () => {
      persist(!open);
    },
    close: () => {
      persist(false);
    },
    setOpen: (next: boolean) => {
      persist(next);
    },
  };
}
