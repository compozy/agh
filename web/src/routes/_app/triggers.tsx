import { Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import { useAutomationTriggersPage } from "@/hooks/routes/use-automation-page";

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

export const Route = createFileRoute("/_app/triggers")({
  validateSearch: parseCreateSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Triggers", icon: Zap },
  }),
  component: TriggersPage,
});

function TriggersPage() {
  const { create, loop } = Route.useSearch();
  const page = useAutomationTriggersPage(create === "loop" && loop ? { loop } : {});

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
