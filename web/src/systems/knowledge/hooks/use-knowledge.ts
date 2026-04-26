import { useQuery } from "@tanstack/react-query";

import { memoriesListOptions, memoryDetailOptions } from "@/systems/knowledge/lib/query-options";
import type { MemoryScope } from "@/systems/knowledge/types";

interface KnowledgeQueryOptions {
  enabled?: boolean;
}

export function useMemories(
  scope?: MemoryScope,
  workspace?: string,
  options?: KnowledgeQueryOptions
) {
  return useQuery({
    ...memoriesListOptions(scope, workspace),
    enabled: options?.enabled ?? true,
  });
}

export function useMemory(
  scope?: MemoryScope,
  filename?: string,
  workspace?: string,
  options?: KnowledgeQueryOptions
) {
  return useQuery({
    ...memoryDetailOptions(scope, filename, workspace),
    enabled: (options?.enabled ?? true) && Boolean(scope && filename),
  });
}
