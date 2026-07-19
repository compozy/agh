import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { TaskEditRoute } from "./-tasks-edit-route";

export const Route = createFileRoute("/_app/tasks/$id/edit")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Edit" } },
  }),
  component: TaskEditRoute,
});
