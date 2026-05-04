import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import { storyAgentNames, storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
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
  const activeWorkspaceId = storyDefaultWorkspaceId;
  const [draft, setDraft] = useState<ReturnType<typeof createAutomationJobDraft>>(() => ({
    ...createAutomationJobDraft(activeWorkspaceId),
    name: "launch-command-digest",
    agent_name: storyAgentNames.product,
    prompt:
      "Summarize launch blockers, approvals, and the next cutover milestone for the launch room.",
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
        onChange: nextDraft => setDraft(nextDraft),
        onSubmit,
      }}
    />
  );
}

function AutomationEditorTriggerHarness({ onSubmit = fn() }: { onSubmit?: () => void }) {
  const activeWorkspaceId = storyDefaultWorkspaceId;
  const [draft, setDraft] = useState<ReturnType<typeof createAutomationTriggerDraft>>(() => ({
    ...createAutomationTriggerDraft(activeWorkspaceId),
    name: "support-sla-breach",
    agent_name: storyAgentNames.support,
    event: "support.launch.sla_breach",
    filter: { "data.sla_minutes": ">=4" },
    prompt: "Investigate the launch support lane when SLA exceeds {{ .Data.sla_minutes }} minutes.",
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
        onChange: nextDraft => setDraft(nextDraft),
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
