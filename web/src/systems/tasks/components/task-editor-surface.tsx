import { Link } from "@tanstack/react-router";
import { ArrowLeft, Loader2, Pencil, Plus } from "lucide-react";

import {
  Button,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  Pill,
  NativeSelect,
  NativeSelectOption,
  PageHeader,
  PillGroup,
  Section,
  type PillGroupItem,
  Textarea,
} from "@agh/ui";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import { pillToneFromLegacyTone } from "@/lib/pill-variant";
import type { TaskOwnerKind, TaskPriority, TaskRecord, TaskScope } from "../types";
import { TASK_TEMPLATES, type TaskTemplate, type TaskTemplateId } from "../lib/task-templates";
import { taskStatusLabel, taskStatusSignal, taskStatusTone } from "../lib/task-formatters";
import { useTasksCreateModalForm } from "./use-tasks-create-modal-form";

const PRIORITY_OPTIONS: PillGroupItem<TaskPriority>[] = [
  { value: "low", label: "Low", testId: "task-editor-priority-low" },
  { value: "medium", label: "Medium", testId: "task-editor-priority-medium" },
  { value: "high", label: "High", testId: "task-editor-priority-high" },
  { value: "urgent", label: "Urgent", testId: "task-editor-priority-urgent" },
];

const OWNER_KIND_OPTIONS: TaskOwnerKind[] = [
  "agent_session",
  "human",
  "automation",
  "extension",
  "network_peer",
  "pool",
];

const APPROVAL_OPTIONS: PillGroupItem<"none" | "manual">[] = [
  { value: "none", label: "No approval", testId: "task-editor-approval-none" },
  {
    value: "manual",
    label: "Human-in-the-loop",
    testId: "task-editor-approval-manual",
  },
];

const ATTEMPT_ITEMS: PillGroupItem<string>[] = [
  { value: "1", label: "1", testId: "task-editor-attempts-1" },
  { value: "2", label: "2", testId: "task-editor-attempts-2" },
  { value: "3", label: "3", testId: "task-editor-attempts-3" },
  { value: "5", label: "5", testId: "task-editor-attempts-5" },
  { value: "default", label: "Default", testId: "task-editor-attempts-default" },
];

interface TaskEditorSurfaceProps {
  mode: "create" | "edit";
  draft: CreateTaskDraftInput;
  onDraftChange: (
    next: CreateTaskDraftInput | ((current: CreateTaskDraftInput) => CreateTaskDraftInput)
  ) => void;
  onSubmit: (draft: CreateTaskDraftInput, asDraft: boolean) => Promise<unknown> | void;
  canSubmit?: boolean;
  isSubmitting?: boolean;
  workspaceName?: string | null;
  templateId?: TaskTemplateId;
  template?: TaskTemplate;
  onTemplateChange?: (templateId: TaskTemplateId) => void;
  task?: TaskRecord | null;
}

