import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";

import { useAgent } from "@/systems/agent/hooks/use-agents";
import { useAgentSessions } from "@/systems/agent/hooks/use-agent-sessions";
import { useAgentCreateHost } from "@/systems/agent/hooks/use-agent-create-host";
import { useAgentDeleteFlow } from "@/systems/agent/hooks/use-agent-delete-flow";
import type {
  AgentDetailSearch,
  AgentDetailTab,
  AgentInstructionFile,
  AgentSessionFilter,
  ResolvedAgentDetailSearch,
} from "@/systems/agent/lib/agent-detail-search";
import { resolveAgentDetailSearch } from "@/systems/agent/lib/agent-detail-search";
import type { AgentSettingsSection } from "@/systems/agent/lib/agent-settings-search";
import type { AgentPayload } from "@/systems/agent/types";
import { useSessionCreate, type SessionPayload } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

export interface UseAgentDetailPageResult {
  agent: AgentPayload | undefined;
  agentLoading: boolean;
  agentError: Error | null;
  sessions: SessionPayload[];
  sessionsTotal: number;
  activeSessionsTotal: number;
  resumableSessionsTotal: number;
  lastSessionActivityAt: string | null;
  hasMoreSessions: boolean;
  isLoadingMoreSessions: boolean;
  onLoadMoreSessions: () => void;
  sessionsLoading: boolean;
  sessionsError: boolean;
  search: ResolvedAgentDetailSearch;
  setTab: (tab: AgentDetailTab) => void;
  setFile: (file: AgentInstructionFile) => void;
  setFilter: (filter: AgentSessionFilter) => void;
  isCreatingForAgent: boolean;
  newSessionDisabled: boolean;
  onNewSession: () => void;
  onEditSettings: (section?: AgentSettingsSection) => void;
  onDuplicate: () => void;
  onDelete: () => void;
  onBackToAgents: () => void;
  deleteDialog: ReturnType<typeof useAgentDeleteFlow>["confirmDialog"];
}

export function useAgentDetailPage(
  name: string,
  rawSearch: AgentDetailSearch
): UseAgentDetailPageResult {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const sessionCreate = useSessionCreate();
  const createHost = useAgentCreateHost();
  const search = resolveAgentDetailSearch(rawSearch);
  const {
    data: agent,
    isLoading: agentLoading,
    error: agentError,
  } = useAgent(name, activeWorkspaceId);
  const {
    sessions,
    total: sessionsTotal,
    activeTotal: activeSessionsTotal,
    resumableTotal: resumableSessionsTotal,
    lastActivityAt: lastSessionActivityAt,
    hasMore: hasMoreSessions,
    isLoadingMore: isLoadingMoreSessions,
    loadMore: onLoadMoreSessions,
    isLoading: sessionsLoading,
    isError: sessionsError,
  } = useAgentSessions(activeWorkspaceId, name);

  const deleteFlow = useAgentDeleteFlow({
    agent,
    workspaceId: activeWorkspaceId,
  });

  const setTab = useCallback(
    (tab: AgentDetailTab) => {
      void navigate({
        to: "/agents/$name",
        params: { name },
        search: {
          tab,
          file: search.file,
          filter: search.filter,
        },
        replace: true,
      });
    },
    [name, navigate, search.file, search.filter]
  );

  const setFile = useCallback(
    (file: AgentInstructionFile) => {
      void navigate({
        to: "/agents/$name",
        params: { name },
        search: {
          tab: "instructions",
          file,
          filter: search.filter,
        },
        replace: true,
      });
    },
    [name, navigate, search.filter]
  );

  const setFilter = useCallback(
    (filter: AgentSessionFilter) => {
      void navigate({
        to: "/agents/$name",
        params: { name },
        search: {
          tab: "sessions",
          file: search.file,
          filter,
        },
        replace: true,
      });
    },
    [name, navigate, search.file]
  );

  const onNewSession = useCallback(() => {
    sessionCreate.openForAgent(name);
  }, [sessionCreate, name]);

  const onEditSettings = useCallback(
    (section?: AgentSettingsSection) => {
      void navigate({
        to: "/agents/$name/settings",
        params: { name },
        search: section ? { section } : {},
      });
    },
    [name, navigate]
  );

  const onDuplicate = useCallback(() => {
    if (!agent) return;
    createHost.openForDuplicate(agent);
  }, [agent, createHost]);

  const onBackToAgents = useCallback(() => {
    void navigate({ to: "/agents" });
  }, [navigate]);

  return {
    agent,
    agentLoading,
    agentError: (agentError as Error | null) ?? null,
    sessions,
    sessionsTotal,
    activeSessionsTotal,
    resumableSessionsTotal,
    lastSessionActivityAt,
    hasMoreSessions,
    isLoadingMoreSessions,
    onLoadMoreSessions,
    sessionsLoading,
    sessionsError,
    search,
    setTab,
    setFile,
    setFilter,
    isCreatingForAgent: sessionCreate.isCreating && sessionCreate.pendingAgentName === name,
    newSessionDisabled: !sessionCreate.hasActiveWorkspace,
    onNewSession,
    onEditSettings,
    onDuplicate,
    onDelete: deleteFlow.openDialog,
    onBackToAgents,
    deleteDialog: deleteFlow.confirmDialog,
  };
}
