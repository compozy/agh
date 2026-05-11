// Types
export type {
  KnowledgeAgentTier,
  KnowledgeFilter,
  KnowledgeMemoryItem,
  KnowledgeScope,
  KnowledgeSelector,
  MemoryAgentTier,
  MemoryDecision,
  MemoryDecisionOp,
  MemoryDecisionRevertRequest,
  MemoryDecisionRevertResponse,
  MemoryDecisionsResponse,
  MemoryDecisionSource,
  MemoryDeleteResponse,
  MemoryDreamTriggerResponse,
  MemoryEditRequest,
  MemoryEditResponse,
  MemoryHeader,
  MemoryReadResponse,
  MemoryScope,
  MemorySearchRequest,
  MemorySearchResponse,
  MemorySearchResult,
  MemoryType,
  MemoryWriteRequest,
  MemoryWriteResponse,
} from "./types";

// Adapters
export {
  deleteMemory,
  editMemory,
  KnowledgeApiError,
  listMemories,
  listMemoryDecisions,
  readMemory,
  revertMemoryDecision,
  searchMemory,
  triggerMemoryDream,
  writeMemory,
  type ListMemoryDecisionsParams,
} from "./adapters/knowledge-api";

// Query infrastructure
export { knowledgeKeys } from "./lib/query-keys";
export {
  memoriesListOptions,
  memoryDecisionsOptions,
  memoryDetailOptions,
  memorySearchOptions,
} from "./lib/query-options";

// Hooks
export {
  useMemories,
  useMemory,
  useMemoryDecisions,
  useMemorySearch,
  type UseMemorySearchOptions,
} from "./hooks/use-knowledge";
export {
  useDeleteMemory,
  useEditMemory,
  useRevertMemoryDecision,
  useTriggerMemoryDream,
  useWriteMemory,
  type EditMemoryParams,
} from "./hooks/use-knowledge-actions";

// Components
export { KnowledgeListPanel } from "./components/knowledge-list-panel";
export { KnowledgeDetailPanel } from "./components/knowledge-detail-panel";
export { KnowledgeCreateDialog } from "./components/knowledge-create-dialog";
export { KnowledgeDeleteDialog } from "./components/knowledge-delete-dialog";
export { KnowledgeEditDialog } from "./components/knowledge-edit-dialog";
export { KnowledgeDecisionsSection } from "./components/knowledge-decisions-section";

// Lib
export {
  compareKnowledgeScope,
  decisionOpLabel,
  decisionSourceLabel,
  knowledgeAgentTierLabel,
  knowledgeAgentTierShortLabel,
  knowledgeMemoryKey,
  knowledgeScopeLabel,
  knowledgeScopeShortLabel,
  memoryScopeTone,
  memoryTypeTone,
} from "./lib/knowledge-formatters";
export {
  filterKnowledgeMemories,
  groupKnowledgeMemoriesByScope,
  sortKnowledgeMemories,
} from "./lib/knowledge-list";
export {
  KNOWLEDGE_TYPE_TONE,
  knowledgeTypeFor,
  type KnowledgeType,
  type KnowledgeTypeTone,
} from "./lib/knowledge-type-tone";
