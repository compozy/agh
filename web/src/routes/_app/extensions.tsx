import { Outlet, createFileRoute } from "@tanstack/react-router";

import {
  useExtensionsRoutePage,
  validateExtensionsSearch,
  type ExtensionsRouteSearch,
} from "@/hooks/routes/use-extensions-route-page";
import { ExtensionsInventory } from "@/systems/extensions";
import type { TopbarRouteContext } from "@/types/topbar";

export type { ExtensionsRouteSearch };

export const Route = createFileRoute("/_app/extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Extensions", to: "/extensions" } },
  }),
  validateSearch: validateExtensionsSearch,
  component: ExtensionsRoute,
});

function ExtensionsRoute() {
  const { child, setView, tab, view } = useExtensionsRoutePage(Route.useSearch());
  return child ? <Outlet /> : <ExtensionsInventory onViewChange={setView} tab={tab} view={view} />;
}
