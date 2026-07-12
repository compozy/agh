import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import { useAgentCreateHost } from "@/systems/agent/hooks/use-agent-create-host";
import { useAgentCatalog } from "@/systems/agent/hooks/use-agents";
import { projectAgentFleetRows } from "@/systems/agent/lib/agent-fleet-projection";
import {
  hasActiveAgentFleetFilters,
  type AgentsFleetSearch,
} from "@/systems/agent/lib/agent-fleet-search";
import { useSessionCreate } from "@/systems/session";
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
  const sessionCreate = useSessionCreate();
  const searchInputRef = useRef<HTMLInputElement | null>(null);
  const [draftQuery, setDraftQuery] = useState(search.q ?? "");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const agentsEnabled = workspaceId !== "" && !hasChildMatch;
  const catalogQuery = useAgentCatalog(
    workspaceId,
    {
      q: search.q,
      category: search.category,
      status: search.status,
      limit: 50,
    },
    { enabled: agentsEnabled }
  );
  const agents = catalogQuery.agents.map(item => item.agent);
  const sessionsPartial = catalogQuery.isSuccess && !catalogQuery.sessionsAvailable;
  const view: ListingViewMode = search.view ?? "rows";

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
    if (hasChildMatch) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isEditableTarget(event.target)) return;
      if (event.target instanceof Element && event.target.closest('[role="dialog"]')) return;
      event.preventDefault();
      searchInputRef.current?.focus();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [hasChildMatch]);

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

  const setView = useCallback(
    (nextView: ListingViewMode) => {
      updateSearch(current => ({
        ...current,
        view: nextView === "rows" ? undefined : nextView,
      }));
    },
    [updateSearch]
  );

  const clearFilters = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setDraftQuery("");
    updateSearch(current => ({
      view: current.view,
    }));
  }, [updateSearch]);

  const openNewSession = useCallback(
    (agentName: string) => {
      sessionCreate.openForAgent(agentName);
    },
    [sessionCreate]
  );

  const rows = useMemo(
    () =>
      projectAgentFleetRows({
        items: catalogQuery.agents,
        sessionsAvailable: catalogQuery.sessionsAvailable,
      }),
    [catalogQuery.agents, catalogQuery.sessionsAvailable]
  );

  const filtersActive = hasActiveAgentFleetFilters(search);
  const isLoading = catalogQuery.isLoading;
  const overallTotal = catalogQuery.facets?.total ?? 0;
  const isFirstRunEmpty =
    !isLoading && !catalogQuery.isError && overallTotal === 0 && !filtersActive;
  const isFilteredEmpty =
    !isLoading && !catalogQuery.isError && catalogQuery.total === 0 && filtersActive;
  const showFacets =
    !isLoading && !isFirstRunEmpty && !(catalogQuery.isError && agents.length === 0);

  return {
    hasChildMatch,
    workspaceId,
    catalogQuery,
    agents,
    fleetTotal: catalogQuery.total,
    rows,
    categoryOptions: catalogQuery.facets?.categories ?? [],
    search,
    draftQuery,
    searchInputRef,
    view,
    setDraftQuery: setDraftQueryAndDebounce,
    setFilters,
    setView,
    clearFilters,
    openCreate: openDialog,
    openNewSession,
    newSessionDisabled: !sessionCreate.hasActiveWorkspace,
    isLoading,
    isFirstRunEmpty,
    isFilteredEmpty,
    sessionsPartial,
    hasMore: catalogQuery.hasNextPage,
    isLoadingMore: catalogQuery.isFetchingNextPage,
    loadMore: () => {
      void catalogQuery.fetchNextPage();
    },
    showFacets,
    agentsError: catalogQuery.error,
    retryAgents: () => {
      void catalogQuery.refetch();
    },
  };
}

export { useAgentsFleetPage };
export type { AgentsFleetSearch };
