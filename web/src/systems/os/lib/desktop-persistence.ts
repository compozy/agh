import type { components } from "@/generated/agh-openapi";

import type { DesktopStoreApi } from "../stores/desktop-store";
import type { OsStateClient, OsStateClientCallbacks } from "./os-state-client";
import { encodeDesktopPayload, encodeWindowPayload } from "./os-state-payloads";
import {
  OS_DESKTOP_KEY,
  osWindowKey,
  windowIdFromKey,
  type OsDesktopStore,
  type OsStateEntry,
} from "./os-types";

type ApplyOp = components["schemas"]["DesktopStateApplyOp"];

/**
 * Bridges the WM store and the desktop-state client in both directions.
 *
 * Local → daemon: subscribes to the store, diffs transitions per key, and
 * classifies them — rect-only churn debounces (invariant 15); anything else
 * (open/close/focus/zoom/minimize/rail/location) ships as ONE atomic apply
 * batch (invariant 4/B-003). Remote → local transitions run muted so they
 * never echo back as writes.
 */
export function createDesktopPersistence(store: DesktopStoreApi): {
  callbacks: OsStateClientCallbacks;
  bind(client: OsStateClient): () => void;
} {
  let muted = false;

  const mutedRun = (fn: () => void) => {
    muted = true;
    try {
      fn();
    } finally {
      muted = false;
    }
  };

  const readLocal = (key: string): Record<string, unknown> | null => {
    const state = store.getState();
    if (key === OS_DESKTOP_KEY) {
      return encodeDesktopPayload(state);
    }
    const windowId = windowIdFromKey(key);
    if (windowId === null) return null;
    const win = state.windows[windowId];
    return win ? encodeWindowPayload(win) : null;
  };

  const callbacks: OsStateClientCallbacks = {
    onSnapshot: (entries, _asOfSeq, preserveKeys) => {
      mutedRun(() => {
        const adopted = entries.filter(entry => !preserveKeys.has(entry.key));
        const synthesized: OsStateEntry[] = [];
        for (const key of preserveKeys) {
          const value = readLocal(key);
          if (value === null) continue;
          synthesized.push({
            key,
            value,
            rev: 0,
            seq: 0,
            deleted: false,
            updated_at: new Date(0).toISOString(),
          });
        }
        store.getState().hydrate([...adopted, ...synthesized]);
      });
    },
    onEvent: event => {
      mutedRun(() => store.getState().applyRemote(event));
    },
    onHydration: hydration => {
      mutedRun(() => store.getState().setHydration(hydration));
    },
    onSettled: (key, seq) => {
      mutedRun(() => store.getState().markSettled(key, seq));
    },
    readLocal,
  };

  const bind = (client: OsStateClient): (() => void) => {
    return store.subscribe((state, prev) => {
      if (muted) return;
      const diff = diffTransition(state, prev);
      if (diff === null) return;
      if (diff.rectOnlyKeys) {
        for (const key of diff.rectOnlyKeys) client.schedulePut(key);
        return;
      }
      client.applyNow(diff.batch);
    });
  };

  const diffTransition = (
    state: OsDesktopStore,
    prev: OsDesktopStore
  ): { rectOnlyKeys?: string[]; batch: ApplyOp[] } | null => {
    const batch: ApplyOp[] = [];
    const changedKeys: string[] = [];
    let rectOnly = true;

    for (const id of Object.keys(prev.windows)) {
      if (!state.windows[id]) {
        batch.push({ kind: "delete", key: osWindowKey(id) });
        rectOnly = false;
      }
    }
    for (const [id, win] of Object.entries(state.windows)) {
      const prevWin = prev.windows[id];
      if (prevWin === win) continue;
      const key = osWindowKey(id);
      changedKeys.push(key);
      batch.push({ kind: "put", key, value: encodeWindowPayload(win) });
      if (!prevWin || !isRectOnlyChange(win, prevWin)) rectOnly = false;
    }
    const desktopChanged =
      state.focusedId !== prev.focusedId ||
      state.railCollapsedAgentIds !== prev.railCollapsedAgentIds ||
      state.wallpaper !== prev.wallpaper ||
      state.reduceMotion !== prev.reduceMotion ||
      state.dockMagnify !== prev.dockMagnify;
    if (desktopChanged) {
      batch.push({ kind: "put", key: OS_DESKTOP_KEY, value: encodeDesktopPayload(state) });
      rectOnly = false;
    }
    if (batch.length === 0) return null;
    if (rectOnly) return { rectOnlyKeys: changedKeys, batch: [] };
    return { batch };
  };

  return { callbacks, bind };
}

function isRectOnlyChange(
  win: OsDesktopStore["windows"][string],
  prev: OsDesktopStore["windows"][string]
): boolean {
  return (
    win.rect !== prev.rect &&
    win.z === prev.z &&
    win.minimized === prev.minimized &&
    win.maximized === prev.maximized &&
    win.location === prev.location &&
    win.prevRect === prev.prevRect &&
    win.snap === prev.snap
  );
}
