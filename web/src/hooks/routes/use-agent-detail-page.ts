import { useCallback, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";

import { agentKeys, useAgent, useAgentSessions, type AgentPayload } from "@/systems/agent";
import {
  sessionKeys,
  useSessionCreate,
  type SessionCreateContextValue,
  type SessionPayload,
} from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

export interface UseAgentDetailPageResult {
  agent: AgentPayload | undefined;
  agentLoading: boolean;
  agentError: Error | null;
  sessions: SessionPayload[];
  sessionsLoading: boolean;
  sessionsError: boolean;
  isRefreshing: boolean;
  isCreatingForAgent: boolean;
  newSessionDisabled: boolean;
  sessionCreate: SessionCreateContextValue;
  onRefresh: () => void;
  onConfigure: () => void;
  onNewSession: () => void;
  onGoHome: () => void;
}

export function useAgentDetailPage(name: string): UseAgentDetailPageResult {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const sessionCreate = useSessionCreate();
  const { data: agent, isLoading: agentLoading, error: agentError } = useAgent(name);
  const {
    sessions,
    isLoading: sessionsLoading,
    isError: sessionsError,
  } = useAgentSessions(activeWorkspaceId, name);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const onRefresh = useCallback(() => {
    setIsRefreshing(true);
    void Promise.all([
      queryClient.invalidateQueries({ queryKey: sessionKeys.lists() }),
      queryClient.invalidateQueries({ queryKey: agentKeys.all }),
    ]).finally(() => setIsRefreshing(false));
  }, [queryClient]);

  const onConfigure = useCallback(() => {
    void navigate({ to: "/settings" });
  }, [navigate]);

  const onNewSession = useCallback(() => {
    sessionCreate.openForAgent(name);
  }, [sessionCreate, name]);

  const onGoHome = useCallback(() => {
    void navigate({ to: "/" });
  }, [navigate]);

  return {
    agent,
    agentLoading,
    agentError: (agentError as Error | null) ?? null,
    sessions,
    sessionsLoading,
    sessionsError,
    isRefreshing,
    isCreatingForAgent: sessionCreate.isCreating && sessionCreate.pendingAgentName === name,
    newSessionDisabled: !sessionCreate.hasActiveWorkspace,
    sessionCreate,
    onRefresh,
    onConfigure,
    onNewSession,
    onGoHome,
  };
}
