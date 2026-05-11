import { Plus } from "lucide-react";
import * as React from "react";

import { Button, Pill } from "@agh/ui";

import { cn } from "@/lib/utils";

import type { TaskKanbanColumn as TaskKanbanColumnDef } from "../lib/task-grouping";

import type { PillTone } from "@agh/ui";

export interface TaskKanbanColumnProps {
  column: TaskKanbanColumnDef;
  count: number;
  tone: PillTone;
  onAdd?: () => void;
  emptyState?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
}

export function TaskKanbanColumn({
  column,
  count,
  tone,
  onAdd,
  emptyState,
  children,
  className,
}: TaskKanbanColumnProps) {
  const isEmpty = React.Children.count(children) === 0;

  return (
    <section
      className={cn(
        "flex min-w-0 flex-col overflow-hidden rounded-lg bg-(--canvas-soft)",
        "min-h-[460px] max-h-[calc(100vh-220px)]",
        className
      )}
      data-testid={`tasks-kanban-column-${column.id}`}
      role="listitem"
    >
      <header className="flex shrink-0 items-center gap-2 px-3 pt-[11px] pb-[9px]">
        <Pill.Dot tone={tone} />
        <h2 className="text-[12px] font-medium tracking-[-0.005em] text-(--fg-strong)">
          {column.label}
        </h2>
        <span
          className="font-mono text-[10.5px] tracking-normal tabular-nums text-(--faint)"
          data-testid={`tasks-kanban-column-count-${column.id}`}
        >
          {count}
        </span>
        {onAdd ? (
          <Button
            aria-label={`Add task to ${column.label}`}
            className="ml-auto"
            data-testid={`tasks-kanban-column-add-${column.id}`}
            onClick={onAdd}
            size="icon-xs"
            type="button"
            variant="ghost"
          >
            <Plus />
          </Button>
        ) : null}
      </header>

      <div
        className="flex min-h-0 flex-1 flex-col gap-[6px] overflow-y-auto px-2 pt-1 pb-3"
        data-testid={`tasks-kanban-column-body-${column.id}`}
      >
        {isEmpty
          ? (emptyState ?? (
              <div
                className="flex flex-1 items-center justify-center rounded-md border border-dashed border-(--line) px-3 py-8 text-center text-[12px] text-(--subtle)"
                data-testid={`tasks-kanban-column-empty-${column.id}`}
              >
                No tasks
              </div>
            ))
          : children}
      </div>
    </section>
  );
}
