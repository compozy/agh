import { useDesktop } from "../../hooks/use-desktop";
import { parseTaskWindowLocation } from "./task-window-location";
import { TaskCreateLocation, TaskEditLocation } from "./task-editor-locations";
import { TaskDetailLocation } from "./task-detail-location";
import { TaskRunLocation } from "./task-run-location";
import { TasksCatalogLocation } from "./tasks-catalog-location";

/** Tasks app controller driven exclusively by the logical window's WM location. */
export function TasksWindow({ windowId }: { windowId: string }) {
  const location = useDesktop(
    state => state.windows[windowId]?.location ?? { pathname: "/tasks", search: {} }
  );
  const parsed = parseTaskWindowLocation(location);

  if (parsed.kind === "create") return <TaskCreateLocation search={parsed.search} />;
  if (parsed.kind === "edit") return <TaskEditLocation taskId={parsed.taskId} />;
  if (parsed.kind === "detail") {
    return <TaskDetailLocation search={parsed.search} taskId={parsed.taskId} />;
  }
  if (parsed.kind === "run") {
    return <TaskRunLocation runId={parsed.runId} taskId={parsed.taskId} />;
  }
  return <TasksCatalogLocation mode={parsed.mode} />;
}
