import { Outlet, createFileRoute } from "@tanstack/react-router";
import { Puzzle } from "lucide-react";

import { useExtensionsRoutePage } from "@/hooks/routes/use-extensions-route-page";
import { ExtensionsInventory } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export interface ExtensionsRouteSearch {
  tab?: "bundles";
}

function validateExtensionsSearch(search: Record<string, unknown>): ExtensionsRouteSearch {
  return { tab: search.tab === "bundles" ? "bundles" : undefined };
}

export const Route = createFileRoute("/_app/extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Extensions", icon: Puzzle },
  }),
  validateSearch: validateExtensionsSearch,
  component: ExtensionsRoute,
});

function ExtensionsRoute() {
  const { child, tab } = useExtensionsRoutePage(Route.useSearch());
  return child ? <Outlet /> : <ExtensionsInventory tab={tab} />;
}
