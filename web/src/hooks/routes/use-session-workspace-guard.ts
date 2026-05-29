import { useEffect, useRef } from "react";
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
  const sessionWorkspaceRef = useRef<string | null>(null);

  useEffect(() => {
    if (sessionWorkspaceId) {
      sessionWorkspaceRef.current = sessionWorkspaceId;
    }
  }, [sessionWorkspaceId]);

  useEffect(() => {
    if (
      sessionWorkspaceRef.current &&
      activeWorkspaceId &&
      activeWorkspaceId !== sessionWorkspaceRef.current
    ) {
      void navigate({ to: "/agents/$name", params: { name: agentName }, replace: true });
    }
  }, [activeWorkspaceId, navigate, agentName]);
}
