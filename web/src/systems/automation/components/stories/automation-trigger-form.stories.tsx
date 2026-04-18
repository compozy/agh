import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { createAutomationTriggerDraft } from "@/systems/automation";

import { AutomationTriggerForm } from "../automation-trigger-form";

const meta: Meta<typeof AutomationTriggerForm> = {
  title: "systems/automation/AutomationTriggerForm",
  component: AutomationTriggerForm,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AutomationTriggerFormHarness({
  activeWorkspaceId,
  initialDraft,
  mode = "create",
}: {
  activeWorkspaceId?: string | null;
  initialDraft?: ReturnType<typeof createAutomationTriggerDraft>;
  mode?: "create" | "edit";
}) {
  const [draft, setDraft] = useState(
    initialDraft ?? {
      ...createAutomationTriggerDraft(activeWorkspaceId),
      name: "qa-trigger-browser",
      agent_name: "reviewer",
      event: "ext.github.push",
      filter: { "data.branch": "main" },
      prompt: "Review the pushed branch {{ .Data.branch }}.",
    }
  );

  return (
    <CenteredSurface className="items-start justify-center">
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
        <AutomationTriggerForm
          activeWorkspaceId={activeWorkspaceId}
          draft={draft}
          isPending={false}
          mode={mode}
          onCancel={() => undefined}
          onChange={setDraft}
          onSubmit={() => undefined}
        />
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => <AutomationTriggerFormHarness activeWorkspaceId="ws_storybook" />,
};

export const ValidationState: Story = {
  render: () => (
    <AutomationTriggerFormHarness
      activeWorkspaceId={null}
      initialDraft={{
        ...createAutomationTriggerDraft(null),
        scope: "workspace",
        workspace_id: undefined,
      }}
    />
  ),
};
