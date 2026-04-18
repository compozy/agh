import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { useAutomationPage } from "@/hooks/routes/use-automation-page";
import { createAutomationJobDraft } from "@/systems/automation";

import { AutomationEditorDialog } from "../automation-editor-dialog";

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
  const page = useAutomationPage();
  const activeWorkspaceId = page.editorDialogProps.activeWorkspaceId;
  const [draft, setDraft] = useState(() => createAutomationJobDraft(activeWorkspaceId));

  return (
    <AutomationEditorDialog
      {...page.editorDialogProps}
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
