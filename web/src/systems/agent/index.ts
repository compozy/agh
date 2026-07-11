// Types
export type {
  AgentHeartbeatHistoryResponse,
  AgentHeartbeatPayload,
  AgentHeartbeatStatusPayload,
  AgentMCPServer,
  AgentPayload,
  AgentResponse,
  AgentSoulHistoryResponse,
  AgentSoulPayload,
  AgentsResponse,
  CreateAgentParams,
  DeleteAgentHeartbeatParams,
  DeleteAgentHeartbeatResponse,
  DeleteAgentResponse,
  DeleteAgentSoulParams,
  DeleteAgentSoulResponse,
  DuplicateAgentParams,
  PutAgentHeartbeatParams,
  PutAgentHeartbeatResponse,
  PutAgentSoulParams,
  PutAgentSoulResponse,
  RollbackAgentHeartbeatParams,
  RollbackAgentHeartbeatResponse,
  RollbackAgentSoulParams,
  RollbackAgentSoulResponse,
  UpdateAgentParams,
  ValidateAgentHeartbeatParams,
  ValidateAgentHeartbeatResponse,
  ValidateAgentSoulParams,
  ValidateAgentSoulResponse,
  WakeAgentHeartbeatParams,
  WakeAgentHeartbeatResponse,
} from "./types";

// Adapters
export {
  AgentApiError,
  AgentDigestConflictError,
  AgentTargetExistsError,
  createAgent,
  deleteAgent,
  duplicateAgent,
  fetchAgent,
  fetchAgents,
  isAgentDigestConflict,
  isAgentTargetExists,
  updateAgent,
} from "./adapters/agent-api";
export {
  deleteAgentSoul,
  fetchAgentSoul,
  fetchAgentSoulHistory,
  putAgentSoul,
  rollbackAgentSoul,
  validateAgentSoul,
} from "./adapters/agent-soul-api";
export {
  deleteAgentHeartbeat,
  fetchAgentHeartbeat,
  fetchAgentHeartbeatHistory,
  fetchAgentHeartbeatStatus,
  putAgentHeartbeat,
  rollbackAgentHeartbeat,
  validateAgentHeartbeat,
  wakeAgentHeartbeat,
  type FetchAgentHeartbeatStatusParams,
} from "./adapters/agent-heartbeat-api";

// Query infrastructure
export { agentKeys } from "./lib/query-keys";
export {
  agentDetailOptions,
  agentHeartbeatHistoryOptions,
  agentHeartbeatOptions,
  agentHeartbeatStatusOptions,
  agentSoulHistoryOptions,
  agentSoulOptions,
  agentsListOptions,
} from "./lib/query-options";

// Lib
export {
  AGENT_CREATE_PERMISSION_OPTIONS,
  appendAgentCreateTokens,
  buildCreateAgentParams,
  buildDraftFromAgentPayload,
  buildDuplicateAgentParams,
  createDefaultAgentCreateDraft,
  parseAgentCreateCategoryPath,
  removeAgentCreateToken,
  splitAgentCreateTokens,
  updateAgentCreateScope,
  validateAgentCreateDraft,
  type AgentCreateDialogDraft,
  type AgentCreatePermission,
  type AgentCreatePermissionChoice,
  type AgentCreateScope,
  type AgentCreateStep,
  type AgentCreateValidation,
  type AgentCreateValidationContext,
} from "./lib/agent-create-draft";
export {
  deriveAgentFleetSignals,
  type AgentFleetSignals,
  type AgentFleetStatus,
} from "./lib/fleet-signals";
export {
  getAgentSessionStatus,
  type AgentSessionStatus,
  type AgentSessionStatusKind,
} from "./lib/session-status";
export {
  AGENT_CATEGORY_FOLDER_ID_PREFIX,
  AGENT_CATEGORY_LEAF_ID_PREFIX,
  AGENT_CATEGORY_LABEL_SEPARATOR,
  buildAgentCategoryTree,
  formatCategoryLabel,
  getAgentCategoryFolderId,
  getAgentLeafId,
  isAgentRootLevel,
  joinAgentCategorySegments,
  type AgentCategoryFolderNode,
  type AgentCategoryLeafNode,
  type AgentCategoryNode,
} from "./lib/agent-category";
export {
  hasActiveAgentFleetFilters,
  parseAgentFleetCategoryFilter,
  parseAgentFleetStatusFilter,
  validateAgentsFleetSearch,
  type AgentsFleetSearch,
} from "./lib/agent-fleet-search";
export {
  collectAgentCategoryOptions,
  compareAgentNamesStable,
  formatAgentFleetAriaLabel,
  formatAgentFleetCardMeta,
  formatAgentFleetMeta,
  formatAgentOriginLabel,
  formatCategoryMetaSegment,
  indexSessionsByAgent,
  projectAgentFleetRows,
  sortAgentsByNameStable,
  type AgentFleetRowModel,
} from "./lib/agent-fleet-projection";
export {
  agentFleetFiltersToChips,
  agentFleetChipsToFilters,
  buildAgentFleetFilterFields,
  type AgentFleetFilterFieldKey,
  type AgentFleetFilterValues,
} from "./lib/agent-fleet-filters";

