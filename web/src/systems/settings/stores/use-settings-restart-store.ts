import { create } from "zustand";

import { createSettingsRestartStore, type SettingsRestartStore } from "./settings-restart-store";

export const useSettingsRestartStore = create<SettingsRestartStore>(createSettingsRestartStore);
