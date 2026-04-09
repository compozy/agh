import { create } from "zustand";

export interface SidebarState {
  collapsed: boolean;
}

export interface SidebarActions {
  toggle: () => void;
  setCollapsed: (collapsed: boolean) => void;
}

export type SidebarStore = SidebarState & SidebarActions;

export const useSidebarStore = create<SidebarStore>(set => ({
  collapsed: false,

  toggle: () => set(state => ({ collapsed: !state.collapsed })),
  setCollapsed: (collapsed: boolean) => set({ collapsed }),
}));
