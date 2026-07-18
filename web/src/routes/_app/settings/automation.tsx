import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationSettingsPage } from "./-automation-settings-page";

export const Route = createFileRoute("/_app/settings/automation")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Automation" } },
  }),
  component: AutomationSettingsPage,
});
