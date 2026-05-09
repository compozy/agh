import { useMemo, useState } from "react";

import {
  knowledgeMemoryKey,
  sortKnowledgeMemories,
  useDeleteMemory,
  useEditMemory,
  useMemories,
  useMemory,
  useMemoryDecisions,
  useMemorySearch,
  useRevertMemoryDecision,
  useWriteMemory,
  type EditMemoryParams,
  type KnowledgeAgentTier,
  type KnowledgeMemoryItem,
  type KnowledgeScope,
  type KnowledgeSelector,
  type MemoryDecision,
  type MemoryEditRequest,
  type MemoryHeader,
  type MemoryType,
  type MemoryWriteRequest,
} from "@/systems/knowledge";
import { useActiveWorkspace } from "@/systems/workspace";

interface DecorateOptions {
  scope: KnowledgeScope;
  agentTier?: KnowledgeAgentTier;
  agentName?: string;
  workspaceId?: string;
}

function decorateKnowledgeMemories(
  memories: MemoryHeader[] | undefined,
  defaults: DecorateOptions
): KnowledgeMemoryItem[] {
  return (memories ?? []).map(memory => {
    const decorated: KnowledgeMemoryItem = {
      ...memory,
      scope: memory.scope ?? defaults.scope,
      agent_tier: memory.agent_tier ?? defaults.agentTier,
      agent_name: memory.agent_name ?? defaults.agentName,
      workspace_id: memory.workspace_id ?? defaults.workspaceId,
    };
    decorated.key = decorated.key ?? knowledgeMemoryKey(decorated);
    return decorated;
  });
}

function selectorFromMemory(memory: KnowledgeMemoryItem): KnowledgeSelector {
  return {
    scope: memory.scope,
    workspaceId: memory.workspace_id,
    agentName: memory.agent_name,
    agentTier: memory.agent_tier,
  };
}

