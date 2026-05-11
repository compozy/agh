import { Link } from "@tanstack/react-router";
import { AlertCircle, Radio } from "lucide-react";

import {
  BlockLoading,
  Button,
  CodeBlock,
  DescriptionCard,
  Metric,
  MonoId,
  Pill,
  Section,
  Time,
} from "@agh/ui";

import {
  runCoordinationChannelLabel,
  runIsCoordinated,
  taskApprovalStateLabel,
  taskHandoffActionCopy,
  taskHasApprovalPending,
  taskIsDraft,
  taskLifecyclePhase,
  taskLifecyclePhaseDescription,
  taskLifecyclePhaseLabel,
  taskLifecyclePhaseTone,
  taskOwnerLabel,
  taskPriorityLabel,
  taskPriorityTone,
  taskShortId,
  taskStatusLabel,
  taskStatusSignal,
  taskStatusTone,
} from "../lib/task-formatters";
import type { TaskDetailView, TaskListItem, TaskRecord } from "../types";
import { TaskDeleteAction } from "./task-delete-action";

export interface TasksDetailPreviewPanelProps {
  task: TaskListItem | null;
  detail: TaskDetailView | null;
  isLoading?: boolean;
  errorMessage?: string | null;
  onDeleteTask?: (taskId: string) => void;
  onCancelTask?: (taskId: string) => void;
  onEnqueueRun?: (taskId: string) => void;
  onPublishTask?: (taskId: string) => void;
  isDeletePending?: boolean;
  isPublishPending?: boolean;
}

type PreviewRecord = Pick<
  TaskRecord,
  | "id"
  | "identifier"
  | "title"
  | "status"
  | "priority"
  | "approval_state"
  | "owner"
  | "scope"
  | "updated_at"
  | "created_by"
  | "origin"
> & { kind?: string | null };

/**
 * Inline preview rendered on `/tasks` when a list row is selected but no detail
 * route is active. Mirrors the detail-route hero anatomy at a tighter density:
 * status dot + 18 px title, MonoId/pill row, sentence meta line, Metric grid,
 * `<DescriptionCard>` and a code preview of the task scope + prompt.
 */
