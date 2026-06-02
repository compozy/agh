import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";

import { useActiveWorkspace } from "@/systems/workspace";

interface UseSessionWorkspaceGuardOptions {
  sessionWorkspaceId: string | undefined;
  agentName: string;
}

export function useSessionWorkspaceGuard({
  sessionWorkspaceId,
  agentName,
}: UseSessionWorkspaceGuardOptions): void {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();

  useEffect(() => {
    if (sessionWorkspaceId && activeWorkspaceId && activeWorkspaceId !== sessionWorkspaceId) {
      void navigate({ to: "/agents/$name", params: { name: agentName }, replace: true });
    }
  }, [activeWorkspaceId, sessionWorkspaceId, navigate, agentName]);
}
