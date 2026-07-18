import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsMemoryRoute } from "../-settings-preload";
import { MemorySettingsPage } from "./-memory-settings-page";

export const Route = createFileRoute("/_app/settings/memory")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Memory" } },
  }),
  loader: ({ context }) => preloadSettingsMemoryRoute(context.queryClient),
  component: MemorySettingsPage,
});
