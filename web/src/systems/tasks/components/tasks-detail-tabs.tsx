import { Tabs, TabsList, TabsTrigger } from "@agh/ui";

import type { TaskDetailPanel } from "@/hooks/routes/use-task-detail-page";

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
    <Tabs
      aria-label="Task detail panels"
      className="border-b border-line px-9"
      data-testid="tasks-detail-tabs"
      onValueChange={value => onChange(value as TaskDetailPanel)}
      value={active}
    >
      <TabsList className="h-10">
        {items.map(item => (
          <TabsTrigger
            count={item.count}
            className="gap-2"
            data-testid={`tasks-detail-tab-${item.id}`}
            key={item.id}
            liveLabel={item.live ? "Live" : undefined}
            value={item.id}
          >
            {item.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
