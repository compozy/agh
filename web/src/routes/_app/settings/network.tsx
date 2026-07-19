import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { NetworkSettingsPage } from "./-network-settings-page";

export const Route = createFileRoute("/_app/settings/network")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Network" } },
  }),
  component: NetworkSettingsPage,
});
