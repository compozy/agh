import { ChevronsUpDown } from "lucide-react";
import { toast } from "sonner";

import {
  cn,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
  Switch,
} from "@agh/ui";

import { useUpdateTask } from "../hooks/use-task-actions";
import { taskPriorityLabel } from "../lib/task-formatters";
import type { TaskPriority, TaskRecord } from "../types";

const PRIORITY_DOT: Record<TaskPriority, string> = {
  urgent: "bg-danger",
  high: "bg-warning",
  medium: "bg-muted",
  low: "bg-faint",
};

const PRIORITY_ORDER: readonly TaskPriority[] = ["urgent", "high", "medium", "low"];

/** Inline PATCH-backed priority editor for the properties rail. */
export function TaskPriorityEditor({ task }: { task: TaskRecord }) {
  const updateTask = useUpdateTask();
  const priority = task.priority ?? "medium";

  const handleSelect = async (next: string) => {
    if (next === priority) return;
    try {
      await updateTask.mutateAsync({ id: task.id, data: { priority: next as TaskPriority } });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Couldn't change priority");
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Change priority"
        data-testid="tasks-rail-priority"
        disabled={updateTask.isPending}
        render={
          <button
            className={cn(
              "-mr-1.5 inline-flex items-center gap-1.5 rounded-sm px-1.5 py-0.5",
              "text-small-body font-medium text-fg transition-colors duration-fast",
              "hover:bg-row-hover focus-visible:outline-none focus-visible:shadow-focus-ring",
              "disabled:cursor-not-allowed disabled:opacity-50"
            )}
            type="button"
          />
        }
      >
        <span aria-hidden="true" className={cn("size-1.5 rounded-full", PRIORITY_DOT[priority])} />
        {taskPriorityLabel(priority)}
        <ChevronsUpDown aria-hidden="true" className="size-[11px] text-faint" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" data-testid="tasks-rail-priority-menu">
        <DropdownMenuRadioGroup onValueChange={value => void handleSelect(value)} value={priority}>
          {PRIORITY_ORDER.map(option => (
            <DropdownMenuRadioItem
              data-testid={`tasks-rail-priority-${option}`}
              key={option}
              value={option}
            >
              <span
                aria-hidden="true"
                className={cn("size-1.5 rounded-full", PRIORITY_DOT[option])}
              />
              {taskPriorityLabel(option)}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Inline PATCH-backed auto-enqueue toggle (instant-effect boolean → switch). */
export function TaskAutoEnqueueSwitch({ task }: { task: TaskRecord }) {
  const updateTask = useUpdateTask();
  const enabled = Boolean(task.auto_enqueue_on_ready);

  const handleToggle = async (next: boolean) => {
    try {
      await updateTask.mutateAsync({ id: task.id, data: { auto_enqueue_on_ready: next } });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Couldn't change auto-enqueue");
    }
  };

  return (
    <Switch
      aria-label="Auto-enqueue when ready"
      checked={enabled}
      data-testid="tasks-rail-auto-enqueue"
      disabled={updateTask.isPending}
      onCheckedChange={next => void handleToggle(next)}
    />
  );
}
