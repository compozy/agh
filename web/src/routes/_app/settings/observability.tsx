import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsObservabilityRoute } from "../-settings-preload";
import { ObservabilitySettingsPage } from "./-observability-settings-page";

export const Route = createFileRoute("/_app/settings/observability")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Observability" } },
  }),
  loader: ({ context }) => preloadSettingsObservabilityRoute(context.queryClient),
  component: ObservabilitySettingsPage,
});
