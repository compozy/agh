"use client";

import { useCallback, useId, useMemo } from "react";
import {
  Calendar,
  Layers,
  ListChecks,
  Network,
  Plus,
  Sparkles,
  UserCheck,
  type LucideIcon,
} from "lucide-react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  Field,
  FieldDescription,
  FieldLabel,
  FormSection,
  Input,
  NativeSelect,
  NativeSelectOption,
  PillGroup,
  RadioCard,
  Spinner,
  Textarea,
  type PillGroupItem,
} from "@agh/ui";

import type { TaskEditorDraft } from "../lib/task-editor";
import {
  TASK_TEMPLATES,
  getTaskTemplate,
  type TaskTemplate,
  type TaskTemplateId,
} from "../lib/task-templates";
import type { TaskOwnerKind, TaskPriority, TaskRecord, TaskScope } from "../types";
import { useTasksCreateModalForm } from "./use-tasks-create-modal-form";

export type TaskEditorModalMode = "new" | "edit";

export interface TaskEditorModalProps {
  /** `new` shows the template picker first; `edit` opens directly on the form body. */
  mode: TaskEditorModalMode;
  /** Whether the modal is open. */
  open: boolean;
  /** Callback fired when the modal requests to close (scrim click, ESC, Cancel). */
  onOpenChange: (open: boolean) => void;
  /** Working draft. */
  draft: TaskEditorDraft;
  /** Draft setter / updater. */
  onDraftChange: (next: TaskEditorDraft | ((current: TaskEditorDraft) => TaskEditorDraft)) => void;
  /** Submit handler. `asDraft` selects Save Draft semantics for templates that gate enqueue. */
  onSubmit: (draft: TaskEditorDraft, asDraft: boolean) => Promise<unknown> | void;
  /** Disable submit when false (e.g. empty title). */
  canSubmit?: boolean;
  /** Show a spinner on the submit button. */
  isSubmitting?: boolean;
  /** Active workspace name — surfaces in the Scope chip and identifier hint. */
  workspaceName?: string | null;
  /** Required in `new` mode — the currently selected template. */
  templateId?: TaskTemplateId;
  /** Required in `new` mode — the resolved template metadata. */
  template?: TaskTemplate;
  /** Required in `new` mode — fires when the operator selects a new template card. */
  onTemplateChange?: (templateId: TaskTemplateId) => void;
  /** Required in `edit` mode — the persisted task being edited. */
  task?: TaskRecord | null;
}

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
  { value: "manual", label: "Human-in-the-loop", testId: "task-editor-approval-manual" },
];

const ATTEMPT_VALUES = [1, 2, 3, 5] as const;
const ATTEMPT_ITEMS: PillGroupItem<string>[] = ATTEMPT_VALUES.map(value => ({
  value: String(value),
  label: <span className="font-mono tabular-nums">{value}</span>,
  testId: `task-editor-attempts-${value}`,
}));

const TEMPLATE_ICONS: Record<TaskTemplateId, LucideIcon> = {
  one_shot: Sparkles,
  recurring: Calendar,
  epic: Layers,
  remote_peer: Network,
  human_in_loop: UserCheck,
  blank: ListChecks,
};

const FOOTER_HINT =
  "The contract is durable — runs descend from this task and respect dependencies.";

interface TaskEditorScopeOptionsArgs {
  workspaceName?: string | null;
}

function buildScopeOptions({
  workspaceName,
}: TaskEditorScopeOptionsArgs): PillGroupItem<TaskScope>[] {
  return [
    {
      value: "workspace",
      label: workspaceName ? `Workspace · ${workspaceName}` : "Workspace",
      testId: "task-editor-scope-workspace",
    },
    { value: "global", label: "Global", testId: "task-editor-scope-global" },
  ];
}

function resolveAttemptsValue(maxAttempts: number | null): string {
  if (typeof maxAttempts === "number" && ATTEMPT_VALUES.includes(maxAttempts as 1 | 2 | 3 | 5)) {
    return String(maxAttempts);
  }
  return "1";
}

function resolveSubmitLabel(mode: TaskEditorModalMode, template?: TaskTemplate): string {
  if (mode === "edit") {
    return "Save changes";
  }
  if (template?.preview.enqueueOnSubmit) {
    return "Enqueue task";
  }
  return "Save draft";
}

/**
 * Task authoring modal — 720 px overlay covering `/tasks`. Switches anatomy
 * via `mode: "new" | "edit"`:
 *   - `new` opens on a 2-col `<RadioCard>` template picker, then reveals the
 *     form blocks once a template is selected.
 *   - `edit` hides the template picker entirely and opens directly on the
 *     form body, pre-populated from the task record.
 *
 * Footer carries the durable-contract hint, a left-aligned Cancel button,
 * and a single primary action whose label is gated by
 * `template.preview.enqueueOnSubmit` (`Enqueue task` vs `Save draft`).
 * Edit mode always renders `Save changes`.
 */
