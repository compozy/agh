import { useEffect, useState } from "react";
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

export function useActiveWorkspaceStoreHasHydrated(): boolean {
  const [hasHydrated, setHasHydrated] = useState(() =>
    useActiveWorkspaceStore.persist.hasHydrated()
  );

  useEffect(() => {
    const unsubscribe = useActiveWorkspaceStore.persist.onFinishHydration(() => {
      setHasHydrated(true);
    });
    setHasHydrated(useActiveWorkspaceStore.persist.hasHydrated());
    return unsubscribe;
  }, []);

  return hasHydrated;
}
