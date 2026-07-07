import { useState } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import { useLoops } from "@/systems/loops";
import type { LoopCatalogEntry, LoopCatalogFilter } from "@/systems/loops";
import { useActiveWorkspace } from "@/systems/workspace";

import { useLoopBindingIndex } from "./use-loop-bindings";

const INITIAL_FILTER: LoopCatalogFilter = { kind: "all", category: null };

/** View-model for the Loops catalog route: data, binding badges, filter, and Run launch. */
export function useLoopsCatalog() {
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const navigate = useNavigate();
  const [filter, setFilter] = useState<LoopCatalogFilter>(INITIAL_FILTER);
  // Skip catalog + binding fetches while a child route (detail/editor/run) owns the view.
  const loopsQuery = useLoops(workspaceId, workspaceId !== "" && !hasChildMatch);
  const bindingIndex = useLoopBindingIndex(hasChildMatch ? "" : workspaceId);

  const handleRun = (entry: LoopCatalogEntry) => {
    void navigate({ to: "/loops/$name/run", params: { name: entry.name } });
  };

  return { hasChildMatch, workspaceId, loopsQuery, bindingIndex, filter, setFilter, handleRun };
}
