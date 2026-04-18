import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { createAutomationJobDraft } from "@/systems/automation";
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

function AutomationEditorDialogHarness() {
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
        onSubmit: () => undefined,
      }}
    />
  );
}

export const Default: Story = {
  args: {},
  render: () => <AutomationEditorDialogHarness />,
};
