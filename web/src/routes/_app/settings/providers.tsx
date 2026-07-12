import { createFileRoute } from "@tanstack/react-router";
import { Database } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsProvidersRoute } from "../-settings-preload";
import { ProvidersSettingsPage } from "./-providers-settings-page";

export const Route = createFileRoute("/_app/settings/providers")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Providers", icon: Database },
  }),
  loader: ({ context }) => preloadSettingsProvidersRoute(context.queryClient),
  component: ProvidersSettingsPage,
});
