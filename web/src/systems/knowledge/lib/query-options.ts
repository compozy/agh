import { queryOptions } from "@tanstack/react-query";

import { listMemories, readMemory } from "@/systems/knowledge/adapters/knowledge-api";
import { knowledgeKeys } from "@/systems/knowledge/lib/query-keys";
import type { MemoryScope } from "@/systems/knowledge/types";

export function memoriesListOptions(scope?: MemoryScope, workspace?: string) {
  return queryOptions({
    queryKey: knowledgeKeys.list(scope, workspace),
    queryFn: ({ signal }) => listMemories(scope, workspace, signal),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function memoryDetailOptions(scope?: MemoryScope, filename?: string, workspace?: string) {
  return queryOptions({
    queryKey: knowledgeKeys.detail(scope, filename, workspace),
    queryFn: ({ signal }) => {
      if (!scope || !filename) {
        throw new Error("Memory detail query requires both scope and filename");
      }
      return readMemory(scope, filename, workspace, signal);
    },
    staleTime: 30_000,
    enabled: Boolean(scope && filename),
  });
}
