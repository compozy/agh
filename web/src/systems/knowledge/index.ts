// Types
export type {
  KnowledgeFilter,
  KnowledgeMemoryItem,
  KnowledgeScope,
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
export { KnowledgeDeleteDialog } from "./components/knowledge-delete-dialog";

// Lib
export {
  compareKnowledgeScope,
  deriveScopeFromFilename,
  knowledgeMemoryKey,
  formatKnowledgeDateTime,
  formatKnowledgeRelativeTime,
  knowledgeScopeLabel,
  knowledgeScopeShortLabel,
  memoryScopeTone,
  memoryTypeTone,
  resolveKnowledgeScope,
} from "./lib/knowledge-formatters";
export {
  filterKnowledgeMemories,
  groupKnowledgeMemoriesByScope,
  sortKnowledgeMemories,
} from "./lib/knowledge-list";
