import { createFileRoute } from "@tanstack/react-router";
import { Settings as SettingsIcon } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsGeneralRoute } from "../-settings-preload";
import { GeneralSettingsPage } from "./-general-settings-page";

export const Route = createFileRoute("/_app/settings/general")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "General settings", icon: SettingsIcon },
  }),
  loader: ({ context }) => preloadSettingsGeneralRoute(context.queryClient),
  component: GeneralSettingsPage,
});
