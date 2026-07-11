import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import { useAgentCreateHost } from "@/systems/agent/hooks/use-agent-create-host";
import { useAgents } from "@/systems/agent/hooks/use-agents";
import {
  collectAgentCategoryOptions,
  projectAgentFleetRows,
} from "@/systems/agent/lib/agent-fleet-projection";
import {
  hasActiveAgentFleetFilters,
  type AgentsFleetSearch,
} from "@/systems/agent/lib/agent-fleet-search";
import { useSessions } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";
import { normalizeListingSearchValue } from "@/lib/listing-search";

const SEARCH_DEBOUNCE_MS = 200;

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

function useAgentsFleetPage(search: AgentsFleetSearch = {}) {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const navigate = useNavigate({ from: "/agents" });
  const { openDialog } = useAgentCreateHost();
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const [draftQuery, setDraftQuery] = useState(search.q ?? "");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const agentsEnabled = workspaceId !== "" && !hasChildMatch;
  const agentsQuery = useAgents(workspaceId || null, { enabled: agentsEnabled });
  const sessionsQuery = useSessions(workspaceId || null, { enabled: agentsEnabled });

  const agents = agentsQuery.data ?? [];
  const sessionsAvailable = !sessionsQuery.isError;
  const sessions = sessionsAvailable ? (sessionsQuery.data ?? []) : null;
  const sessionsPartial = agentsQuery.isSuccess && sessionsQuery.isError;

  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }
    setDraftQuery(search.q ?? "");
  }, [search.q]);

  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isEditableTarget(event.target)) return;
      if (event.target instanceof Element && event.target.closest('[role="dialog"]')) return;
      event.preventDefault();
      searchInputRef.current?.focus();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  const updateSearch = useCallback(
    (updater: (current: AgentsFleetSearch) => AgentsFleetSearch) => {
      void navigate({
        search: current => updater(current),
        to: "/agents",
      });
    },
    [navigate]
  );

  const setDraftQueryAndDebounce = useCallback(
    (nextQuery: string) => {
      setDraftQuery(nextQuery);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        debounceRef.current = null;
        updateSearch(current => ({
          ...current,
          q: normalizeListingSearchValue(nextQuery),
        }));
      }, SEARCH_DEBOUNCE_MS);
    },
    [updateSearch]
  );

  const setFilters = useCallback(
    (next: Pick<AgentsFleetSearch, "category" | "status">) => {
      updateSearch(current => ({
        ...current,
        category: next.category,
        status: next.status,
      }));
    },
    [updateSearch]
  );

  const clearFilters = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setDraftQuery("");
    updateSearch(() => ({}));
  }, [updateSearch]);

  const categoryOptions = useMemo(() => collectAgentCategoryOptions(agents), [agents]);
  const rows = useMemo(
    () =>
      projectAgentFleetRows({
        agents,
        sessions,
        search,
      }),
    [agents, search, sessions]
  );

  const filtersActive = hasActiveAgentFleetFilters(search);
  const isLoading = agentsQuery.isLoading || (agents.length > 0 && sessionsQuery.isLoading);
  const isFirstRunEmpty = !isLoading && !agentsQuery.isError && agents.length === 0;
  const isFilteredEmpty =
    !isLoading && !agentsQuery.isError && agents.length > 0 && rows.length === 0 && filtersActive;

  return {
    hasChildMatch,
    workspaceId,
    agentsQuery,
    sessionsQuery,
    agents,
    fleetTotal: agents.length,
    rows,
    categoryOptions,
    search,
    draftQuery,
    searchInputRef,
    setDraftQuery: setDraftQueryAndDebounce,
    setFilters,
    clearFilters,
    openCreate: openDialog,
    isLoading,
    isFirstRunEmpty,
    isFilteredEmpty,
    sessionsPartial,
    agentsError: agentsQuery.error,
    retryAgents: () => {
      void agentsQuery.refetch();
    },
  };
}

export { useAgentsFleetPage };
export type { AgentsFleetSearch };
