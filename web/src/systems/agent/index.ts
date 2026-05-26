// Types
export type {
  AgentMCPServer,
  AgentPayload,
  AgentResponse,
  AgentsResponse,
  CreateAgentParams,
} from "./types";

// Adapters
export { AgentApiError, createAgent, fetchAgent, fetchAgents } from "./adapters/agent-api";

// Query infrastructure
export { agentKeys } from "./lib/query-keys";
export { agentDetailOptions, agentsListOptions } from "./lib/query-options";

// Lib
export {
  AGENT_CREATE_PERMISSION_OPTIONS,
  appendAgentCreateTokens,
  buildCreateAgentParams,
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

// Hooks
export { useAgent, useAgents, useCreateAgent } from "./hooks/use-agents";
export {
  useAgentCreateDialog,
  type AgentCreateDialogApi,
  type AgentCreateDialogState,
} from "./hooks/use-agent-create-dialog";
export { useAgentSessions } from "./hooks/use-agent-sessions";

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
export { AgentCategoryTree, type AgentCategoryTreeProps } from "./components/agent-category-tree";
export {
  AgentCommandSelect,
  type AgentCommandSelectProps,
} from "./components/agent-command-select";
export {
  AgentCommandMultiSelect,
  type AgentCommandMultiSelectProps,
} from "./components/agent-command-multi-select";
export { AgentCreateDialog, type AgentCreateDialogProps } from "./components/agent-create-dialog";
