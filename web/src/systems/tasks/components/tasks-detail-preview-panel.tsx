import { Link } from "@tanstack/react-router";
import { AlertCircle, Radio } from "lucide-react";

import { BlockLoading, Button, CodeBlock, Metric, Pill, Section } from "@agh/ui";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";

import {
  formatRelativeTime,
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
 * route is active. Composes `Pill.Dot`, `Pill`, `Metric`, `Section`, and a
 * `CodeBlock` preview of the task prompt + scope + agent.
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
        className="flex flex-1 items-center justify-center px-6 py-10 text-sm text-(--color-text-tertiary)"
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
        className="flex flex-1 items-center justify-center"
        data-testid="tasks-detail-preview-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-(--color-danger)" />
          <p className="text-sm text-(--color-text-tertiary)">{errorMessage}</p>
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
      className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto bg-(--color-canvas) px-6 py-5"
      data-testid="tasks-detail-preview-panel"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <Pill.Dot tone={signal.tone} pulse={signal.pulse} />
            <h2
              className="truncate text-ui-title-lg font-semibold tracking-tight text-(--color-text-primary)"
              data-testid="tasks-detail-preview-title"
            >
              {record.title}
            </h2>
            <Pill mono>{identifier}</Pill>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-small-body text-(--color-text-secondary)">
            <Pill tone={pillToneFromLegacyTone(taskStatusTone(record.status))}>
              {taskStatusLabel(record.status)}
            </Pill>
            <Pill
              data-testid="tasks-detail-preview-lifecycle"
              title={taskLifecyclePhaseDescription(lifecyclePhase)}
              tone={pillToneFromLegacyTone(taskLifecyclePhaseTone(lifecyclePhase))}
            >
              {taskLifecyclePhaseLabel(lifecyclePhase)}
            </Pill>
            {record.priority ? (
              <Pill tone={pillToneFromLegacyTone(taskPriorityTone(record.priority))}>
                {taskPriorityLabel(record.priority)}
              </Pill>
            ) : null}
            {taskHasApprovalPending(record) ? (
              <Pill tone="accent">{taskApprovalStateLabel(record.approval_state)}</Pill>
            ) : null}
            {channelLabel ? (
              <Pill
                data-testid="tasks-detail-preview-coordination"
                title="Coordination channel is bound to the active run. Channel messages support coordination only -- task ownership stays in the task service."
                tone={pillToneFromLegacyTone("violet")}
              >
                <span className="inline-flex items-center gap-1">
                  <Radio className="size-3" aria-hidden="true" />
                  Channel: {channelLabel}
                </span>
              </Pill>
            ) : null}
            <span>Owner {ownerLabel}</span>
            <span>· Scope {record.scope}</span>
            <span>· Updated {formatRelativeTime(record.updated_at)}</span>
          </div>
        </div>
        <div
          className="flex shrink-0 flex-wrap items-center gap-2"
          data-testid="tasks-detail-preview-actions"
        >
          <Link
            data-testid="tasks-detail-preview-edit-link"
            params={{ id: record.id }}
            to="/tasks/$id/edit"
          >
            <Button size="sm" type="button" variant="outline">
              Edit
            </Button>
          </Link>
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
              variant="outline"
            >
              {startCopy.label}
            </Button>
          ) : null}
          {canCancel && onCancelTask ? (
            <Button
              data-testid="tasks-detail-preview-cancel"
              onClick={() => onCancelTask(record.id)}
              size="sm"
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
          ) : null}
        </div>
      </header>

      <p
        className="text-xs text-(--color-text-tertiary)"
        data-testid="tasks-detail-preview-lifecycle-hint"
      >
        {taskLifecyclePhaseDescription(lifecyclePhase)}
      </p>

      <div className="grid gap-4 md:grid-cols-3">
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
        <Metric
          data-testid="tasks-detail-preview-counts-runs"
          label="Runs"
          value={runs.length}
          tone={runs.length > 0 ? "accent" : "default"}
        />
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
          <p className="whitespace-pre-wrap text-small-body leading-relaxed text-(--color-text-secondary)">
            {description}
          </p>
        ) : (
          <p className="text-small-body italic text-(--color-text-tertiary)">
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
