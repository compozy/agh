import { useQuery } from "@tanstack/react-query";

import { memoriesListOptions, memoryDetailOptions } from "@/systems/knowledge/lib/query-options";
import type { MemoryScope } from "@/systems/knowledge/types";

export function useMemories(scope?: MemoryScope, workspace?: string) {
  return useQuery(memoriesListOptions(scope, workspace));
}

export function useMemory(scope?: MemoryScope, filename?: string, workspace?: string) {
  return useQuery(memoryDetailOptions(scope, filename, workspace));
}
