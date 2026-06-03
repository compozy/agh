"use client";

import { Check, ClipboardCheck } from "lucide-react";
import { useCallback, useState } from "react";

import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Eyebrow,
  Field,
  FieldLabel,
  Input,
  Spinner,
} from "@agh/ui";

import type { WorkspaceCommandSelectOption } from "@/systems/workspace";

import type { TaskEditorDraft } from "../lib/task-editor";
import {
  SIMPLE_TASK_TEMPLATE_IDS,
  type TaskTemplate,
  type TaskTemplateId,
} from "../lib/task-templates";
import type { TaskRecord } from "../types";
import { ContractSection } from "./task-form/contract-section";
import { ExecutionCollapsible } from "./task-form/execution-collapsible";
import { IngressIdentitySection } from "./task-form/ingress-identity-section";
import { ModeToolbar, type TaskFormMode } from "./task-form/mode-toolbar";
import { NumberedSection } from "./task-form/numbered-section";
import { PlacementSection } from "./task-form/placement-section";
import { PrioritySection } from "./task-form/priority-section";
import { QueueOwnershipSection } from "./task-form/queue-ownership-section";
import { TemplateCards } from "./task-form/template-cards";
import { useTasksCreateModalForm } from "./use-tasks-create-modal-form";

export type TaskEditorModalMode = "new" | "edit";

export interface TaskEditorModalProps {
  /** `new` shows the mode toolbar + template grid; `edit` opens on the contract fields. */
  mode: TaskEditorModalMode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  draft: TaskEditorDraft;
  onDraftChange: (next: TaskEditorDraft | ((current: TaskEditorDraft) => TaskEditorDraft)) => void;
  onSubmit: (draft: TaskEditorDraft, asDraft: boolean) => Promise<unknown> | void;
  canSubmit?: boolean;
  isSubmitting?: boolean;
  /** Registered workspaces selectable for new workspace-scoped tasks. */
  workspaces?: ReadonlyArray<WorkspaceCommandSelectOption>;
  /** Required in `new` mode — the selected template id. */
  templateId?: TaskTemplateId;
  /** Required in `new` mode — the resolved template metadata. */
  template?: TaskTemplate;
  /** Required in `new` mode — fires when the operator selects a new template card. */
  onTemplateChange?: (templateId: TaskTemplateId) => void;
  /** Required in `edit` mode — the persisted task being edited. */
  task?: TaskRecord | null;
}

const MODAL_CONTENT_CLASS =
  "text-fg w-(--width-modal-md) max-w-[calc(100vw-2rem)] sm:max-w-(--width-modal-md) grid-rows-[auto_minmax(0,1fr)] h-(--height-modal-md) max-h-[min(var(--height-modal-md),calc(100vh-2rem))]";

const TASK_DESCRIPTION =
  "A task is a durable contract — a unit of work that gets claimed and run by an owner. Runs descend from it and respect its dependencies.";

function resolveSubmitLabel(mode: TaskEditorModalMode, saveAsDraft: boolean): string {
  if (mode === "edit") {
    return "Save changes";
  }
  return saveAsDraft ? "Save draft" : "Enqueue task";
}

