import { Tabs, TabsList, TabsTrigger } from "@agh/ui";

import type { TaskDetailPanel } from "@/hooks/routes/use-task-detail-page";
import { cn } from "@/lib/utils";

export interface TasksDetailTabItem {
  id: TaskDetailPanel;
  label: string;
  count?: number;
  live?: boolean;
}

export interface TasksDetailTabsProps {
  items: TasksDetailTabItem[];
  active: TaskDetailPanel;
  onChange: (next: TaskDetailPanel) => void;
}

/**
 * Detail panel tab bar — `@agh/ui` `Tabs` (Base UI primitive) rendered as a
 * line-variant bar with optional count badges + pulsing live indicator per tab.
 */
export function TasksDetailTabs({ items, active, onChange }: TasksDetailTabsProps) {
  return (
    <Tabs
      aria-label="Task detail panels"
      className="border-b border-[color:var(--color-divider)] px-4"
      data-testid="tasks-detail-tabs"
      onValueChange={value => onChange(value as TaskDetailPanel)}
      value={active}
    >
      <TabsList variant="line" className="h-10">
        {items.map(item => (
          <TabsTrigger
            className="gap-2"
            data-testid={`tasks-detail-tab-${item.id}`}
            key={item.id}
            value={item.id}
          >
            <span>{item.label}</span>
            {typeof item.count === "number" ? (
              <span
                className={cn(
                  "inline-flex h-5 min-w-[20px] items-center justify-center rounded-md px-1.5 font-mono text-[10px]",
                  "bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]",
                  "group-data-[active=true]:bg-[color:var(--color-surface-elevated)] group-data-[active=true]:text-[color:var(--color-text-primary)]"
                )}
                data-testid={`tasks-detail-tab-count-${item.id}`}
              >
                {item.count}
              </span>
            ) : null}
            {item.live ? (
              <span
                className="inline-flex h-5 items-center gap-1 rounded-md bg-[color:var(--color-accent-tint)] px-1.5 font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-accent)]"
                data-testid={`tasks-detail-tab-live-${item.id}`}
              >
                <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--color-accent)]" />
                Live
              </span>
            ) : null}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
