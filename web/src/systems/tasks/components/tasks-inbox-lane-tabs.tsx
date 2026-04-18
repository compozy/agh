import { cn } from "@/lib/utils";

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
      className="flex flex-wrap items-center gap-1.5 border-b border-[color:var(--color-divider)] px-4 py-2.5"
      data-testid="tasks-inbox-lane-tabs"
      role="tablist"
    >
      <LaneTab
        active={value === "all"}
        count={inbox?.total ?? 0}
        label="All"
        onSelect={() => onChange("all")}
        testId="tasks-inbox-lane-all"
        unread={inbox?.unread_total ?? 0}
      />
      {lanes.map(lane => {
        const counts = groupCounts.get(lane);
        const label = taskInboxLaneLabel(lane);

        return (
          <LaneTab
            active={value === lane}
            count={counts?.count ?? 0}
            key={lane}
            label={label}
            onSelect={() => onChange(lane)}
            testId={`tasks-inbox-lane-${lane}`}
            unread={counts?.unread ?? 0}
          />
        );
      })}
    </div>
  );
}

interface LaneTabProps {
  label: string;
  count: number;
  unread: number;
  active: boolean;
  onSelect: () => void;
  testId: string;
}

function LaneTab({ label, count, unread, active, onSelect, testId }: LaneTabProps) {
  return (
    <button
      aria-selected={active}
      className={cn(
        "inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 font-mono text-[0.64rem] uppercase tracking-[0.14em] transition-colors",
        active
          ? "border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-text-primary)]"
          : "border-transparent text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-divider)] hover:text-[color:var(--color-text-primary)]"
      )}
      data-testid={testId}
      onClick={onSelect}
      role="tab"
      type="button"
    >
      <span>{label}</span>
      <span
        className="rounded-md bg-[color:var(--color-surface)] px-1.5 py-0.5 text-[0.58rem] font-semibold text-[color:var(--color-text-primary)]"
        data-testid={`${testId}-count`}
      >
        {count}
      </span>
      {unread > 0 ? (
        <span
          className="size-1.5 rounded-full bg-[color:var(--color-warning)]"
          data-testid={`${testId}-unread`}
        />
      ) : null}
    </button>
  );
}
