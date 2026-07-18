import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsGeneralRoute } from "../-settings-preload";
import { GeneralSettingsPage } from "./-general-settings-page";

export const Route = createFileRoute("/_app/settings/general")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "General" } },
  }),
  loader: ({ context }) => preloadSettingsGeneralRoute(context.queryClient),
  component: GeneralSettingsPage,
});
