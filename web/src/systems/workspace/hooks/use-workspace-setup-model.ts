import { useAgents } from "@/systems/agent";
import { useSettingsSandboxes } from "@/systems/settings";

import { useWorkspaceSetupContent } from "./use-workspace-setup-content";

interface UseWorkspaceSetupModelOptions {
  onWorkspaceResolved: (workspaceId: string) => void;
  onSuccessClose?: () => void;
}

/**
 * One data model behind both workspace-setup hosts: the split dialog and the
 * first-run page render the same location and defaults panes from this.
 */
export function useWorkspaceSetupModel(options: UseWorkspaceSetupModelOptions) {
  const setup = useWorkspaceSetupContent(options);
  const agentsQuery = useAgents();
  const sandboxesQuery = useSettingsSandboxes();

  return {
    setup,
    agents: agentsQuery.data ?? [],
    sandboxes: (sandboxesQuery.data?.sandboxes ?? []).map(entry => ({
      name: entry.name,
      backend: entry.profile.backend,
    })),
  };
}
