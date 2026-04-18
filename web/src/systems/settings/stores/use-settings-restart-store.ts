import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import {
  createSettingsRestartStore,
  initialSettingsRestartState,
  type SettingsRestartState,
  type SettingsRestartStore,
} from "./settings-restart-store";

const settingsRestartStorageKey = "agh:settings:restart";

type PersistedSettingsRestartState = Pick<
  SettingsRestartState,
  "operationId" | "status" | "activeSessionCount" | "failureReason" | "lastMutation"
>;

const settingsRestartStorage = createJSONStorage<PersistedSettingsRestartState>(() => {
  if (typeof window === "undefined") {
    throw new Error("sessionStorage is unavailable");
  }

  return window.sessionStorage;
});

function resetStateSnapshot(): SettingsRestartStore {
  const { startRestart, updateRestart, clearRestart, recordMutation } =
    useSettingsRestartStore.getState();

  return {
    ...initialSettingsRestartState,
    startRestart,
    updateRestart,
    clearRestart,
    recordMutation,
  };
}

export const useSettingsRestartStore = create<SettingsRestartStore>()(
  persist(createSettingsRestartStore, {
    name: settingsRestartStorageKey,
    storage: settingsRestartStorage,
    partialize: state => ({
      operationId: state.operationId,
      status: state.status,
      activeSessionCount: state.activeSessionCount,
      failureReason: state.failureReason,
      lastMutation: state.lastMutation,
    }),
  })
);

export function resetSettingsRestartStore() {
  useSettingsRestartStore.setState(resetStateSnapshot());
  useSettingsRestartStore.persist.clearStorage();
}

export { settingsRestartStorageKey };
