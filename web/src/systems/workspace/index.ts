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
export { OptionCard } from "./components/option-card";
export type {
  OptionCardHeaderProps,
  OptionCardIconProps,
  OptionCardProps,
  OptionCardSize,
  OptionCardTone,
} from "./components/option-card";
export { WorkspaceOnboarding, WorkspaceSetupDialog } from "./components/workspace-setup";