export function TasksDetailPreviewPanel({
  task,
  detail,
  isLoading = false,
  errorMessage = null,
  onDeleteTask,
  onCancelTask,
  onEnqueueRun,
  onPublishTask,
  isDeletePending = false,
  isPublishPending = false,
}: TasksDetailPreviewPanelProps) {
  if (!task) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-9 py-10 text-small-body text-subtle"
        data-testid="tasks-detail-preview-empty"
      >
        Select a task to inspect its overview.
      </div>
    );
  }

  if (isLoading && !detail) {
    return (
      <BlockLoading
        className="flex-1"
        label="Loading task preview"
        size="md"
        surface="bare"
        data-testid="tasks-detail-preview-loading"
      />
    );
  }

  if (errorMessage && !detail) {
    return (
      <div
        className="flex flex-1 items-center justify-center px-9 py-10"
        data-testid="tasks-detail-preview-error"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <AlertCircle className="size-5 text-danger" strokeWidth={1.75} />
          <p className="text-small-body text-muted">{errorMessage}</p>
        </div>
      </div>
    );
  }

  const record = (detail?.task ?? task) as PreviewRecord & {
    description?: string | null;
    draft?: boolean;
  };
  const childCount = detail?.children?.length ?? task.child_count ?? 0;
  const dependencyReferences = detail?.dependency_references ?? detail?.dependencies ?? [];
  const dependencyCount = dependencyReferences.length || (task.dependency_count ?? 0);
  const runs = detail?.runs ?? [];
  const isDraft = taskIsDraft(record);
  const signal = taskStatusSignal(record.status);
  const identifier = taskShortId({ id: record.id, identifier: record.identifier });
  const description = detail?.task.description ?? null;
  const ownerLabel = taskOwnerLabel(record.owner);
  const previewLanguage =
    (record as { kind?: string | null }).kind === "yaml" ? "yaml" : "markdown";
  const previewCode = buildPreviewCode({
    record,
    description,
    ownerLabel,
  });
  const canCancel =
    record.status === "ready" || record.status === "in_progress" || record.status === "blocked";
  const activeRun =
    detail?.summary?.active_run ?? (task.active_run as TaskListItem["active_run"] | null);
  const lifecyclePhase = taskLifecyclePhase({
    status: record.status,
    approval_state: record.approval_state,
    draft: record.draft,
    active_run: activeRun ?? null,
  });
  const channelLabel = runIsCoordinated(activeRun) ? runCoordinationChannelLabel(activeRun) : null;
  const publishCopy = taskHandoffActionCopy("publish");
  const startCopy = taskHandoffActionCopy("start");

  return (
    <section
      className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto bg-canvas px-9 py-7"
      data-testid="tasks-detail-preview-panel"
    >
      <header className="flex min-w-0 flex-wrap items-start gap-4">
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
            <h2
              className="min-w-0 truncate text-empty-h1 font-medium leading-tight tracking-empty-h1 text-fg-strong"
              data-testid="tasks-detail-preview-title"
              style={{ fontWeight: 510 }}
            >
              {record.title}
            </h2>
          </div>
          <div className="flex min-w-0 flex-wrap items-center gap-1.5">
            <MonoId data-testid="tasks-detail-preview-id" value={identifier} />
            <Pill tone={taskStatusTone(record.status)}>{taskStatusLabel(record.status)}</Pill>
            <Pill
              data-testid="tasks-detail-preview-lifecycle"
              title={taskLifecyclePhaseDescription(lifecyclePhase)}
              tone={lifecyclePhase === "running" ? "info" : taskLifecyclePhaseTone(lifecyclePhase)}
            >
              {taskLifecyclePhaseLabel(lifecyclePhase)}
            </Pill>
            {record.priority ? (
              <Pill tone={taskPriorityTone(record.priority)}>
                {taskPriorityLabel(record.priority)}
              </Pill>
            ) : null}
            {taskHasApprovalPending(record) ? (
              <Pill tone="warning">{taskApprovalStateLabel(record.approval_state)}</Pill>
            ) : null}
            {channelLabel ? (
              <Pill
                data-testid="tasks-detail-preview-coordination"
                title="Coordination channel is bound to the active run. Channel messages support coordination only -- task ownership stays in the task service."
                tone="info"
              >
                <span className="inline-flex items-center gap-1">
                  <Radio className="size-3" aria-hidden="true" strokeWidth={1.75} />
                  Channel: {channelLabel}
                </span>
              </Pill>
            ) : null}
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted">
            <span>Owner {ownerLabel}</span>
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span>Scope {record.scope}</span>
            <span aria-hidden="true" className="text-faint">
              ·
            </span>
            <span className="inline-flex items-center gap-1">
              Updated <Time iso={record.updated_at} mode="relative" />
            </span>
          </div>
        </div>
        <div
          className="flex shrink-0 flex-wrap items-center gap-2"
          data-testid="tasks-detail-preview-actions"
        >
          <Link params={{ id: record.id }} to="/tasks/$id/edit">
            <Button
              data-testid="tasks-detail-preview-edit-link"
              size="sm"
              type="button"
              variant="neutral"
            >
              Edit
            </Button>
          </Link>
          {canCancel && onCancelTask ? (
            <Button
              data-testid="tasks-detail-preview-cancel"
              onClick={() => onCancelTask(record.id)}
              size="sm"
              type="button"
              variant="neutral"
            >
              Cancel
            </Button>
          ) : null}
          {onDeleteTask ? (
            <TaskDeleteAction
              taskId={record.id}
              taskTitle={record.title}
              onDelete={onDeleteTask}
              isPending={isDeletePending}
              triggerTestId="tasks-detail-preview-delete"
              dialogTestId="tasks-detail-preview-delete-dialog"
              cancelTestId="tasks-detail-preview-delete-cancel"
              confirmTestId="tasks-detail-preview-delete-confirm"
            />
          ) : null}
          {isDraft && onPublishTask ? (
            <Button
              data-testid="tasks-detail-preview-publish"
              disabled={isPublishPending}
              onClick={() => onPublishTask(record.id)}
              size="sm"
              title={publishCopy.tooltip}
              type="button"
            >
              {publishCopy.label}
            </Button>
          ) : null}
          {!isDraft && onEnqueueRun ? (
            <Button
              data-testid="tasks-detail-preview-enqueue"
              onClick={() => onEnqueueRun(record.id)}
              size="sm"
              title={startCopy.tooltip}
              type="button"
            >
              {startCopy.label}
            </Button>
          ) : null}
        </div>
      </header>

      <div className="grid gap-3 md:grid-cols-3">
        <Metric
          data-testid="tasks-detail-preview-counts-children"
          label="Children"
          value={childCount}
        />
        <Metric
          data-testid="tasks-detail-preview-counts-deps"
          label="Dependencies"
          value={dependencyCount}
        />
        <Metric data-testid="tasks-detail-preview-counts-runs" label="Runs" value={runs.length} />
      </div>

      <Section
        data-testid="tasks-detail-preview-overview"
        label="Overview"
        right={
          <Pill.Link
            data-testid="tasks-detail-preview-deeplink"
            render={<Link params={{ id: record.id }} to="/tasks/$id" />}
          >
            Open detail
          </Pill.Link>
        }
      >
        {description ? (
          <DescriptionCard data-testid="tasks-detail-preview-description">
            {description}
          </DescriptionCard>
        ) : (
          <p className="text-small-body italic text-subtle">
            No description provided yet. Open the full detail view to inspect timeline, runs, and
            dependencies.
          </p>
        )}
      </Section>

      <Section data-testid="tasks-detail-preview-preview" label="Preview">
        <CodeBlock
          code={previewCode}
          copyable={false}
          data-testid="tasks-detail-preview-code"
          language={previewLanguage}
          showPrompt={false}
        />
      </Section>
    </section>
  );
}

interface BuildPreviewCodeParams {
  record: PreviewRecord & { description?: string | null };
  description: string | null;
  ownerLabel: string;
}

function buildPreviewCode({ record, description, ownerLabel }: BuildPreviewCodeParams): string {
  const scope = record.scope ?? "workspace";
  const owner = ownerLabel.toLowerCase().replace(/\s+/g, "-");
  const origin = record.origin?.kind ?? "unknown";
  const prompt = (description ?? "").trim() || "--";
  return [
    `# scope    ${scope}`,
    `# owner    ${owner}`,
    `# origin   ${origin}`,
    `# prompt`,
    prompt,
  ].join("\n");
}
