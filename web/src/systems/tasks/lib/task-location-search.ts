import { z } from "zod";

import type { TaskViewMode } from "../types";
import type { TaskTemplateId } from "./task-templates";

const taskRouteModeSchema = z.enum(["kanban", "dashboard", "inbox"]);
const taskTemplateIdSchema = z.enum([
  "one_shot",
  "recurring",
  "epic",
  "remote_peer",
  "human_in_loop",
  "blank",
]);

/** Object type, not an interface: it is spread straight into navigation search. */
export type TasksRouteSearch = {
  mode?: "kanban" | "dashboard" | "inbox";
};

/**
 * Create-dialog location. It inherits `mode` because the dialog is an overlay on
 * the tasks catalog: the view underneath has to survive in the location, or
 * opening the editor would silently snap the background back to the list.
 */
export interface TaskCreateSearch extends TasksRouteSearch {
  template?: TaskTemplateId;
}

export function validateTasksSearch(search: Record<string, unknown>): TasksRouteSearch {
  const result = taskRouteModeSchema.safeParse(search.mode);
  return result.success ? { mode: result.data } : {};
}

export function parseTasksSurfaceMode(search: Record<string, unknown>): TaskViewMode {
  return validateTasksSearch(search).mode ?? "list";
}

export function validateTaskCreateSearch(search: Record<string, unknown>): TaskCreateSearch {
  const template = taskTemplateIdSchema.safeParse(search.template);
  const mode = taskRouteModeSchema.safeParse(search.mode);
  return {
    ...(template.success ? { template: template.data } : {}),
    ...(mode.success ? { mode: mode.data } : {}),
  };
}

/** Search a create location carries back to the catalog it was opened from. */
export function taskCatalogSearchFor(mode: TaskViewMode): TasksRouteSearch {
  return mode === "list" ? {} : { mode };
}