export function TaskEditorModal({
  mode,
  open,
  onOpenChange,
  draft,
  onDraftChange,
  onSubmit,
  canSubmit = true,
  isSubmitting = false,
  workspaces,
  templateId,
  onTemplateChange,
}: TaskEditorModalProps) {
  const form = useTasksCreateModalForm({ draft, onDraftChange, onSubmit });
  const [formMode, setFormMode] = useState<TaskFormMode>("simple");
  const isNewMode = mode === "new";

  const setField = useCallback(
    (field: keyof TaskEditorDraft) => (value: string) =>
      onDraftChange(current => ({ ...current, [field]: value })),
    [onDraftChange]
  );

  const handleCancel = useCallback(() => onOpenChange(false), [onOpenChange]);

  const handleModeChange = useCallback(
    (nextMode: TaskFormMode) => {
      setFormMode(nextMode);
      // Dropping to Simple while on an advanced-only template snaps back to one-shot.
      if (
        nextMode === "simple" &&
        templateId &&
        !SIMPLE_TASK_TEMPLATE_IDS.includes(templateId) &&
        onTemplateChange
      ) {
        onTemplateChange("one_shot");
      }
    },
    [onTemplateChange, templateId]
  );

  const advanced = isNewMode && formMode === "advanced";
  const submitLabel = resolveSubmitLabel(mode, draft.saveAsDraft);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        unframed
        className={MODAL_CONTENT_CLASS}
        data-mode={mode}
        data-testid="task-editor-modal"
        showCloseButton={false}
      >
        <DialogHeader variant="ruled">
          <div className="flex items-start gap-4">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accent-tint text-accent-strong ring-1 ring-accent-dim ring-inset">
              <ClipboardCheck aria-hidden="true" className="size-4" />
            </div>
            <div className="min-w-0 flex-1">
              <Eyebrow className="text-accent-strong">Autonomy · Task</Eyebrow>
              <DialogTitle className="mt-1" data-testid="task-editor-modal-title">
                {isNewMode ? "Create task" : "Edit task"}
              </DialogTitle>
              <DialogDescription className="mt-1">{TASK_DESCRIPTION}</DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <form
          className="flex min-h-0 flex-col"
          data-testid="task-editor-modal-form"
          onSubmit={form.submitForm}
        >
          {isNewMode ? (
            <ModeToolbar
              mode={formMode}
              onModeChange={handleModeChange}
              onScopeChange={form.updateScope}
              onWorkspaceChange={form.updateWorkspace}
              scope={draft.scope}
              workspaceId={draft.workspaceId}
              workspaces={workspaces}
            />
          ) : null}

          <div
            className="min-h-0 flex-1 overflow-y-auto px-6 py-5"
            data-testid="task-editor-modal-body"
          >
            <NumberedSection first index="01" subtitle="What should get done?" title="The contract">
              <ContractSection
                description={draft.description}
                onDescription={setField("description")}
                onTitle={setField("title")}
                title={draft.title}
              />
            </NumberedSection>

            {isNewMode && templateId && onTemplateChange ? (
              <NumberedSection
                index="02"
                subtitle={
                  advanced
                    ? "Presets the contract fields below — tweak any of them."
                    : "Pick the closest fit — you can change details after."
                }
                title={advanced ? "Template" : "How should it run?"}
              >
                <TemplateCards
                  mode={formMode}
                  onSelect={onTemplateChange}
                  templateId={templateId}
                />
              </NumberedSection>
            ) : null}

            <NumberedSection
              index={isNewMode ? "03" : "02"}
              subtitle="Higher priority gets claimed sooner."
              title="Priority"
            >
              <PrioritySection onPriority={form.updatePriority} priority={draft.priority} />
            </NumberedSection>

            {advanced ? (
              <>
                <NumberedSection
                  index="04"
                  subtitle="Where it sits in the task hierarchy."
                  title="Placement"
                >
                  <PlacementSection
                    onParentTaskId={setField("parentTaskId")}
                    parentTaskId={draft.parentTaskId}
                  />
                </NumberedSection>
                <NumberedSection
                  index="05"
                  subtitle="Who runs it, and how retries behave."
                  title="Queue &amp; ownership"
                >
                  <QueueOwnershipSection
                    approvalPolicy={draft.approvalPolicy}
                    maxAttempts={draft.maxAttempts}
                    onApprovalPolicy={form.updateApprovalPolicy}
                    onMaxAttempts={form.updateMaxAttempts}
                    onOwnerKind={form.updateOwnerKind}
                    onOwnerRef={setField("ownerRef")}
                    ownerKind={draft.ownerKind}
                    ownerRef={draft.ownerRef}
                  />
                </NumberedSection>
                <NumberedSection
                  index="06"
                  subtitle="Optional — for peer ingress and stable references."
                  title="Ingress &amp; identity"
                >
                  <IngressIdentitySection
                    identifier={draft.identifier}
                    networkChannel={draft.networkChannel}
                    onIdentifier={setField("identifier")}
                    onNetworkChannel={setField("networkChannel")}
                  />
                </NumberedSection>
                <ExecutionCollapsible
                  autoEnqueueOnReady={draft.autoEnqueueOnReady}
                  onAutoEnqueue={form.updateAutoEnqueue}
                  onSaveAsDraft={form.updateSaveAsDraft}
                  saveAsDraft={draft.saveAsDraft}
                />
              </>
            ) : null}

            {!isNewMode ? (
              <>
                <NumberedSection
                  index="03"
                  subtitle="Who runs it, and how retries behave."
                  title="Queue &amp; ownership"
                >
                  <QueueOwnershipSection
                    approvalPolicy={draft.approvalPolicy}
                    maxAttempts={draft.maxAttempts}
                    onApprovalPolicy={form.updateApprovalPolicy}
                    onMaxAttempts={form.updateMaxAttempts}
                    onOwnerKind={form.updateOwnerKind}
                    onOwnerRef={setField("ownerRef")}
                    ownerKind={draft.ownerKind}
                    ownerRef={draft.ownerRef}
                  />
                </NumberedSection>
                <NumberedSection index="04" subtitle="Peer ingress channel." title="Channel">
                  <Field>
                    <FieldLabel htmlFor="task-editor-network-input">Network channel</FieldLabel>
                    <Input
                      className="font-mono"
                      data-testid="task-network-input"
                      id="task-editor-network-input"
                      onChange={event => setField("networkChannel")(event.target.value)}
                      placeholder="ingress channel"
                      value={draft.networkChannel}
                    />
                  </Field>
                </NumberedSection>
              </>
            ) : null}
          </div>

          <DialogFooter variant="ruled">
            <div
              className="flex flex-1 items-center gap-2 text-form-hint text-subtle"
              data-testid="task-editor-modal-hint"
            >
              <span>
                {draft.saveAsDraft ? (
                  <>
                    Saved as a <b className="font-medium text-muted">draft</b>; no run is queued
                    until you enqueue it.
                  </>
                ) : (
                  <>
                    The contract is durable; runs descend from this task and respect dependencies.
                  </>
                )}
              </span>
            </div>
            <Button
              data-testid="task-editor-modal-cancel"
              onClick={handleCancel}
              type="button"
              variant="outline"
            >
              Cancel
            </Button>
            <Button
              className="min-w-32"
              data-testid="task-editor-modal-submit"
              disabled={!canSubmit || isSubmitting}
              type="submit"
            >
              {isSubmitting ? (
                <Spinner className="size-3" />
              ) : mode === "new" ? (
                <Check aria-hidden="true" className="size-4" />
              ) : null}
              {submitLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