function describeError(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

function useKnowledgePage() {
  const { activeWorkspaceId } = useActiveWorkspace();

  const [activeScope, setActiveScope] = useState<KnowledgeScope>("global");
  const [agentName, setAgentName] = useState("");
  const [agentTier, setAgentTier] = useState<KnowledgeAgentTier>("workspace");
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedMemoryKey, setSelectedMemoryKey] = useState<string | null>(null);
  const [actionTargetKey, setActionTargetKey] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [revertingDecisionId, setRevertingDecisionId] = useState<string | null>(null);

  const trimmedAgentName = agentName.trim();
  const trimmedSearchQuery = searchQuery.trim();

  const selector: KnowledgeSelector | null = useMemo(() => {
    if (activeScope === "workspace") {
      if (!activeWorkspaceId) {
        return null;
      }
      return { scope: "workspace", workspaceId: activeWorkspaceId };
    }
    if (activeScope === "agent") {
      if (trimmedAgentName === "") {
        return null;
      }
      return {
        scope: "agent",
        agentName: trimmedAgentName,
        agentTier,
        workspaceId: agentTier === "workspace" ? (activeWorkspaceId ?? undefined) : undefined,
      };
    }
    return { scope: "global" };
  }, [activeScope, agentTier, activeWorkspaceId, trimmedAgentName]);

  const decorateOptions: DecorateOptions = useMemo(() => {
    return {
      scope: activeScope,
      agentTier: activeScope === "agent" ? agentTier : undefined,
      agentName: activeScope === "agent" ? trimmedAgentName : undefined,
      workspaceId: selector?.workspaceId,
    };
  }, [activeScope, agentTier, selector?.workspaceId, trimmedAgentName]);

  const memoriesQuery = useMemories(selector ?? undefined, { enabled: Boolean(selector) });
  const searchEnabled = Boolean(selector) && trimmedSearchQuery.length > 0;
  const searchQueryResult = useMemorySearch(selector ?? undefined, trimmedSearchQuery, {
    enabled: searchEnabled,
  });

  const {
    error: deleteMutationError,
    isPending: isDeletePending,
    mutateAsync: deleteMemoryMutate,
    reset: resetDeleteMutation,
  } = useDeleteMemory();

  const {
    error: editMutationError,
    isPending: isEditPending,
    mutateAsync: editMemoryMutate,
    reset: resetEditMutation,
  } = useEditMemory();

  const {
    error: writeMutationError,
    isPending: isCreatePending,
    mutateAsync: writeMemoryMutate,
    reset: resetWriteMutation,
  } = useWriteMemory();

  const {
    error: revertMutationError,
    isPending: isRevertPending,
    mutateAsync: revertMemoryDecisionMutate,
    reset: resetRevertMutation,
  } = useRevertMemoryDecision();

  const listMemories = useMemo<KnowledgeMemoryItem[]>(() => {
    if (searchEnabled) {
      const results = searchQueryResult.data?.results ?? [];
      return results.map(result => {
        const decorated: KnowledgeMemoryItem = {
          ...result.memory,
          scope: result.memory.scope ?? activeScope,
          agent_tier: result.memory.agent_tier ?? decorateOptions.agentTier,
          agent_name: result.memory.agent_name ?? decorateOptions.agentName,
          workspace_id: result.memory.workspace_id ?? decorateOptions.workspaceId,
        };
        decorated.key = knowledgeMemoryKey(decorated);
        return decorated;
      });
    }
    return decorateKnowledgeMemories(memoriesQuery.data, decorateOptions);
  }, [
    activeScope,
    decorateOptions,
    memoriesQuery.data,
    searchEnabled,
    searchQueryResult.data?.results,
  ]);

  const visibleMemories = useMemo(() => sortKnowledgeMemories(listMemories), [listMemories]);

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

  const detailSelector = selectedMemory ? selectorFromMemory(selectedMemory) : null;
  const memoryDetailQuery = useMemory(detailSelector ?? undefined, selectedMemory?.filename, {
    enabled: Boolean(detailSelector && selectedMemory),
  });
  const decisionsQuery = useMemoryDecisions(
    detailSelector ? { ...detailSelector, limit: 10 } : undefined,
    { enabled: Boolean(detailSelector) }
  );

  const isListLoading = searchEnabled ? searchQueryResult.isLoading : memoriesQuery.isLoading;
  const listError = searchEnabled ? searchQueryResult.error : memoriesQuery.error;

  const isLoading = isListLoading;
  const error = listError ?? null;

  const decisionsForSelected: MemoryDecision[] = useMemo(() => {
    const decisions = decisionsQuery.data?.decisions ?? [];
    if (!selectedMemory) return [];
    return decisions.filter(
      decision =>
        decision.target_filename === selectedMemory.filename ||
        decision.frontmatter.filename === selectedMemory.filename
    );
  }, [decisionsQuery.data?.decisions, selectedMemory]);

  const clearActionState = () => {
    if (actionTargetKey !== null || deleteMutationError !== null) {
      resetDeleteMutation();
    }
    if (editMutationError !== null) {
      resetEditMutation();
    }
    if (writeMutationError !== null) {
      resetWriteMutation();
    }
    if (revertMutationError !== null) {
      resetRevertMutation();
    }
    setActionTargetKey(null);
    setRevertingDecisionId(null);
  };

  const handleSetActiveScope = (nextScope: KnowledgeScope) => {
    clearActionState();
    setActiveScope(nextScope);
  };

  const handleSetAgentName = (next: string) => {
    clearActionState();
    setAgentName(next);
  };

  const handleSetAgentTier = (next: KnowledgeAgentTier) => {
    clearActionState();
    setAgentTier(next);
  };

  const handleSetSearchQuery = (next: string) => {
    clearActionState();
    setSearchQuery(next);
  };

  const handleSetSelectedMemoryKey = (next: string | null) => {
    clearActionState();
    setSelectedMemoryKey(next);
  };

  const handleSetCreateOpen = (next: boolean) => {
    if (next) {
      resetWriteMutation();
    }
    setCreateOpen(next);
  };

  const handleCreate = async (input: {
    type: MemoryType;
    name: string;
    description?: string;
    content: string;
  }) => {
    if (!selector) {
      return;
    }
    resetWriteMutation();
    const body: MemoryWriteRequest = {
      scope: selector.scope,
      type: input.type,
      name: input.name,
      description: input.description,
      content: input.content,
      workspace_id: selector.workspaceId,
      agent_name: selector.agentName,
      agent_tier: selector.agentTier,
    };
    const response = await writeMemoryMutate(body);
    const filename = response.decision.target_filename ?? response.decision.frontmatter.filename;
    setSearchQuery("");
    setSelectedMemoryKey(`${selector.scope}:${filename}`);
    setCreateOpen(false);
  };

  const handleDelete = async (memory: KnowledgeMemoryItem) => {
    const memorySelector = selectorFromMemory(memory);
    if (memorySelector.scope === "workspace" && !memorySelector.workspaceId) {
      return;
    }
    const memoryKey = knowledgeMemoryKey(memory);
    resetDeleteMutation();
    setActionTargetKey(memoryKey);
    await deleteMemoryMutate({ selector: memorySelector, filename: memory.filename });
    setActionTargetKey(prev => (prev === memoryKey ? null : prev));
  };

  const handleEdit = async (
    memory: KnowledgeMemoryItem,
    input: { content: string; description?: string }
  ) => {
    const memoryKey = knowledgeMemoryKey(memory);
    resetEditMutation();
    setActionTargetKey(memoryKey);
    const body: MemoryEditRequest = {
      content: input.content,
      description: input.description,
      scope: memory.scope,
      type: memory.type,
      name: memory.name,
      workspace_id: memory.workspace_id,
      agent_name: memory.agent_name,
      agent_tier: memory.agent_tier,
    };
    const params: EditMemoryParams = { filename: memory.filename, body };
    await editMemoryMutate(params);
    setActionTargetKey(prev => (prev === memoryKey ? null : prev));
  };

  const handleRevertDecision = async (decision: MemoryDecision) => {
    if (isRevertPending || revertingDecisionId !== null) {
      return;
    }

    resetRevertMutation();
    setRevertingDecisionId(decision.id);
    try {
      await revertMemoryDecisionMutate({
        decisionID: decision.id,
        body: { reason: "operator reverted from Knowledge" },
      });
      const filename = decision.target_filename ?? decision.frontmatter.filename;
      setSelectedMemoryKey(`${decision.scope}:${filename}`);
    } finally {
      setRevertingDecisionId(prev => (prev === decision.id ? null : prev));
    }
  };

  const selectedTargetMatches = (() => {
    if (!selectedMemory) return false;
    const key = knowledgeMemoryKey(selectedMemory);
    return actionTargetKey === key;
  })();

  const deleteError =
    selectedTargetMatches && deleteMutationError
      ? describeError(deleteMutationError, "Failed to delete knowledge entry")
      : null;
  const editError =
    selectedTargetMatches && editMutationError
      ? describeError(editMutationError, "Failed to edit knowledge entry")
      : null;
  const createError = writeMutationError
    ? describeError(writeMutationError, "Failed to create knowledge entry")
    : null;
  const revertError = revertMutationError
    ? describeError(revertMutationError, "Failed to revert memory decision")
    : null;

  const requiresWorkspace = activeScope === "workspace" && !activeWorkspaceId;
  const requiresAgentName = activeScope === "agent" && trimmedAgentName === "";
  const guardMessage = requiresWorkspace
    ? "Select an active workspace to view workspace memories."
    : requiresAgentName
      ? "Enter an agent name to view agent-scoped memories."
      : null;

  const searchInfo = searchEnabled
    ? `Recall ${searchQueryResult.data?.results.length ?? 0} of top-K`
    : null;

  return {
    activeScope,
    setActiveScope: handleSetActiveScope,
    agentName,
    setAgentName: handleSetAgentName,
    agentTier,
    setAgentTier: handleSetAgentTier,
    searchQuery,
    setSearchQuery: handleSetSearchQuery,
    setSelectedMemoryKey: handleSetSelectedMemoryKey,
    effectiveSelectedMemoryKey,
    memories: visibleMemories,
    memoryCount: visibleMemories.length,
    isLoading,
    error,
    selectedMemory,
    selectedScope: selectedMemory?.scope,
    selectedContent: memoryDetailQuery.data?.content,
    isContentLoading: memoryDetailQuery.isLoading && Boolean(selectedMemory),
    contentError: memoryDetailQuery.error,
    handleDelete,
    isDeletePending,
    deleteError,
    handleEdit,
    isEditPending,
    editError,
    createOpen,
    setCreateOpen: handleSetCreateOpen,
    handleCreate,
    isCreatePending,
    createError,
    createDefaultType: (activeScope === "workspace" ? "project" : "user") as MemoryType,
    canCreateMemory: Boolean(selector),
    decisions: decisionsForSelected,
    decisionsError: decisionsQuery.error,
    isDecisionsLoading: decisionsQuery.isLoading && Boolean(selectedMemory),
    handleRevertDecision,
    revertingDecisionId,
    isRevertPending,
    revertError,
    searchActive: searchEnabled,
    searchInfo,
    guardMessage,
  };
}

export { useKnowledgePage };
