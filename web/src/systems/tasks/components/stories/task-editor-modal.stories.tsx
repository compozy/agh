import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyWorkspaceIds, storyWorkspaceNames } from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";

import { TaskEditorModal, type TaskEditorModalMode } from "../task-editor-modal";
import { createTaskEditorDraft, type TaskEditorDraft } from "../../lib/task-editor";
import {
  DEFAULT_TASK_TEMPLATE_ID,
  getTaskTemplate,
  type TaskTemplateId,
} from "../../lib/task-templates";
import type { TaskRecord } from "../../types";

const meta: Meta<typeof TaskEditorModal> = {
  title: "systems/tasks/TaskEditorModal",
  component: TaskEditorModal,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const storyWorkspaces = [
  { id: storyWorkspaceIds.hq, name: storyWorkspaceNames.hq },
  { id: storyWorkspaceIds.risk, name: storyWorkspaceNames.risk },
  { id: storyWorkspaceIds.growth, name: storyWorkspaceNames.growth },
];

interface TaskEditorModalHarnessProps {
  mode?: TaskEditorModalMode;
  templateId?: TaskTemplateId;
  initialDraft?: TaskEditorDraft;
  isSubmitting?: boolean;
  task?: TaskRecord | null;
}

/**
 * Controlled Storybook harness — holds the editor draft and the active template
 * id in local state, mirroring the live `useTasksPage` orchestration. The modal
 * stays open and renders on the centered story surface; submit / open-change are
 * no-ops so the surface is stable for visual capture.
 */
function TaskEditorModalHarness({
  mode = "new",
  templateId = DEFAULT_TASK_TEMPLATE_ID,
  initialDraft,
  isSubmitting = false,
  task = null,
}: TaskEditorModalHarnessProps) {
  const [activeTemplate, setActiveTemplate] = useState<TaskTemplateId>(templateId);
  const [draft, setDraft] = useState<TaskEditorDraft>(
    () => initialDraft ?? createTaskEditorDraft(templateId, storyWorkspaceIds.hq)
  );

  const isNewMode = mode === "new";
  const canSubmit = draft.title.trim().length > 0;

  return (
    <CenteredSurface className="items-start justify-center p-0">
      <TaskEditorModal
        canSubmit={canSubmit}
        draft={draft}
        isSubmitting={isSubmitting}
        mode={mode}
        onDraftChange={setDraft}
        onOpenChange={() => undefined}
        onSubmit={() => Promise.resolve()}
        onTemplateChange={
          isNewMode
            ? next => {
                setActiveTemplate(next);
                setDraft(createTaskEditorDraft(next, storyWorkspaceIds.hq));
              }
            : undefined
        }
        open
        task={task}
        template={isNewMode ? getTaskTemplate(activeTemplate) : undefined}
        templateId={isNewMode ? activeTemplate : undefined}
        workspaces={storyWorkspaces}
      />
    </CenteredSurface>
  );
}

const editTask = {
  id: "task_42",
  identifier: "TASK-42",
  title: "Summarize launch-room escalations",
  status: "in_progress",
  scope: "workspace",
  origin: { kind: "cli", ref: "op" },
  workspace_id: storyWorkspaceIds.hq,
  created_at: "2026-04-17T09:00:00Z",
  updated_at: "2026-04-17T09:30:00Z",
  created_by: { kind: "human", ref: "pedro" },
  priority: "high",
  description: "Compress the escalation thread into five bullets the launch room can act on.",
  max_attempts: 3,
} as unknown as TaskRecord;

function buildEditDraft(): TaskEditorDraft {
  return {
    ...createTaskEditorDraft("blank", storyWorkspaceIds.hq),
    title: "Summarize launch-room escalations",
    description: "Compress the escalation thread into five bullets the launch room can act on.",
    priority: "high",
    maxAttempts: 3,
    networkChannel: "launch-room",
  };
}

/**
 * Default create surface — Simple mode with the three approachable template
 * cards. Click the Advanced pill (`task-mode-advanced`) to reveal the numbered
 * Placement / Queue / Ingress sections and the Execution collapsible.
 */
export const NewSimple: Story = {
  render: () => <TaskEditorModalHarness />,
};

/**
 * Advanced create surface — drives the internal Simple/Advanced toggle through a
 * play function so the numbered advanced sections render on load.
 */
export const NewAdvanced: Story = {
  render: () => <TaskEditorModalHarness />,
  play: async ({ canvasElement }) => {
    const advancedPill = canvasElement.querySelector<HTMLButtonElement>(
      "[data-testid='task-mode-advanced']"
    );
    advancedPill?.click();
  },
};

/**
 * Human-in-the-loop template — manual approval is preset; the Queue & ownership
 * approval hint flips to the approval gate in Advanced mode.
 */
export const HumanInLoopTemplate: Story = {
  render: () => <TaskEditorModalHarness templateId="human_in_loop" />,
};

/**
 * Recurring template — saves as a draft so Automation can attach the schedule
 * later; the submit label reads `Save draft`.
 */
export const RecurringDraft: Story = {
  render: () => <TaskEditorModalHarness templateId="recurring" />,
};

/** Edit an existing task — no mode toolbar, no template grid, channel input present. */
export const EditMode: Story = {
  render: () => (
    <TaskEditorModalHarness mode="edit" initialDraft={buildEditDraft()} task={editTask} />
  ),
};

/** Validation disabled — empty title keeps the submit action disabled. */
export const ValidationDisabled: Story = {
  render: () => (
    <TaskEditorModalHarness
      initialDraft={createTaskEditorDraft(DEFAULT_TASK_TEMPLATE_ID, storyWorkspaceIds.hq)}
    />
  ),
};
