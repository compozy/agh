import { Activity, AlertCircle } from "lucide-react";
import { useState } from "react";

import {
  BlockLoading,
  Button,
  Empty,
  LiveBadge,
  PillGroup,
  Section,
  type PillGroupItem,
} from "@agh/ui";

import {
  humanizeTaskEvent,
  matchesActivityFilter,
  TASK_ACTIVITY_FILTERS,
  type TaskActivityFilter,
} from "../lib/task-activity-copy";
import type { TaskTimelineItem } from "../types";
import { TaskActivityItem } from "./task-activity-item";

export interface TaskActivityPanelProps {
  items: TaskTimelineItem[];
  isLoading?: boolean;
  errorMessage?: string | null;
  isLive?: boolean;
  onLoadMore?: () => void;
  canLoadMore?: boolean;
}

const FILTER_LABEL: Record<TaskActivityFilter, string> = {
  all: "All",
  runs: "Runs",
  reviews: "Reviews",
  changes: "Changes",
};

const FILTER_ITEMS: ReadonlyArray<PillGroupItem<TaskActivityFilter>> = TASK_ACTIVITY_FILTERS.map(
  value => ({
    value,
    label: FILTER_LABEL[value],
    testId: `tasks-activity-filter-${value}`,
  })
);

function emptyCopyFor(filter: TaskActivityFilter): { title: string; description: string } {
  if (filter === "reviews") {
    return {
      title: "No review activity yet",
      description: "Reviews appear here when a review policy is on.",
    };
  }
  return {
    title: "No activity recorded yet",
    description: "Events land here as this task moves through its runs.",
  };
}

/**
 * Activity tab (§4.6): humanized newest-first feed with one filter group
 * (filtering replaces the old grouping modes) and the page's single LiveBadge
 * while the task stream is attached.
 */
export function TaskActivityPanel({
  items,
  isLoading = false,
  errorMessage = null,
  isLive = false,
  onLoadMore,
  canLoadMore = false,
}: TaskActivityPanelProps) {
  const [filter, setFilter] = useState<TaskActivityFilter>("all");

  if (isLoading && items.length === 0) {
    return (
      <BlockLoading
        label="Loading activity"
        size="md"
        surface="bare"
        data-testid="tasks-activity-loading"
      />
    );
  }

  if (errorMessage && items.length === 0) {
    return (
      <Empty
        icon={AlertCircle}
        title="Couldn't load activity"
        description={errorMessage}
        data-testid="tasks-activity-error"
      />
    );
  }

  if (items.length === 0) {
    return (
      <Empty
        icon={Activity}
        title="No activity recorded yet"
        description="Events land here as this task moves through its runs."
        data-testid="tasks-activity-empty"
      />
    );
  }

  const visible = items.filter(item => matchesActivityFilter(humanizeTaskEvent(item), filter));
  const emptyCopy = emptyCopyFor(filter);

  return (
    <Section
      aria-label="Task activity"
      data-testid="tasks-activity-panel"
      right={isLive ? <LiveBadge data-testid="tasks-activity-live" /> : undefined}
      tabs={
        <PillGroup<TaskActivityFilter>
          aria-label="Filter activity"
          data-testid="tasks-activity-filters"
          items={FILTER_ITEMS}
          onChange={setFilter}
          size="sm"
          value={filter}
        />
      }
    >
      <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
        {visible.length === 0 ? (
          <p
            className="px-4 py-4 text-small-body text-muted"
            data-testid="tasks-activity-filter-empty"
          >
            {emptyCopy.title}. {emptyCopy.description}
          </p>
        ) : (
          visible.map(item => <TaskActivityItem isLive={isLive} item={item} key={item.event_id} />)
        )}
      </div>
      {canLoadMore && onLoadMore ? (
        <div className="flex items-center justify-center pt-1">
          <Button
            data-testid="tasks-activity-load-more"
            onClick={onLoadMore}
            size="sm"
            type="button"
            variant="neutral"
          >
            Load more
          </Button>
        </div>
      ) : null}
    </Section>
  );
}
