// Types
export type {
  SessionProviderOption,
  WorkspaceDetailPayload,
  WorkspacePayload,
  WorkspaceResponse,
  WorkspacesResponse,
} from "./types";

// Adapters
export {
  deleteWorkspace,
  fetchWorkspace,
  fetchWorkspaces,
  resolveWorkspace,
} from "./adapters/workspace-api";
export type { ResolveWorkspaceParams } from "./adapters/workspace-api";

// Query infrastructure
export { isHomeWorkspace, splitHomeWorkspace } from "./lib/home-workspace";
export { toWorkspaceCommandSelectOptions } from "./lib/workspace-command-select-options";
export type { HomeWorkspacePartition, WorkspaceHomeCandidate } from "./lib/home-workspace";
export { workspaceKeys } from "./lib/query-keys";
export { workspaceDetailOptions, workspacesListOptions } from "./lib/query-options";

// Hooks
export { useActiveWorkspace } from "./hooks/use-active-workspace";
export {
  useActiveWorkspaceStore,
  useActiveWorkspaceStoreHasHydrated,
} from "./hooks/use-active-workspace-store";
export { useUserHomeDir } from "./hooks/use-user-home-dir";
export { resetUserHomeDirStore, useUserHomeDirStore } from "./hooks/use-user-home-dir-store";
export {
  useDeleteWorkspace,
  useResolveWorkspace,
  useWorkspace,
  useWorkspaces,
} from "./hooks/use-workspaces";

// Components
export { OptionCard } from "./components/option-card";
export type {
  OptionCardHeaderProps,
  OptionCardIconProps,
  OptionCardProps,
  OptionCardSize,
  OptionCardTone,
} from "./components/option-card";
export {
  ScopeSelector,
  type ScopeSelectorProps,
  type ScopeSelectorScope,
} from "./components/scope-selector";
export {
  WorkspaceCommandSelect,
  type WorkspaceCommandSelectOption,
  type WorkspaceCommandSelectProps,
} from "./components/workspace-command-select";
export { WorkspaceOnboarding, WorkspaceSetupDialog } from "./components/workspace-setup";
