import type { StateCreator } from "zustand";

import type { SettingsMutationResult, SettingsRestartStatusName } from "../types";

export interface PendingSettingsMutation {
  section: SettingsMutationResult["section"];
  restartRequired: boolean;
  restartScope?: string;
  warnings: string[];
  completedAt: string;
}

export interface SettingsRestartState {
  operationId: string | null;
  status: SettingsRestartStatusName | null;
  activeSessionCount: number;
  failureReason?: string;
  lastMutation: PendingSettingsMutation | null;
}

export interface SettingsRestartActions {
  startRestart: (payload: {
    operationId: string;
    status: SettingsRestartStatusName;
    activeSessionCount: number;
  }) => void;
  updateRestart: (payload: {
    status: SettingsRestartStatusName;
    activeSessionCount?: number;
    failureReason?: string;
  }) => void;
  clearRestart: () => void;
  recordMutation: (payload: PendingSettingsMutation | null) => void;
}

export type SettingsRestartStore = SettingsRestartState & SettingsRestartActions;

export const initialSettingsRestartState: SettingsRestartState = {
  operationId: null,
  status: null,
  activeSessionCount: 0,
  failureReason: undefined,
  lastMutation: null,
};

export const createSettingsRestartStore: StateCreator<SettingsRestartStore> = set => ({
  ...initialSettingsRestartState,
  startRestart: ({ operationId, status, activeSessionCount }) =>
    set({
      operationId,
      status,
      activeSessionCount,
      failureReason: undefined,
    }),
  updateRestart: ({ status, activeSessionCount, failureReason }) =>
    set(state => ({
      status,
      activeSessionCount: activeSessionCount ?? state.activeSessionCount,
      failureReason,
    })),
  clearRestart: () =>
    set(state => ({
      ...initialSettingsRestartState,
      lastMutation: state.lastMutation,
    })),
  recordMutation: lastMutation => set({ lastMutation }),
});
