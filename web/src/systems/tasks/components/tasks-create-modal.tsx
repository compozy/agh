import { Loader2, Plus } from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  NativeSelect,
  NativeSelectOption,
  Pill,
  Pills,
  Section,
  type PillsItem,
  Textarea,
} from "@agh/ui";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import { pillVariantFromTone } from "@/lib/pill-variant";
import type { TaskOwnerKind, TaskPriority, TaskScope } from "../types";
import { TASK_TEMPLATES, type TaskTemplate, type TaskTemplateId } from "../lib/task-templates";
import { useTasksCreateModalForm } from "./use-tasks-create-modal-form";

const PRIORITY_OPTIONS: PillsItem<TaskPriority>[] = [
  { value: "low", label: "Low", testId: "tasks-create-modal-priority-low" },
  { value: "medium", label: "Medium", testId: "tasks-create-modal-priority-medium" },
  { value: "high", label: "High", testId: "tasks-create-modal-priority-high" },
  { value: "urgent", label: "Urgent", testId: "tasks-create-modal-priority-urgent" },
];

const OWNER_KIND_OPTIONS: TaskOwnerKind[] = [
  "agent_session",
  "human",
  "automation",
  "extension",
  "network_peer",
  "pool",
];

const ATTEMPT_VALUES = ["1", "2", "3", "5", "default"] as const;
type AttemptValue = (typeof ATTEMPT_VALUES)[number];

const APPROVAL_OPTIONS: PillsItem<"none" | "manual">[] = [
  { value: "none", label: "No approval", testId: "tasks-create-modal-approval-none" },
  {
    value: "manual",
    label: "Human-in-the-loop",
    testId: "tasks-create-modal-approval-manual",
  },
];

export interface TasksCreateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: TaskTemplate;
  templateId: TaskTemplateId;
  onTemplateChange: (templateId: TaskTemplateId) => void;
  draft: CreateTaskDraftInput;
  onDraftChange: (
    next: CreateTaskDraftInput | ((current: CreateTaskDraftInput) => CreateTaskDraftInput)
  ) => void;
  onSubmit: (draft: CreateTaskDraftInput, asDraft: boolean) => Promise<unknown> | void;
  workspaceName?: string | null;
  isSubmitting?: boolean;
  canSubmit?: boolean;
}

