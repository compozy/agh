import { createFileRoute } from "@tanstack/react-router";

import {
  MARKETPLACE_KIND_LABEL,
  isMarketplaceKind,
  marketplaceRouteKindFor,
} from "@/systems/marketplace";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";

interface MarketplaceDetailSearch {
  installed_name?: string;
  scope?: "global" | "workspace";
  workspace_id?: string;
}

function validateMarketplaceDetailSearch(search: Record<string, unknown>): MarketplaceDetailSearch {
  const scope =
    search.scope === "global" || search.scope === "workspace" ? search.scope : undefined;
  const workspaceId =
    scope === "workspace" && typeof search.workspace_id === "string"
      ? search.workspace_id.trim() || undefined
      : undefined;
  const installedName =
    typeof search.installed_name === "string"
      ? search.installed_name.trim() || undefined
      : undefined;
  return { installed_name: installedName, scope, workspace_id: workspaceId };
}

export const Route = createFileRoute("/_app/marketplace/$kind/$entryId")({
  validateSearch: validateMarketplaceDetailSearch,
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: {
      parentCrumb: isMarketplaceKind(params.kind)
        ? {
            label: MARKETPLACE_KIND_LABEL[params.kind],
            to: `/marketplace/${marketplaceRouteKindFor(params.kind)}`,
          }
        : undefined,
      crumb: { label: params.entryId },
    },
  }),
  component: createOsRouteSync("marketplace"),
});
