import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, within } from "storybook/test";

import {
  AgentCreateDialog,
  createDefaultAgentCreateDraft,
  type AgentCreateDialogDraft,
  type AgentCreateStep,
} from "@/systems/agent";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

const providerOptions = workspaceDetailFixture.providers ?? [];

const validDraft: AgentCreateDialogDraft = {
  ...createDefaultAgentCreateDraft(true),
  name: "release-captain",
  provider: "codex",
  model: "gpt-5.4",
  prompt: "Own release readiness, canary evidence, and rollback guardrails.",
  permissions: "approve-reads",
  tools: ["agh__skill_view"],
  toolsets: ["agh__catalog"],
  denyTools: ["agh__task_*"],
  disabledSkills: ["draft-blog-post"],
};

const meta: Meta<typeof AgentCreateDialog> = {
  title: "systems/agent/AgentCreateDialog",
  component: AgentCreateDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AgentCreateDialogHarness({
  initialDraft,
  initialStep,
  isSubmitting = false,
  submitError = null,
}: {
  initialDraft?: AgentCreateDialogDraft;
  initialStep?: AgentCreateStep;
  isSubmitting?: boolean;
  submitError?: string | null;
}) {
  const [draft, setDraft] = useState(initialDraft ?? createDefaultAgentCreateDraft(true));

  return (
    <AgentCreateDialog
      draft={draft}
      hasActiveWorkspace
      initialStep={initialStep}
      isSubmitting={isSubmitting}
      modelCatalogError={null}
      modelCatalogLoading={false}
      modelOptions={["gpt-5.4", "gpt-5.4-mini", "claude-sonnet-4-6"]}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onSubmit={fn()}
      open
      providerOptions={providerOptions}
      providersError={null}
      providersLoading={false}
      submitError={submitError}
      workspaceName={workspaceDetailFixture.workspace.name}
    />
  );
}

export const Default: Story = {
  render: () => <AgentCreateDialogHarness />,
};

export const ValidationError: Story = {
  render: () => (
    <AgentCreateDialogHarness
      initialDraft={{
        ...createDefaultAgentCreateDraft(true),
        name: "../release",
        categoryPath: "Engineering//Release",
      }}
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("agent-create-name-error")).toHaveTextContent(
      "Agent names cannot be . or .."
    );
    await expect(canvas.getByTestId("agent-create-category-path-error")).toHaveTextContent(
      "Category path cannot contain blank segments."
    );
  },
};

export const Submitting: Story = {
  render: () => (
    <AgentCreateDialogHarness initialDraft={validDraft} initialStep="access" isSubmitting />
  ),
};

export const DuplicateError: Story = {
  render: () => (
    <AgentCreateDialogHarness
      initialDraft={validDraft}
      initialStep="access"
      submitError="agent definition already exists"
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("agent-create-submit-error")).toHaveTextContent(
      "agent definition already exists"
    );
  },
};
