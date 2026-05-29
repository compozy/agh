import { create } from "zustand";
import { persist } from "zustand/middleware";
import {
  createActiveWorkspaceStore,
  type ActiveWorkspaceStore,
} from "../stores/active-workspace-store";

export const useActiveWorkspaceStore = create<ActiveWorkspaceStore>()(
  persist(createActiveWorkspaceStore, {
    name: "agh:active-workspace",
    partialize: state => ({ selectedWorkspaceId: state.selectedWorkspaceId }),
  })
);
