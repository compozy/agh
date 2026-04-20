import { Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationTriggersPage } from "@/hooks/routes/use-automation-page";

export const Route = createFileRoute("/_app/triggers")({
  component: TriggersPage,
});

function TriggersPage() {
  const page = useAutomationTriggersPage();

  return (
    <AutomationOperationsPage
      createButtonTestId="create-trigger-btn"
      createLabel="Trigger"
      icon={Zap}
      page={page}
      title="Triggers"
      titlePrefix="triggers"
    />
  );
}
