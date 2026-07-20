import { createFileRoute } from "@tanstack/react-router";

import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import type { TaskTemplateId } from "@/systems/tasks/lib/task-templates";

export const Route = createFileRoute("/_app/tasks/new")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "New task" } },
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
  component: createOsRouteSync("tasks"),
});
