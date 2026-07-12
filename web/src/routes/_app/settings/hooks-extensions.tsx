import { createFileRoute } from "@tanstack/react-router";
import { Puzzle } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsHooksExtensionsRoute } from "../-settings-preload";
import { HooksExtensionsSettingsPage } from "./-hooks-extensions-settings-page";

export const Route = createFileRoute("/_app/settings/hooks-extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Hooks & extensions", icon: Puzzle },
  }),
  loader: ({ context }) => preloadSettingsHooksExtensionsRoute(context.queryClient),
  component: HooksExtensionsSettingsPage,
});
