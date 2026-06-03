import { Pencil } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { TaskEditRoute } from "./-tasks-edit-route";

export const Route = createFileRoute("/_app/tasks/$id/edit")({
  beforeLoad: ({ params }): { topbar: TopbarRouteContext } => ({
    topbar: { title: `Task ${params.id}`, icon: Pencil },
  }),
  component: TaskEditRoute,
});
