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

export function TasksDetailOrchestrationPanel({
  profile,
  reviews,
  notifications,
  stream,
}: TasksDetailOrchestrationPanelProps) {
  return (
    <div
      className="flex w-full flex-col gap-8 px-6 py-5"
      data-testid="tasks-detail-orchestration-panel"
    >
      <TasksExecutionProfileCard {...profile} />
      <TasksReviewsCard {...reviews} />
      <TasksBridgeNotificationsCard {...notifications} />
      <TasksStreamResumeCard {...stream} />
    </div>
  );
}
