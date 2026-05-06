import { queryOptions } from "@tanstack/react-query";

import {
  listMemories,
  listMemoryDecisions,
  readMemory,
  searchMemory,
  type ListMemoryDecisionsParams,
} from "@/systems/knowledge/adapters/knowledge-api";
import { knowledgeKeys } from "@/systems/knowledge/lib/query-keys";
import type { KnowledgeSelector } from "@/systems/knowledge/types";

export function memoriesListOptions(selector?: KnowledgeSelector) {
  return queryOptions({
    queryKey: knowledgeKeys.list(selector),
    queryFn: ({ signal }) => listMemories(selector, signal),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function memoryDetailOptions(selector: KnowledgeSelector | undefined, filename?: string) {
  const enabled = Boolean(selector?.scope && filename);
  return queryOptions({
    queryKey: knowledgeKeys.detail(filename ?? "", selector),
    queryFn: ({ signal }) => {
      if (!selector || !filename) {
        throw new Error("Memory detail query requires both selector and filename");
      }
      return readMemory(selector, filename, signal);
    },
    staleTime: 30_000,
    enabled,
  });
}

export function memorySearchOptions(
  selector: KnowledgeSelector | undefined,
  queryText: string,
  options?: { topK?: number; includeSystem?: boolean; explain?: boolean }
) {
  const trimmed = queryText.trim();
  const enabled = trimmed.length > 0 && Boolean(selector?.scope);
  return queryOptions({
    queryKey: knowledgeKeys.search(trimmed, selector),
    queryFn: ({ signal }) => {
      if (!selector || trimmed.length === 0) {
        throw new Error("Memory search query requires a selector and a non-empty query");
      }
      return searchMemory(
        {
          query_text: trimmed,
          scope: selector.scope,
          workspace_id: selector.workspaceId,
          agent_name: selector.agentName,
          agent_tier: selector.agentTier,
          top_k: options?.topK,
          include_system: options?.includeSystem,
          explain: options?.explain,
        },
        signal
      );
    },
    staleTime: 15_000,
    enabled,
  });
}

export function memoryDecisionsOptions(params: ListMemoryDecisionsParams | undefined) {
  const enabled = Boolean(params?.scope);
  return queryOptions({
    queryKey: knowledgeKeys.decisionsFor("", params),
    queryFn: ({ signal }) => {
      if (!params) {
        throw new Error("Memory decisions query requires a selector");
      }
      return listMemoryDecisions(params, signal);
    },
    staleTime: 15_000,
    enabled,
  });
}
