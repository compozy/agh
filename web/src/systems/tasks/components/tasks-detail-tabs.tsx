import { LaneTabs, type LaneTabsItem } from "@agh/ui";

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
  const tabs: ReadonlyArray<LaneTabsItem<TaskDetailPanel>> = items.map(item => ({
    value: item.id,
    label: item.label,
    count: item.count,
    liveLabel: item.live ? "Live" : undefined,
    testId: `tasks-detail-tab-${item.id}`,
  }));

  return (
    <div className="border-b border-line px-9" data-testid="tasks-detail-tabs">
      <LaneTabs<TaskDetailPanel>
        ariaLabel="Task detail panels"
        className="border-b-0"
        items={tabs}
        onChange={onChange}
        value={active}
      />
    </div>
  );
}