// Hooks
export {
  useAgent,
  useAgents,
  useCreateAgent,
  useDeleteAgent,
  useDuplicateAgent,
  useUpdateAgent,
  type DeleteAgentVariables,
  type DuplicateAgentVariables,
  type UpdateAgentVariables,
} from "./hooks/use-agents";
export {
  useAgentCreateDialog,
  type AgentCreateDialogApi,
  type AgentCreateDialogState,
} from "./hooks/use-agent-create-dialog";
export { useAgentSessions } from "./hooks/use-agent-sessions";
export {
  useAgentSoul,
  useAgentSoulHistory,
  useDeleteAgentSoul,
  usePutAgentSoul,
  useRollbackAgentSoul,
  useValidateAgentSoul,
} from "./hooks/use-agent-soul";
export {
  useAgentHeartbeat,
  useAgentHeartbeatHistory,
  useAgentHeartbeatStatus,
  useDeleteAgentHeartbeat,
  usePutAgentHeartbeat,
  useRollbackAgentHeartbeat,
  useValidateAgentHeartbeat,
  useWakeAgentHeartbeat,
} from "./hooks/use-agent-heartbeat";
export {
  useUnsavedGuard,
  type UseUnsavedGuardOptions,
  type UseUnsavedGuardResult,
} from "./hooks/use-unsaved-guard";

// Components
export { AgentIcon, providerIconMap } from "./components/agent-icon";
export {
  AgentPageActions,
  AgentPageStatusPill,
  type AgentPageActionsProps,
  type AgentPageStatusPillProps,
} from "./components/agent-page-header";
export { AgentSessionsList, type AgentSessionsListProps } from "./components/agent-sessions-list";
export {
  AgentInfoInspector,
  type AgentInfoInspectorProps,
} from "./components/agent-info-inspector";
export { AgentStatsGrid, type AgentStatsGridProps } from "./components/agent-stats-grid";
export {
  AgentCommandSelect,
  type AgentCommandSelectProps,
} from "./components/agent-command-select";
export {
  AgentCommandMultiSelect,
  type AgentCommandMultiSelectProps,
} from "./components/agent-command-multi-select";
export { AgentCreateDialog, type AgentCreateDialogProps } from "./components/agent-create-dialog";
export { TokenListField, type TokenListFieldProps } from "./components/token-list-field";
export {
  AgentCreateHostProvider,
  type AgentCreateHostProviderProps,
  type AgentCreateHostValue,
} from "./components/agent-create-host";
export { useAgentCreateHost } from "./hooks/use-agent-create-host";
export { AgentFleetRow, type AgentFleetRowProps } from "./components/agent-fleet-row";
export { AgentFleetCard, type AgentFleetCardProps } from "./components/agent-fleet-card";
export {
  AgentFleetNewSessionButton,
  type AgentFleetNewSessionButtonProps,
} from "./components/agent-fleet-new-session-button";
export { AgentFleetToolbar, type AgentFleetToolbarProps } from "./components/agent-fleet-toolbar";
export { AgentFleetList, type AgentFleetListProps } from "./components/agent-fleet-list";
