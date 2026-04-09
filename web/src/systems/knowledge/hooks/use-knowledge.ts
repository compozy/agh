import { useQuery } from "@tanstack/react-query";

import { memoriesListOptions, memoryDetailOptions } from "../lib/query-options";

export function useMemories(scope?: string, workspace?: string) {
  return useQuery(memoriesListOptions(scope, workspace));
}

export function useMemory(scope: string, filename: string, workspace?: string) {
  return useQuery(memoryDetailOptions(scope, filename, workspace));
}