export function TaskEditorSurface({
  mode,
  draft,
  onDraftChange,
  onSubmit,
  canSubmit = true,
  isSubmitting = false,
  workspaceName,
  templateId,
  template,
  onTemplateChange,
  task = null,
}: TaskEditorSurfaceProps) {
  const form = useTasksCreateModalForm({ draft, onDraftChange, onSubmit });

  const isCreateMode = mode === "create";
  const title = isCreateMode ? "New task" : "Edit task";
  const description = isCreateMode
    ? `Start from ${template?.label ?? "the default"} template and stage the contract before it enters the queue.`
    : "Change the mutable task fields here. Scope, parent, and identifier stay visible as task context.";
  const noticeText =
    isCreateMode && template
      ? (template.preview.notice ??
        (template.preview.enqueueOnSubmit
          ? "This template enqueues its first run as soon as the task is created."
          : "This template creates a draft first so you can publish it later."))
      : "Changes are saved directly to the task record and become visible across the list, detail, inbox, and dashboard views.";

  const scopeItems: PillGroupItem<TaskScope>[] = [
    {
      value: "workspace",
      label: workspaceName ? `Workspace · ${workspaceName}` : "Workspace",
      testId: "task-editor-scope-workspace",
    },
    { value: "global", label: "Global", testId: "task-editor-scope-global" },
  ];

  const templateItems: PillGroupItem<TaskTemplateId>[] = TASK_TEMPLATES.map(option => ({
    value: option.id,
    label: option.label,
    testId: `task-editor-template-${option.id}`,
  }));

  const attemptsValue = draft.maxAttempts === null ? "default" : String(draft.maxAttempts);
  const signal = task ? taskStatusSignal(task.status) : null;

  const headerMeta = task ? (
    <div className="flex flex-wrap items-center gap-2 text-[13px] text-[color:var(--color-text-secondary)]">
      {task.identifier ? <Pill mono>{task.identifier}</Pill> : null}
      <Pill tone={pillToneFromLegacyTone(taskStatusTone(task.status))}>
        {taskStatusLabel(task.status)}
      </Pill>
      <span>Scope {task.scope}</span>
      {task.parent_task_id ? <span>· Parent {task.parent_task_id}</span> : null}
    </div>
  ) : null;

  return (
    <section
      className="flex min-h-0 flex-1 flex-col bg-[color:var(--color-canvas)]"
      data-testid="task-editor-surface"
    >
      <nav
        aria-label="Breadcrumb"
        className="flex items-center gap-2 border-b border-[color:var(--color-divider)] px-6 py-2 font-mono text-[11px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
      >
        {task ? (
          <Link
            className="inline-flex items-center gap-1.5 hover:text-[color:var(--color-text-secondary)]"
            data-testid="task-editor-back-link"
            params={{ id: task.id }}
            to="/tasks/$id"
          >
            <ArrowLeft className="size-3" />
            Back to tasks
          </Link>
        ) : (
          <Link
            className="inline-flex items-center gap-1.5 hover:text-[color:var(--color-text-secondary)]"
            data-testid="task-editor-back-link"
            to="/tasks"
          >
            <ArrowLeft className="size-3" />
            Back to tasks
          </Link>
        )}
      </nav>

      <PageHeader
        icon={isCreateMode ? Plus : Pencil}
        meta={headerMeta}
        title={
          <span className="flex min-w-0 items-center gap-2">
            {signal ? <Pill.Dot pulse={signal.pulse} tone={signal.tone} /> : null}
            <span
              className="truncate text-[15px] font-semibold text-[color:var(--color-text-primary)]"
              data-testid="task-editor-title"
            >
              {title}
            </span>
          </span>
        }
      />

      <p className="border-b border-[color:var(--color-divider)] px-6 py-3 text-[13px] leading-6 text-[color:var(--color-text-secondary)]">
        {description}
      </p>

      <form className="flex min-h-0 flex-1 flex-col" onSubmit={form.submitForm}>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="grid gap-6 px-6 py-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.85fr)]">
            <div className="flex min-w-0 flex-col gap-6">
              <Section label="Task contract">
                <div className="flex flex-col gap-5 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
                  {isCreateMode && template && templateId && onTemplateChange ? (
                    <Field>
                      <FieldLabel data-testid="task-editor-template-label">Template</FieldLabel>
                      <PillGroup
                        aria-label="Task template"
                        className="w-full flex-wrap"
                        data-testid="task-editor-template-options"
                        items={templateItems}
                        onChange={onTemplateChange}
                        size="sm"
                        value={templateId}
                      />
                    </Field>
                  ) : null}

                  <Field>
                    <div className="flex items-center justify-between gap-3">
                      <FieldLabel
                        data-testid="task-editor-title-label"
                        htmlFor="task-editor-title-input"
                      >
                        Title
                      </FieldLabel>
                      <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
                        Required
                      </span>
                    </div>
                    <Input
                      className="h-10"
                      data-testid="task-editor-title-input"
                      id="task-editor-title-input"
                      onChange={form.updateText("title")}
                      placeholder="Generate API client for payments-v3"
                      required
                      value={draft.title}
                    />
                  </Field>

                  <Field>
                    <FieldLabel
                      data-testid="task-editor-description-label"
                      htmlFor="task-editor-description-input"
                    >
                      Description
                    </FieldLabel>
                    <FieldDescription>
                      Describe the expected outcome, constraints, and completion criteria.
                    </FieldDescription>
                    <Textarea
                      className="min-h-[168px]"
                      data-testid="task-editor-description-input"
                      id="task-editor-description-input"
                      onChange={form.updateText("description")}
                      placeholder="Describe the task contract for the agent."
                      value={draft.description}
                    />
                  </Field>

                  {isCreateMode ? (
                    <div className="grid gap-5 md:grid-cols-2">
                      <Field>
                        <FieldLabel data-testid="task-editor-scope-label">Scope</FieldLabel>
                        {workspaceName ? (
                          <FieldDescription>Workspace default: {workspaceName}</FieldDescription>
                        ) : null}
                        <PillGroup
                          aria-label="Task scope"
                          className="w-full flex-wrap"
                          items={scopeItems}
                          onChange={form.updateScope}
                          size="sm"
                          value={draft.scope}
                        />
                      </Field>

                      <Field>
                        <FieldLabel data-testid="task-editor-priority-label">Priority</FieldLabel>
                        <PillGroup
                          aria-label="Task priority"
                          className="w-full flex-wrap"
                          items={PRIORITY_OPTIONS}
                          onChange={form.updatePriority}
                          size="sm"
                          value={draft.priority}
                        />
                      </Field>
                    </div>
                  ) : null}
                </div>
              </Section>

              {isCreateMode ? (
                <Section label="Queue settings">
                  <div className="flex flex-col gap-5 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-5">
                    <div className="grid gap-5 md:grid-cols-2">
                      <Field>
                        <FieldLabel
                          data-testid="task-editor-owner-label"
                          htmlFor="task-editor-owner-kind"
                        >
                          Owner
                        </FieldLabel>
                        <NativeSelect
                          aria-label="Owner kind"
                          className="w-full"
                          data-testid="task-editor-owner-kind"
                          id="task-editor-owner-kind"
                          onChange={event =>
                            form.updateOwnerKind(event.target.value as TaskOwnerKind | "")
                          }
                          value={draft.ownerKind}
                        >
                          <NativeSelectOption value="">Unassigned</NativeSelectOption>
                          {OWNER_KIND_OPTIONS.map(kind => (
                            <NativeSelectOption key={kind} value={kind}>
                              {kind}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                        <Input
                          className="h-10"
                          data-testid="task-editor-owner-ref"
                          onChange={form.updateText("ownerRef")}
                          placeholder="Owner reference (for example: coder)"
                          value={draft.ownerRef}
                        />
                      </Field>

                      <Field>
                        <FieldLabel data-testid="task-editor-attempts-label">Attempts</FieldLabel>
                        <PillGroup
                          aria-label="Max attempts"
                          className="w-full flex-wrap"
                          items={ATTEMPT_ITEMS}
                          onChange={next => {
                            if (next === "default") {
                              form.updateMaxAttempts(null);
                              return;
                            }
                            form.updateMaxAttempts(Number(next));
                          }}
                          size="sm"
                          value={attemptsValue}
                        />
                      </Field>
                    </div>

                    <div className="grid gap-5 md:grid-cols-2">
                      <Field>
                        <FieldLabel data-testid="task-editor-approval-label">Approval</FieldLabel>
                        <PillGroup
                          aria-label="Approval policy"
                          className="w-full flex-wrap"
                          items={APPROVAL_OPTIONS}
                          onChange={form.updateApprovalPolicy}
                          size="sm"
                          value={draft.approvalPolicy}
                        />
                      </Field>

                      <Field>
                        <FieldLabel
                          data-testid="task-editor-parent-label"
                          htmlFor="task-editor-parent-input"
                        >
                          Parent task
                        </FieldLabel>
                        <Input
                          className="h-10"
                          data-testid="task-editor-parent-input"
                          id="task-editor-parent-input"
                          onChange={form.updateText("parentTaskId")}
                          placeholder="Search by identifier or task id"
                          value={draft.parentTaskId}
                        />
                      </Field>
                    </div>
                  </div>
                </Section>
              ) : null}
            </div>

            <div className="flex min-w-0 flex-col gap-6">
              <Section label={isCreateMode ? "Submission" : "Editable fields"}>
                <div className="flex flex-col gap-5 rounded-[var(--radius-diagram)] border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-5">
                  <p
                    className="text-[13px] leading-5 text-[color:var(--color-text-secondary)]"
                    data-testid="task-editor-notice"
                  >
                    {noticeText}
                  </p>

                  <Field>
                    <FieldLabel
                      data-testid="task-editor-network-label"
                      htmlFor="task-editor-network-input"
                    >
                      Network channel
                    </FieldLabel>
                    <Input
                      className="h-10"
                      data-testid="task-editor-network-input"
                      id="task-editor-network-input"
                      onChange={form.updateText("networkChannel")}
                      placeholder="ingress channel"
                      value={draft.networkChannel}
                    />
                  </Field>

                  {isCreateMode ? (
                    <Field>
                      <FieldLabel
                        data-testid="task-editor-identifier-label"
                        htmlFor="task-editor-identifier-input"
                      >
                        Identifier override
                      </FieldLabel>
                      <Input
                        className="h-10"
                        data-testid="task-editor-identifier-input"
                        id="task-editor-identifier-input"
                        onChange={form.updateText("identifier")}
                        placeholder="TASK-123"
                        value={draft.identifier}
                      />
                    </Field>
                  ) : null}

                  {!isCreateMode ? (
                    <>
                      <Field>
                        <FieldLabel data-testid="task-editor-priority-label">Priority</FieldLabel>
                        <PillGroup
                          aria-label="Task priority"
                          className="w-full flex-wrap"
                          items={PRIORITY_OPTIONS}
                          onChange={form.updatePriority}
                          size="sm"
                          value={draft.priority}
                        />
                      </Field>

                      <Field>
                        <FieldLabel data-testid="task-editor-approval-label">Approval</FieldLabel>
                        <PillGroup
                          aria-label="Approval policy"
                          className="w-full flex-wrap"
                          items={APPROVAL_OPTIONS}
                          onChange={form.updateApprovalPolicy}
                          size="sm"
                          value={draft.approvalPolicy}
                        />
                      </Field>

                      <Field>
                        <FieldLabel data-testid="task-editor-attempts-label">Attempts</FieldLabel>
                        <PillGroup
                          aria-label="Max attempts"
                          className="w-full flex-wrap"
                          items={ATTEMPT_ITEMS}
                          onChange={next => {
                            if (next === "default") {
                              form.updateMaxAttempts(null);
                              return;
                            }
                            form.updateMaxAttempts(Number(next));
                          }}
                          size="sm"
                          value={attemptsValue}
                        />
                      </Field>
                    </>
                  ) : null}

                  <div className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
                    <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                      Read-only context
                    </p>
                    <dl className="mt-3 space-y-2 text-[13px] text-[color:var(--color-text-secondary)]">
                      <ContextRow label="Scope" value={draft.scope} />
                      <ContextRow label="Parent task" value={draft.parentTaskId || "None"} />
                      <ContextRow label="Identifier" value={draft.identifier || "Auto-generated"} />
                      <ContextRow
                        label="Workspace"
                        value={workspaceName ?? "No active workspace"}
                      />
                    </dl>
                  </div>
                </div>
              </Section>
            </div>
          </div>
        </div>

        <footer className="border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-6 py-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-[13px] text-[color:var(--color-text-secondary)]">
              {isCreateMode
                ? "Review the contract before you enqueue work."
                : "Saving updates refreshes the list, detail, inbox, and dashboard views."}
            </p>

            <div className="flex flex-wrap items-center gap-2">
              {task ? (
                <Link params={{ id: task.id }} to="/tasks/$id">
                  <Button type="button" variant="outline">
                    Cancel
                  </Button>
                </Link>
              ) : (
                <Link to="/tasks">
                  <Button type="button" variant="outline">
                    Cancel
                  </Button>
                </Link>
              )}

              {isCreateMode ? (
                <Button
                  data-testid="task-editor-save-draft"
                  disabled={!canSubmit || isSubmitting}
                  onClick={form.submitDraft}
                  type="button"
                  variant="outline"
                >
                  {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
                  Save draft
                </Button>
              ) : null}

              <Button
                data-testid="task-editor-submit"
                disabled={!canSubmit || isSubmitting}
                type="submit"
              >
                {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
                {isCreateMode
                  ? template?.preview.enqueueOnSubmit
                    ? "Create & enqueue"
                    : "Create task"
                  : "Save changes"}
              </Button>
            </div>
          </div>
        </footer>
      </form>
    </section>
  );
}

function ContextRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </dt>
      <dd className="truncate text-[color:var(--color-text-primary)]">{value}</dd>
    </div>
  );
}
