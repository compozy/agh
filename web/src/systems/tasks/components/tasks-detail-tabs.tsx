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

export function TasksDetailTabs({ items, active, onChange }: TasksDetailTabsProps) {
  return (
    <div
      aria-label="Task detail panels"
      className="flex items-center gap-6 border-b border-[color:var(--color-divider)] px-6"
      data-testid="tasks-detail-tabs"
      role="tablist"
    >
      {items.map(item => {
        const isActive = item.id === active;
        return (
          <button
            aria-selected={isActive}
            className={cn(
              "group relative flex items-center gap-2 py-3 text-sm transition-colors",
              isActive
                ? "text-[color:var(--color-text-primary)]"
                : "text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
            )}
            data-testid={`tasks-detail-tab-${item.id}`}
            key={item.id}
            onClick={() => onChange(item.id)}
            role="tab"
            type="button"
          >
            <span>{item.label}</span>
            {typeof item.count === "number" ? (
              <span
                className={cn(
                  "inline-flex h-5 min-w-[20px] items-center justify-center rounded-md px-1.5 font-mono text-[0.64rem]",
                  isActive
                    ? "bg-[color:var(--color-surface-elevated)] text-[color:var(--color-text-primary)]"
                    : "bg-[color:var(--color-surface)] text-[color:var(--color-text-secondary)]"
                )}
                data-testid={`tasks-detail-tab-count-${item.id}`}
              >
                {item.count}
              </span>
            ) : null}
            {item.live ? (
              <span
                className="inline-flex h-5 items-center gap-1 rounded-md bg-[color:var(--color-accent-tint)] px-1.5 font-mono text-[0.6rem] uppercase tracking-[0.16em] text-[color:var(--color-accent)]"
                data-testid={`tasks-detail-tab-live-${item.id}`}
              >
                <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--color-accent)]" />
                Live
              </span>
            ) : null}
            {isActive ? (
              <span
                aria-hidden="true"
                className="absolute inset-x-0 -bottom-px h-0.5 rounded-full bg-[color:var(--color-accent)]"
              />
            ) : null}
          </button>
        );
      })}
    </div>
  );
}
