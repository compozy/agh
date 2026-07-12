import { createFileRoute } from "@tanstack/react-router";
import { Bot } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationSettingsPage } from "./-automation-settings-page";

export const Route = createFileRoute("/_app/settings/automation")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Automation settings", icon: Bot },
  }),
  component: AutomationSettingsPage,
});
