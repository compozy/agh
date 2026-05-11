import {
  TasksBridgeNotificationsCard,
  type TasksBridgeNotificationsCardProps,
} from "./tasks-bridge-notifications-card";
import {
  TasksExecutionProfileCard,
  type TasksExecutionProfileCardProps,
} from "./tasks-execution-profile-card";
import { TasksReviewsCard, type TasksReviewsCardProps } from "./tasks-reviews-card";
import { TasksStreamResumeCard, type TasksStreamResumeCardProps } from "./tasks-stream-resume-card";

export interface TasksDetailOrchestrationPanelProps {
  profile: TasksExecutionProfileCardProps;
  reviews: TasksReviewsCardProps;
  notifications: TasksBridgeNotificationsCardProps;
  stream: TasksStreamResumeCardProps;
}

/**
 * Orchestration tab — stacks the four operational cards (execution profile,
 * reviews, bridge notifications, stream resume) with a single rhythmic gap.
 * Each child card carries its own framing; this panel only owns vertical
 * spacing and the outer page padding so the cards align with the rest of the
 * task detail tabs.
 */
export function TasksDetailOrchestrationPanel({
  profile,
  reviews,
  notifications,
  stream,
}: TasksDetailOrchestrationPanelProps) {
  return (
    <div
      className="flex w-full flex-col gap-6 px-6 py-5"
      data-testid="tasks-detail-orchestration-panel"
    >
      <TasksExecutionProfileCard {...profile} />
      <TasksReviewsCard {...reviews} />
      <TasksBridgeNotificationsCard {...notifications} />
      <TasksStreamResumeCard {...stream} />
    </div>
  );
}
