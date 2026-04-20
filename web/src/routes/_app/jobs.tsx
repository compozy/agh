import { Clock3 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";

export const Route = createFileRoute("/_app/jobs")({
  component: JobsPage,
});

function JobsPage() {
  const page = useAutomationJobsPage();

  return (
    <AutomationOperationsPage
      createButtonTestId="create-job-btn"
      createLabel="Job"
      icon={Clock3}
      page={page}
      title="Jobs"
      titlePrefix="jobs"
    />
  );
}
