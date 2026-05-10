import { Eyebrow, Item, ItemActions, ItemContent, ItemGroup, Pill, Section } from "@agh/ui";

import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import { formatPercent, taskStatusLabel, taskStatusTone } from "../lib/task-formatters";
import type { TaskDashboardView } from "../types";

export interface TasksDashboardStatusBreakdownProps {
  dashboard: TaskDashboardView;
}

export function TasksDashboardStatusBreakdown({ dashboard }: TasksDashboardStatusBreakdownProps) {
  const entries = dashboard.status_breakdown ?? [];
  const total = entries.reduce((sum, entry) => sum + entry.count, 0);

  return (
    <Section
      data-testid="tasks-dashboard-status-breakdown"
      label="Status breakdown"
      right={
        total > 0 ? (
          <Eyebrow data-testid="tasks-dashboard-status-breakdown-total">total {total}</Eyebrow>
        ) : undefined
      }
    >
      {entries.length === 0 ? (
        <p
          className="px-1 py-6 text-sm text-(--muted)"
          data-testid="tasks-dashboard-status-breakdown-empty"
        >
          No task activity yet.
        </p>
      ) : (
        <ItemGroup className="gap-2 pt-2">
          {entries.map(entry => (
            <Item
              className="justify-between gap-3 p-0"
              data-testid={`tasks-dashboard-status-row-${entry.status}`}
              key={entry.status}
            >
              <ItemContent className="flex-row">
                <Pill
                  data-testid={`tasks-dashboard-status-pill-${entry.status}`}
                  size="sm"
                  tone={pillToneFromLegacyTone(taskStatusTone(entry.status))}
                >
                  {taskStatusLabel(entry.status)}
                  <span
                    className="ml-1 opacity-80"
                    data-testid={`tasks-dashboard-status-count-${entry.status}`}
                  >
                    {entry.count}
                  </span>
                </Pill>
              </ItemContent>
              <ItemActions className="font-mono text-eyebrow text-(--subtle)">
                {formatPercent(entry.share_percent)}
              </ItemActions>
            </Item>
          ))}
        </ItemGroup>
      )}
    </Section>
  );
}
