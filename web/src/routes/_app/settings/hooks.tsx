import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsHooksRoute } from "../-settings-preload";
import { HooksSettingsPage } from "./-hooks-settings-page";

export const Route = createFileRoute("/_app/settings/hooks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({ topbar: { crumb: { label: "Hooks" } } }),
  loader: ({ context }) => preloadSettingsHooksRoute(context.queryClient),
  component: HooksSettingsPage,
});
