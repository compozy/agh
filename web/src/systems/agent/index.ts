// Types
export type { AgentMCPServer, AgentPayload, AgentResponse, AgentsResponse } from "./types";

// Adapters
export { fetchAgent, fetchAgents } from "./adapters/agent-api";

// Query infrastructure
export { agentKeys } from "./lib/query-keys";
export { agentDetailOptions, agentsListOptions } from "./lib/query-options";

// Lib
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
export { useAgent, useAgents } from "./hooks/use-agents";
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
