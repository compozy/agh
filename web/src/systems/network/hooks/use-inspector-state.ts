import { useCallback, useEffect, useState } from "react";

export type InspectorTab = "members" | "work" | "activity";

interface InspectorChannelState {
  open: boolean;
  tab: InspectorTab;
}

const STORAGE_KEY = "network:inspector-state";
const DEFAULT_STATE: InspectorChannelState = { open: false, tab: "members" };

type Store = Record<string, InspectorChannelState>;

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
      if (value === null || typeof value !== "object") {
        continue;
      }
      const { open, tab } = value as { open?: unknown; tab?: unknown };
      if (typeof open !== "boolean") {
        continue;
      }
      const safeTab: InspectorTab =
        tab === "members" || tab === "work" || tab === "activity" ? tab : DEFAULT_STATE.tab;
      out[key] = { open, tab: safeTab };
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

export interface UseInspectorStateResult {
  open: boolean;
  tab: InspectorTab;
  toggle: () => void;
  close: () => void;
  openWith: (tab: InspectorTab) => void;
  setTab: (tab: InspectorTab) => void;
}

/**
 * Per-channel inspector state — open/closed + active sub-tab — persisted in
 * localStorage per `_design.md` §5.8.3 + the user's "Inspector default closed,
 * remembered per channel" decision.
 */
export function useInspectorState(channel: string | null | undefined): UseInspectorStateResult {
  const key = channel ?? "";
  const [state, setState] = useState<InspectorChannelState>(() => {
    if (!key) {
      return DEFAULT_STATE;
    }
    return readStore()[key] ?? DEFAULT_STATE;
  });

  useEffect(() => {
    if (!key) {
      setState(DEFAULT_STATE);
      return;
    }
    setState(readStore()[key] ?? DEFAULT_STATE);
  }, [key]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return undefined;
    }
    function handleStorage(event: StorageEvent) {
      if (event.key !== STORAGE_KEY) {
        return;
      }
      if (!key) {
        setState(DEFAULT_STATE);
        return;
      }
      setState(readStore()[key] ?? DEFAULT_STATE);
    }
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [key]);

  const persist = useCallback(
    (next: InspectorChannelState) => {
      setState(next);
      if (!key) {
        return;
      }
      const store = readStore();
      store[key] = next;
      writeStore(store);
    },
    [key]
  );

  const toggle = useCallback(() => {
    persist({ open: !state.open, tab: state.tab });
  }, [persist, state.open, state.tab]);

  const close = useCallback(() => {
    persist({ open: false, tab: state.tab });
  }, [persist, state.tab]);

  const openWith = useCallback(
    (tab: InspectorTab) => {
      persist({ open: true, tab });
    },
    [persist]
  );

  const setTab = useCallback(
    (tab: InspectorTab) => {
      persist({ open: state.open, tab });
    },
    [persist, state.open]
  );

  return { open: state.open, tab: state.tab, toggle, close, openWith, setTab };
}
