import { createFileRoute } from "@tanstack/react-router";
import { ListChecks } from "lucide-react";

import type { TopbarRouteContext } from "@/types/topbar";
import { TasksRoute } from "./-tasks-route";

export const Route = createFileRoute("/_app/tasks")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Tasks", icon: ListChecks },
  }),
  component: TasksRoute,
});
