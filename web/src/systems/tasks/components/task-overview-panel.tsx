import { ChevronRight } from "lucide-react";

import { Button, DescriptionCard, JsonViewer, Markdown, Section } from "@agh/ui";

import type { TaskDetailView, TaskRun, TaskTimelineItem } from "../types";
import { TaskActivityItem } from "./task-activity-item";
import { TaskDependenciesSection } from "./task-dependencies-section";
import { TaskNowStrip, type TaskNowStripHandlers } from "./task-now-strip";
import { TaskSubtasksSection } from "./task-subtasks-section";

export const TASK_RESULT_ANCHOR_ID = "tasks-detail-result";

const RECENT_ACTIVITY_LIMIT = 5;

export interface TaskOverviewPanelProps {
  detail: TaskDetailView;
  runs: TaskRun[];
  timeline: TaskTimelineItem[];
  isLive: boolean;
  nowHandlers: TaskNowStripHandlers;
  nowPending?: Parameters<typeof TaskNowStrip>[0]["pending"];
  onViewAllActivity: () => void;
}

function TaskResultSection({ runs }: { runs: readonly TaskRun[] }) {
  const lastCompleted = [...runs]
    .filter(run => run.status === "completed")
    .sort((a, b) => b.attempt - a.attempt)[0];
  const result = lastCompleted?.result;
  if (result == null) return null;

  return (
    <Section data-testid="tasks-detail-result-section" id={TASK_RESULT_ANCHOR_ID} label="Result">
      {typeof result === "string" ? (
        <div className="rounded-lg border border-line bg-canvas-soft px-4 py-3.5">
          <Markdown>{result}</Markdown>
        </div>
      ) : (
        <JsonViewer data-testid="tasks-detail-result-json" value={result} />
      )}
    </Section>
  );
}

/**
 * Overview tab (§4.3): Now strip → result (terminal) → description → subtasks
 * → dependencies → recent activity. Sections self-suppress when empty so a
 * simple task reads as a single quiet column.
 */
export function TaskOverviewPanel({
  detail,
  runs,
  timeline,
  isLive,
  nowHandlers,
  nowPending,
  onViewAllActivity,
}: TaskOverviewPanelProps) {
  const record = detail.task;
  const children = detail.children ?? [];
  const dependencies = detail.dependency_references ?? [];
  const recent = timeline.slice(0, RECENT_ACTIVITY_LIMIT);
  const description = record.description?.trim();

  return (
    <div className="flex flex-col gap-6" data-testid="tasks-detail-overview">
      <TaskNowStrip detail={detail} handlers={nowHandlers} pending={nowPending} runs={runs} />

      {record.status === "completed" ? <TaskResultSection runs={runs} /> : null}

      <Section data-testid="tasks-detail-description" label="Description">
        {description ? (
          <DescriptionCard className="border border-line px-4 py-3.5">
            {description}
          </DescriptionCard>
        ) : (
          <p className="text-small-body text-subtle">No description.</p>
        )}
      </Section>

      <TaskSubtasksSection items={children} />
      <TaskDependenciesSection dependencies={dependencies} />

      {recent.length > 0 ? (
        <Section
          data-testid="tasks-detail-recent-activity"
          label="Recent activity"
          right={
            <Button
              className="-mr-1.5 h-auto px-1.5 py-0.5 text-eyebrow font-medium text-muted"
              data-testid="tasks-detail-view-all-activity"
              onClick={onViewAllActivity}
              size="sm"
              type="button"
              variant="ghost"
            >
              View all
              <ChevronRight aria-hidden="true" className="size-3" />
            </Button>
          }
        >
          <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
            {recent.map(item => (
              <TaskActivityItem isLive={isLive} item={item} key={item.event_id} />
            ))}
          </div>
        </Section>
      ) : null}
    </div>
  );
}
