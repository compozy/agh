import { queryOptions } from "@tanstack/react-query";

import { listMemories, readMemory } from "../adapters/knowledge-api";
import { knowledgeKeys } from "./query-keys";

export function memoriesListOptions(scope?: string, workspace?: string) {
  return queryOptions({
    queryKey: knowledgeKeys.list(scope, workspace),
    queryFn: ({ signal }) => listMemories(scope, workspace, signal),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function memoryDetailOptions(scope: string, filename: string, workspace?: string) {
  return queryOptions({
    queryKey: knowledgeKeys.detail(scope, filename, workspace),
    queryFn: ({ signal }) => readMemory(scope, filename, workspace, signal),
    staleTime: 30_000,
    enabled: !!scope && !!filename,
  });
}