export function TaskEditorModal({
  mode,
  open,
  onOpenChange,
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
}: TaskEditorModalProps) {
  const titleId = useId();
  const form = useTasksCreateModalForm({ draft, onDraftChange, onSubmit });
  const isNewMode = mode === "new";

  const handleCancel = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const handleDialogOpenChange = useCallback(
    (next: boolean) => {
      onOpenChange(next);
    },
    [onOpenChange]
  );

  const scopeItems = useMemo(() => buildScopeOptions({ workspaceName }), [workspaceName]);
  const attemptsValue = resolveAttemptsValue(draft.maxAttempts);
  const submitLabel = resolveSubmitLabel(mode, template);
  const title = isNewMode ? "New task" : "Edit task";

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent
        aria-labelledby={titleId}
        data-testid="task-editor-modal"
        data-mode={mode}
        showCloseButton={false}
        unframed
        className="w-(--width-modal-md) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-md) grid-rows-[auto_1fr_auto] max-h-[min(720px,calc(100vh-2rem))]"
      >
        <header
          data-slot="task-editor-modal-head"
          className="flex items-center justify-between gap-3 border-b border-line px-[18px] py-[13px]"
        >
          <DialogTitle
            id={titleId}
            data-testid="task-editor-modal-title"
            className="text-modal-title tracking-modal-title text-fg-strong"
            style={{ fontWeight: 510 }}
          >
            {title}
          </DialogTitle>
          {isNewMode && template ? (
            <span data-testid="task-editor-modal-template-hint" className="text-[12px] text-muted">
              {template.label}
            </span>
          ) : task?.identifier ? (
            <span
              data-testid="task-editor-modal-task-id"
              className="font-mono text-[12px] text-muted"
            >
              {task.identifier}
            </span>
          ) : null}
        </header>

        <form
          className="flex min-h-0 flex-col overflow-hidden"
          data-testid="task-editor-modal-form"
          onSubmit={form.submitForm}
        >
          <div
            className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-[18px]"
            data-testid="task-editor-modal-body"
          >
            {isNewMode && templateId && onTemplateChange ? (
              <TemplatePicker
                onSelect={onTemplateChange}
                templateId={templateId}
                workspaceName={workspaceName ?? null}
              />
            ) : null}

            <FormSection
              data-testid="task-editor-modal-section-contract"
              size="compact"
              title="Contract"
            >
              <Field>
                <div className="flex items-center justify-between gap-3">
                  <FieldLabel
                    data-testid="task-editor-title-label"
                    htmlFor="task-editor-title-input"
                  >
                    Title
                  </FieldLabel>
                  <span className="text-eyebrow text-subtle">Required</span>
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
                  className="min-h-[136px]"
                  data-testid="task-editor-description-input"
                  id="task-editor-description-input"
                  onChange={form.updateText("description")}
                  placeholder="Describe the task contract for the agent."
                  value={draft.description}
                />
              </Field>

              <div className="grid gap-4 md:grid-cols-2">
                <Field>
                  <FieldLabel data-testid="task-editor-scope-label">Scope</FieldLabel>
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
            </FormSection>

            <FormSection
              data-testid="task-editor-modal-section-queue"
              size="compact"
              title="Queue & ownership"
            >
              <div className="grid gap-4 md:grid-cols-2">
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
                    className="mt-2 h-10"
                    data-testid="task-editor-owner-ref"
                    onChange={form.updateText("ownerRef")}
                    placeholder="Owner reference (e.g. coder)"
                    value={draft.ownerRef}
                  />
                </Field>

                <Field>
                  <FieldLabel data-testid="task-editor-attempts-label">Max attempts</FieldLabel>
                  <PillGroup
                    aria-label="Max attempts"
                    className="w-full flex-wrap"
                    data-testid="task-editor-attempts-options"
                    items={ATTEMPT_ITEMS}
                    onChange={next => {
                      const numeric = Number(next);
                      if (ATTEMPT_VALUES.includes(numeric as 1 | 2 | 3 | 5)) {
                        form.updateMaxAttempts(numeric);
                      }
                    }}
                    size="sm"
                    value={attemptsValue}
                  />
                </Field>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
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

                {isNewMode ? (
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
                ) : null}
              </div>
            </FormSection>

            <FormSection
              data-testid="task-editor-modal-section-context"
              size="compact"
              title="Channel & identifier"
            >
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
              {isNewMode ? (
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
            </FormSection>
          </div>

          <footer
            data-slot="task-editor-modal-foot"
            data-testid="task-editor-modal-footer"
            className="flex flex-wrap items-center justify-between gap-3 border-t border-line bg-canvas-soft px-[18px] py-[11px]"
          >
            <p
              className="min-w-0 flex-1 text-[12px] text-muted"
              data-testid="task-editor-modal-hint"
            >
              {FOOTER_HINT}
            </p>
            <div className="flex shrink-0 items-center gap-2">
              <Button
                data-testid="task-editor-modal-cancel"
                onClick={handleCancel}
                type="button"
                variant="outline"
              >
                Cancel
              </Button>
              <Button
                data-testid="task-editor-modal-submit"
                disabled={!canSubmit || isSubmitting}
                type="submit"
              >
                {isSubmitting ? <Spinner className="size-4" /> : null}
                {submitLabel}
              </Button>
            </div>
          </footer>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface TemplatePickerProps {
  templateId: TaskTemplateId;
  onSelect: (templateId: TaskTemplateId) => void;
  workspaceName: string | null;
}

function TemplatePicker({ onSelect, templateId, workspaceName }: TemplatePickerProps) {
  return (
    <FormSection
      data-testid="task-editor-modal-template-picker"
      icon={Plus}
      rightLabel={workspaceName ? `Workspace · ${workspaceName}` : undefined}
      size="compact"
      title="Template"
    >
      <div className="grid gap-2 sm:grid-cols-2" data-testid="task-editor-modal-template-grid">
        {TASK_TEMPLATES.map(option => {
          const Icon = TEMPLATE_ICONS[option.id];
          const selected = option.id === templateId;
          return (
            <RadioCard
              data-testid={`task-editor-template-${option.id}`}
              description={option.description}
              icon={Icon}
              key={option.id}
              onSelect={() => onSelect(option.id)}
              selected={selected}
              title={option.label}
            />
          );
        })}
      </div>
    </FormSection>
  );
}

export { getTaskTemplate, TASK_TEMPLATES };
export type { TaskTemplate };
