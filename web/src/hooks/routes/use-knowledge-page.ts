import { useMemo, useState } from "react";

import {
  filterKnowledgeMemories,
  knowledgeMemoryKey,
  sortKnowledgeMemories,
  useDeleteMemory,
  useMemories,
  useMemory,
  type KnowledgeMemoryItem,
  type KnowledgeScope,
} from "@/systems/knowledge";
import { useActiveWorkspace } from "@/systems/workspace";

type Tab = "all" | "global" | "workspace";

function decorateKnowledgeMemories(
  memories: KnowledgeMemoryItem[] | undefined,
  scope: KnowledgeScope
): KnowledgeMemoryItem[] {
  return (memories ?? []).map(memory => ({
    ...memory,
    scope,
    key: memory.key ?? knowledgeMemoryKey({ ...memory, scope }),
  }));
}

function useKnowledgePage() {
  const [activeTab, setActiveTab] = useState<Tab>("all");
  const [selectedMemoryKey, setSelectedMemoryKey] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [deleteTargetKey, setDeleteTargetKey] = useState<string | null>(null);

  const { activeWorkspace } = useActiveWorkspace();
  const activeWorkspacePath = activeWorkspace?.root_dir ?? null;

  const globalMemoriesQuery = useMemories("global");
  const workspaceMemoriesQuery = useMemories("workspace", activeWorkspacePath ?? undefined, {
    enabled: Boolean(activeWorkspacePath),
  });
  const {
    error: deleteMutationError,
    isPending: isDeletePending,
    mutateAsync: deleteMemory,
    reset: resetDeleteMutation,
  } = useDeleteMemory();

  const relevantMemories = useMemo(() => {
    const globalMemories = decorateKnowledgeMemories(globalMemoriesQuery.data, "global");
    const workspaceMemories = decorateKnowledgeMemories(workspaceMemoriesQuery.data, "workspace");

    if (activeTab === "global") {
      return globalMemories;
    }
    if (activeTab === "workspace") {
      return workspaceMemories;
    }
    return [...globalMemories, ...workspaceMemories];
  }, [activeTab, globalMemoriesQuery.data, workspaceMemoriesQuery.data]);

  const visibleMemories = useMemo(() => {
    return sortKnowledgeMemories(filterKnowledgeMemories(relevantMemories, searchQuery));
  }, [relevantMemories, searchQuery]);

  const effectiveSelectedMemoryKey = useMemo(() => {
    if (
      selectedMemoryKey &&
      visibleMemories.some(memory => knowledgeMemoryKey(memory) === selectedMemoryKey)
    ) {
      return selectedMemoryKey;
    }

    return visibleMemories[0] ? knowledgeMemoryKey(visibleMemories[0]) : null;
  }, [selectedMemoryKey, visibleMemories]);

  const selectedMemory = useMemo(
    () => visibleMemories.find(memory => knowledgeMemoryKey(memory) === effectiveSelectedMemoryKey),
    [effectiveSelectedMemoryKey, visibleMemories]
  );

  const selectedScope = selectedMemory?.scope;
  const selectedWorkspace =
    selectedScope === "workspace" ? (activeWorkspacePath ?? undefined) : undefined;
  const {
    data: selectedContent,
    isLoading: isContentLoading,
    error: contentError,
  } = useMemory(selectedScope, selectedMemory?.filename ?? "", selectedWorkspace);

  const isLoading =
    activeTab === "global"
      ? globalMemoriesQuery.isLoading
      : activeTab === "workspace"
        ? workspaceMemoriesQuery.isLoading
        : globalMemoriesQuery.isLoading || workspaceMemoriesQuery.isLoading;

  const error =
    activeTab === "global"
      ? (globalMemoriesQuery.error ?? null)
      : activeTab === "workspace"
        ? (workspaceMemoriesQuery.error ?? null)
        : (globalMemoriesQuery.error ?? workspaceMemoriesQuery.error ?? null);

  const clearDeleteState = () => {
    if (deleteTargetKey !== null || deleteMutationError !== null) {
      resetDeleteMutation();
    }
    setDeleteTargetKey(null);
  };

  const handleSetActiveTab = (nextTab: Tab) => {
    clearDeleteState();
    setActiveTab(nextTab);
  };

  const handleSetSearchQuery = (nextQuery: string) => {
    clearDeleteState();
    setSearchQuery(nextQuery);
  };

  const handleSetSelectedMemoryKey = (nextMemoryKey: string | null) => {
    clearDeleteState();
    setSelectedMemoryKey(nextMemoryKey);
  };

  const handleDelete = async (memory: KnowledgeMemoryItem) => {
    const scope = memory.scope;
    if (!scope) {
      return;
    }

    const memoryKey = knowledgeMemoryKey(memory);
    resetDeleteMutation();
    setDeleteTargetKey(memoryKey);
    await deleteMemory({
      scope,
      filename: memory.filename,
      workspace: scope === "workspace" ? (activeWorkspacePath ?? undefined) : undefined,
    });
    setDeleteTargetKey(null);
  };

  return {
    activeTab,
    contentError,
    effectiveSelectedMemoryKey,
    error,
    handleDelete,
    isContentLoading: isContentLoading && effectiveSelectedMemoryKey !== null,
    isDeletePending,
    deleteError:
      deleteTargetKey !== null &&
      selectedMemory &&
      deleteTargetKey === knowledgeMemoryKey(selectedMemory) &&
      deleteMutationError
        ? deleteMutationError instanceof Error
          ? deleteMutationError.message
          : "Failed to delete knowledge entry"
        : null,
    isLoading,
    memoryCount: visibleMemories.length,
    memories: visibleMemories,
    searchQuery,
    selectedContent,
    selectedMemory,
    selectedScope,
    setActiveTab: handleSetActiveTab,
    setSearchQuery: handleSetSearchQuery,
    setSelectedMemoryKey: handleSetSelectedMemoryKey,
  };
}

export { useKnowledgePage };
