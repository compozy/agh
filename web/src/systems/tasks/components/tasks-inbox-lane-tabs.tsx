import { Tabs, TabsList, TabsTrigger } from "@agh/ui";

import type { InboxLaneFilter } from "@/hooks/routes/use-tasks-page";

import { taskInboxLaneLabel } from "../lib/task-formatters";
import type { TaskInboxLane, TaskInboxView } from "../types";

export interface TasksInboxLaneTabsProps {
  inbox: TaskInboxView | null;
  value: InboxLaneFilter;
  onChange: (next: InboxLaneFilter) => void;
  showArchive?: boolean;
}

const LANE_ORDER: TaskInboxLane[] = ["my_work", "approvals", "failed_runs", "blocked", "archived"];

/**
 * Lane tabs for the Inbox. Counts render as muted inline `(N)` text beside the
 * label — no bg-colored count pills, no leading StatusDots. Unread totals are
 * reflected at the row level via the accent left-rail, not on the tab itself.
 */
export function TasksInboxLaneTabs({
  inbox,
  value,
  onChange,
  showArchive = true,
}: TasksInboxLaneTabsProps) {
  const groupCounts = new Map<TaskInboxLane, { count: number; unread: number }>();
  for (const group of inbox?.groups ?? []) {
    groupCounts.set(group.lane, { count: group.count, unread: group.unread_count });
  }

  const lanes: TaskInboxLane[] = LANE_ORDER.filter(lane => showArchive || lane !== "archived");

  return (
    <div
      className="border-b border-[color:var(--color-divider)] px-4 py-2.5"
      data-testid="tasks-inbox-lane-tabs"
    >
      <Tabs
        onValueChange={next => onChange(next as InboxLaneFilter)}
        orientation="horizontal"
        value={value}
      >
        <TabsList className="h-8 overflow-x-auto" variant="line">
          <TabsTrigger
            className="flex-none gap-1.5 font-mono text-[11px] uppercase tracking-[0.12em]"
            data-testid="tasks-inbox-lane-all"
            value="all"
          >
            <span>All</span>
            <LaneCount testId="tasks-inbox-lane-all-count" value={inbox?.total ?? 0} />
          </TabsTrigger>
          {lanes.map(lane => {
            const counts = groupCounts.get(lane);
            const label = taskInboxLaneLabel(lane);
            return (
              <TabsTrigger
                className="flex-none gap-1.5 font-mono text-[11px] uppercase tracking-[0.12em]"
                data-testid={`tasks-inbox-lane-${lane}`}
                key={lane}
                value={lane}
              >
                <span>{label}</span>
                <LaneCount testId={`tasks-inbox-lane-${lane}-count`} value={counts?.count ?? 0} />
              </TabsTrigger>
            );
          })}
        </TabsList>
      </Tabs>
    </div>
  );
}

interface LaneCountProps {
  value: number;
  testId: string;
}

function LaneCount({ value, testId }: LaneCountProps) {
  return (
    <span
      className="font-mono text-[10px] text-[color:var(--color-text-tertiary)]"
      data-testid={testId}
    >
      ({value})
    </span>
  );
}
