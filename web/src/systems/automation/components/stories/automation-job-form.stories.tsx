import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames, storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";
import { createAutomationJobDraft } from "@/systems/automation";

import { AutomationJobForm } from "../automation-job-form";

const meta: Meta<typeof AutomationJobForm> = {
  title: "systems/automation/AutomationJobForm",
  component: AutomationJobForm,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AutomationJobFormHarness({
  activeWorkspaceId,
  initialDraft,
  mode = "create",
}: {
  activeWorkspaceId?: string | null;
  initialDraft?: ReturnType<typeof createAutomationJobDraft>;
  mode?: "create" | "edit";
}) {
  const [draft, setDraft] = useState(
    initialDraft ?? {
      ...createAutomationJobDraft(activeWorkspaceId),
      name: "payout-watchlist",
      agent_name: storyAgentNames.fraud,
      prompt: "Review payout holds above the reserve threshold and draft the operator summary.",
    }
  );

  return (
    <CenteredSurface className="items-start justify-center">
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-(--line) bg-(--canvas-soft)">
        <AutomationJobForm
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
  render: () => <AutomationJobFormHarness activeWorkspaceId={storyDefaultWorkspaceId} />,
};

export const ValidationState: Story = {
  args: {},
  render: () => (
    <AutomationJobFormHarness
      activeWorkspaceId={null}
      initialDraft={{
        ...createAutomationJobDraft(null),
        scope: "workspace",
        workspace_id: undefined,
      }}
    />
  ),
};
