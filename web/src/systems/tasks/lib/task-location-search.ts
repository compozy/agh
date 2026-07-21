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

export interface TasksRouteSearch {
  mode?: "kanban" | "dashboard" | "inbox";
}

export interface TaskCreateSearch {
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
  const result = taskTemplateIdSchema.safeParse(search.template);
  return result.success ? { template: result.data } : {};
}
