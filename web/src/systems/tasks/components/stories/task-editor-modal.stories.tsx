import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { TaskEditorModal } from "../task-editor-modal";
import {
  EMPTY_TASK_EDITOR_DRAFT,
  createTaskEditorDraft,
  type TaskEditorDraft,
} from "../../lib/task-editor";
import { getTaskTemplate, type TaskTemplateId } from "../../lib/task-templates";
import type { TaskRecord } from "../../types";

const meta: Meta<typeof TaskEditorModal> = {
  title: "systems/tasks/TaskEditorModal",
  parameters: {
    layout: "fullscreen",
  },
  component: TaskEditorModal,
};

export default meta;
type Story = StoryObj<typeof TaskEditorModal>;

function NewModeStory({ templateId = "one_shot" }: { templateId?: TaskTemplateId }) {
  const [draft, setDraft] = useState<TaskEditorDraft>(() =>
    createTaskEditorDraft(templateId, "ws_alpha")
  );
  const [activeTemplate, setActiveTemplate] = useState<TaskTemplateId>(templateId);
  return (
    <div className="min-h-screen bg-(--canvas) p-6">
      <TaskEditorModal
        canSubmit={draft.title.trim().length > 0}
        draft={draft}
        mode="new"
        onDraftChange={setDraft}
        onOpenChange={() => undefined}
        onSubmit={() => Promise.resolve()}
        onTemplateChange={next => {
          setActiveTemplate(next);
          setDraft(createTaskEditorDraft(next, "ws_alpha"));
        }}
        open
        task={null}
        template={getTaskTemplate(activeTemplate)}
        templateId={activeTemplate}
        workspaceName="Alpha"
      />
    </div>
  );
}

const editTask = {
  id: "task_42",
  identifier: "TASK-42",
  title: "Summarize review feedback",
  status: "in_progress",
  scope: "workspace",
  origin: { kind: "cli", ref: "op" },
  workspace_id: "ws_alpha",
  created_at: "2026-04-11T09:00:00Z",
  updated_at: "2026-04-11T09:30:00Z",
  created_by: { kind: "human", ref: "pedro@" },
  priority: "medium",
  description: "Compress the long thread into 5 bullets the team can act on by Friday.",
  max_attempts: 3,
} as unknown as TaskRecord;

function EditModeStory() {
  const [draft, setDraft] = useState<TaskEditorDraft>(() => ({
    ...EMPTY_TASK_EDITOR_DRAFT,
    title: editTask.title,
    description: "Compress the long thread into 5 bullets the team can act on by Friday.",
    priority: "medium",
    maxAttempts: 3,
  }));
  return (
    <div className="min-h-screen bg-(--canvas) p-6">
      <TaskEditorModal
        canSubmit={draft.title.trim().length > 0}
        draft={draft}
        mode="edit"
        onDraftChange={setDraft}
        onOpenChange={() => undefined}
        onSubmit={() => Promise.resolve()}
        open
        task={editTask}
        workspaceName="Alpha"
      />
    </div>
  );
}

export const NewOneShot: Story = {
  name: "New · one-shot (Enqueue)",
  render: () => <NewModeStory templateId="one_shot" />,
};

export const NewRecurring: Story = {
  name: "New · recurring (Save draft)",
  render: () => <NewModeStory templateId="recurring" />,
};

export const Edit: Story = {
  name: "Edit · existing task",
  render: () => <EditModeStory />,
};