export function TasksCreateModal({
  open,
  onOpenChange,
  template,
  templateId,
  onTemplateChange,
  draft,
  onDraftChange,
  onSubmit,
  workspaceName,
  isSubmitting = false,
  canSubmit = true,
}: TasksCreateModalProps) {
  const form = useTasksCreateModalForm({ draft, onDraftChange, onSubmit });

  const noticeText =
    template.preview.notice ??
    (template.preview.enqueueOnSubmit
      ? "Will enqueue 1 run immediately on submit."
      : "Saves a draft. You can publish it later from the task list.");

  const templateItems: PillsItem<TaskTemplateId>[] = TASK_TEMPLATES.map(option => ({
    value: option.id,
    label: option.label,
    testId: `tasks-create-modal-template-${option.id}`,
  }));

  const scopeItems: PillsItem<TaskScope>[] = [
    {
      value: "workspace",
      label: workspaceName ? `Workspace · ${workspaceName}` : "Workspace",
      testId: "tasks-create-modal-scope-workspace",
    },
    { value: "global", label: "Global", testId: "tasks-create-modal-scope-global" },
  ];

  const attemptsValue: AttemptValue =
    draft.maxAttempts === null ? "default" : (String(draft.maxAttempts) as AttemptValue);

  const attemptItems: PillsItem<AttemptValue>[] = [
    { value: "1", label: "1", testId: "tasks-create-modal-attempts-1" },
    { value: "2", label: "2", testId: "tasks-create-modal-attempts-2" },
    { value: "3", label: "3", testId: "tasks-create-modal-attempts-3" },
    { value: "5", label: "5", testId: "tasks-create-modal-attempts-5" },
    { value: "default", label: "Default", testId: "tasks-create-modal-attempts-default" },
  ];

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="gap-0 p-0 text-[color:var(--color-text-primary)] sm:max-w-[38rem]"
        data-testid="tasks-create-modal"
      >
        <DialogHeader className="flex-row items-center gap-3 border-b border-[color:var(--color-divider)] px-5 py-4">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
            <Plus className="size-4" />
          </span>
          <div className="flex min-w-0 flex-col gap-0.5">
            <DialogTitle>New task</DialogTitle>
            <DialogDescription data-testid="tasks-create-modal-template-label">
              Starting from {template.label} template
            </DialogDescription>
          </div>
        </DialogHeader>

        <form className="flex max-h-[min(85vh,960px)] flex-col" onSubmit={form.submitForm}>
          <div className="flex-1 space-y-5 overflow-y-auto px-5 py-5">
            <Section label="Template">
              <Pills
                aria-label="Task template"
                className="flex-wrap"
                data-testid="tasks-create-modal-template-pills"
                items={templateItems}
                onChange={next => onTemplateChange(next)}
                size="sm"
                value={templateId}
              />
            </Section>

            <Field>
              <div className="flex items-center justify-between gap-3">
                <FieldLabel htmlFor="tasks-create-title">Title</FieldLabel>
                <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
                  Required
                </span>
              </div>
              <Input
                className="h-10"
                data-testid="tasks-create-modal-title"
                id="tasks-create-title"
                onChange={form.updateText("title")}
                placeholder="e.g. Generate API client for payments-v3"
                required
                value={draft.title}
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="tasks-create-description">Description</FieldLabel>
              <Textarea
                className="min-h-[96px]"
                data-testid="tasks-create-modal-description"
                id="tasks-create-description"
                onChange={form.updateText("description")}
                placeholder="Describe the task contract for the agent."
                value={draft.description}
              />
            </Field>

            <div className="grid gap-5 md:grid-cols-2">
              <Field>
                <FieldLabel>Scope</FieldLabel>
                <Pills
                  aria-label="Task scope"
                  className="w-full flex-wrap"
                  items={scopeItems}
                  onChange={form.updateScope}
                  size="sm"
                  value={draft.scope}
                />
              </Field>

              <Field>
                <FieldLabel>Priority</FieldLabel>
                <Pills
                  aria-label="Task priority"
                  className="w-full flex-wrap"
                  data-testid="tasks-create-modal-priorities"
                  items={PRIORITY_OPTIONS}
                  onChange={form.updatePriority}
                  size="sm"
                  value={draft.priority}
                />
              </Field>
            </div>

            <div className="grid gap-5 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="tasks-create-owner-kind">Owner</FieldLabel>
                <NativeSelect
                  aria-label="Owner kind"
                  className="w-full"
                  data-testid="tasks-create-modal-owner-kind"
                  id="tasks-create-owner-kind"
                  onChange={event => form.updateOwnerKind(event.target.value as TaskOwnerKind | "")}
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
                  data-testid="tasks-create-modal-owner-ref"
                  onChange={form.updateText("ownerRef")}
                  placeholder="Owner reference (e.g. agent name)"
                  value={draft.ownerRef}
                />
              </Field>

              <Field>
                <FieldLabel>Attempts</FieldLabel>
                <Pills
                  aria-label="Max attempts"
                  className="w-full flex-wrap"
                  data-testid="tasks-create-modal-attempts"
                  items={attemptItems}
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

            <Field>
              <FieldLabel htmlFor="tasks-create-parent">Parent task</FieldLabel>
              <FieldDescription>Optional — link this task to an existing parent.</FieldDescription>
              <Input
                className="h-10"
                data-testid="tasks-create-modal-parent"
                id="tasks-create-parent"
                onChange={form.updateText("parentTaskId")}
                placeholder="Search by identifier or task id"
                value={draft.parentTaskId}
              />
            </Field>

            <Field>
              <FieldLabel>Approval</FieldLabel>
              <Pills
                aria-label="Approval policy"
                className="w-full flex-wrap"
                data-testid="tasks-create-modal-approval"
                items={APPROVAL_OPTIONS}
                onChange={form.updateApprovalPolicy}
                size="sm"
                value={draft.approvalPolicy}
              />
            </Field>

            <div className="grid gap-5 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="tasks-create-network">Network channel</FieldLabel>
                <FieldDescription>Optional ingress channel.</FieldDescription>
                <Input
                  className="h-10"
                  data-testid="tasks-create-modal-network-channel"
                  id="tasks-create-network"
                  onChange={form.updateText("networkChannel")}
                  placeholder="ingress channel"
                  value={draft.networkChannel}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="tasks-create-identifier">Identifier override</FieldLabel>
                <FieldDescription>Optional — replace the auto-generated id.</FieldDescription>
                <Input
                  className="h-10"
                  data-testid="tasks-create-modal-identifier"
                  id="tasks-create-identifier"
                  onChange={form.updateText("identifier")}
                  placeholder="TASK-123"
                  value={draft.identifier}
                />
              </Field>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-3 py-2 text-[13px] text-[color:var(--color-text-secondary)]">
              <span data-testid="tasks-create-modal-notice">{noticeText}</span>
              <Pill
                variant={pillVariantFromTone(
                  template.preview.enqueueOnSubmit ? "violet" : "neutral"
                )}
              >
                {template.preview.enqueueOnSubmit ? "Enqueues" : "Draft"}
              </Pill>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-2 border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-5 py-3">
            <Button onClick={() => onOpenChange(false)} type="button" variant="outline">
              Cancel
            </Button>
            <Button
              data-testid="tasks-create-modal-save-draft"
              disabled={!canSubmit || isSubmitting}
              onClick={form.submitDraft}
              type="button"
              variant="outline"
            >
              {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
              Save draft
            </Button>
            <Button
              data-testid="tasks-create-modal-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
              {template.preview.enqueueOnSubmit ? "Create & enqueue" : "Create task"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
