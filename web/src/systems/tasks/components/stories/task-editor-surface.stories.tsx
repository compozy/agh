import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { storyAgentNames, storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import { EMPTY_TASK_EDITOR_DRAFT, taskEditorDraftFromTask } from "../../lib/task-editor";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  getTaskTemplate,
  type TaskTemplateId,
} from "../../lib/task-templates";
import type { TaskRecord } from "../../types";
import { TaskEditorSurface } from "../task-editor-surface";

const meta: Meta<typeof TaskEditorSurface> = {
  title: "systems/tasks/TaskEditorSurface",
  component: TaskEditorSurface,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function CreateEditor(props: {
  initialDraft?: CreateTaskDraftInput;
  initialTemplate?: TaskTemplateId;
  isSubmitting?: boolean;
}) {
  const [draft, setDraft] = useState<CreateTaskDraftInput>(
    props.initialDraft ?? EMPTY_TASK_EDITOR_DRAFT
  );
  const [templateId, setTemplateId] = useState<TaskTemplateId>(
    props.initialTemplate ?? DEFAULT_TASK_TEMPLATE_ID
  );
  const template = getTaskTemplate(templateId);
  return (
    <PanelSurface className="min-h-[820px] p-0">
      <TaskEditorSurface
        canSubmit={draft.title.trim().length > 0}
        draft={draft}
        isSubmitting={props.isSubmitting ?? false}
        mode="create"
        onDraftChange={setDraft}
        onSubmit={() => undefined}
        onTemplateChange={setTemplateId}
        task={null}
        template={template}
        templateId={templateId}
        workspaceName={storyDefaultWorkspaceName}
      />
    </PanelSurface>
  );
}

const EDIT_TASK: TaskRecord = {
  id: "task_abc",
  identifier: "TASK-42",
  title: "Summarize review feedback",
  status: "ready",
  scope: "workspace",
  priority: "high",
  origin: { kind: "web", ref: "op" },
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:00:00Z",
  owner: { kind: "agent_session", ref: storyAgentNames.fraud },
  description: "Review reserve reasons, confirm settlement timing, and draft the operator summary.",
} as unknown as TaskRecord;

function EditEditor(props: { isSubmitting?: boolean }) {
  const [draft, setDraft] = useState<CreateTaskDraftInput>(taskEditorDraftFromTask(EDIT_TASK));
  return (
    <PanelSurface className="min-h-[820px] p-0">
      <TaskEditorSurface
        canSubmit={draft.title.trim().length > 0}
        draft={draft}
        isSubmitting={props.isSubmitting ?? false}
        mode="edit"
        onDraftChange={setDraft}
        onSubmit={() => undefined}
        task={EDIT_TASK}
        workspaceName={storyDefaultWorkspaceName}
      />
    </PanelSurface>
  );
}

export const CreateEmpty: Story = {
  name: "Empty",
  render: () => <CreateEditor />,
};

export const CreatePopulated: Story = {
  name: "Populated",
  render: () => (
    <CreateEditor
      initialDraft={{
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Prepare VIP merchant callback for delayed settlement",
        description:
          "Draft the callback notes, reserve rationale, and next checkpoint for support.",
        priority: "high",
        ownerKind: "agent_session",
        ownerRef: storyAgentNames.support,
        maxAttempts: 3,
      }}
    />
  ),
};

export const Submitting: Story = {
  name: "Pending",
  render: () => (
    <CreateEditor
      initialDraft={{
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Prepare VIP merchant callback for delayed settlement",
      }}
      isSubmitting
    />
  ),
};

export const ValidationError: Story = {
  render: () => <CreateEditor />,
};

export const EditMode: Story = {
  name: "Error",
  render: () => <EditEditor />,
};
