import { createFileRoute } from "@tanstack/react-router";
import { Network as NetworkIcon } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { NetworkSettingsPage } from "./-network-settings-page";

export const Route = createFileRoute("/_app/settings/network")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Network settings", icon: NetworkIcon },
  }),
  component: NetworkSettingsPage,
});
