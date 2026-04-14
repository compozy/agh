// Types
export type {
  WorkspaceDetailPayload,
  WorkspacePayload,
  WorkspaceResponse,
  WorkspacesResponse,
} from "./types";

// Adapters
export type { ResolveWorkspaceParams } from "./adapters/workspace-api";
export { fetchWorkspace, fetchWorkspaces, resolveWorkspace } from "./adapters/workspace-api";

// Query infrastructure
export { workspaceKeys } from "./lib/query-keys";
export { workspaceDetailOptions, workspacesListOptions } from "./lib/query-options";

// Hooks
export { useActiveWorkspace } from "./hooks/use-active-workspace";
export { useResolveWorkspace, useWorkspace, useWorkspaces } from "./hooks/use-workspaces";

// Components
export { WorkspacePageShell } from "./components/workspace-page-shell";
export { WorkspaceSelector } from "./components/workspace-selector";
export { WorkspaceOnboarding, WorkspaceSetupDialog } from "./components/workspace-setup";
