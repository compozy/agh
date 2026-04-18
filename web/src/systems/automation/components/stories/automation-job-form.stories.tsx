import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

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
      name: "nightly-docs",
      agent_name: "reviewer",
      prompt: "Review open stories and summarize risks.",
    }
  );

  return (
    <CenteredSurface className="items-start justify-center">
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
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
  render: () => <AutomationJobFormHarness activeWorkspaceId="ws_storybook" />,
};

export const ValidationState: Story = {
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
