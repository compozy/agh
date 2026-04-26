import {
  compareKnowledgeScope,
  knowledgeScopeLabel,
  resolveKnowledgeScope,
} from "@/systems/knowledge/lib/knowledge-formatters";
import type { KnowledgeMemoryItem, KnowledgeScope } from "@/systems/knowledge/types";

export interface KnowledgeMemoryGroup {
  scope: KnowledgeScope;
  label: string;
  memories: KnowledgeMemoryItem[];
}

export function filterKnowledgeMemories(
  memories: KnowledgeMemoryItem[],
  query: string
): KnowledgeMemoryItem[] {
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

export function groupKnowledgeMemoriesByScope(
  memories: KnowledgeMemoryItem[]
): KnowledgeMemoryGroup[] {
  const buckets = new Map<KnowledgeScope, KnowledgeMemoryItem[]>();

  for (const memory of memories) {
    const scope = resolveKnowledgeScope(memory);
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

export function sortKnowledgeMemories(memories: KnowledgeMemoryItem[]): KnowledgeMemoryItem[] {
  return groupKnowledgeMemoriesByScope(memories).flatMap(group => group.memories);
}
