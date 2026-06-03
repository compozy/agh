import { Plus } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";
import { TaskCreateRoute } from "./-tasks-new-route";

export const Route = createFileRoute("/_app/tasks/new")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Tasks", icon: Plus },
  }),
  validateSearch: search => ({
    template:
      typeof search.template === "string" &&
      ["one_shot", "recurring", "epic", "remote_peer", "human_in_loop", "blank"].includes(
        search.template
      )
        ? (search.template as TaskTemplateId)
        : undefined,
  }),
  component: TaskCreateRoute,
});
