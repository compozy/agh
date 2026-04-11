// Types
export type {
  WorkspaceDetailPayload,
  WorkspacePayload,
  WorkspaceResponse,
  WorkspacesResponse,
} from "./types";

// Adapters
export type { ResolveWorkspaceParams } from "./adapters/workspace-api";
export { fetchWorkspaces, resolveWorkspace } from "./adapters/workspace-api";

// Query infrastructure
export { workspaceKeys } from "./lib/query-keys";
export { workspacesListOptions } from "./lib/query-options";

// Hooks
export { useActiveWorkspace } from "./hooks/use-active-workspace";
export { useResolveWorkspace, useWorkspaces } from "./hooks/use-workspaces";

// Components
export { WorkspaceSelector } from "./components/workspace-selector";
export { WorkspaceOnboarding, WorkspaceSetupDialog } from "./components/workspace-setup";
