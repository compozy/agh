import { createFileRoute } from "@tanstack/react-router";
import { Webhook } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSettingsHooksRoute } from "../-settings-preload";
import { HooksSettingsPage } from "./-hooks-settings-page";

export const Route = createFileRoute("/_app/settings/hooks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({ topbar: { title: "Hooks", icon: Webhook } }),
  loader: ({ context }) => preloadSettingsHooksRoute(context.queryClient),
  component: HooksSettingsPage,
});
