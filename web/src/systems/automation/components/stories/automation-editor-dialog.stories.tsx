import { useEffect, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { useAutomationPage } from "@/hooks/routes/use-automation-page";

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
  const openedRef = useRef(false);

  useEffect(() => {
    if (openedRef.current) {
      return;
    }

    openedRef.current = true;
    page.handleCreate();
  }, [page]);

  return <AutomationEditorDialog {...page.editorDialogProps} />;
}

export const Default: Story = {
  render: () => <AutomationEditorDialogHarness />,
};
