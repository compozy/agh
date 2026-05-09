import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames, storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
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
      name: "chargeback-spike",
      agent_name: storyAgentNames.compliance,
      event: "payments.chargeback.spike",
      filter: { "data.rate": ">=0.018" },
      prompt: "Review the chargeback spike for merchant {{ .Data.merchant_id }}.",
    }
  );

  return (
    <CenteredSurface className="items-start justify-center">
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-(--color-divider) bg-(--color-surface)">
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
  args: {},
  render: () => <AutomationTriggerFormHarness activeWorkspaceId={storyDefaultWorkspaceId} />,
};

export const ValidationState: Story = {
  args: {},
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
