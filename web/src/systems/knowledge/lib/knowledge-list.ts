import {
  compareKnowledgeScope,
  deriveScopeFromFilename,
  knowledgeScopeLabel,
  type KnowledgeScope,
} from "@/systems/knowledge/lib/knowledge-formatters";
import type { MemoryHeader } from "@/systems/knowledge/types";

export interface KnowledgeMemoryGroup {
  scope: KnowledgeScope;
  label: string;
  memories: MemoryHeader[];
}

export function filterKnowledgeMemories(memories: MemoryHeader[], query: string): MemoryHeader[] {
  const normalized = query.trim().toLowerCase();
  if (normalized === "") {
    return memories;
  }

  return memories.filter(memory => {
    return (
      memory.name.toLowerCase().includes(normalized) ||
      (memory.description ?? "").toLowerCase().includes(normalized) ||
      memory.type.toLowerCase().includes(normalized)
    );
  });
}

export function groupKnowledgeMemoriesByScope(memories: MemoryHeader[]): KnowledgeMemoryGroup[] {
  const buckets = new Map<KnowledgeScope, MemoryHeader[]>();

  for (const memory of memories) {
    const scope = deriveScopeFromFilename(memory.filename);
    const list = buckets.get(scope);
    if (list) {
      list.push(memory);
      continue;
    }

    buckets.set(scope, [memory]);
  }

  return Array.from(buckets.entries())
    .sort(([left], [right]) => compareKnowledgeScope(left, right))
    .map(([scope, items]) => ({
      scope,
      label: knowledgeScopeLabel(scope),
      memories: items.slice().sort((left, right) => left.name.localeCompare(right.name)),
    }));
}

export function sortKnowledgeMemories(memories: MemoryHeader[]): MemoryHeader[] {
  return groupKnowledgeMemoriesByScope(memories).flatMap(group => group.memories);
}
