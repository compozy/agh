// Types
export type {
  KnowledgeFilter,
  MemoryConsolidateResponse,
  MemoryHeader,
  MemoryMutationResponse,
  MemoryReadResponse,
  MemoryScope,
  MemoryType,
} from "./types";

// Adapters
export {
  consolidateMemory,
  deleteMemory,
  KnowledgeApiError,
  listMemories,
  readMemory,
  writeMemory,
} from "./adapters/knowledge-api";

// Query infrastructure
export { knowledgeKeys } from "./lib/query-keys";
export { memoriesListOptions, memoryDetailOptions } from "./lib/query-options";

// Hooks
export { useMemories, useMemory } from "./hooks/use-knowledge";
export { useConsolidateMemory, useDeleteMemory } from "./hooks/use-knowledge-actions";

// Components
export { KnowledgeListPanel } from "./components/knowledge-list-panel";
export { KnowledgeDetailPanel } from "./components/knowledge-detail-panel";
