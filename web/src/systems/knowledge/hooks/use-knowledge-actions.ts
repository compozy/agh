import { useMutation, useQueryClient } from "@tanstack/react-query";

import { consolidateMemory, deleteMemory } from "@/systems/knowledge/adapters/knowledge-api";
import { knowledgeKeys } from "@/systems/knowledge/lib/query-keys";
import type { MemoryScope } from "@/systems/knowledge/types";

interface DeleteMemoryParams {
  scope: Exclude<MemoryScope, undefined>;
  filename: string;
  workspace?: string;
}

export function useDeleteMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ scope, filename, workspace }: DeleteMemoryParams) =>
      deleteMemory(scope, filename, workspace),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all });
    },
  });
}

interface ConsolidateMemoryParams {
  workspace?: string;
}

export function useConsolidateMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ workspace }: ConsolidateMemoryParams) => consolidateMemory(workspace),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all });
    },
  });
}
