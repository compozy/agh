import { AlertCircle, ClipboardList } from "lucide-react";

import {
  Button,
  cn,
  Empty,
  LaneTabs,
  PAGE_CONTENT_GUTTER,
  Skeleton,
  TabsContent,
  type LaneTabsItem,
} from "@agh/ui";

import {
  TaskActivityPanel,
  TaskOverviewPanel,
  TaskPropertiesRail,
  TaskRunsPanel,
  TasksDetailSubhead,
  type TaskDetailSearch,
  type TaskDetailTab,
  type TaskRunReview,
} from "@/systems/tasks";

import { TaskDetailOverlays } from "./task-detail-overlays";
import { TaskDetailTopbar } from "./task-detail-topbar";
import { useTaskDetailLocation } from "./use-task-detail-location";

const TAB_ITEMS = (runCount: number): ReadonlyArray<LaneTabsItem<TaskDetailTab>> => [
  { value: "overview", label: "Overview", testId: "tasks-detail-tab-overview" },
  { value: "runs", label: "Runs", count: runCount, testId: "tasks-detail-tab-runs" },
  { value: "activity", label: "Activity", testId: "tasks-detail-tab-activity" },
];

function groupReviewsByRun(
  reviews: readonly TaskRunReview[]
): ReadonlyMap<string, readonly TaskRunReview[]> {
  const map = new Map<string, TaskRunReview[]>();
  for (const review of reviews) {
    if (!review.run_id) continue;
    const bucket = map.get(review.run_id);
    if (bucket) {
      bucket.push(review);
    } else {
      map.set(review.run_id, [review]);
    }
  }
  return map;
}

function TaskDetailLoading() {
  return (
    <div
      className={cn(PAGE_CONTENT_GUTTER, "flex min-h-0 flex-1 flex-col gap-4 py-5")}
      data-testid="tasks-detail-loading"
    >
      <Skeleton className="h-6 w-72" />
      <Skeleton className="h-10 w-full max-w-md" />
      <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div className="flex flex-col gap-3">
          <Skeleton className="h-20 rounded-lg" />
          <Skeleton className="h-32 rounded-lg" />
          <Skeleton className="h-40 rounded-lg" />
        </div>
        <Skeleton className="hidden h-80 rounded-lg lg:block" />
      </div>
    </div>
  );
}

export function TaskDetailLocation({
  taskId,
  rawSearch,
}: {
  taskId: string;
  rawSearch: TaskDetailSearch;
}) {
  const controller = useTaskDetailLocation(taskId, rawSearch);
  const { page, detail, record, command } = controller;

  if (page.detailLoading) {
    return <TaskDetailLoading />;
  }

  if (page.notFound || !detail || !record || !command) {
    return (
      <div
        className={cn(PAGE_CONTENT_GUTTER, "flex flex-1 items-center justify-center py-8")}
        data-testid="tasks-detail-not-found"
      >
        <Empty
          icon={AlertCircle}
          title="Task not found"
          description={page.fatalError?.message ?? `No task with id "${taskId}" in this workspace.`}
          action={
            <Button onClick={controller.backToTasks} size="sm" type="button" variant="ghost">
              <ClipboardList aria-hidden="true" className="size-3" />
              Back to tasks
            </Button>
          }
        />
      </div>
    );
  }

  const reviewsByRun = groupReviewsByRun(page.reviews);
  const canStartRun = command.primary?.kind === "start";

  return (
    <div
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
      data-testid="tasks-detail-content"
    >
      <TaskDetailTopbar controller={controller} />
      <TaskDetailOverlays controller={controller} />

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className={cn(PAGE_CONTENT_GUTTER, "@container pt-5 pb-20")}>
          <TasksDetailSubhead detail={detail} />
          <LaneTabs<TaskDetailTab>
            ariaLabel="Task views"
            className="gap-0"
            data-testid="tasks-detail-tabs"
            items={TAB_ITEMS(page.runs.length)}
            listClassName="w-full"
            onChange={controller.setTab}
            value={controller.search.tab}
          >
            <div className="grid items-start gap-8 pt-5 @min-[64rem]:grid-cols-[minmax(0,1fr)_320px]">
              <main className="min-w-0" data-testid="tasks-detail-panels">
                <TabsContent value="overview">
                  <TaskOverviewPanel
                    detail={detail}
                    isLive={page.isLive}
                    nowHandlers={{
                      onOpenRun: controller.openRun,
                      onOpenTask: controller.openTask,
                      onApprove: () => void page.handleApproveTask(),
                      onReject: () => void page.handleRejectTask(),
                      onResume: () => void page.handleResumeTask(),
                      onRecover: () => void page.handleRecoverTask(),
                      onClearBlock: blockId => void page.handleClearBlock(blockId),
                      onViewResult: controller.scrollToResult,
                    }}
                    nowPending={{
                      approve: page.isApprovePending,
                      reject: page.isRejectPending,
                      resume: page.isResumePending,
                      recover: page.isRecoverPending,
                      clearBlock: page.isClearBlockPending,
                    }}
                    onViewAllActivity={() => controller.setTab("activity")}
                    runs={page.runs}
                    timeline={page.timeline}
                  />
                </TabsContent>
                <TabsContent value="runs">
                  <TaskRunsPanel
                    errorMessage={page.runsError?.message ?? null}
                    isLoading={page.runsLoading}
                    onStartRun={canStartRun ? () => void page.handleEnqueueRun() : undefined}
                    reviewsByRun={reviewsByRun}
                    runs={page.runs}
                    taskId={taskId}
                    workerName={page.profile?.worker?.agent_name ?? null}
                  />
                </TabsContent>
                <TabsContent value="activity">
                  <TaskActivityPanel
                    canLoadMore={page.isTimelineSaturated}
                    errorMessage={page.timelineError?.message ?? null}
                    isLive={page.isLive}
                    isLoading={page.timelineLoading}
                    items={page.timeline}
                    onLoadMore={page.handleTimelineLoadMore}
                  />
                </TabsContent>
              </main>
              <aside className="min-w-0 @min-[64rem]:sticky @min-[64rem]:top-0">
                <TaskPropertiesRail
                  approvalPending={{
                    approve: page.isApprovePending,
                    reject: page.isRejectPending,
                  }}
                  detail={detail}
                  onApprove={() => void page.handleApproveTask()}
                  onEditSetup={() => controller.setSetupOpen(true)}
                  onInspect={() => controller.setInspectOpen(true)}
                  onReject={() => void page.handleRejectTask()}
                  profile={page.profile}
                  runs={page.runs}
                />
              </aside>
            </div>
          </LaneTabs>
        </div>
      </div>
    </div>
  );
}
