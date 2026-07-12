import { createFileRoute } from "@tanstack/react-router";
import { Activity } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsObservabilityRoute } from "../-settings-preload";
import { ObservabilitySettingsPage } from "./-observability-settings-page";

export const Route = createFileRoute("/_app/settings/observability")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Observability", icon: Activity },
  }),
  loader: ({ context }) => preloadSettingsObservabilityRoute(context.queryClient),
  component: ObservabilitySettingsPage,
});
