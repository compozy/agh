import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { createAutomationJobDraft, createAutomationTriggerDraft } from "@/systems/automation";
import { AutomationEditorDialog } from "@/systems/automation/components/automation-editor-dialog";

const meta: Meta<typeof AutomationEditorDialog> = {
  title: "systems/automation/AutomationEditorDialog",
  component: AutomationEditorDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AutomationEditorJobHarness({ onSubmit = fn() }: { onSubmit?: () => void }) {
  const activeWorkspaceId = "ws_storybook";
  const [draft, setDraft] = useState(() => ({
    ...createAutomationJobDraft(activeWorkspaceId),
    name: "nightly-docs",
    agent_name: "reviewer",
    prompt: "Review open stories and summarize risks.",
  }));

  return (
    <AutomationEditorDialog
      activeWorkspaceId={activeWorkspaceId}
      editor={{
        draft,
        isPending: false,
        kind: "jobs",
        mode: "create",
        onCancel: () => undefined,
        onChange: setDraft,
        onSubmit,
      }}
    />
  );
}

function AutomationEditorTriggerHarness({ onSubmit = fn() }: { onSubmit?: () => void }) {
  const activeWorkspaceId = "ws_storybook";
  const [draft, setDraft] = useState(() => ({
    ...createAutomationTriggerDraft(activeWorkspaceId),
    name: "push-review",
    agent_name: "reviewer",
    event: "ext.github.push",
    prompt: "Review push event {{ .Data.branch }}.",
  }));

  return (
    <AutomationEditorDialog
      activeWorkspaceId={activeWorkspaceId}
      editor={{
        draft,
        isPending: false,
        kind: "triggers",
        mode: "edit",
        onCancel: () => undefined,
        onChange: setDraft,
        onSubmit,
      }}
    />
  );
}

export const CreateJob: Story = {
  args: {},
  render: () => <AutomationEditorJobHarness />,
};

export const EditTrigger: Story = {
  args: {},
  render: () => <AutomationEditorTriggerHarness />,
};

export const CreateJobSubmit: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <AutomationEditorJobHarness onSubmit={fn()} />,
  play: async ({ canvasElement }) => {
    const canvas = within(document.body);
    const form = await canvas.findByTestId("automation-job-form");
    void canvasElement;
    const submit = await within(form).findByTestId("submit-job-form");
    await expect(submit).toBeEnabled();
    await userEvent.click(submit);
  },
};
