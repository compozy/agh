import { useEffect, useMemo, useState } from "react";

import { useDeleteMemory, useMemories, useMemory } from "@/systems/knowledge";
import {
  filterKnowledgeMemories,
  sortKnowledgeMemories,
} from "@/systems/knowledge/lib/knowledge-list";
import type { MemoryScope } from "@/systems/knowledge/types";
import { useActiveWorkspace } from "@/systems/workspace";
import { deriveScopeFromFilename } from "@/systems/knowledge/lib/knowledge-formatters";

type Tab = "all" | "global" | "workspace";

function useKnowledgePage() {
  const [activeTab, setActiveTab] = useState<Tab>("all");
  const [selectedFilename, setSelectedFilename] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [deleteTargetFilename, setDeleteTargetFilename] = useState<string | null>(null);

  const { activeWorkspaceId } = useActiveWorkspace();
  const scopeFilter = activeTab === "all" ? undefined : activeTab;

  const {
    data: memories,
    isLoading,
    error,
  } = useMemories(scopeFilter, activeWorkspaceId || undefined);
  const {
    error: deleteMutationError,
    isPending: isDeletePending,
    mutateAsync: deleteMemory,
    reset: resetDeleteMutation,
  } = useDeleteMemory();
  const visibleMemories = useMemo(() => {
    return sortKnowledgeMemories(filterKnowledgeMemories(memories ?? [], searchQuery));
  }, [memories, searchQuery]);

  const effectiveSelectedFilename = useMemo(() => {
    if (selectedFilename && visibleMemories.some(memory => memory.filename === selectedFilename)) {
      return selectedFilename;
    }

    return visibleMemories[0]?.filename ?? null;
  }, [selectedFilename, visibleMemories]);

  const selectedMemory = useMemo(
    () => visibleMemories.find(memory => memory.filename === effectiveSelectedFilename),
    [visibleMemories, effectiveSelectedFilename]
  );

  const selectedScope = selectedMemory
    ? (deriveScopeFromFilename(selectedMemory.filename) as Exclude<MemoryScope, undefined>)
    : undefined;
  const {
    data: selectedContent,
    isLoading: isContentLoading,
    error: contentError,
  } = useMemory(selectedScope, effectiveSelectedFilename ?? "", activeWorkspaceId || undefined);

  useEffect(() => {
    if (!deleteTargetFilename || isDeletePending) {
      return;
    }

    if (selectedMemory?.filename === deleteTargetFilename) {
      return;
    }

    resetDeleteMutation();
    setDeleteTargetFilename(null);
  }, [deleteTargetFilename, isDeletePending, resetDeleteMutation, selectedMemory?.filename]);

  const handleDelete = async (filename: string) => {
    const scope =
      selectedMemory?.filename === filename ? selectedScope : deriveScopeFromFilename(filename);
    if (!scope) {
      return;
    }

    setDeleteTargetFilename(filename);
    resetDeleteMutation();
    await deleteMemory({
      scope,
      filename,
      workspace: activeWorkspaceId || undefined,
    });
    setDeleteTargetFilename(null);
  };

  return {
    activeTab,
    contentError,
    effectiveSelectedFilename,
    error,
    handleDelete,
    isContentLoading: isContentLoading && effectiveSelectedFilename !== null,
    isDeletePending,
    deleteError:
      deleteTargetFilename === selectedMemory?.filename && deleteMutationError
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
    setActiveTab,
    setSearchQuery,
    setSelectedFilename,
  };
}

export { useKnowledgePage };
