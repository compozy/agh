import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  deleteMemory,
  editMemory,
  triggerMemoryDream,
} from "@/systems/knowledge/adapters/knowledge-api";
import { knowledgeKeys } from "@/systems/knowledge/lib/query-keys";
import type { KnowledgeSelector, MemoryEditRequest } from "@/systems/knowledge/types";

interface DeleteMemoryParams {
  selector: KnowledgeSelector;
  filename: string;
}

export function useDeleteMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ selector, filename }: DeleteMemoryParams) => deleteMemory(selector, filename),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all });
    },
  });
}

export interface EditMemoryParams {
  filename: string;
  body: MemoryEditRequest;
}

export function useEditMemory() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ filename, body }: EditMemoryParams) => editMemory(filename, body),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all });
    },
  });
}

interface TriggerMemoryDreamParams {
  workspaceID?: string;
}

export function useTriggerMemoryDream() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ workspaceID }: TriggerMemoryDreamParams) => triggerMemoryDream(workspaceID),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.all });
    },
  });
}
