import { useCallback } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import { useLoops } from "@/systems/loops";
import type {
  LoopCatalogEntry,
  LoopCatalogFilter,
  LoopKindFilter,
  LoopStatusFilter,
} from "@/systems/loops";
import {
  parseLoopCategoryFilter,
  parseLoopKindFilter,
  parseLoopStatusFilter,
} from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";
import { normalizeListingSearchValue } from "@/lib/listing-search";

import { useLoopBindingIndex } from "./use-loop-bindings";

export interface LoopsRouteSearch {
  q?: string;
  view?: ListingViewMode;
  kind?: Exclude<LoopKindFilter, "all">;
  category?: string;
  status?: LoopStatusFilter;
}

/** View-model for the Loops catalog route: URL state, data, bindings, and Run launch. */
function useLoopsCatalog(search: LoopsRouteSearch = {}) {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspaceId, activeWorkspace } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const navigate = useNavigate({ from: "/loops" });

  const searchQuery = search.q ?? "";
  const view: ListingViewMode = search.view ?? "rows";
  const filter: LoopCatalogFilter = {
    kind: search.kind ?? "all",
    category: search.category ?? null,
    status: search.status ?? null,
  };

  // Skip catalog + binding fetches while a child route (detail/editor/run) owns the view.
  const loopsQuery = useLoops(workspaceId, workspaceId !== "" && !hasChildMatch);
  const bindingIndex = useLoopBindingIndex(hasChildMatch ? "" : workspaceId);

  const updateSearch = useCallback(
    (updater: (current: LoopsRouteSearch) => LoopsRouteSearch) => {
      void navigate({
        search: current => updater(current),
        to: "/loops",
      });
    },
    [navigate]
  );

  const setSearchQuery = useCallback(
    (nextQuery: string) => {
      updateSearch(current => ({
        ...current,
        q: normalizeListingSearchValue(nextQuery),
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

  const setKindFilter = useCallback(
    (next: LoopKindFilter) => {
      updateSearch(current => ({
        ...current,
        kind: next === "all" ? undefined : next,
      }));
    },
    [updateSearch]
  );

  const setCategoryFilter = useCallback(
    (next: string | null) => {
      updateSearch(current => ({
        ...current,
        category: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const setStatusFilter = useCallback(
    (next: LoopStatusFilter | null) => {
      updateSearch(current => ({
        ...current,
        status: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const clearFilters = useCallback(() => {
    updateSearch(current => ({
      ...current,
      q: undefined,
      kind: undefined,
      category: undefined,
      status: undefined,
    }));
  }, [updateSearch]);

  const handleRefresh = useCallback(() => {
    void loopsQuery.refetch();
  }, [loopsQuery]);

  const handleRun = useCallback(
    (entry: LoopCatalogEntry) => {
      void navigate({ to: "/loops/$name/run", params: { name: entry.name } });
    },
    [navigate]
  );

  return {
    activeWorkspace,
    bindingIndex,
    clearFilters,
    filter,
    handleRefresh,
    handleRun,
    hasChildMatch,
    loopsQuery,
    searchQuery,
    setCategoryFilter,
    setKindFilter,
    setSearchQuery,
    setStatusFilter,
    setView,
    view,
    workspaceId,
  };
}

export { parseLoopCategoryFilter, parseLoopKindFilter, parseLoopStatusFilter, useLoopsCatalog };
