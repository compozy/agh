import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsExtensionsRoute } from "../-settings-preload";
import { ExtensionsSettingsPage } from "./-extensions-settings-page";

export const Route = createFileRoute("/_app/settings/extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Extensions" } },
  }),
  loader: ({ context }) => preloadSettingsExtensionsRoute(context.queryClient),
  component: ExtensionsSettingsPage,
});
