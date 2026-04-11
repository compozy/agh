import { useEffect, useMemo } from "react";

import { useActiveWorkspaceStore } from "../stores/active-workspace-store";
import { useWorkspaces } from "./use-workspaces";

export function useActiveWorkspace() {
  const selectedWorkspaceId = useActiveWorkspaceStore(state => state.selectedWorkspaceId);
  const setSelectedWorkspaceId = useActiveWorkspaceStore(state => state.setSelectedWorkspaceId);
  const clearSelectedWorkspaceId = useActiveWorkspaceStore(state => state.clearSelectedWorkspaceId);
  const query = useWorkspaces();

  const activeWorkspace = useMemo(() => {
    if (!query.data || query.data.length === 0) {
      return undefined;
    }

    if (selectedWorkspaceId) {
      const selectedWorkspace = query.data.find(workspace => workspace.id === selectedWorkspaceId);
      if (selectedWorkspace) {
        return selectedWorkspace;
      }
    }

    return query.data[0];
  }, [query.data, selectedWorkspaceId]);

  useEffect(() => {
    if (!selectedWorkspaceId || !query.data) {
      return;
    }

    if (!query.data.some(workspace => workspace.id === selectedWorkspaceId)) {
      clearSelectedWorkspaceId();
    }
  }, [clearSelectedWorkspaceId, query.data, selectedWorkspaceId]);

  return {
    ...query,
    workspaces: query.data ?? [],
    hasWorkspaces: (query.data?.length ?? 0) > 0,
    activeWorkspace,
    activeWorkspaceId: activeWorkspace?.id ?? null,
    setActiveWorkspaceId: setSelectedWorkspaceId,
    clearActiveWorkspaceSelection: clearSelectedWorkspaceId,
  };
}
