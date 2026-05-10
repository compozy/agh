import { Clock3 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";

export const Route = createFileRoute("/_app/jobs")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Jobs", icon: Clock3 },
  }),
  component: JobsPage,
});

function JobsPage() {
  const page = useAutomationJobsPage();

  return (
    <AutomationOperationsPage
      createButtonTestId="create-job-btn"
      createLabel="Job"
      page={page}
      title="Jobs"
      titlePrefix="jobs"
    />
  );
}
