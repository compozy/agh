import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { StorySurface } from "@/storybook/story-layout";
import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import { EMPTY_TASK_EDITOR_DRAFT } from "../../lib/task-editor";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  getTaskTemplate,
  type TaskTemplateId,
} from "../../lib/task-templates";
import { TasksCreateModal } from "../tasks-create-modal";

const meta: Meta<typeof TasksCreateModal> = {
  title: "systems/tasks/TasksCreateModal",
  component: TasksCreateModal,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ControlledModal(props: {
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
    <StorySurface className="min-h-[760px]">
      <TasksCreateModal
        canSubmit={draft.title.trim().length > 0}
        draft={draft}
        isSubmitting={props.isSubmitting ?? false}
        onDraftChange={setDraft}
        onOpenChange={() => undefined}
        onSubmit={() => undefined}
        onTemplateChange={setTemplateId}
        open
        template={template}
        templateId={templateId}
        workspaceName="Polybot"
      />
    </StorySurface>
  );
}

export const Empty: Story = {
  render: () => <ControlledModal />,
};

export const Populated: Story = {
  render: () => (
    <ControlledModal
      initialDraft={{
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Generate API client for payments-v3",
        description:
          "Generate a typed TypeScript client from the OpenAPI spec and wire it into @/integrations/payments.",
        priority: "high",
        ownerKind: "agent_session",
        ownerRef: "Coder",
        maxAttempts: 3,
      }}
    />
  ),
};

export const Submitting: Story = {
  name: "Pending",
  render: () => (
    <ControlledModal
      initialDraft={{
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Generate API client for payments-v3",
      }}
      isSubmitting
    />
  ),
};

export const ValidationError: Story = {
  render: () => <ControlledModal initialDraft={{ ...EMPTY_TASK_EDITOR_DRAFT, title: "" }} />,
};

export const RecurringTemplate: Story = {
  name: "Error",
  render: () => (
    <ControlledModal
      initialDraft={{
        ...EMPTY_TASK_EDITOR_DRAFT,
        title: "Daily digest",
        description: "Summarize activity once per day.",
      }}
      initialTemplate="recurring"
    />
  ),
};
