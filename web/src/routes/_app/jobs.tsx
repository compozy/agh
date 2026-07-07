import { Clock3 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";

/** Optional deep-link that pre-targets the create sheet at a Loop (§9.14 CTA). */
interface AutomationCreateSearch {
  create?: "loop";
  loop?: string;
}

function parseCreateSearch(search: Record<string, unknown>): AutomationCreateSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    loop: typeof search.loop === "string" ? search.loop : undefined,
  };
}

export const Route = createFileRoute("/_app/jobs")({
  validateSearch: parseCreateSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Jobs", icon: Clock3 },
  }),
  component: JobsPage,
});

function JobsPage() {
  const { create, loop } = Route.useSearch();
  const page = useAutomationJobsPage(create === "loop" && loop ? { loop } : {});

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
