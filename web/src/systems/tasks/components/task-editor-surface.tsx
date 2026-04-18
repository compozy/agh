import { Link } from "@tanstack/react-router";
import { ArrowLeft, Loader2, Pencil, Plus } from "lucide-react";

import {
  Pill,
  Panel,
  PanelBody,
  PanelDescription,
  PanelHeader,
  PanelTitle,
} from "@/components/design-system";
import { Textarea } from "@/components/ui/textarea";
import { Button, Input } from "@agh/ui";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import type { TaskOwnerKind, TaskPriority, TaskRecord, TaskScope } from "../types";
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

  return (
    <section
      className="flex min-h-0 flex-1 flex-col bg-[color:var(--color-canvas)]"
      data-testid="task-editor-surface"
    >
      <header className="border-b border-[color:var(--color-divider)] px-6 py-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 space-y-3">
            {task ? (
              <Link
                className="inline-flex items-center gap-2 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-secondary)] transition-colors hover:text-[color:var(--color-text-primary)]"
                data-testid="task-editor-back-link"
                params={{ id: task.id }}
                to="/tasks/$id"
              >
                <ArrowLeft className="size-3.5" />
                Back to tasks
              </Link>
            ) : (
              <Link
                className="inline-flex items-center gap-2 font-mono text-[0.66rem] uppercase tracking-[0.14em] text-[color:var(--color-text-secondary)] transition-colors hover:text-[color:var(--color-text-primary)]"
                data-testid="task-editor-back-link"
                to="/tasks"
              >
                <ArrowLeft className="size-3.5" />
                Back to tasks
              </Link>
            )}
            <div className="space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="flex size-9 items-center justify-center rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] text-[color:var(--color-accent)]">
                  {isCreateMode ? <Plus className="size-4" /> : <Pencil className="size-4" />}
                </span>
                <h1
                  className="text-2xl font-semibold tracking-[-0.03em] text-[color:var(--color-text-primary)]"
                  data-testid="task-editor-title"
                >
                  {title}
                </h1>
              </div>
              <p className="max-w-2xl text-sm leading-6 text-[color:var(--color-text-secondary)]">
                {description}
              </p>
            </div>
          </div>

          {task ? (
            <div className="flex flex-wrap items-center gap-2 text-xs text-[color:var(--color-text-secondary)]">
              {task.identifier ? (
                <Pill emphasis="strong" kind="state" tone="neutral">
                  {task.identifier}
                </Pill>
              ) : null}
              <span>Scope {task.scope}</span>
              {task.parent_task_id ? <span>· Parent {task.parent_task_id}</span> : null}
              <span>· Status {task.status}</span>
            </div>
          ) : null}
        </div>
      </header>

      <form className="flex min-h-0 flex-1 flex-col" onSubmit={form.submitForm}>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="grid gap-5 px-6 py-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.85fr)]">
            <div className="flex min-w-0 flex-col gap-5">
              <Panel data-testid="task-editor-main-panel">
                <PanelHeader>
                  <PanelTitle>Task contract</PanelTitle>
                  <PanelDescription>
                    Define the headline and the instructions the assigned worker will receive.
                  </PanelDescription>
                </PanelHeader>
                <PanelBody className="gap-5">
                  {isCreateMode && template && templateId && onTemplateChange ? (
                    <Fieldset label="Template" labelTestId="task-editor-template-label">
                      <div
                        className="flex flex-wrap gap-2"
                        data-testid="task-editor-template-options"
                      >
                        {TASK_TEMPLATES.map(option => {
                          const active = option.id === templateId;
                          return (
                            <button
                              aria-pressed={active}
                              className={
                                active
                                  ? "rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-1.5 font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-accent)]"
                                  : "rounded-full border border-[color:var(--color-divider)] bg-transparent px-3 py-1.5 font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-secondary)] transition-colors hover:border-[color:var(--color-text-label)] hover:text-[color:var(--color-text-primary)]"
                              }
                              data-testid={`task-editor-template-${option.id}`}
                              key={option.id}
                              onClick={() => onTemplateChange(option.id)}
                              type="button"
                            >
                              {option.label}
                            </button>
                          );
                        })}
                      </div>
                    </Fieldset>
                  ) : null}

                  <Fieldset helper="Required" label="Title" labelTestId="task-editor-title-label">
                    <Input
                      className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                      data-testid="task-editor-title-input"
                      onChange={form.updateText("title")}
                      placeholder="Generate API client for payments-v3"
                      required
                      value={draft.title}
                    />
                  </Fieldset>

                  <Fieldset
                    helper="Describe the expected outcome, constraints, and completion criteria."
                    label="Description"
                    labelTestId="task-editor-description-label"
                  >
                    <Textarea
                      className="min-h-[168px] border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                      data-testid="task-editor-description-input"
                      onChange={form.updateText("description")}
                      placeholder="Describe the task contract for the agent."
                      value={draft.description}
                    />
                  </Fieldset>

                  {isCreateMode ? (
                    <div className="grid gap-5 md:grid-cols-2">
                      <Fieldset
                        helper={workspaceName ? `Workspace default: ${workspaceName}` : undefined}
                        label="Scope"
                        labelTestId="task-editor-scope-label"
                      >
                        <div className="grid gap-2 sm:grid-cols-2">
                          {SCOPE_OPTIONS.map(option => (
                            <ChoiceButton
                              active={draft.scope === option}
                              dataTestId={`task-editor-scope-${option}`}
                              key={option}
                              label={
                                option === "workspace"
                                  ? `Workspace${workspaceName ? ` · ${workspaceName}` : ""}`
                                  : "Global"
                              }
                              onClick={() => form.updateScope(option)}
                            />
                          ))}
                        </div>
                      </Fieldset>

                      <Fieldset label="Priority" labelTestId="task-editor-priority-label">
                        <div className="grid gap-2 grid-cols-2">
                          {PRIORITY_OPTIONS.map(option => (
                            <ChoiceButton
                              active={draft.priority === option}
                              dataTestId={`task-editor-priority-${option}`}
                              key={option}
                              label={option}
                              onClick={() => form.updatePriority(option)}
                            />
                          ))}
                        </div>
                      </Fieldset>
                    </div>
                  ) : null}
                </PanelBody>
              </Panel>

              {isCreateMode ? (
                <Panel>
                  <PanelHeader>
                    <PanelTitle>Queue settings</PanelTitle>
                    <PanelDescription>
                      Decide how the task should be routed, how many attempts it gets, and whether
                      it needs approval before execution.
                    </PanelDescription>
                  </PanelHeader>
                  <PanelBody className="gap-5">
                    <div className="grid gap-5 md:grid-cols-2">
                      <Fieldset label="Owner" labelTestId="task-editor-owner-label">
                        <select
                          aria-label="Owner kind"
                          className="h-10 w-full rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-3 text-sm text-[color:var(--color-text-primary)] outline-none"
                          data-testid="task-editor-owner-kind"
                          onChange={event =>
                            form.updateOwnerKind(event.target.value as TaskOwnerKind | "")
                          }
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
                          className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                          data-testid="task-editor-owner-ref"
                          onChange={form.updateText("ownerRef")}
                          placeholder="Owner reference (for example: coder)"
                          value={draft.ownerRef}
                        />
                      </Fieldset>

                      <Fieldset label="Attempts" labelTestId="task-editor-attempts-label">
                        <div className="grid grid-cols-5 gap-2">
                          {ATTEMPT_OPTIONS.map(option => (
                            <ChoiceButton
                              active={draft.maxAttempts === option}
                              dataTestId={`task-editor-attempts-${option}`}
                              key={option}
                              label={option.toString()}
                              onClick={() => form.updateMaxAttempts(option)}
                            />
                          ))}
                          <ChoiceButton
                            active={draft.maxAttempts === null}
                            dataTestId="task-editor-attempts-default"
                            label="Default"
                            onClick={() => form.updateMaxAttempts(null)}
                          />
                        </div>
                      </Fieldset>
                    </div>

                    <div className="grid gap-5 md:grid-cols-2">
                      <Fieldset label="Approval" labelTestId="task-editor-approval-label">
                        <div className="grid gap-2 sm:grid-cols-2">
                          <ChoiceButton
                            active={draft.approvalPolicy === "none"}
                            dataTestId="task-editor-approval-none"
                            label="No approval"
                            onClick={() => form.updateApprovalPolicy("none")}
                          />
                          <ChoiceButton
                            active={draft.approvalPolicy === "manual"}
                            dataTestId="task-editor-approval-manual"
                            label="Human-in-the-loop"
                            onClick={() => form.updateApprovalPolicy("manual")}
                          />
                        </div>
                      </Fieldset>

                      <Fieldset label="Parent task" labelTestId="task-editor-parent-label">
                        <Input
                          className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                          data-testid="task-editor-parent-input"
                          onChange={form.updateText("parentTaskId")}
                          placeholder="Search by identifier or task id"
                          value={draft.parentTaskId}
                        />
                      </Fieldset>
                    </div>
                  </PanelBody>
                </Panel>
              ) : null}
            </div>

            <div className="flex min-w-0 flex-col gap-5">
              <Panel tone="elevated">
                <PanelHeader>
                  <PanelTitle>{isCreateMode ? "Submission" : "Editable fields"}</PanelTitle>
                  <PanelDescription data-testid="task-editor-notice">{noticeText}</PanelDescription>
                </PanelHeader>
                <PanelBody className="gap-5">
                  <Fieldset label="Network channel" labelTestId="task-editor-network-label">
                    <Input
                      className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                      data-testid="task-editor-network-input"
                      onChange={form.updateText("networkChannel")}
                      placeholder="ingress channel"
                      value={draft.networkChannel}
                    />
                  </Fieldset>

                  {isCreateMode ? (
                    <Fieldset
                      label="Identifier override"
                      labelTestId="task-editor-identifier-label"
                    >
                      <Input
                        className="h-10 border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]"
                        data-testid="task-editor-identifier-input"
                        onChange={form.updateText("identifier")}
                        placeholder="TASK-123"
                        value={draft.identifier}
                      />
                    </Fieldset>
                  ) : null}

                  {!isCreateMode ? (
                    <>
                      <Fieldset label="Priority" labelTestId="task-editor-priority-label">
                        <div className="grid gap-2 grid-cols-2">
                          {PRIORITY_OPTIONS.map(option => (
                            <ChoiceButton
                              active={draft.priority === option}
                              dataTestId={`task-editor-priority-${option}`}
                              key={option}
                              label={option}
                              onClick={() => form.updatePriority(option)}
                            />
                          ))}
                        </div>
                      </Fieldset>

                      <Fieldset label="Approval" labelTestId="task-editor-approval-label">
                        <div className="grid gap-2 sm:grid-cols-2">
                          <ChoiceButton
                            active={draft.approvalPolicy === "none"}
                            dataTestId="task-editor-approval-none"
                            label="No approval"
                            onClick={() => form.updateApprovalPolicy("none")}
                          />
                          <ChoiceButton
                            active={draft.approvalPolicy === "manual"}
                            dataTestId="task-editor-approval-manual"
                            label="Human-in-the-loop"
                            onClick={() => form.updateApprovalPolicy("manual")}
                          />
                        </div>
                      </Fieldset>

                      <Fieldset label="Attempts" labelTestId="task-editor-attempts-label">
                        <div className="grid grid-cols-5 gap-2">
                          {ATTEMPT_OPTIONS.map(option => (
                            <ChoiceButton
                              active={draft.maxAttempts === option}
                              dataTestId={`task-editor-attempts-${option}`}
                              key={option}
                              label={option.toString()}
                              onClick={() => form.updateMaxAttempts(option)}
                            />
                          ))}
                          <ChoiceButton
                            active={draft.maxAttempts === null}
                            dataTestId="task-editor-attempts-default"
                            label="Default"
                            onClick={() => form.updateMaxAttempts(null)}
                          />
                        </div>
                      </Fieldset>
                    </>
                  ) : null}

                  <div className="rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-4 py-3">
                    <p className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
                      Read-only context
                    </p>
                    <dl className="mt-3 space-y-2 text-sm text-[color:var(--color-text-secondary)]">
                      <ContextRow label="Scope" value={draft.scope} />
                      <ContextRow label="Parent task" value={draft.parentTaskId || "None"} />
                      <ContextRow label="Identifier" value={draft.identifier || "Auto-generated"} />
                      <ContextRow
                        label="Workspace"
                        value={workspaceName ?? "No active workspace"}
                      />
                    </dl>
                  </div>
                </PanelBody>
              </Panel>
            </div>
          </div>
        </div>

        <footer className="border-t border-[color:var(--color-divider)] bg-[color:var(--color-surface)] px-6 py-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-[color:var(--color-text-secondary)]">
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

function Fieldset({
  label,
  labelTestId,
  helper,
  children,
}: {
  label: string;
  labelTestId?: string;
  helper?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-2.5">
      <div className="flex items-center justify-between gap-3">
        <span
          className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
          data-testid={labelTestId}
        >
          {label}
        </span>
        {helper ? (
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{helper}</span>
        ) : null}
      </div>
      <div className="flex flex-col gap-2">{children}</div>
    </label>
  );
}

function ChoiceButton({
  active,
  label,
  onClick,
  dataTestId,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
  dataTestId: string;
}) {
  return (
    <button
      aria-pressed={active}
      className={
        active
          ? "rounded-xl border border-[color:var(--color-accent)] bg-[color:var(--color-accent-tint)] px-3 py-2.5 text-sm font-medium text-[color:var(--color-text-primary)] transition-colors"
          : "rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-3 py-2.5 text-sm text-[color:var(--color-text-secondary)] transition-colors hover:border-[color:var(--color-text-label)] hover:text-[color:var(--color-text-primary)]"
      }
      data-testid={dataTestId}
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}

function ContextRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="font-mono text-[0.62rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
        {label}
      </dt>
      <dd className="truncate text-[color:var(--color-text-primary)]">{value}</dd>
    </div>
  );
}
