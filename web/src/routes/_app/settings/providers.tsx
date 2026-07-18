import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsProvidersRoute } from "../-settings-preload";
import { ProvidersSettingsPage } from "./-providers-settings-page";

export const Route = createFileRoute("/_app/settings/providers")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Providers" } },
  }),
  loader: ({ context }) => preloadSettingsProvidersRoute(context.queryClient),
  component: ProvidersSettingsPage,
});
