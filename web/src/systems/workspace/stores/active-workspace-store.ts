import { create } from "zustand";

interface ActiveWorkspaceState {
  selectedWorkspaceId: string | null;
}

interface ActiveWorkspaceActions {
  setSelectedWorkspaceId: (workspaceId: string | null) => void;
  clearSelectedWorkspaceId: () => void;
}

type ActiveWorkspaceStore = ActiveWorkspaceState & ActiveWorkspaceActions;

export const useActiveWorkspaceStore = create<ActiveWorkspaceStore>(set => ({
  selectedWorkspaceId: null,

  setSelectedWorkspaceId: selectedWorkspaceId => set({ selectedWorkspaceId }),
  clearSelectedWorkspaceId: () => set({ selectedWorkspaceId: null }),
}));
