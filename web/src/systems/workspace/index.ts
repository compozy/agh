// Types
export type {
  SessionProviderOption,
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
export { ProviderCommandList } from "./components/provider-command-list";
export type { ProviderCommandListProps } from "./components/provider-command-list";
export { ProviderCommandSelect } from "./components/provider-command-select";
export type { ProviderCommandSelectProps } from "./components/provider-command-select";
export { WorkspaceOnboarding, WorkspaceSetupDialog } from "./components/workspace-setup";
