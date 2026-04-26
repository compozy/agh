import type { UpdateTaskRequest } from "../types";
import type { TaskTemplateId } from "./task-templates";
import { applyTemplateToCreatePayload, getTaskTemplate } from "./task-templates";
import type {
  CreateChildTaskRequest,
  CreateTaskRequest,
  TaskOwnerKind,
  TaskPriority,
  TaskRecord,
  TaskScope,
} from "../types";

export interface TaskEditorDraft {
  title: string;
  description: string;
  scope: TaskScope;
  priority: TaskPriority;
  ownerKind: TaskOwnerKind | "";
  ownerRef: string;
  parentTaskId: string;
  maxAttempts: number | null;
  approvalPolicy: "none" | "manual";
  networkChannel: string;
  identifier: string;
}

export const EMPTY_TASK_EDITOR_DRAFT: TaskEditorDraft = {
  title: "",
  description: "",
  scope: "workspace",
  priority: "medium",
  ownerKind: "",
  ownerRef: "",
  parentTaskId: "",
  maxAttempts: null,
  approvalPolicy: "none",
  networkChannel: "",
  identifier: "",
};

export function createTaskEditorDraft(
  templateId: TaskTemplateId,
  activeWorkspaceId?: string | null
): TaskEditorDraft {
  const template = getTaskTemplate(templateId);

  return {
    ...EMPTY_TASK_EDITOR_DRAFT,
    scope: activeWorkspaceId ? "workspace" : "global",
    priority: template.defaults.priority ?? "medium",
    maxAttempts:
      typeof template.defaults.max_attempts === "number" ? template.defaults.max_attempts : null,
    approvalPolicy: template.defaults.approval_policy ?? "none",
    networkChannel: template.defaults.network_channel ?? "",
  };
}

export function applyTemplateDefaultsToTaskEditorDraft(
  draft: TaskEditorDraft,
  templateId: TaskTemplateId
): TaskEditorDraft {
  const template = getTaskTemplate(templateId);

  return {
    ...draft,
    priority: template.defaults.priority ?? draft.priority,
    maxAttempts:
      typeof template.defaults.max_attempts === "number"
        ? template.defaults.max_attempts
        : draft.maxAttempts,
    approvalPolicy: template.defaults.approval_policy ?? draft.approvalPolicy,
    networkChannel: template.defaults.network_channel ?? draft.networkChannel,
  };
}

export function taskEditorDraftFromTask(task: TaskRecord): TaskEditorDraft {
  return {
    title: task.title,
    description: task.description ?? "",
    scope: task.scope,
    priority: task.priority ?? "medium",
    ownerKind: task.owner?.kind ?? "",
    ownerRef: task.owner?.ref ?? "",
    parentTaskId: task.parent_task_id ?? "",
    maxAttempts: task.max_attempts ?? null,
    approvalPolicy: task.approval_policy === "manual" ? "manual" : "none",
    networkChannel: task.network_channel ?? "",
    identifier: task.identifier ?? "",
  };
}

function resolveOwnerInput(draft: TaskEditorDraft) {
  const ownerKind = draft.ownerKind || undefined;
  const ownerRef = draft.ownerRef.trim();

  return {
    owner:
      ownerKind && ownerRef
        ? { kind: ownerKind, ref: ownerRef }
        : ownerKind === undefined && ownerRef === ""
          ? undefined
          : null,
    ownerIsEmpty: ownerKind === undefined && ownerRef === "",
  };
}

export function buildCreateTaskRequest(
  draft: TaskEditorDraft,
  options: {
    activeWorkspaceId?: string | null;
    templateId: TaskTemplateId;
    asDraft: boolean;
  }
): CreateTaskRequest {
  const { owner } = resolveOwnerInput(draft);

  const basePayload: CreateTaskRequest = {
    title: draft.title.trim(),
    description: draft.description.trim() || undefined,
    scope: draft.scope,
    workspace: draft.scope === "workspace" ? (options.activeWorkspaceId ?? undefined) : undefined,
    priority: draft.priority,
    max_attempts: draft.maxAttempts ?? undefined,
    draft: options.asDraft || options.templateId === "recurring",
    owner,
    approval_policy: draft.approvalPolicy === "manual" ? "manual" : undefined,
    network_channel: draft.networkChannel.trim() || undefined,
    identifier: draft.identifier.trim() || undefined,
  };

  return applyTemplateToCreatePayload(basePayload, options.templateId);
}

export function buildCreateChildTaskRequest(
  draft: TaskEditorDraft,
  options: {
    activeWorkspaceId?: string | null;
    templateId: TaskTemplateId;
    asDraft: boolean;
  }
): CreateChildTaskRequest {
  const { owner } = resolveOwnerInput(draft);

  const basePayload: CreateChildTaskRequest = {
    title: draft.title.trim(),
    description: draft.description.trim() || undefined,
    scope: draft.scope,
    workspace: draft.scope === "workspace" ? (options.activeWorkspaceId ?? undefined) : undefined,
    priority: draft.priority,
    max_attempts: draft.maxAttempts ?? undefined,
    draft: options.asDraft || options.templateId === "recurring",
    owner,
    approval_policy: draft.approvalPolicy === "manual" ? "manual" : undefined,
    network_channel: draft.networkChannel.trim() || undefined,
    identifier: draft.identifier.trim() || undefined,
  };

  return applyTemplateToCreatePayload(basePayload, options.templateId);
}

export function buildUpdateTaskRequest(draft: TaskEditorDraft): UpdateTaskRequest {
  const { owner, ownerIsEmpty } = resolveOwnerInput(draft);

  return {
    title: draft.title.trim() || undefined,
    description: draft.description.trim() || null,
    priority: draft.priority,
    owner: ownerIsEmpty ? undefined : owner,
    clear_owner: ownerIsEmpty ? true : undefined,
    max_attempts: draft.maxAttempts ?? null,
    approval_policy: draft.approvalPolicy === "manual" ? "manual" : "none",
    network_channel: draft.networkChannel.trim() || null,
  };
}
