import { Loader2, Plus } from "lucide-react";

import { Button, Input } from "@agh/ui";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Pill } from "@/components/design-system";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import type { TaskOwnerKind, TaskPriority, TaskScope } from "../types";
import { TASK_TEMPLATES, type TaskTemplate, type TaskTemplateId } from "../lib/task-templates";
import { useTasksCreateModalForm } from "./use-tasks-create-modal-form";

const PRIORITY_OPTIONS: TaskPriority[] = ["low", "medium", "high", "urgent"];
const SCOPE_OPTIONS: TaskScope[] = ["workspace", "global"];
const OWNER_KIND_OPTIONS: TaskOwnerKind[] = [
  "agent_session",
  "human",
  "automation",
  "extension",
  "network_peer",
  "pool",
];

const ATTEMPT_OPTIONS = [1, 2, 3, 5];

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

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent
        className="max-w-[calc(100%-2rem)] gap-0 border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-0 text-[color:var(--color-text-primary)] ring-0 sm:max-w-[36rem]"
        data-testid="tasks-create-modal"
      >
        <DialogHeader className="border-b border-[color:var(--color-divider)] px-5 py-4">
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] text-[color:var(--color-accent)]">
              <Plus className="size-4" />
            </span>
            <div>
              <DialogTitle>New task</DialogTitle>
              <DialogDescription
                className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]"
                data-testid="tasks-create-modal-template-label"
              >
                Starting from {template.label} template
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form className="flex max-h-[min(85vh,960px)] flex-col" onSubmit={form.submitForm}>
          <div className="flex-1 space-y-5 overflow-y-auto px-5 py-4">
            <div data-testid="tasks-create-modal-template-pills">
              <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                Template
              </p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {TASK_TEMPLATES.map(option => {
                  const active = option.id === templateId;
                  return (
                    <button
                      aria-pressed={active}
                      className={
                        active
                          ? "rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-accent)] px-3 py-1 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-accent-ink)]"
                          : "rounded-full border border-[color:var(--color-divider)] bg-transparent px-3 py-1 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)] hover:text-[color:var(--color-text-primary)]"
                      }
                      data-testid={`tasks-create-modal-template-${option.id}`}
                      key={option.id}
                      onClick={() => onTemplateChange(option.id)}
                      type="button"
                    >
                      {option.label}
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="space-y-2">
              <label
                className="flex items-center justify-between text-xs text-[color:var(--color-text-secondary)]"
                htmlFor="tasks-create-title"
              >
                <span>Title</span>
                <span className="font-mono text-[0.6rem] uppercase tracking-[0.14em] text-[color:var(--color-text-tertiary)]">
                  Required
                </span>
              </label>
              <Input
                className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                data-testid="tasks-create-modal-title"
                id="tasks-create-title"
                onChange={form.updateText("title")}
                placeholder="e.g. Generate API client for payments-v3"
                required
                value={draft.title}
              />
            </div>

            <div className="space-y-2">
              <label
                className="text-xs text-[color:var(--color-text-secondary)]"
                htmlFor="tasks-create-description"
              >
                Description
              </label>
              <Textarea
                className="min-h-[96px] border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                data-testid="tasks-create-modal-description"
                id="tasks-create-description"
                onChange={form.updateText("description")}
                placeholder="Describe the task contract for the agent."
                value={draft.description}
              />
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <FieldGroup label="Scope">
                <div className="flex gap-1.5">
                  {SCOPE_OPTIONS.map(option => (
                    <button
                      aria-pressed={draft.scope === option}
                      className={
                        draft.scope === option
                          ? "flex-1 rounded-lg border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs font-medium text-[color:var(--color-text-primary)]"
                          : "flex-1 rounded-lg border border-[color:var(--color-divider)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)]"
                      }
                      data-testid={`tasks-create-modal-scope-${option}`}
                      key={option}
                      onClick={() => form.updateScope(option)}
                      type="button"
                    >
                      {option === "workspace"
                        ? `Workspace${workspaceName ? ` · ${workspaceName}` : ""}`
                        : "Global"}
                    </button>
                  ))}
                </div>
              </FieldGroup>

              <FieldGroup label="Priority">
                <div className="flex flex-wrap gap-1.5" data-testid="tasks-create-modal-priorities">
                  {PRIORITY_OPTIONS.map(option => (
                    <button
                      aria-pressed={draft.priority === option}
                      className={
                        draft.priority === option
                          ? "rounded-lg border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs font-medium text-[color:var(--color-text-primary)]"
                          : "rounded-lg border border-[color:var(--color-divider)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)]"
                      }
                      data-testid={`tasks-create-modal-priority-${option}`}
                      key={option}
                      onClick={() => form.updatePriority(option)}
                      type="button"
                    >
                      {option}
                    </button>
                  ))}
                </div>
              </FieldGroup>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <FieldGroup label="Owner">
                <select
                  aria-label="Owner kind"
                  className="h-10 w-full rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-3 text-sm text-[color:var(--color-text-primary)]"
                  data-testid="tasks-create-modal-owner-kind"
                  onChange={event => form.updateOwnerKind(event.target.value as TaskOwnerKind | "")}
                  value={draft.ownerKind}
                >
                  <option value="">Unassigned</option>
                  {OWNER_KIND_OPTIONS.map(kind => (
                    <option key={kind} value={kind}>
                      {kind}
                    </option>
                  ))}
                </select>
                <Input
                  className="mt-2 h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                  data-testid="tasks-create-modal-owner-ref"
                  onChange={form.updateText("ownerRef")}
                  placeholder="Owner reference (e.g. agent name)"
                  value={draft.ownerRef}
                />
              </FieldGroup>

              <FieldGroup label="Attempts">
                <div className="flex flex-wrap gap-1.5" data-testid="tasks-create-modal-attempts">
                  {ATTEMPT_OPTIONS.map(option => (
                    <button
                      aria-pressed={draft.maxAttempts === option}
                      className={
                        draft.maxAttempts === option
                          ? "rounded-lg border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs font-medium text-[color:var(--color-text-primary)]"
                          : "rounded-lg border border-[color:var(--color-divider)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)]"
                      }
                      data-testid={`tasks-create-modal-attempts-${option}`}
                      key={option}
                      onClick={() => form.updateMaxAttempts(option)}
                      type="button"
                    >
                      {option}
                    </button>
                  ))}
                  <button
                    aria-pressed={draft.maxAttempts === null}
                    className={
                      draft.maxAttempts === null
                        ? "rounded-lg border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs font-medium text-[color:var(--color-text-primary)]"
                        : "rounded-lg border border-[color:var(--color-divider)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)]"
                    }
                    data-testid="tasks-create-modal-attempts-default"
                    onClick={() => form.updateMaxAttempts(null)}
                    type="button"
                  >
                    default
                  </button>
                </div>
              </FieldGroup>
            </div>

            <FieldGroup label="Parent task (optional)">
              <Input
                className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                data-testid="tasks-create-modal-parent"
                onChange={form.updateText("parentTaskId")}
                placeholder="Search by identifier or task id"
                value={draft.parentTaskId}
              />
            </FieldGroup>

            <FieldGroup label="Approval">
              <div className="flex gap-1.5" data-testid="tasks-create-modal-approval">
                {(["none", "manual"] as const).map(option => (
                  <button
                    aria-pressed={draft.approvalPolicy === option}
                    className={
                      draft.approvalPolicy === option
                        ? "flex-1 rounded-lg border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2 text-xs font-medium text-[color:var(--color-text-primary)]"
                        : "flex-1 rounded-lg border border-[color:var(--color-divider)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)]"
                    }
                    data-testid={`tasks-create-modal-approval-${option}`}
                    key={option}
                    onClick={() => form.updateApprovalPolicy(option)}
                    type="button"
                  >
                    {option === "manual" ? "Human-in-the-loop" : "No approval"}
                  </button>
                ))}
              </div>
            </FieldGroup>

            <div className="grid gap-4 md:grid-cols-2">
              <FieldGroup label="Network channel (optional)">
                <Input
                  className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                  data-testid="tasks-create-modal-network-channel"
                  onChange={form.updateText("networkChannel")}
                  placeholder="ingress channel"
                  value={draft.networkChannel}
                />
              </FieldGroup>
              <FieldGroup label="Identifier override (optional)">
                <Input
                  className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                  data-testid="tasks-create-modal-identifier"
                  onChange={form.updateText("identifier")}
                  placeholder="TASK-123"
                  value={draft.identifier}
                />
              </FieldGroup>
            </div>

            <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] px-3 py-2 text-xs text-[color:var(--color-text-secondary)]">
              <span data-testid="tasks-create-modal-notice">{noticeText}</span>
              <Pill kind="state" tone={template.preview.enqueueOnSubmit ? "violet" : "neutral"}>
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

interface FieldGroupProps {
  label: string;
  children: React.ReactNode;
}

function FieldGroup({ label, children }: FieldGroupProps) {
  return (
    <div className="space-y-2">
      <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </p>
      {children}
    </div>
  );
}
