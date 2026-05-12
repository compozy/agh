import { Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationTriggersPage } from "@/hooks/routes/use-automation-page";

export const Route = createFileRoute("/_app/triggers")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Triggers", icon: Zap },
  }),
  component: TriggersPage,
});

function TriggersPage() {
  const page = useAutomationTriggersPage();

  return (
    <AutomationOperationsPage
      createButtonTestId="create-trigger-btn"
      createLabel="Trigger"
      page={page}
      title="Triggers"
      titlePrefix="triggers"
    />
  );
}
